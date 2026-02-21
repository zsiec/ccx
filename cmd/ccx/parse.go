package main

import (
	"io"

	"github.com/zsiec/ccx"
)

type codec int

const (
	codecUnknown codec = iota
	codecH264
	codecH265
)

func (c codec) String() string {
	switch c {
	case codecH264:
		return "H.264"
	case codecH265:
		return "H.265"
	default:
		return "unknown"
	}
}

type captionEvent struct {
	NALIndex int
	Codec    codec
	Data     *ccx.CaptionData
	Frame608 *ccx.CaptionFrame
	Frame708 *ccx.CaptionFrame
}

type streamParser struct {
	codec     codec
	dec608    [5]*ccx.CEA608Decoder // indexed by channel (1-4); 0 unused
	dec708    *ccx.CEA708Decoder
	nalIndex  int
}

func newStreamParser(c codec) *streamParser {
	sp := &streamParser{
		codec:  c,
		dec708: ccx.NewCEA708Decoder(),
	}
	for i := 1; i <= 4; i++ {
		sp.dec608[i] = ccx.NewCEA608Decoder()
	}
	return sp
}

// parseStream reads an Annex B bitstream and yields caption events.
func parseStream(r io.Reader, forceCodec codec, emit func(captionEvent)) error {
	buf, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	nals := splitAnnexB(buf)
	if len(nals) == 0 {
		return nil
	}

	detectedCodec := forceCodec
	if detectedCodec == codecUnknown {
		detectedCodec = detectCodec(nals)
	}

	sp := newStreamParser(detectedCodec)

	for _, nal := range nals {
		if len(nal) == 0 {
			continue
		}
		sp.nalIndex++
		sp.processNAL(nal, emit)
	}
	return nil
}

func (sp *streamParser) processNAL(nal []byte, emit func(captionEvent)) {
	var cd *ccx.CaptionData

	switch sp.codec {
	case codecH264:
		if !ccx.IsH264SEI(nal[0]) {
			return
		}
		cd = ccx.ExtractCaptions(nal)
	case codecH265:
		if len(nal) < 2 || !ccx.IsHEVCSEI(nal[0]) {
			return
		}
		cd = ccx.ExtractCaptionsHEVC(nal)
	default:
		return
	}

	if cd == nil {
		return
	}

	ev := captionEvent{
		NALIndex: sp.nalIndex,
		Codec:    sp.codec,
		Data:     cd,
	}

	for _, pair := range cd.CC608Pairs {
		ch := pair.Channel
		if ch < 1 || ch > 4 {
			continue
		}
		dec := sp.dec608[ch]
		text := dec.Decode(pair.Data[0], pair.Data[1])
		if text != "" {
			regions := dec.StyledRegions()
			ev.Frame608 = &ccx.CaptionFrame{
				Text:    text,
				Channel: pair.Channel,
				Regions: regions,
			}
		}
	}

	for _, pair := range cd.DTVCC {
		text := sp.dec708.AddTriplet(pair)
		if text != "" {
			regions := sp.dec708.StyledRegions()
			ev.Frame708 = &ccx.CaptionFrame{
				Text:    text,
				Regions: regions,
			}
		}
	}

	if ev.Frame608 != nil || ev.Frame708 != nil || len(cd.CC608Pairs) > 0 || len(cd.DTVCC) > 0 {
		emit(ev)
	}
}

// splitAnnexB splits a byte stream on 0x000001 and 0x00000001 start codes.
func splitAnnexB(data []byte) [][]byte {
	var nals [][]byte
	i := 0
	n := len(data)

	start := -1
	for i < n {
		if i+2 < n && data[i] == 0x00 && data[i+1] == 0x00 {
			scLen := 0
			if data[i+2] == 0x01 {
				scLen = 3
			} else if i+3 < n && data[i+2] == 0x00 && data[i+3] == 0x01 {
				scLen = 4
			}
			if scLen > 0 {
				if start >= 0 {
					nals = append(nals, data[start:i])
				}
				start = i + scLen
				i += scLen
				continue
			}
		}
		i++
	}
	if start >= 0 && start < n {
		nals = append(nals, data[start:])
	}
	return nals
}

// detectCodec inspects NAL units to determine if the stream is H.264 or
// H.265. It uses a voting approach because the single-byte H.264 NAL type
// mask (0x1F) can collide with HEVC 2-byte headers (e.g. HEVC IDR_W_RADL
// type 19 has byte 0 = 0x26, which looks like H.264 SEI type 6).
func detectCodec(nals [][]byte) codec {
	h264Votes := 0
	h265Votes := 0

	for _, nal := range nals {
		if len(nal) < 2 {
			continue
		}

		// HEVC: forbidden_zero_bit(1) | type(6) | layer_id(6) in byte 0-1,
		// nuh_temporal_id_plus1(3) in low bits of byte 1 (must be >= 1).
		if nal[0]&0x80 == 0 && nal[1]&0x07 != 0 {
			hevcType := (nal[0] >> 1) & 0x3F
			switch hevcType {
			case 32, 33, 34: // VPS, SPS, PPS
				h265Votes += 10
			case 39, 40: // PREFIX_SEI, SUFFIX_SEI
				h265Votes += 10
			case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, // TRAIL/TSA/STSA
				16, 17, 18, 19, 20, 21: // BLA/IDR/CRA
				h265Votes++
			}
		}

		// H.264: forbidden_zero_bit(1) | nal_ref_idc(2) | type(5)
		if nal[0]&0x80 == 0 {
			h264Type := nal[0] & 0x1F
			switch h264Type {
			case 7, 8: // SPS, PPS
				h264Votes += 10
			case 6: // SEI
				h264Votes += 10
			case 1, 2, 3, 4, 5: // slice types
				h264Votes++
			}
		}
	}

	if h265Votes > h264Votes {
		return codecH265
	}
	if h264Votes > 0 {
		return codecH264
	}
	return codecUnknown
}
