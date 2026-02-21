package ccx

import (
	"bytes"
	"os"
	"testing"
)

func buildHEVCSEINAL(triplets []byte, ccCount int) []byte {
	t35 := []byte{
		0xB5, 0x00, 0x31,
		'G', 'A', '9', '4',
		0x03,
		0x40 | byte(ccCount&0x1F),
		0xFF,
	}
	t35 = append(t35, triplets...)

	nalType := byte(39) // PREFIX_SEI
	nalHeader0 := nalType << 1
	nalHeader1 := byte(0x01)
	nalData := []byte{nalHeader0, nalHeader1, 0x04, byte(len(t35))}
	nalData = append(nalData, t35...)
	return nalData
}

func buildHEVCSuffixSEINAL(triplets []byte, ccCount int) []byte {
	t35 := []byte{
		0xB5, 0x00, 0x31,
		'G', 'A', '9', '4',
		0x03,
		0x40 | byte(ccCount&0x1F),
		0xFF,
	}
	t35 = append(t35, triplets...)

	nalType := byte(40) // SUFFIX_SEI
	nalHeader0 := nalType << 1
	nalHeader1 := byte(0x01)
	nalData := []byte{nalHeader0, nalHeader1, 0x04, byte(len(t35))}
	nalData = append(nalData, t35...)
	return nalData
}

func TestExtractCaptionsHEVC_Basic(t *testing.T) {
	nalData := buildHEVCSEINAL([]byte{
		0xFC, 'H', 'i',
		0xFC, '!', 0x00,
	}, 2)

	cd := ExtractCaptionsHEVC(nalData)
	if cd == nil {
		t.Fatal("expected non-nil CaptionData")
	}
	if len(cd.CC608Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(cd.CC608Pairs))
	}
	if cd.CC608Pairs[0].Data[0] != 'H' || cd.CC608Pairs[0].Data[1] != 'i' {
		t.Errorf("pair 0: got [%02x, %02x], want ['H', 'i']",
			cd.CC608Pairs[0].Data[0], cd.CC608Pairs[0].Data[1])
	}
	if cd.CC608Pairs[0].Channel != 1 {
		t.Errorf("pair 0 channel: got %d, want 1", cd.CC608Pairs[0].Channel)
	}
}

func TestExtractCaptionsHEVC_NullPairsSkipped(t *testing.T) {
	nalData := buildHEVCSEINAL([]byte{
		0xFC, 0x00, 0x00,
		0xFC, 'A', 'B',
	}, 2)

	cd := ExtractCaptionsHEVC(nalData)
	if cd == nil {
		t.Fatal("expected non-nil CaptionData")
	}
	if len(cd.CC608Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(cd.CC608Pairs))
	}
	if cd.CC608Pairs[0].Data[0] != 'A' {
		t.Errorf("expected 'A', got 0x%02x", cd.CC608Pairs[0].Data[0])
	}
}

func TestExtractCaptionsHEVC_DTVCC(t *testing.T) {
	nalData := buildHEVCSEINAL([]byte{
		0xFE, 0xAA, 0xBB, // cc_type=2 (DTVCC data)
		0xFF, 0xCC, 0xDD, // cc_type=3 (DTVCC start)
		0xFC, 'X', 'Y', // cc_type=0 (CEA-608 field 1)
	}, 3)

	cd := ExtractCaptionsHEVC(nalData)
	if cd == nil {
		t.Fatal("expected non-nil CaptionData")
	}
	if len(cd.CC608Pairs) != 1 {
		t.Fatalf("expected 1 CEA-608 pair, got %d", len(cd.CC608Pairs))
	}
	if len(cd.DTVCC) != 2 {
		t.Fatalf("expected 2 DTVCC pairs, got %d", len(cd.DTVCC))
	}
	if !cd.DTVCC[1].Start {
		t.Error("DTVCC[1] should have Start=true (cc_type=3)")
	}
}

func TestExtractCaptionsHEVC_ChannelRouting(t *testing.T) {
	nalData := buildHEVCSEINAL([]byte{
		0xFC, 0x94, 0xA5, // CC1 RU2 (with parity)
		0xFC, 'A', 'B', // inherits CC1
		0xFC, 0x9C, 0xA5, // CC2 RU2
		0xFC, 'C', 'D', // inherits CC2
	}, 4)

	cd := ExtractCaptionsHEVC(nalData)
	if cd == nil {
		t.Fatal("expected non-nil CaptionData")
	}
	if len(cd.CC608Pairs) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(cd.CC608Pairs))
	}

	wantChannels := []int{1, 1, 2, 2}
	for i, want := range wantChannels {
		if cd.CC608Pairs[i].Channel != want {
			t.Errorf("pair %d: channel=%d, want %d", i, cd.CC608Pairs[i].Channel, want)
		}
	}
}

func TestExtractCaptionsHEVC_SuffixSEI(t *testing.T) {
	nalData := buildHEVCSuffixSEINAL([]byte{
		0xFC, 'S', 'X',
	}, 1)

	cd := ExtractCaptionsHEVC(nalData)
	if cd == nil {
		t.Fatal("suffix SEI should be parseable")
	}
	if len(cd.CC608Pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(cd.CC608Pairs))
	}
	if cd.CC608Pairs[0].Data[0] != 'S' {
		t.Errorf("expected 'S', got 0x%02x", cd.CC608Pairs[0].Data[0])
	}
}

func TestExtractCaptionsHEVC_TooShort(t *testing.T) {
	if ExtractCaptionsHEVC(nil) != nil {
		t.Error("nil should return nil")
	}
	if ExtractCaptionsHEVC([]byte{0x4E}) != nil {
		t.Error("1 byte should return nil")
	}
	if ExtractCaptionsHEVC([]byte{0x4E, 0x01}) != nil {
		t.Error("2 bytes (header only) should return nil")
	}
}

func TestExtractCaptionsHEVC_WrongPayloadType(t *testing.T) {
	nalType := byte(39)
	nalHeader0 := nalType << 1
	nalData := []byte{nalHeader0, 0x01, 0x05, 0x03, 0x01, 0x02, 0x03}
	cd := ExtractCaptionsHEVC(nalData)
	if cd != nil {
		t.Error("non-T35 payload type should return nil")
	}
}

func TestExtractCaptionsHEVC_ParityStripped(t *testing.T) {
	nalData := buildHEVCSEINAL([]byte{
		0xFC, 0xC8, 0xE9, // 'H' and 'i' with parity set
	}, 1)

	cd := ExtractCaptionsHEVC(nalData)
	if cd == nil {
		t.Fatal("expected non-nil")
	}
	if cd.CC608Pairs[0].Data[0] != 'H' || cd.CC608Pairs[0].Data[1] != 'i' {
		t.Errorf("parity not stripped: got [%02x, %02x]",
			cd.CC608Pairs[0].Data[0], cd.CC608Pairs[0].Data[1])
	}
}

func TestIsH264SEI(t *testing.T) {
	if !IsH264SEI(0x06) {
		t.Error("0x06 should be H.264 SEI")
	}
	if !IsH264SEI(0x66) {
		t.Error("0x66 (forbidden=0, ref=3, type=6) should be H.264 SEI")
	}
	if IsH264SEI(0x65) {
		t.Error("0x65 (type=5, IDR) should not be SEI")
	}
}

func TestIsHEVCSEI(t *testing.T) {
	if !IsHEVCSEI(39 << 1) {
		t.Error("type 39 should be HEVC prefix SEI")
	}
	if !IsHEVCSEI(40 << 1) {
		t.Error("type 40 should be HEVC suffix SEI")
	}
	if IsHEVCSEI(19 << 1) {
		t.Error("type 19 (IDR) should not be SEI")
	}
	if IsHEVCSEI(0x00) {
		t.Error("type 0 should not be SEI")
	}
}

// --- End-to-end tests with real bitstream files ---

func splitAnnexBNALs(data []byte) [][]byte {
	var nals [][]byte
	i := 0
	for i < len(data) {
		start := -1
		headerLen := 0
		if i+4 <= len(data) && bytes.Equal(data[i:i+4], []byte{0, 0, 0, 1}) {
			start = i + 4
			headerLen = 4
		} else if i+3 <= len(data) && bytes.Equal(data[i:i+3], []byte{0, 0, 1}) {
			start = i + 3
			headerLen = 3
		}
		if start < 0 {
			i++
			continue
		}

		end := len(data)
		for j := start; j < len(data)-2; j++ {
			if data[j] == 0 && data[j+1] == 0 && (data[j+2] == 1 || (j+3 < len(data) && data[j+2] == 0 && data[j+3] == 1)) {
				end = j
				break
			}
		}
		nals = append(nals, data[i+headerLen:end])
		i = end
	}
	return nals
}

func TestE2E_H264_RealBitstream(t *testing.T) {
	data, err := os.ReadFile("testdata/h264_caption_sei.h264")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	nals := splitAnnexBNALs(data)
	if len(nals) == 0 {
		t.Fatal("no NALs found in test file")
	}

	dec := NewCEA608Decoder()
	var allText []string
	seiCount := 0

	for _, nal := range nals {
		if len(nal) == 0 {
			continue
		}
		if !IsH264SEI(nal[0]) {
			continue
		}
		seiCount++

		cd := ExtractCaptions(nal)
		if cd == nil {
			continue
		}

		for _, pair := range cd.CC608Pairs {
			text := dec.Decode(pair.Data[0], pair.Data[1])
			if text != "" {
				allText = append(allText, text)
			}
		}
	}

	if seiCount == 0 {
		t.Fatal("no SEI NALs found in H.264 bitstream")
	}

	if len(allText) == 0 {
		t.Fatal("no caption text decoded from H.264 bitstream")
	}

	finalText := allText[len(allText)-1]
	if finalText != "Hello!" {
		t.Errorf("final decoded text: got %q, want %q", finalText, "Hello!")
	}

	t.Logf("H.264 E2E: found %d SEI NALs, decoded %d text updates, final=%q",
		seiCount, len(allText), finalText)
}

func TestE2E_H264_StyledRegions(t *testing.T) {
	data, err := os.ReadFile("testdata/h264_caption_sei.h264")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	nals := splitAnnexBNALs(data)
	dec := NewCEA608Decoder()

	for _, nal := range nals {
		if len(nal) == 0 || !IsH264SEI(nal[0]) {
			continue
		}
		cd := ExtractCaptions(nal)
		if cd == nil {
			continue
		}
		for _, pair := range cd.CC608Pairs {
			dec.Decode(pair.Data[0], pair.Data[1])
		}
	}

	regions := dec.StyledRegions()
	if len(regions) == 0 {
		t.Fatal("expected styled regions after decoding")
	}

	totalText := ""
	for _, row := range regions[0].Rows {
		for _, span := range row.Spans {
			totalText += span.Text
		}
	}
	if totalText != "Hello!" {
		t.Errorf("styled text: got %q, want %q", totalText, "Hello!")
	}

	span := regions[0].Rows[0].Spans[0]
	if span.FgColor == "" {
		t.Error("span should have a foreground color")
	}
	if span.BgColor == "" {
		t.Error("span should have a background color")
	}

	frame := &CaptionFrame{Channel: 1, Regions: regions}
	wire := frame.Serialize()
	decoded := DeserializeCaptionFrame(wire)
	if decoded == nil {
		t.Fatal("failed to deserialize")
	}
	if decoded.PlainText() != "Hello!" {
		t.Errorf("round-trip text: got %q, want %q", decoded.PlainText(), "Hello!")
	}

	t.Logf("H.264 E2E styled: %d region(s), %d row(s), serialized=%d bytes",
		len(regions), len(regions[0].Rows), len(wire))
}

func TestE2E_H265_RealBitstream(t *testing.T) {
	data, err := os.ReadFile("testdata/h265_caption_sei.h265")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	nals := splitAnnexBNALs(data)
	if len(nals) == 0 {
		t.Fatal("no NALs found in test file")
	}

	dec := NewCEA608Decoder()
	var allText []string
	seiCount := 0

	for _, nal := range nals {
		if len(nal) < 2 {
			continue
		}
		if !IsHEVCSEI(nal[0]) {
			continue
		}
		seiCount++

		cd := ExtractCaptionsHEVC(nal)
		if cd == nil {
			continue
		}

		for _, pair := range cd.CC608Pairs {
			text := dec.Decode(pair.Data[0], pair.Data[1])
			if text != "" {
				allText = append(allText, text)
			}
		}
	}

	if seiCount == 0 {
		t.Fatal("no SEI NALs found in H.265 bitstream")
	}

	if len(allText) == 0 {
		t.Fatal("no caption text decoded from H.265 bitstream")
	}

	finalText := allText[len(allText)-1]
	if finalText != "World!" {
		t.Errorf("final decoded text: got %q, want %q", finalText, "World!")
	}

	t.Logf("H.265 E2E: found %d SEI NALs, decoded %d text updates, final=%q",
		seiCount, len(allText), finalText)
}

func TestE2E_H265_StyledRegions(t *testing.T) {
	data, err := os.ReadFile("testdata/h265_caption_sei.h265")
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	nals := splitAnnexBNALs(data)
	dec := NewCEA608Decoder()

	for _, nal := range nals {
		if len(nal) < 2 || !IsHEVCSEI(nal[0]) {
			continue
		}
		cd := ExtractCaptionsHEVC(nal)
		if cd == nil {
			continue
		}
		for _, pair := range cd.CC608Pairs {
			dec.Decode(pair.Data[0], pair.Data[1])
		}
	}

	regions := dec.StyledRegions()
	if len(regions) == 0 {
		t.Fatal("expected styled regions after decoding H.265 captions")
	}

	totalText := ""
	for _, row := range regions[0].Rows {
		for _, span := range row.Spans {
			totalText += span.Text
		}
	}
	if totalText != "World!" {
		t.Errorf("styled text: got %q, want %q", totalText, "World!")
	}

	frame := &CaptionFrame{Channel: 1, Regions: regions}
	wire := frame.Serialize()
	decoded := DeserializeCaptionFrame(wire)
	if decoded == nil {
		t.Fatal("failed to deserialize")
	}
	if decoded.PlainText() != "World!" {
		t.Errorf("round-trip text: got %q, want %q", decoded.PlainText(), "World!")
	}

	t.Logf("H.265 E2E styled: %d region(s), %d row(s), serialized=%d bytes",
		len(regions), len(regions[0].Rows), len(wire))
}

func TestE2E_H264_vs_H265_SamePayload(t *testing.T) {
	triplets := []byte{
		0xFC, 0x14, 0x25, // RU2
		0xFC, 'T', 'e',
		0xFC, 's', 't',
	}

	h264NAL := buildSEINAL(triplets, 3)
	h265NAL := buildHEVCSEINAL(triplets, 3)

	cd264 := ExtractCaptions(h264NAL)
	cd265 := ExtractCaptionsHEVC(h265NAL)

	if cd264 == nil || cd265 == nil {
		t.Fatal("both should return non-nil")
	}

	if len(cd264.CC608Pairs) != len(cd265.CC608Pairs) {
		t.Fatalf("pair count mismatch: h264=%d, h265=%d",
			len(cd264.CC608Pairs), len(cd265.CC608Pairs))
	}

	for i := range cd264.CC608Pairs {
		if cd264.CC608Pairs[i] != cd265.CC608Pairs[i] {
			t.Errorf("pair %d mismatch: h264=%v, h265=%v",
				i, cd264.CC608Pairs[i], cd265.CC608Pairs[i])
		}
	}

	dec264 := NewCEA608Decoder()
	dec265 := NewCEA608Decoder()
	var text264, text265 string

	for _, pair := range cd264.CC608Pairs {
		if t := dec264.Decode(pair.Data[0], pair.Data[1]); t != "" {
			text264 = t
		}
	}
	for _, pair := range cd265.CC608Pairs {
		if t := dec265.Decode(pair.Data[0], pair.Data[1]); t != "" {
			text265 = t
		}
	}

	if text264 != text265 {
		t.Errorf("decoded text differs: h264=%q, h265=%q", text264, text265)
	}
	if text264 != "Test" {
		t.Errorf("decoded text: got %q, want %q", text264, "Test")
	}
}
