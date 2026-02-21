package ccx

import "testing"

func TestCaptionFrame_SerializeDeserialize_Legacy(t *testing.T) {
	f := &CaptionFrame{Channel: 1, Text: "Hello captions"}

	data := f.Serialize()
	got := DeserializeCaptionFrame(data)
	if got == nil {
		t.Fatal("DeserializeCaptionFrame returned nil")
	}
	if got.Channel != 1 {
		t.Errorf("Channel: got %d, want 1", got.Channel)
	}
	if got.Text != "Hello captions" {
		t.Errorf("Text: got %q, want %q", got.Text, "Hello captions")
	}
}

func TestCaptionFrame_SerializeDeserialize_Structured(t *testing.T) {
	f := &CaptionFrame{
		Channel: 7,
		Regions: []CaptionRegion{
			{
				ID: 0, Justify: 2, ScrollDirection: 1, WordWrap: true,
				FillOpacity: 1, BorderType: 3,
				FillColor: "330000", BorderColor: "ff00ff",
				AnchorV: 50, AnchorH: 100, AnchorID: 7,
				RelativeToggle: true, Priority: 5,
				Rows: []CaptionRow{
					{
						Row: 0,
						Spans: []CaptionSpan{
							{Text: "Hello ", FgColor: "ffffff", BgColor: "000000", PenSize: 1, FontTag: 2, Offset: 1, EdgeColor: "000000"},
							{Text: "World", FgColor: "ff0000", BgColor: "0000ff", BgOpacity: 2, Italic: true, Underline: true, Flash: true, PenSize: 2, FontTag: 5, Offset: 2, EdgeType: 3, EdgeColor: "00ff00"},
						},
					},
					{Row: 1, Spans: []CaptionSpan{{Text: "Line two", FgColor: "00ff00", BgColor: "000000", EdgeColor: "000000"}}},
				},
			},
		},
	}

	data := f.Serialize()
	got := DeserializeCaptionFrame(data)

	if got == nil {
		t.Fatal("DeserializeCaptionFrame returned nil")
	}
	if got.Channel != 7 {
		t.Errorf("Channel: got %d, want 7", got.Channel)
	}
	if len(got.Regions) != 1 {
		t.Fatalf("Regions: got %d, want 1", len(got.Regions))
	}

	reg := got.Regions[0]
	if reg.Justify != 2 {
		t.Errorf("Justify: got %d, want 2", reg.Justify)
	}
	if reg.ScrollDirection != 1 {
		t.Errorf("ScrollDirection: got %d, want 1", reg.ScrollDirection)
	}
	if !reg.WordWrap {
		t.Error("WordWrap should be true")
	}
	if reg.FillColor != "330000" {
		t.Errorf("FillColor: got %q, want %q", reg.FillColor, "330000")
	}
	if reg.BorderColor != "ff00ff" {
		t.Errorf("BorderColor: got %q, want %q", reg.BorderColor, "ff00ff")
	}
	if reg.AnchorV != 50 {
		t.Errorf("AnchorV: got %d, want 50", reg.AnchorV)
	}
	if reg.AnchorH != 100 {
		t.Errorf("AnchorH: got %d, want 100", reg.AnchorH)
	}
	if reg.AnchorID != 7 {
		t.Errorf("AnchorID: got %d, want 7", reg.AnchorID)
	}
	if !reg.RelativeToggle {
		t.Error("RelativeToggle should be true")
	}
	if reg.Priority != 5 {
		t.Errorf("Priority: got %d, want 5", reg.Priority)
	}

	if len(reg.Rows) != 2 {
		t.Fatalf("Rows: got %d, want 2", len(reg.Rows))
	}

	s0 := reg.Rows[0].Spans[0]
	if s0.Text != "Hello " {
		t.Errorf("Span[0] Text: got %q", s0.Text)
	}
	if s0.PenSize != 1 {
		t.Errorf("Span[0] PenSize: got %d, want 1", s0.PenSize)
	}
	if s0.FontTag != 2 {
		t.Errorf("Span[0] FontTag: got %d, want 2", s0.FontTag)
	}

	s1 := reg.Rows[0].Spans[1]
	if !s1.Italic || !s1.Underline || !s1.Flash {
		t.Error("Span[1] should be italic, underline, flash")
	}
	if s1.BgOpacity != 2 {
		t.Errorf("Span[1] BgOpacity: got %d, want 2", s1.BgOpacity)
	}
	if s1.EdgeType != 3 {
		t.Errorf("Span[1] EdgeType: got %d, want 3", s1.EdgeType)
	}
	if s1.EdgeColor != "00ff00" {
		t.Errorf("Span[1] EdgeColor: got %q, want %q", s1.EdgeColor, "00ff00")
	}
}

func TestCaptionFrame_PlainText(t *testing.T) {
	f := &CaptionFrame{
		Regions: []CaptionRegion{
			{Rows: []CaptionRow{
				{Spans: []CaptionSpan{{Text: "Hello "}, {Text: "World"}}},
				{Spans: []CaptionSpan{{Text: "Line 2"}}},
			}},
		},
	}
	got := f.PlainText()
	want := "Hello World\nLine 2"
	if got != want {
		t.Errorf("PlainText: got %q, want %q", got, want)
	}
}

func TestCaptionFrame_PlainText_Legacy(t *testing.T) {
	f := &CaptionFrame{Text: "legacy text"}
	if f.PlainText() != "legacy text" {
		t.Errorf("got %q", f.PlainText())
	}
}

func TestHexToRGB(t *testing.T) {
	tests := []struct {
		hex     string
		r, g, b byte
	}{
		{"ffffff", 0xFF, 0xFF, 0xFF}, {"000000", 0x00, 0x00, 0x00},
		{"ff0000", 0xFF, 0x00, 0x00}, {"00ff00", 0x00, 0xFF, 0x00},
		{"0000ff", 0x00, 0x00, 0xFF}, {"aabb55", 0xAA, 0xBB, 0x55},
	}
	for _, tt := range tests {
		rgb := hexToRGB(tt.hex)
		if rgb[0] != tt.r || rgb[1] != tt.g || rgb[2] != tt.b {
			t.Errorf("hexToRGB(%q): got [%02x,%02x,%02x], want [%02x,%02x,%02x]",
				tt.hex, rgb[0], rgb[1], rgb[2], tt.r, tt.g, tt.b)
		}
	}
}

func TestRGBToHex(t *testing.T) {
	tests := []struct {
		r, g, b byte
		want    string
	}{
		{0xFF, 0xFF, 0xFF, "ffffff"}, {0x00, 0x00, 0x00, "000000"},
		{0xFF, 0x00, 0x00, "ff0000"}, {0xAA, 0xBB, 0x55, "aabb55"},
	}
	for _, tt := range tests {
		got := rgbToHex(tt.r, tt.g, tt.b)
		if got != tt.want {
			t.Errorf("rgbToHex(%02x,%02x,%02x): got %q, want %q", tt.r, tt.g, tt.b, got, tt.want)
		}
	}
}

func TestCaptionFrame_EdgeType_AllValues(t *testing.T) {
	for et := 0; et <= 5; et++ {
		f := &CaptionFrame{
			Channel: 1,
			Regions: []CaptionRegion{
				{Rows: []CaptionRow{
					{Row: 0, Spans: []CaptionSpan{
						{Text: "x", FgColor: "ffffff", BgColor: "000000", EdgeColor: "ff0000", EdgeType: et},
					}},
				}},
			},
		}
		data := f.Serialize()
		got := DeserializeCaptionFrame(data)
		if got == nil {
			t.Fatalf("EdgeType %d: deserialize returned nil", et)
		}
		span := got.Regions[0].Rows[0].Spans[0]
		if span.EdgeType != et {
			t.Errorf("EdgeType %d: round-trip got %d", et, span.EdgeType)
		}
	}
}
