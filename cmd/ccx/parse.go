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
	codec    codec
	dec608   *ccx.CEA608Decoder
	dec708   *ccx.CEA708Decoder
	nalIndex int
}

func newStreamParser(c codec) *streamParser {
	return &streamParser{
		codec:  c,
		dec608: ccx.NewCEA608Decoder(),
		dec708: ccx.NewCEA708Decoder(),
	}
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
		text := sp.dec608.Decode(pair.Data[0], pair.Data[1])
		if text != "" {
			regions := sp.dec608.StyledRegions()
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

// detectCodec inspects the first NAL unit to determine if the stream is
// H.264 or H.265. H.264 NAL headers are 1 byte; H.265 uses 2 bytes with
// the type in the upper 6 bits of byte 0.
func detectCodec(nals [][]byte) codec {
	for _, nal := range nals {
		if len(nal) == 0 {
			continue
		}

		h264Type := nal[0] & 0x1F
		if h264Type == 7 || h264Type == 8 {
			return codecH264
		}
		if h264Type == 6 {
			return codecH264
		}

		if len(nal) >= 2 {
			hevcType := (nal[0] >> 1) & 0x3F
			if hevcType == 32 || hevcType == 33 || hevcType == 34 {
				return codecH265
			}
			if hevcType == 39 || hevcType == 40 {
				return codecH265
			}
		}
	}
	return codecUnknown
}
