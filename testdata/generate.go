//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

func cc608(cc1, cc2 byte) []byte { return []byte{0xFC, cc1, cc2} }

func cc608Ctrl(cc1, cc2 byte) []byte { return cc608(cc1, cc2) }

func cc608Text(c1, c2 byte) []byte { return cc608(c1, c2) }

func cc608SingleChar(c byte) []byte { return cc608(c, 0x80) }

func dtvccStart(b1, b2 byte) []byte { return []byte{0xFF, b1, b2} }

func dtvccCont(b1, b2 byte) []byte { return []byte{0xFE, b1, b2} }

func buildDTVCCPacket(serviceData []byte) []byte {
	blockSize := len(serviceData)
	serviceHeader := byte((1 << 5) | (blockSize & 0x1F))
	totalPayload := 1 + 1 + blockSize // header + service_header + data
	sizeCode := (totalPayload + 1) / 2
	if sizeCode == 0 {
		sizeCode = 1
	}
	header := byte(sizeCode & 0x3F)
	packet := []byte{header, serviceHeader}
	packet = append(packet, serviceData...)
	totalNeeded := sizeCode * 2
	for len(packet) < totalNeeded {
		packet = append(packet, 0x00)
	}
	return packet
}

func dtvccTriplets(serviceData []byte) [][]byte {
	packet := buildDTVCCPacket(serviceData)
	var triplets [][]byte
	for i := 0; i < len(packet); i += 2 {
		b1 := packet[i]
		b2 := byte(0x00)
		if i+1 < len(packet) {
			b2 = packet[i+1]
		}
		if i == 0 {
			triplets = append(triplets, dtvccStart(b1, b2))
		} else {
			triplets = append(triplets, dtvccCont(b1, b2))
		}
	}
	return triplets
}

type scenario struct {
	name   string
	frames [][]byte
}

func buildA53CaptionPayload(triplets []byte, ccCount int) []byte {
	payload := []byte{
		0xB5, 0x00, 0x31,
		'G', 'A', '9', '4',
		0x03,
		0x40 | byte(ccCount&0x1F),
		0xFF,
	}
	payload = append(payload, triplets...)
	return payload
}

func buildSEIMessage(payloadType int, payload []byte) []byte {
	var msg []byte
	pt := payloadType
	for pt >= 255 {
		msg = append(msg, 0xFF)
		pt -= 255
	}
	msg = append(msg, byte(pt))
	ps := len(payload)
	for ps >= 255 {
		msg = append(msg, 0xFF)
		ps -= 255
	}
	msg = append(msg, byte(ps))
	msg = append(msg, payload...)
	return msg
}

func addEPB(data []byte) []byte {
	var out []byte
	zeroCount := 0
	for _, b := range data {
		if zeroCount >= 2 && b <= 0x03 {
			out = append(out, 0x03)
			zeroCount = 0
		}
		out = append(out, b)
		if b == 0x00 {
			zeroCount++
		} else {
			zeroCount = 0
		}
	}
	return out
}

func writeAnnexBNAL(f *os.File, nal []byte) {
	f.Write([]byte{0x00, 0x00, 0x00, 0x01})
	f.Write(nal)
}

func generateBitstream(name, codec string, frames [][]byte) {
	ext := "." + codec
	filename := codec + "_" + name + ext
	f, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create %s: %v\n", filename, err)
		return
	}
	defer f.Close()

	if codec == "h264" {
		sps := []byte{0x67, 0x42, 0xC0, 0x1E, 0xD9, 0x00, 0xA0, 0x47, 0xFE, 0xC8}
		pps := []byte{0x68, 0xCE, 0x38, 0x80}
		writeAnnexBNAL(f, sps)
		writeAnnexBNAL(f, pps)
	} else {
		vps := []byte{0x40, 0x01, 0x0C, 0x01, 0xFF, 0xFF, 0x01, 0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x96, 0x09, 0x00}
		sps := []byte{0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x96, 0xA0, 0x01, 0x40, 0x20, 0x06, 0x11, 0x48, 0xFC, 0xBC}
		pps := []byte{0x44, 0x01, 0xC1, 0x73, 0x18, 0x31}
		writeAnnexBNAL(f, vps)
		writeAnnexBNAL(f, sps)
		writeAnnexBNAL(f, pps)
	}

	for _, frame := range frames {
		ccCount := len(frame) / 3
		a53 := buildA53CaptionPayload(frame, ccCount)
		seiMsg := buildSEIMessage(4, a53)
		seiBody := addEPB(seiMsg)

		var seiNAL []byte
		if codec == "h264" {
			seiNAL = append([]byte{0x06}, seiBody...)
		} else {
			seiNAL = append([]byte{39 << 1, 0x01}, seiBody...)
		}
		writeAnnexBNAL(f, seiNAL)

		if codec == "h264" {
			idr := make([]byte, 33)
			idr[0] = 0x65
			writeAnnexBNAL(f, idr)
		} else {
			idr := make([]byte, 34)
			idr[0] = 19 << 1
			idr[1] = 0x01
			binary.BigEndian.PutUint32(idr[2:], 0x80000000)
			writeAnnexBNAL(f, idr)
		}
	}
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func gen608RollUp2() scenario {
	return scenario{
		name: "608_rollup2",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25), // RU2
			cc608Ctrl(0x14, 0x25), // RU2 dedup
			cc608Text('L', '1'),
			cc608Ctrl(0x14, 0x2D), // CR
			cc608Ctrl(0x14, 0x2D), // CR dedup
			cc608Text('L', '2'),
		},
	}
}

func gen608RollUp3() scenario {
	return scenario{
		name: "608_rollup3",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x26), // RU3
			cc608Ctrl(0x14, 0x26), // RU3 dedup
			cc608Text('A', 'A'),
			cc608Ctrl(0x14, 0x2D), // CR
			cc608Ctrl(0x14, 0x2D),
			cc608Text('B', 'B'),
			cc608Ctrl(0x14, 0x2D), // CR
			cc608Ctrl(0x14, 0x2D),
			cc608Text('C', 'C'),
		},
	}
}

func gen608RollUp4() scenario {
	return scenario{
		name: "608_rollup4",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x27), // RU4
			cc608Ctrl(0x14, 0x27),
			cc608Text('R', '1'),
			cc608Ctrl(0x14, 0x2D),
			cc608Ctrl(0x14, 0x2D),
			cc608Text('R', '2'),
			cc608Ctrl(0x14, 0x2D),
			cc608Ctrl(0x14, 0x2D),
			cc608Text('R', '3'),
			cc608Ctrl(0x14, 0x2D),
			cc608Ctrl(0x14, 0x2D),
			cc608Text('R', '4'),
		},
	}
}

func gen608PopOn() scenario {
	return scenario{
		name: "608_popon",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x20),
			cc608Ctrl(0x14, 0x20),
			cc608Ctrl(0x15, 0x70), // PAC row 3 indent 0
			cc608Text('T', 'o'),
			cc608Text('p', 0x80),
			cc608Ctrl(0x17, 0x70), // PAC row 9 indent 0
			cc608Text('B', 'o'),
			cc608Text('t', 0x80),
			cc608Ctrl(0x14, 0x2F), // EOC
			cc608Ctrl(0x14, 0x2F),
		},
	}
}

func gen608PaintOn() scenario {
	return scenario{
		name: "608_painton",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x29), // RDC
			cc608Ctrl(0x14, 0x29),
			cc608Text('O', 'n'),
			cc608Text('e', 0x80),
			cc608Ctrl(0x14, 0x2D), // CR
			cc608Ctrl(0x14, 0x2D),
			cc608Text('T', 'w'),
			cc608Text('o', 0x80),
		},
	}
}

func gen608ModeSwitch() scenario {
	return scenario{
		name: "608_mode_switch",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x20), // RCL (pop-on)
			cc608Ctrl(0x14, 0x20),
			cc608Text('B', 'u'),
			cc608Text('f', 0x80),
			cc608Ctrl(0x14, 0x2F), // EOC
			cc608Ctrl(0x14, 0x2F),
			cc608Ctrl(0x14, 0x25), // RU2 (switches mode, clears display)
			cc608Ctrl(0x14, 0x25),
			cc608Text('N', 'w'),
		},
	}
}

func gen608SpecialChars() scenario {
	var frames [][]byte
	frames = append(frames, cc608Ctrl(0x14, 0x25)) // RU2
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	for b := byte(0x30); b <= 0x3F; b++ {
		frames = append(frames, cc608SingleChar('X'))
		frames = append(frames, cc608Ctrl(0x11, b))
		frames = append(frames, cc608Ctrl(0x11, b))
	}
	return scenario{name: "608_special_chars", frames: frames}
}

func gen608ExtendedSpanish() scenario {
	var frames [][]byte
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	for b := byte(0x20); b <= 0x3F; b++ {
		frames = append(frames, cc608SingleChar('X'))
		frames = append(frames, cc608Ctrl(0x12, b))
		frames = append(frames, cc608Ctrl(0x12, b))
	}
	return scenario{name: "608_extended_spanish", frames: frames}
}

func gen608ExtendedPortuguese() scenario {
	var frames [][]byte
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	for b := byte(0x20); b <= 0x3F; b++ {
		frames = append(frames, cc608SingleChar('X'))
		frames = append(frames, cc608Ctrl(0x13, b))
		frames = append(frames, cc608Ctrl(0x13, b))
	}
	return scenario{name: "608_extended_portuguese", frames: frames}
}

func gen608G0Overrides() scenario {
	return scenario{
		name: "608_g0_overrides",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25),
			cc608Ctrl(0x14, 0x25),
			cc608SingleChar(0x27), // right single quote
			cc608SingleChar(0x2A), // á
			cc608SingleChar(0x5C), // é
			cc608SingleChar(0x5E), // í
			cc608SingleChar(0x5F), // ó
			cc608SingleChar(0x60), // ú
			cc608SingleChar(0x7B), // ç
			cc608SingleChar(0x7C), // ÷
			cc608SingleChar(0x7D), // Ñ
			cc608SingleChar(0x7E), // ñ
			cc608SingleChar(0x7F), // full block
		},
	}
}

func gen608Backspace() scenario {
	return scenario{
		name: "608_backspace",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25), // RU2
			cc608Ctrl(0x14, 0x25),
			cc608Text('A', 'B'),
			cc608Text('C', 0x80),
			cc608Ctrl(0x14, 0x21), // BS
			cc608Ctrl(0x14, 0x21),
			cc608Text('X', 0x80),
		},
	}
}

func gen608DeleteToEnd() scenario {
	return scenario{
		name: "608_delete_to_end",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x29), // RDC (paint-on)
			cc608Ctrl(0x14, 0x29),
			cc608Text('A', 'B'),
			cc608Text('C', 'D'),
			cc608Text('E', 'F'),
			cc608Ctrl(0x14, 0x21), // BS -> col 5
			cc608Ctrl(0x14, 0x21),
			cc608Ctrl(0x14, 0x21), // BS -> col 4
			cc608Ctrl(0x14, 0x21),
			cc608Ctrl(0x14, 0x24), // DER from col 4
			cc608Ctrl(0x14, 0x24),
		},
	}
}

func gen608TabOffsets() scenario {
	return scenario{
		name: "608_tab_offsets",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25), // RU2
			cc608Ctrl(0x14, 0x25),
			cc608SingleChar('A'),
			cc608Ctrl(0x17, 0x21), // TO1
			cc608Ctrl(0x17, 0x21),
			cc608SingleChar('B'),
			cc608Ctrl(0x17, 0x22), // TO2
			cc608Ctrl(0x17, 0x22),
			cc608SingleChar('C'),
			cc608Ctrl(0x17, 0x23), // TO3
			cc608Ctrl(0x17, 0x23),
			cc608SingleChar('D'),
		},
	}
}

func gen608EraseDisplayed() scenario {
	return scenario{
		name: "608_erase_displayed",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25), // RU2
			cc608Ctrl(0x14, 0x25),
			cc608Text('H', 'i'),
			cc608Ctrl(0x14, 0x2C), // EDM
			cc608Ctrl(0x14, 0x2C),
		},
	}
}

func gen608EraseNonDisplayed() scenario {
	return scenario{
		name: "608_erase_nondisplayed",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x20), // RCL (pop-on)
			cc608Ctrl(0x14, 0x20),
			cc608Text('B', 'u'),
			cc608Text('f', 0x80),
			cc608Ctrl(0x14, 0x2E), // ENM
			cc608Ctrl(0x14, 0x2E),
			cc608Ctrl(0x14, 0x2F), // EOC
			cc608Ctrl(0x14, 0x2F),
		},
	}
}

func gen608PACColors() scenario {
	type pacEntry struct {
		cc1, cc2 byte
	}
	rows := []pacEntry{
		{0x11, 0x40}, // row 0, white
		{0x11, 0x62}, // row 1, green
		{0x12, 0x44}, // row 2, blue
		{0x12, 0x66}, // row 3, cyan
		{0x15, 0x48}, // row 4, red
		{0x15, 0x6A}, // row 5, yellow
		{0x16, 0x4C}, // row 6, magenta
	}
	var frames [][]byte
	frames = append(frames, cc608Ctrl(0x14, 0x29)) // RDC (paint-on)
	frames = append(frames, cc608Ctrl(0x14, 0x29))
	for _, r := range rows {
		frames = append(frames, cc608Ctrl(r.cc1, r.cc2))
		frames = append(frames, cc608Ctrl(r.cc1, r.cc2))
		frames = append(frames, cc608SingleChar('X'))
	}
	return scenario{name: "608_pac_colors", frames: frames}
}

func gen608PACIndent() scenario {
	return scenario{
		name: "608_pac_indent",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x29), // RDC (paint-on)
			cc608Ctrl(0x14, 0x29),
			cc608Ctrl(0x11, 0x52), // PAC row 0, indent 4
			cc608Ctrl(0x11, 0x52),
			cc608Text('I', '4'),
			cc608Ctrl(0x11, 0x76), // PAC row 1, indent 12
			cc608Ctrl(0x11, 0x76),
			cc608Text('I', 'C'),
		},
	}
}

func gen608PACUnderline() scenario {
	return scenario{
		name: "608_pac_underline",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25), // RU2
			cc608Ctrl(0x14, 0x25),
			cc608Ctrl(0x11, 0x41), // PAC row 0, white, underline
			cc608Ctrl(0x11, 0x41),
			cc608Text('U', 'L'),
		},
	}
}

func gen608PACItalic() scenario {
	return scenario{
		name: "608_pac_italic",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25),
			cc608Ctrl(0x14, 0x25),
			cc608Ctrl(0x11, 0x4E), // PAC row 0, italic
			cc608Ctrl(0x11, 0x4E),
			cc608Text('I', 'T'),
		},
	}
}

func gen608MidrowColors() scenario {
	var frames [][]byte
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	frames = append(frames, cc608Text('W', 0x80))
	midrowCodes := []byte{0x22, 0x24, 0x26, 0x28, 0x2A, 0x2C}
	for _, mc := range midrowCodes {
		frames = append(frames, cc608Ctrl(0x11, mc))
		frames = append(frames, cc608Ctrl(0x11, mc))
		frames = append(frames, cc608SingleChar('X'))
	}
	return scenario{name: "608_midrow_colors", frames: frames}
}

func gen608MidrowItalicUnderline() scenario {
	return scenario{
		name: "608_midrow_italic_underline",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25), // RU2
			cc608Ctrl(0x14, 0x25),
			cc608SingleChar('N'),
			cc608Ctrl(0x11, 0x2E), // midrow italic
			cc608Ctrl(0x11, 0x2E),
			cc608SingleChar('I'),
			cc608Ctrl(0x11, 0x21), // midrow white underline
			cc608Ctrl(0x11, 0x21),
			cc608SingleChar('U'),
			cc608Ctrl(0x11, 0x2F), // midrow italic+underline
			cc608Ctrl(0x11, 0x2F),
			cc608SingleChar('B'),
		},
	}
}

func gen608BackgroundColors() scenario {
	bgCmds := [][2]byte{
		{0x10, 0x20}, // bg black
		{0x10, 0x22}, // bg green
		{0x10, 0x24}, // bg blue
		{0x10, 0x26}, // bg cyan
		{0x10, 0x28}, // bg red
		{0x10, 0x2A}, // bg yellow
		{0x10, 0x2C}, // bg magenta
		{0x10, 0x2E}, // bg white
	}
	var frames [][]byte
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	for _, bg := range bgCmds {
		frames = append(frames, cc608Ctrl(bg[0], bg[1]))
		frames = append(frames, cc608Ctrl(bg[0], bg[1]))
		frames = append(frames, cc608SingleChar('X'))
	}
	return scenario{name: "608_background_colors", frames: frames}
}

func gen608Flash() scenario {
	return scenario{
		name: "608_flash",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25),
			cc608Ctrl(0x14, 0x25),
			cc608SingleChar('N'),
			cc608Ctrl(0x14, 0x28), // FON
			cc608Ctrl(0x14, 0x28),
			cc608SingleChar('F'),
		},
	}
}

func gen608PACRowPositioning() scenario {
	return scenario{
		name: "608_pac_row_positioning",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x29), // RDC (paint-on)
			cc608Ctrl(0x14, 0x29),
			cc608Ctrl(0x11, 0x40), // PAC row 0
			cc608Ctrl(0x11, 0x40),
			cc608Text('R', '0'),
			cc608Ctrl(0x14, 0x60), // PAC row 3
			cc608Ctrl(0x14, 0x60),
			cc608Text('R', '3'),
			cc608Ctrl(0x16, 0x40), // PAC row 7
			cc608Ctrl(0x16, 0x40),
			cc608Text('R', '7'),
			cc608Ctrl(0x17, 0x60), // PAC row 14
			cc608Ctrl(0x17, 0x60),
			cc608Text('R', 'E'),
		},
	}
}

func gen608RollUpPACRelocate() scenario {
	return scenario{
		name: "608_rollup_pac_relocate",
		frames: [][]byte{
			cc608Ctrl(0x14, 0x25), // RU2
			cc608Ctrl(0x14, 0x25),
			cc608Text('O', 'K'),
			cc608Ctrl(0x15, 0x60), // PAC row 3
			cc608Ctrl(0x15, 0x60),
			cc608Text('M', 'V'),
		},
	}
}

func gen608ColumnOverflow() scenario {
	var frames [][]byte
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	frames = append(frames, cc608Ctrl(0x14, 0x25))
	for i := 0; i < 17; i++ {
		frames = append(frames, cc608Text('A'+byte(i%26), 'a'+byte(i%26)))
	}
	return scenario{name: "608_column_overflow", frames: frames}
}

func dfWin(winID int, visible bool, rows, cols int, penStyle, winStyle byte) []byte {
	cmd := byte(0x98 + winID)
	var visBit byte
	if visible {
		visBit = 0x20
	}
	b0 := visBit | 0x03
	b1 := byte(0x00)
	b2 := byte(0x00)
	b3 := byte((rows - 1) & 0x0F)
	b4 := byte((cols - 1) & 0x3F)
	b5 := (winStyle << 3) | penStyle
	return []byte{cmd, b0, b1, b2, b3, b4, b5}
}

func gen708G0MusicNote() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{0x7F},
	)
	return scenario{
		name:   "708_g0_music_note",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708G1Latin1() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{0xC0, 0xC9, 0xD1, 0xE9, 0xFC},
	)
	return scenario{
		name:   "708_g1_latin1",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708G2Full() scenario {
	allCodes := []byte{
		0x25, 0x2A, 0x2C,
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35,
		0x39, 0x3A, 0x3C, 0x3D, 0x3F,
		0x76, 0x77, 0x78, 0x79,
		0x7A, 0x7B, 0x7C, 0x7D, 0x7E, 0x7F,
	}

	svc1 := concat(dfWin(0, true, 4, 42, 1, 1))
	for _, c := range allCodes[:8] {
		svc1 = append(svc1, 0x10, c)
	}

	svc2 := []byte{}
	for _, c := range allCodes[8:16] {
		svc2 = append(svc2, 0x10, c)
	}

	svc3 := []byte{}
	for _, c := range allCodes[16:] {
		svc3 = append(svc3, 0x10, c)
	}
	svc3 = append(svc3, 0x03)

	drain := dtvccTriplets([]byte{0x03})
	var frames [][]byte
	frames = append(frames, tripletFrames(dtvccTriplets(svc1))...)
	frames = append(frames, tripletFrames(dtvccTriplets(svc2))...)
	frames = append(frames, tripletFrames(dtvccTriplets(svc3))...)
	frames = append(frames, tripletFrames(drain)...)
	return scenario{
		name:   "708_g2_full",
		frames: frames,
	}
}

func gen708G3Icon() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{0x10, 0xA0},
		[]byte{0x03},
	)
	return scenario{
		name:   "708_g3_icon",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708P16() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{0x18, 0x00, 0x41},
	)
	return scenario{
		name:   "708_p16",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708Multiwindow() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'W', '0'},
		dfWin(1, true, 2, 32, 1, 1),
		[]byte{'W', '1'},
	)
	return scenario{
		name:   "708_multiwindow",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708HideShow() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'S', 'h', 'o', 'w'},
		[]byte{0x8A, 0x01},
		[]byte{0x89, 0x01},
		[]byte{'!'},
	)
	return scenario{
		name:   "708_hide_show",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708Toggle() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'O', 'n'},
		[]byte{0x8B, 0x01},
	)
	return scenario{
		name:   "708_toggle",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708DeleteWindow() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'G', 'o', 'n', 'e'},
		[]byte{0x8C, 0x01},
	)
	return scenario{
		name:   "708_delete_window",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708ClearWindow() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'T', 'x', 't'},
		[]byte{0x88, 0x01},
		[]byte{'N', 'w'},
	)
	return scenario{
		name:   "708_clear_window",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708Reset() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'O', 'l', 'd'},
		[]byte{0x8F},
	)
	return scenario{
		name:   "708_reset",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708SPAFull() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{
			0x90,
			0x02 | (0x01 << 2) | (0x05 << 4),
			0x03 | (0x02 << 3) | 0x40 | 0x80,
		},
		[]byte{'S', 'T'},
	)
	return scenario{
		name:   "708_spa_full",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708SPCFull() scenario {
	fgByte := byte(0x30 | 0x00)
	bgByte := byte(0x0C | 0x80)
	edgeByte := byte(0x3F)
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{0x91, fgByte, bgByte, edgeByte},
		[]byte{'C', 'L'},
		[]byte{0x03},
	)
	return scenario{
		name:   "708_spc_full",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708SWAFull() scenario {
	fillByte := byte(0x30 | 0x80)
	borderByte := byte(0x0F | 0xC0)
	justByte := byte(0x02 | (0x01 << 2) | (0x02 << 4) | 0x40)
	effectByte := byte(0x01 | (0x02 << 1) | (0x01 << 3) | (0x01 << 6))
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{0x97, fillByte, borderByte, justByte, effectByte},
		[]byte{'S', 'W'},
	)
	return scenario{
		name:   "708_swa_full",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708PredefinedPenStyles() scenario {
	svc1 := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'P', '1'},
		dfWin(1, true, 2, 32, 2, 1),
		[]byte{'P', '2'},
		dfWin(2, true, 2, 32, 3, 1),
		[]byte{'P', '3'},
	)
	svc2 := concat(
		dfWin(3, true, 2, 32, 4, 1),
		[]byte{'P', '4'},
		dfWin(4, true, 2, 32, 5, 1),
		[]byte{'P', '5'},
		dfWin(5, true, 2, 32, 6, 1),
		[]byte{'P', '6'},
	)
	svc3 := concat(
		dfWin(6, true, 2, 32, 7, 1),
		[]byte{'P', '7'},
		[]byte{0x03},
	)
	drain := dtvccTriplets([]byte{0x03})
	var frames [][]byte
	frames = append(frames, tripletFrames(dtvccTriplets(svc1))...)
	frames = append(frames, tripletFrames(dtvccTriplets(svc2))...)
	frames = append(frames, tripletFrames(dtvccTriplets(svc3))...)
	frames = append(frames, tripletFrames(drain)...)
	return scenario{
		name:   "708_predefined_pen_styles",
		frames: frames,
	}
}

func gen708PredefinedWindowStyles() scenario {
	svc1 := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'W', '1'},
		dfWin(1, true, 2, 32, 1, 2),
		[]byte{'W', '2'},
		dfWin(2, true, 2, 32, 1, 3),
		[]byte{'W', '3'},
	)
	svc2 := concat(
		dfWin(3, true, 2, 32, 1, 4),
		[]byte{'W', '4'},
		dfWin(4, true, 2, 32, 1, 5),
		[]byte{'W', '5'},
		dfWin(5, true, 2, 32, 1, 6),
		[]byte{'W', '6'},
	)
	svc3 := concat(
		dfWin(6, true, 2, 32, 1, 7),
		[]byte{'W', '7'},
		[]byte{0x03},
	)
	drain := dtvccTriplets([]byte{0x03})
	var frames [][]byte
	frames = append(frames, tripletFrames(dtvccTriplets(svc1))...)
	frames = append(frames, tripletFrames(dtvccTriplets(svc2))...)
	frames = append(frames, tripletFrames(dtvccTriplets(svc3))...)
	frames = append(frames, tripletFrames(drain)...)
	return scenario{
		name:   "708_predefined_window_styles",
		frames: frames,
	}
}

func gen708Backspace() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'A', 'B', 'C'},
		[]byte{0x08},
		[]byte{'X'},
	)
	return scenario{
		name:   "708_backspace",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708CarriageReturn() scenario {
	svc := concat(
		dfWin(0, true, 4, 32, 1, 1),
		[]byte{'L', '1'},
		[]byte{0x0D},
		[]byte{'L', '2'},
		[]byte{0x0D},
		[]byte{'L', '3'},
		[]byte{0x03},
	)
	return scenario{
		name:   "708_carriage_return",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708FormFeed() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'O', 'l', 'd'},
		[]byte{0x0C},
		[]byte{'N', 'e', 'w'},
	)
	return scenario{
		name:   "708_formfeed",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708HCR() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'O', 'l', 'd'},
		[]byte{0x0E},
		[]byte{'N', 'e', 'w'},
	)
	return scenario{
		name:   "708_hcr",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func gen708ETX() scenario {
	svc := concat(
		dfWin(0, true, 2, 32, 1, 1),
		[]byte{'A', 'B'},
		[]byte{0x03},
	)
	return scenario{
		name:   "708_etx",
		frames: tripletFrames(dtvccTriplets(svc)),
	}
}

func tripletFrames(triplets [][]byte) [][]byte {
	frames := make([][]byte, len(triplets))
	for i, t := range triplets {
		frames[i] = t
	}
	return frames
}

func main() {
	scenarios := []scenario{
		gen608RollUp2(),
		gen608RollUp3(),
		gen608RollUp4(),
		gen608PopOn(),
		gen608PaintOn(),
		gen608ModeSwitch(),

		gen608SpecialChars(),
		gen608ExtendedSpanish(),
		gen608ExtendedPortuguese(),
		gen608G0Overrides(),

		gen608Backspace(),
		gen608DeleteToEnd(),
		gen608TabOffsets(),
		gen608EraseDisplayed(),
		gen608EraseNonDisplayed(),

		gen608PACColors(),
		gen608PACIndent(),
		gen608PACUnderline(),
		gen608PACItalic(),
		gen608MidrowColors(),
		gen608MidrowItalicUnderline(),
		gen608BackgroundColors(),
		gen608Flash(),
		gen608PACRowPositioning(),

		gen608RollUpPACRelocate(),
		gen608ColumnOverflow(),

		gen708G0MusicNote(),
		gen708G1Latin1(),
		gen708G2Full(),
		gen708G3Icon(),
		gen708P16(),

		gen708Multiwindow(),
		gen708HideShow(),
		gen708Toggle(),
		gen708DeleteWindow(),
		gen708ClearWindow(),
		gen708Reset(),

		gen708SPAFull(),
		gen708SPCFull(),
		gen708SWAFull(),
		gen708PredefinedPenStyles(),
		gen708PredefinedWindowStyles(),

		gen708Backspace(),
		gen708CarriageReturn(),
		gen708FormFeed(),
		gen708HCR(),
		gen708ETX(),
	}

	for _, s := range scenarios {
		generateBitstream(s.name, "h264", s.frames)
		generateBitstream(s.name, "h265", s.frames)
	}

	absDir, _ := filepath.Abs(".")
	fmt.Printf("Generated %d scenarios (%d files) in %s\n",
		len(scenarios), len(scenarios)*2, absDir)
}
