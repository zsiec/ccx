package ccx

// ExtractCEA608 extracts CEA-608 closed caption byte pairs from an H.264 SEI
// NAL unit containing ATSC A/53 user data (ITU-T T.35).
func ExtractCEA608(nalData []byte) []CCPair {
	cd := ExtractCaptions(nalData)
	if cd == nil {
		return nil
	}
	return cd.CC608Pairs
}

// ExtractCaptions extracts both CEA-608 and CEA-708/DTVCC caption data from
// an H.264 SEI NAL unit containing ATSC A/53 user data.
func ExtractCaptions(nalData []byte) *CaptionData {
	if len(nalData) < 2 {
		return nil
	}

	raw := removeEPB(nalData[1:])

	i := 0
	var result *CaptionData
	for i < len(raw) {
		payloadType := 0
		for i < len(raw) && raw[i] == 0xFF {
			payloadType += 255
			i++
		}
		if i >= len(raw) {
			break
		}
		payloadType += int(raw[i])
		i++

		payloadSize := 0
		for i < len(raw) && raw[i] == 0xFF {
			payloadSize += 255
			i++
		}
		if i >= len(raw) {
			break
		}
		payloadSize += int(raw[i])
		i++

		if i+payloadSize > len(raw) {
			break
		}

		payload := raw[i : i+payloadSize]
		i += payloadSize

		if payloadType == 4 {
			cd := parseA53Payload(payload)
			if cd != nil {
				result = cd
			}
		}
	}
	return result
}

func parseA53Payload(payload []byte) *CaptionData {
	if len(payload) < 10 {
		return nil
	}

	if payload[0] != 0xB5 {
		return nil
	}
	if payload[1] != 0x00 || payload[2] != 0x31 {
		return nil
	}
	if payload[3] != 'G' || payload[4] != 'A' || payload[5] != '9' || payload[6] != '4' {
		return nil
	}
	if payload[7] != 0x03 {
		return nil
	}

	ccHeader := payload[8]
	if ccHeader&0x40 == 0 {
		return nil
	}
	ccCount := int(ccHeader & 0x1F)

	tripletStart := 10
	if tripletStart+ccCount*3 > len(payload) {
		return nil
	}

	var cd CaptionData
	activeChannel := [2]int{1, 3}

	for j := 0; j < ccCount; j++ {
		offset := tripletStart + j*3
		marker := payload[offset]
		ccData1 := payload[offset+1]
		ccData2 := payload[offset+2]

		if marker&0x04 == 0 {
			continue
		}

		ccType := marker & 0x03

		if ccType >= 2 {
			cd.DTVCC = append(cd.DTVCC, DTVCCPair{
				Data:  [2]byte{ccData1, ccData2},
				Start: ccType == 3,
			})
			continue
		}

		ccData1 &= 0x7F
		ccData2 &= 0x7F

		if ccData1 == 0x00 && ccData2 == 0x00 {
			continue
		}

		var channel int
		if ccData1 >= 0x10 && ccData1 <= 0x1F {
			isPrimary := ccData1&0x08 == 0
			if ccType == 0 {
				if isPrimary {
					channel = 1
				} else {
					channel = 2
				}
			} else {
				if isPrimary {
					channel = 3
				} else {
					channel = 4
				}
			}
			activeChannel[ccType] = channel
		} else {
			channel = activeChannel[ccType]
		}

		cd.CC608Pairs = append(cd.CC608Pairs, CCPair{
			Data:    [2]byte{ccData1, ccData2},
			Channel: channel,
			Field:   ccType,
		})
	}

	if len(cd.CC608Pairs) == 0 && len(cd.DTVCC) == 0 {
		return nil
	}
	return &cd
}

// removeEPB removes H.264 emulation prevention bytes.
// 0x00 0x00 0x03 -> 0x00 0x00
func removeEPB(data []byte) []byte {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if i+2 < len(data) && data[i] == 0x00 && data[i+1] == 0x00 && data[i+2] == 0x03 {
			out = append(out, 0x00, 0x00)
			i += 2
		} else {
			out = append(out, data[i])
		}
	}
	return out
}
