package ccx

import "encoding/binary"

const captionMagic = 0xCC02

// Serialize encodes a CaptionFrame into a compact binary format suitable
// for transport over WebTransport, WebSocket, or any byte-oriented channel.
//
// Wire format (v2):
//
//	[2] magic 0xCC02
//	[1] version (2)
//	[1] channel
//	[1] region count
//	per region:
//	  [1] id
//	  [1] justify | scrollDir<<2 | printDir<<4 | wordWrap<<6 | relativeToggle<<7
//	  [1] fillOpacity<<6 | borderType<<3 | priority
//	  [3] fillColor (RGB)
//	  [3] borderColor (RGB)
//	  [1] anchorV
//	  [1] anchorH
//	  [1] anchorID
//	  [2] row count
//	  per row:
//	    [1] row index
//	    [1] span count
//	    per span:
//	      [2] text length
//	      [n] text (UTF-8)
//	      [3] fgColor (RGB)
//	      [3] bgColor (RGB)
//	      [1] fgOpacity<<6 | bgOpacity<<4 | italic<<3 | underline<<2 | flash<<1 | penSize(bit1)
//	      [1] penSize(bit0)<<7 | fontTag<<4 | offset<<2 | edgeType(lo2)
//	      [1] edgeType(bit2)<<7 (reserved rest)
//	      [3] edgeColor (RGB)
//
// If no regions are present, falls back to legacy format: [1] channel + [n] text.
func (f *CaptionFrame) Serialize() []byte {
	if len(f.Regions) == 0 {
		data := make([]byte, 1+len(f.Text))
		data[0] = byte(f.Channel)
		copy(data[1:], f.Text)
		return data
	}

	buf := make([]byte, 0, 256)
	buf = append(buf, byte(captionMagic>>8), byte(captionMagic&0xFF))
	buf = append(buf, 2)
	buf = append(buf, byte(f.Channel))
	buf = append(buf, byte(len(f.Regions)))

	for _, reg := range f.Regions {
		buf = append(buf, byte(reg.ID))
		flags := byte(reg.Justify&0x03) | byte((reg.ScrollDirection&0x03)<<2) | byte((reg.PrintDirection&0x03)<<4)
		if reg.WordWrap {
			flags |= 0x40
		}
		if reg.RelativeToggle {
			flags |= 0x80
		}
		buf = append(buf, flags)
		fillFlags := byte((reg.FillOpacity&0x03)<<6) | byte((reg.BorderType&0x07)<<3) | byte(reg.Priority&0x07)
		buf = append(buf, fillFlags)
		buf = append(buf, hexToRGB(reg.FillColor)...)
		buf = append(buf, hexToRGB(reg.BorderColor)...)
		buf = append(buf, byte(reg.AnchorV))
		buf = append(buf, byte(reg.AnchorH))
		buf = append(buf, byte(reg.AnchorID))

		rowCount := len(reg.Rows)
		buf = binary.BigEndian.AppendUint16(buf, uint16(rowCount))

		for _, row := range reg.Rows {
			buf = append(buf, byte(row.Row))
			buf = append(buf, byte(len(row.Spans)))

			for _, span := range row.Spans {
				textBytes := []byte(span.Text)
				buf = binary.BigEndian.AppendUint16(buf, uint16(len(textBytes)))
				buf = append(buf, textBytes...)
				buf = append(buf, hexToRGB(span.FgColor)...)
				buf = append(buf, hexToRGB(span.BgColor)...)

				var attr0 byte
				attr0 |= byte((span.FgOpacity & 0x03) << 6)
				attr0 |= byte((span.BgOpacity & 0x03) << 4)
				if span.Italic {
					attr0 |= 0x08
				}
				if span.Underline {
					attr0 |= 0x04
				}
				if span.Flash {
					attr0 |= 0x02
				}
				attr0 |= byte((span.PenSize >> 1) & 0x01)
				buf = append(buf, attr0)

				var attr1 byte
				attr1 |= byte((span.PenSize & 0x01) << 7)
				attr1 |= byte((span.FontTag & 0x07) << 4)
				attr1 |= byte((span.Offset & 0x03) << 2)
				attr1 |= byte(span.EdgeType & 0x03)
				buf = append(buf, attr1)

				attr2 := byte(((span.EdgeType >> 2) & 0x01) << 7)
				buf = append(buf, attr2)

				buf = append(buf, hexToRGB(span.EdgeColor)...)
			}
		}
	}

	return buf
}

// DeserializeCaptionFrame decodes a binary caption frame produced by Serialize.
// Returns nil if the data is too short or malformed.
func DeserializeCaptionFrame(data []byte) *CaptionFrame {
	if len(data) < 2 {
		return nil
	}

	magic := uint16(data[0])<<8 | uint16(data[1])
	if magic != captionMagic {
		return &CaptionFrame{
			Channel: int(data[0]),
			Text:    string(data[1:]),
		}
	}

	if len(data) < 5 {
		return nil
	}

	version := data[2]
	f := &CaptionFrame{}
	f.Channel = int(data[3])
	regionCount := int(data[4])
	pos := 5

	for i := 0; i < regionCount && pos < len(data); i++ {
		if pos+3 > len(data) {
			break
		}

		reg := CaptionRegion{}
		reg.ID = int(data[pos])
		pos++

		flags := data[pos]
		pos++
		reg.Justify = int(flags & 0x03)
		reg.ScrollDirection = int((flags >> 2) & 0x03)
		reg.PrintDirection = int((flags >> 4) & 0x03)
		reg.WordWrap = flags&0x40 != 0
		reg.RelativeToggle = flags&0x80 != 0

		fillFlags := data[pos]
		pos++
		reg.FillOpacity = int((fillFlags >> 6) & 0x03)
		reg.BorderType = int((fillFlags >> 3) & 0x07)
		reg.Priority = int(fillFlags & 0x07)

		if version >= 2 {
			if pos+9 > len(data) {
				break
			}
			reg.FillColor = rgbToHex(data[pos], data[pos+1], data[pos+2])
			pos += 3
			reg.BorderColor = rgbToHex(data[pos], data[pos+1], data[pos+2])
			pos += 3
			reg.AnchorV = int(data[pos])
			pos++
			reg.AnchorH = int(data[pos])
			pos++
			reg.AnchorID = int(data[pos])
			pos++
		}

		if pos+2 > len(data) {
			break
		}
		rowCount := int(binary.BigEndian.Uint16(data[pos:]))
		pos += 2

		for r := 0; r < rowCount && pos < len(data); r++ {
			if pos+2 > len(data) {
				break
			}
			row := CaptionRow{Row: int(data[pos])}
			pos++
			spanCount := int(data[pos])
			pos++

			for s := 0; s < spanCount && pos < len(data); s++ {
				if pos+2 > len(data) {
					break
				}
				textLen := int(binary.BigEndian.Uint16(data[pos:]))
				pos += 2

				if pos+textLen > len(data) {
					break
				}
				span := CaptionSpan{Text: string(data[pos : pos+textLen])}
				pos += textLen

				if pos+9 > len(data) {
					break
				}
				span.FgColor = rgbToHex(data[pos], data[pos+1], data[pos+2])
				pos += 3
				span.BgColor = rgbToHex(data[pos], data[pos+1], data[pos+2])
				pos += 3

				attr0 := data[pos]
				pos++
				span.FgOpacity = int((attr0 >> 6) & 0x03)
				span.BgOpacity = int((attr0 >> 4) & 0x03)
				span.Italic = attr0&0x08 != 0
				span.Underline = attr0&0x04 != 0
				span.Flash = attr0&0x02 != 0
				span.PenSize = int((attr0 & 0x01) << 1)

				attr1 := data[pos]
				pos++
				span.PenSize |= int((attr1 >> 7) & 0x01)
				span.FontTag = int((attr1 >> 4) & 0x07)
				span.Offset = int((attr1 >> 2) & 0x03)
				span.EdgeType = int(attr1 & 0x03)

				attr2 := data[pos]
				pos++
				span.EdgeType |= int((attr2>>7)&0x01) << 2

				if pos+3 > len(data) {
					break
				}
				span.EdgeColor = rgbToHex(data[pos], data[pos+1], data[pos+2])
				pos += 3

				row.Spans = append(row.Spans, span)
			}
			reg.Rows = append(reg.Rows, row)
		}
		f.Regions = append(f.Regions, reg)
	}

	return f
}

func hexToRGB(hex string) []byte {
	if len(hex) != 6 {
		return []byte{0, 0, 0}
	}
	r := hexNibble(hex[0])<<4 | hexNibble(hex[1])
	g := hexNibble(hex[2])<<4 | hexNibble(hex[3])
	b := hexNibble(hex[4])<<4 | hexNibble(hex[5])
	return []byte{r, g, b}
}

func hexNibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func rgbToHex(r, g, b byte) string {
	const hexChars = "0123456789abcdef"
	return string([]byte{
		hexChars[r>>4], hexChars[r&0x0F],
		hexChars[g>>4], hexChars[g&0x0F],
		hexChars[b>>4], hexChars[b&0x0F],
	})
}
