//go:build ignore

package main

import (
	"encoding/binary"
	"os"
)

func main() {
	generateH264()
	generateH265()
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

func generateH264() {
	f, _ := os.Create("h264_caption_sei.h264")
	defer f.Close()

	sps := []byte{0x67, 0x42, 0xC0, 0x1E, 0xD9, 0x00, 0xA0, 0x47, 0xFE, 0xC8}
	pps := []byte{0x68, 0xCE, 0x38, 0x80}

	writeAnnexBNAL(f, sps)
	writeAnnexBNAL(f, pps)

	for i := 0; i < 5; i++ {
		triplets := make([]byte, 0)
		switch i {
		case 0:
			triplets = append(triplets,
				0xFC, 0x14, 0x25, // CC1 RU2
			)
		case 1:
			triplets = append(triplets,
				0xFC, 0x14, 0x25, // CC1 RU2 dedup
			)
		case 2:
			triplets = append(triplets,
				0xFC, 'H', 'e',
			)
		case 3:
			triplets = append(triplets,
				0xFC, 'l', 'l',
			)
		case 4:
			triplets = append(triplets,
				0xFC, 'o', '!',
			)
		}

		a53 := buildA53CaptionPayload(triplets, len(triplets)/3)
		seiMsg := buildSEIMessage(4, a53)

		nalHeader := byte(0x06)
		seiBody := addEPB(seiMsg)
		seiNAL := append([]byte{nalHeader}, seiBody...)
		writeAnnexBNAL(f, seiNAL)

		idrNAL := []byte{0x65}
		for j := 0; j < 100; j++ {
			idrNAL = append(idrNAL, 0x00)
		}
		writeAnnexBNAL(f, idrNAL)
	}
}

func generateH265() {
	f, _ := os.Create("h265_caption_sei.h265")
	defer f.Close()

	vps := []byte{0x40, 0x01, 0x0C, 0x01, 0xFF, 0xFF, 0x01, 0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x96, 0x09, 0x00}
	sps := []byte{0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x96, 0xA0, 0x01, 0x40, 0x20, 0x06, 0x11, 0x48, 0xFC, 0xBC}
	pps := []byte{0x44, 0x01, 0xC1, 0x73, 0x18, 0x31}

	writeAnnexBNAL(f, vps)
	writeAnnexBNAL(f, sps)
	writeAnnexBNAL(f, pps)

	for i := 0; i < 5; i++ {
		triplets := make([]byte, 0)
		switch i {
		case 0:
			triplets = append(triplets,
				0xFC, 0x14, 0x25, // CC1 RU2
			)
		case 1:
			triplets = append(triplets,
				0xFC, 0x14, 0x25, // CC1 RU2 dedup
			)
		case 2:
			triplets = append(triplets,
				0xFC, 'W', 'o',
			)
		case 3:
			triplets = append(triplets,
				0xFC, 'r', 'l',
			)
		case 4:
			triplets = append(triplets,
				0xFC, 'd', '!',
			)
		}

		a53 := buildA53CaptionPayload(triplets, len(triplets)/3)
		seiMsg := buildSEIMessage(4, a53)

		nalType := byte(39)
		nalHeader0 := (nalType << 1)
		nalHeader1 := byte(0x01)
		seiBody := addEPB(seiMsg)
		seiNAL := append([]byte{nalHeader0, nalHeader1}, seiBody...)
		writeAnnexBNAL(f, seiNAL)

		idrType := byte(19)
		idrNAL := []byte{idrType << 1, 0x01}
		idrPayload := make([]byte, 100)
		binary.BigEndian.PutUint32(idrPayload, 0x80000000)
		idrNAL = append(idrNAL, idrPayload...)
		writeAnnexBNAL(f, idrNAL)
	}
}
