package ccx

import (
	"testing"
)

func TestExtractCEA608_ValidSEI(t *testing.T) {
	nalHeader := byte(0x06)
	payloadType := byte(4)

	t35 := []byte{
		0xB5, 0x00, 0x31,
		'G', 'A', '9', '4',
		0x03, 0x43, 0xFF,
		0xFC, 'H', 'i',
		0xFC, '!', 0x00,
		0xF8, 0x00, 0x00,
	}

	payloadSize := byte(len(t35))
	nalData := []byte{nalHeader, payloadType, payloadSize}
	nalData = append(nalData, t35...)

	pairs := ExtractCEA608(nalData)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}

	if pairs[0].Data[0] != 'H' || pairs[0].Data[1] != 'i' {
		t.Errorf("pair 0: got [%02x, %02x], want ['H', 'i']", pairs[0].Data[0], pairs[0].Data[1])
	}
	if pairs[0].Channel != 1 {
		t.Errorf("pair 0 channel: got %d, want 1", pairs[0].Channel)
	}

	if pairs[1].Data[0] != '!' || pairs[1].Data[1] != 0x00 {
		t.Errorf("pair 1: got [%02x, %02x], want ['!', 0x00]", pairs[1].Data[0], pairs[1].Data[1])
	}
}

func TestExtractCEA608_NullPairsSkipped(t *testing.T) {
	nalData := buildSEINAL([]byte{
		0xFC, 0x00, 0x00,
		0xFC, 'A', 'B',
	}, 2)

	pairs := ExtractCEA608(nalData)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Data[0] != 'A' || pairs[0].Data[1] != 'B' {
		t.Errorf("pair 0: got [%02x, %02x], want ['A', 'B']", pairs[0].Data[0], pairs[0].Data[1])
	}
}

func TestExtractCEA608_Field2(t *testing.T) {
	nalData := buildSEINAL([]byte{
		0xFD, 'A', 'B',
		0xFC, 'C', 'D',
	}, 2)

	pairs := ExtractCEA608(nalData)
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}

	if pairs[0].Channel != 3 {
		t.Errorf("pair 0 channel: got %d, want 3", pairs[0].Channel)
	}
	if pairs[1].Channel != 1 {
		t.Errorf("pair 1 channel: got %d, want 1", pairs[1].Channel)
	}
}

func TestExtractCEA608_ChannelRouting(t *testing.T) {
	nalData := buildSEINAL([]byte{
		0xFC, 0x94, 0xA5,
		0xFC, 'H', 'i',
		0xFC, 0x9C, 0xA5,
		0xFC, 'A', 'B',
		0xFC, 0x94, 0xA0,
		0xFC, 'O', 'K',
	}, 6)

	pairs := ExtractCEA608(nalData)
	if len(pairs) != 6 {
		t.Fatalf("expected 6 pairs, got %d", len(pairs))
	}

	wantChannels := []int{1, 1, 2, 2, 1, 1}
	for i, want := range wantChannels {
		if pairs[i].Channel != want {
			t.Errorf("pair %d: got channel %d, want %d (data=[%02x,%02x])",
				i, pairs[i].Channel, want, pairs[i].Data[0], pairs[i].Data[1])
		}
	}
}

func TestExtractCEA608_PACBit0StaysOnChannel(t *testing.T) {
	nalData := buildSEINAL([]byte{
		0xFC, 0x91, 0xE0,
		0xFC, 'A', 'B',
		0xFC, 0x95, 0xE0,
		0xFC, 'C', 'D',
	}, 4)

	pairs := ExtractCEA608(nalData)
	if len(pairs) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}

	for i, pair := range pairs {
		if pair.Channel != 1 {
			t.Errorf("pair %d: got channel %d, want 1 (data=[%02x,%02x])",
				i, pair.Channel, pair.Data[0], pair.Data[1])
		}
	}
}

func TestExtractCEA608_Field2ChannelRouting(t *testing.T) {
	nalData := buildSEINAL([]byte{
		0xFD, 0x94, 0xA5,
		0xFD, 'X', 'Y',
		0xFD, 0x9C, 0xA5,
		0xFD, 'Q', 'R',
	}, 4)

	pairs := ExtractCEA608(nalData)
	if len(pairs) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}

	wantChannels := []int{3, 3, 4, 4}
	for i, want := range wantChannels {
		if pairs[i].Channel != want {
			t.Errorf("pair %d: got channel %d, want %d (data=[%02x,%02x])",
				i, pairs[i].Channel, want, pairs[i].Data[0], pairs[i].Data[1])
		}
	}
}

func TestExtractCEA608_DTVCC_Skipped(t *testing.T) {
	nalData := buildSEINAL([]byte{
		0xFE, 0x01, 0x02,
		0xFF, 0x03, 0x04,
		0xFC, 'X', 'Y',
	}, 3)

	pairs := ExtractCEA608(nalData)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Data[0] != 'X' || pairs[0].Data[1] != 'Y' {
		t.Errorf("pair 0: got [%02x, %02x], want ['X', 'Y']", pairs[0].Data[0], pairs[0].Data[1])
	}
}

func TestExtractCaptions_DTVCC(t *testing.T) {
	nalData := buildSEINAL([]byte{
		0xFE, 0xAA, 0xBB,
		0xFF, 0xCC, 0xDD,
		0xFC, 'H', 'i',
	}, 3)

	cd := ExtractCaptions(nalData)
	if cd == nil {
		t.Fatal("expected non-nil CaptionData")
	}

	if len(cd.CC608Pairs) != 1 {
		t.Fatalf("expected 1 CEA-608 pair, got %d", len(cd.CC608Pairs))
	}
	if cd.CC608Pairs[0].Data[0] != 'H' || cd.CC608Pairs[0].Data[1] != 'i' {
		t.Errorf("CEA-608 pair: got [%02x, %02x], want ['H', 'i']",
			cd.CC608Pairs[0].Data[0], cd.CC608Pairs[0].Data[1])
	}

	if len(cd.DTVCC) != 2 {
		t.Fatalf("expected 2 DTVCC pairs, got %d", len(cd.DTVCC))
	}
	if cd.DTVCC[0].Data != [2]byte{0xAA, 0xBB} || cd.DTVCC[0].Start {
		t.Errorf("DTVCC[0]: got data=%v start=%v, want [AA BB] start=false",
			cd.DTVCC[0].Data, cd.DTVCC[0].Start)
	}
	if cd.DTVCC[1].Data != [2]byte{0xCC, 0xDD} || !cd.DTVCC[1].Start {
		t.Errorf("DTVCC[1]: got data=%v start=%v, want [CC DD] start=true",
			cd.DTVCC[1].Data, cd.DTVCC[1].Start)
	}
}

func TestExtractCEA608_NotT35(t *testing.T) {
	nalData := []byte{0x06, 0x05, 0x03, 0x01, 0x02, 0x03}
	pairs := ExtractCEA608(nalData)
	if pairs != nil {
		t.Errorf("expected nil for non-T35 SEI, got %v", pairs)
	}
}

func TestExtractCEA608_WrongCountryCode(t *testing.T) {
	nalData := buildSEINALCustom(0x00, 0x00, 0x31, []byte{0xFC, 'A', 'B'}, 1)
	pairs := ExtractCEA608(nalData)
	if pairs != nil {
		t.Errorf("expected nil for wrong country code, got %v", pairs)
	}
}

func TestExtractCEA608_ParityStripped(t *testing.T) {
	nalData := buildSEINAL([]byte{
		0xFC, 0xC8, 0xE9,
	}, 1)

	pairs := ExtractCEA608(nalData)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Data[0] != 'H' || pairs[0].Data[1] != 'i' {
		t.Errorf("pair 0: got [%02x, %02x], want ['H', 'i']", pairs[0].Data[0], pairs[0].Data[1])
	}
}

func TestExtractCEA608_EmptyNAL(t *testing.T) {
	pairs := ExtractCEA608(nil)
	if pairs != nil {
		t.Errorf("expected nil for nil input, got %v", pairs)
	}

	pairs = ExtractCEA608([]byte{0x06})
	if pairs != nil {
		t.Errorf("expected nil for short input, got %v", pairs)
	}
}

func TestRemoveEPB(t *testing.T) {
	input := []byte{0x00, 0x00, 0x03, 0x01, 0xFF}
	got := removeEPB(input)
	want := []byte{0x00, 0x00, 0x01, 0xFF}
	if len(got) != len(want) {
		t.Fatalf("removeEPB: got len %d, want len %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("removeEPB[%d]: got 0x%02x, want 0x%02x", i, got[i], want[i])
		}
	}
}

func buildSEINAL(triplets []byte, ccCount int) []byte {
	return buildSEINALCustom(0xB5, 0x00, 0x31, triplets, ccCount)
}

func buildSEINALCustom(country, provHi, provLo byte, triplets []byte, ccCount int) []byte {
	t35 := []byte{
		country,
		provHi, provLo,
		'G', 'A', '9', '4',
		0x03,
		0x40 | byte(ccCount&0x1F),
		0xFF,
	}
	t35 = append(t35, triplets...)

	nalData := []byte{0x06, 0x04, byte(len(t35))}
	nalData = append(nalData, t35...)
	return nalData
}
