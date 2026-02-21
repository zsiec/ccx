package ccx

import "testing"

func TestCEA708_BasicText(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x02, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Hello"))
	text := svc.DisplayText()
	if text != "Hello" {
		t.Errorf("got %q, want %q", text, "Hello")
	}
}

func TestCEA708_G0MusicNote(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x7F})
	text := svc.DisplayText()
	if text != "♪" {
		t.Errorf("got %q, want %q", text, "♪")
	}
}

func TestCEA708_G1Latin1(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0xC9})
	text := svc.DisplayText()
	if text != "É" {
		t.Errorf("got %q, want %q", text, "É")
	}
}

func TestCEA708_Backspace(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("AB"))
	svc.ProcessBlock([]byte{0x08})
	text := svc.DisplayText()
	if text != "A" {
		t.Errorf("got %q, want %q", text, "A")
	}
}

func TestCEA708_CarriageReturn(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x02, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Line1"))
	svc.ProcessBlock([]byte{0x0D})
	svc.ProcessBlock([]byte("Line2"))
	text := svc.DisplayText()
	if text != "Line1\nLine2" {
		t.Errorf("got %q, want %q", text, "Line1\nLine2")
	}
}

func TestCEA708_CarriageReturnRollUp(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x01, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Row1"))
	svc.ProcessBlock([]byte{0x0D})
	svc.ProcessBlock([]byte("Row2"))
	svc.ProcessBlock([]byte{0x0D})
	svc.ProcessBlock([]byte("Row3"))
	text := svc.DisplayText()
	if text != "Row2\nRow3" {
		t.Errorf("got %q, want %q", text, "Row2\nRow3")
	}
}

func TestCEA708_FormFeed(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x02, 0x1F, 0x11})
	svc.ProcessBlock([]byte("OldText"))
	svc.ProcessBlock([]byte{0x0C})
	svc.ProcessBlock([]byte("New"))
	text := svc.DisplayText()
	if text != "New" {
		t.Errorf("got %q, want %q", text, "New")
	}
}

func TestCEA708_HCR(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x02, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Hello"))
	svc.ProcessBlock([]byte{0x0E})
	svc.ProcessBlock([]byte("Bye"))
	text := svc.DisplayText()
	if text != "Bye" {
		t.Errorf("got %q, want %q", text, "Bye")
	}
}

func TestCEA708_SetPenLocation(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x02, 0x1F, 0x11})
	svc.ProcessBlock([]byte("First"))
	svc.ProcessBlock([]byte{0x92, 0x01, 0x00})
	svc.ProcessBlock([]byte("Second"))
	text := svc.DisplayText()
	if text != "First\nSecond" {
		t.Errorf("got %q, want %q", text, "First\nSecond")
	}
}

func TestCEA708_ClearWindow(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x02, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Before"))
	svc.DisplayText()
	svc.ProcessBlock([]byte{0x88, 0x01})
	svc.ProcessBlock([]byte{0x92, 0x00, 0x00})
	svc.ProcessBlock([]byte("After"))
	text := svc.DisplayText()
	if text != "After" {
		t.Errorf("got %q, want %q", text, "After")
	}
}

func TestCEA708_HideShowWindow(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Visible"))
	text := svc.DisplayText()
	if text != "Visible" {
		t.Errorf("got %q, want %q", text, "Visible")
	}

	svc.ProcessBlock([]byte{0x8A, 0x01})
	text = svc.DisplayText()
	if text != "" {
		t.Errorf("after HDW: got %q, want %q", text, "")
	}
}

func TestCEA708_WasHiddenClearsOnText(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x00, 0x00, 0x00, 0x02, 0x29, 0x11})
	svc.ProcessBlock([]byte("OldCaption"))
	svc.ProcessBlock([]byte{0x89, 0x01})
	text := svc.DisplayText()
	if text != "OldCaption" {
		t.Errorf("caption 1: got %q, want %q", text, "OldCaption")
	}

	svc.ProcessBlock([]byte{0x8A, 0xFF})
	svc.ProcessBlock([]byte{0x8B, 0x01})

	text = svc.DisplayText()
	if text != "" {
		t.Errorf("after TGW should not re-emit old: got %q", text)
	}

	svc.ProcessBlock([]byte("NewCaption"))
	text = svc.DisplayText()
	if text != "NewCaption" {
		t.Errorf("new caption: got %q, want %q", text, "NewCaption")
	}
}

func TestCEA708_DeleteWindow(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Data"))

	svc.ProcessBlock([]byte{0x8C, 0x01})
	if svc.windows[0].defined {
		t.Error("window 0 should not be defined after DLW")
	}
}

func TestCEA708_DefineWindowStyle(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x02, 0x1F, 0x11})
	if svc.windows[0].justify != JustifyCenter {
		t.Errorf("justify: got %d, want %d", svc.windows[0].justify, JustifyCenter)
	}
}

func TestCEA708_ETXSignalsChange(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Test"))
	svc.DisplayText()

	changed := svc.ProcessBlock([]byte{0x03})
	if !changed {
		t.Error("ETX should signal change")
	}
}

func TestCEA708_G2Characters(t *testing.T) {
	tests := []struct {
		code byte
		want rune
	}{
		{0x25, '\u2026'}, {0x31, '\u2018'}, {0x32, '\u2019'},
		{0x33, '\u201c'}, {0x34, '\u201d'}, {0x35, '\u2022'},
		{0x39, '\u2122'},
	}

	for _, tt := range tests {
		svc := NewCEA708Service()
		svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
		svc.ProcessBlock([]byte{0x10, tt.code})
		text := svc.DisplayText()
		if text != string(tt.want) {
			t.Errorf("G2 0x%02x: got %q, want %q", tt.code, text, string(tt.want))
		}
	}
}

func TestCEA708_G2ExtendedLatin(t *testing.T) {
	tests := []struct {
		code byte
		want rune
	}{
		{0x40, '\u00c0'}, {0x49, '\u00c9'}, {0x51, '\u00d1'},
		{0x60, '\u00e0'}, {0x69, '\u00e9'}, {0x71, '\u00f1'},
		{0x75, '\u00f5'},
	}

	for _, tt := range tests {
		svc := NewCEA708Service()
		svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
		svc.ProcessBlock([]byte{0x10, tt.code})
		text := svc.DisplayText()
		if text != string(tt.want) {
			t.Errorf("G2 extended 0x%02x: got %q, want %q", tt.code, text, string(tt.want))
		}
	}
}

func TestCEA708_P16Character(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x18, 0x00, 0x41})
	text := svc.DisplayText()
	if text != "A" {
		t.Errorf("got %q, want %q", text, "A")
	}
}

func TestCEA708_MultipleWindows(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Win0"))

	svc.ProcessBlock([]byte{0x99, 0x20, 0x10, 0x10, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Win1"))

	text := svc.DisplayText()
	if text != "Win0\nWin1" {
		t.Errorf("got %q, want %q", text, "Win0\nWin1")
	}
}

func TestCEA708_ColorToHex(t *testing.T) {
	tests := []struct {
		input byte
		want  string
	}{
		{0x00, "000000"}, {0x3F, "ffffff"}, {0x30, "ff0000"},
		{0x0C, "00ff00"}, {0x03, "0000ff"}, {0x15, "555555"},
		{0x2A, "aaaaaa"},
	}
	for _, tt := range tests {
		got := cea708ColorToHex(tt.input)
		if got != tt.want {
			t.Errorf("cea708ColorToHex(0x%02x): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCEA708_ColorToHex_AllValues(t *testing.T) {
	for b := 0; b < 64; b++ {
		hex := cea708ColorToHex(byte(b))
		if len(hex) != 6 {
			t.Errorf("cea708ColorToHex(0x%02x): got len %d, want 6", b, len(hex))
		}
	}
}

func TestCEA708_ParseDTVCCPacket(t *testing.T) {
	header := byte(0x43)
	serviceHeader := byte(1<<5 | 2)
	packet := []byte{header, serviceHeader, 'H', 'i', 0x00, 0x00}

	blocks := ParseDTVCCPacket(packet)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0].ServiceNum != 1 {
		t.Errorf("service: got %d, want 1", blocks[0].ServiceNum)
	}
	if string(blocks[0].Data) != "Hi" {
		t.Errorf("data: got %q, want %q", string(blocks[0].Data), "Hi")
	}
}

func TestCEA708_PacketSize(t *testing.T) {
	tests := []struct {
		header byte
		want   int
	}{
		{0x00, 128}, {0x01, 2}, {0x06, 12}, {0x3F, 126},
	}
	for _, tt := range tests {
		got := DTVCCPacketSize(tt.header)
		if got != tt.want {
			t.Errorf("DTVCCPacketSize(0x%02x): got %d, want %d", tt.header, got, tt.want)
		}
	}
}

func TestCEA708_ReservedC1_SingleByte(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x93, 'A'})
	text := svc.DisplayText()
	if text != "A" {
		t.Errorf("reserved C1 0x93 should consume 1 byte; got %q, want %q", text, "A")
	}
}

func TestCEA708_ReservedC1_AllValues(t *testing.T) {
	for cmd := byte(0x93); cmd <= 0x96; cmd++ {
		svc := NewCEA708Service()
		svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
		svc.ProcessBlock([]byte{cmd, 'X'})
		text := svc.DisplayText()
		if text != "X" {
			t.Errorf("reserved C1 0x%02x should consume 1 byte; got %q, want %q", cmd, text, "X")
		}
	}
}

func TestCEA708_SPA_FullParse(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x90, 0x39, 0xCB})

	w := &svc.windows[svc.currentWin]
	if w.pen.penSize != PenSizeStandard {
		t.Errorf("penSize: got %d, want %d", w.pen.penSize, PenSizeStandard)
	}
	if w.pen.offset != PenOffsetSuperscript {
		t.Errorf("offset: got %d, want %d", w.pen.offset, PenOffsetSuperscript)
	}
	if w.pen.textTag != 3 {
		t.Errorf("textTag: got %d, want 3", w.pen.textTag)
	}
	if w.pen.fontTag != FontMonoSans {
		t.Errorf("fontTag: got %d, want %d", w.pen.fontTag, FontMonoSans)
	}
	if w.pen.edgeType != EdgeRaised {
		t.Errorf("edgeType: got %d, want %d", w.pen.edgeType, EdgeRaised)
	}
	if !w.pen.italic {
		t.Error("italic should be set")
	}
	if !w.pen.underline {
		t.Error("underline should be set")
	}
}

func TestCEA708_SPC_FullParse(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x91, 0x7F, 0xC0, 0x30})

	w := &svc.windows[svc.currentWin]
	if w.pen.fgOpacity != OpacityFlash {
		t.Errorf("fgOpacity: got %d, want %d", w.pen.fgOpacity, OpacityFlash)
	}
	if w.pen.bgOpacity != OpacityTransparent {
		t.Errorf("bgOpacity: got %d, want %d", w.pen.bgOpacity, OpacityTransparent)
	}
	if w.pen.fgColor != "ffffff" {
		t.Errorf("fgColor: got %q, want %q", w.pen.fgColor, "ffffff")
	}
	if w.pen.edgeColor != "ff0000" {
		t.Errorf("edgeColor: got %q, want %q", w.pen.edgeColor, "ff0000")
	}
}

func TestCEA708_SWA_FullParse(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x97, 0x00, 0x00, 0x42, 0x20})

	w := &svc.windows[svc.currentWin]
	if w.justify != JustifyCenter {
		t.Errorf("justify: got %d, want %d", w.justify, JustifyCenter)
	}
	if !w.wordWrap {
		t.Error("wordWrap should be true")
	}
}

func TestCEA708_DefineWindow_Priority(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x23, 0x00, 0x00, 0x02, 0x1F, 0x11})

	w := &svc.windows[0]
	if w.priority != 3 {
		t.Errorf("priority: got %d, want 3", w.priority)
	}
}

func TestCEA708_DefineWindow_Locks(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x38, 0x00, 0x00, 0x02, 0x1F, 0x11})

	w := &svc.windows[0]
	if !w.rowLock {
		t.Error("rowLock should be true")
	}
	if !w.colLock {
		t.Error("colLock should be true")
	}
}

func TestCEA708_PredefinedPenStyle(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x02})
	w := &svc.windows[0]
	if w.pen.fontTag != FontMonoSerif {
		t.Errorf("pen style 2: fontTag got %d, want %d", w.pen.fontTag, FontMonoSerif)
	}

	svc2 := NewCEA708Service()
	svc2.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x05})
	w2 := &svc2.windows[0]
	if w2.pen.fontTag != FontPropSans {
		t.Errorf("pen style 5: fontTag got %d, want %d", w2.pen.fontTag, FontPropSans)
	}
	if w2.pen.bgOpacity != OpacityTransparent {
		t.Errorf("pen style 5: bgOpacity got %d, want %d", w2.pen.bgOpacity, OpacityTransparent)
	}
}

func TestCEA708_PredefinedWindowStyle(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x18})
	w := &svc.windows[0]
	if !w.wordWrap {
		t.Error("window style 3 should enable word wrap")
	}
	if w.justify != JustifyLeft {
		t.Errorf("window style 3: justify got %d, want %d", w.justify, JustifyLeft)
	}
}

func TestCEA708_StyledCharCarriesAllAttributes(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x90, 0x09, 0xC9})
	svc.ProcessBlock([]byte{0x91, 0x7F, 0x80, 0x30})
	svc.ProcessBlock([]byte("X"))

	w := &svc.windows[0]
	cell := w.memory[0][0]
	if cell == nil {
		t.Fatal("expected styled char")
	}
	if !cell.italic {
		t.Error("char should be italic")
	}
	if !cell.underline {
		t.Error("char should be underlined")
	}
	if cell.fgColor != "ffffff" {
		t.Errorf("fgColor: got %q, want %q", cell.fgColor, "ffffff")
	}
	if cell.fgOpacity != OpacityFlash {
		t.Errorf("fgOpacity: got %d, want %d", cell.fgOpacity, OpacityFlash)
	}
	if cell.edgeColor != "ff0000" {
		t.Errorf("edgeColor: got %q, want %q", cell.edgeColor, "ff0000")
	}
}

func TestCEA708_StyledRegions_Basic(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x02, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Hello"))

	regions := svc.StyledRegions()
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	totalText := ""
	for _, span := range regions[0].Rows[0].Spans {
		totalText += span.Text
	}
	if totalText != "Hello" {
		t.Errorf("text: got %q, want %q", totalText, "Hello")
	}
}

func TestCEA708_StyledRegions_WithStyle(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("AB"))
	svc.ProcessBlock([]byte{0x90, 0x01, 0x80})
	svc.ProcessBlock([]byte("CD"))

	regions := svc.StyledRegions()
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	row := regions[0].Rows[0]
	if len(row.Spans) < 2 {
		t.Fatalf("expected at least 2 spans, got %d", len(row.Spans))
	}
	if row.Spans[0].Italic {
		t.Error("first span should not be italic")
	}
	if !row.Spans[1].Italic {
		t.Error("second span should be italic")
	}
}

func TestCEA708_StyledRegions_Invisible(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x00, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Hidden"))

	regions := svc.StyledRegions()
	if len(regions) != 0 {
		t.Errorf("invisible window should produce 0 regions, got %d", len(regions))
	}
}

func TestCEA708_StyledRegions_MultiWindow(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Win0"))
	svc.ProcessBlock([]byte{0x99, 0x20, 0x10, 0x10, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Win1"))

	regions := svc.StyledRegions()
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(regions))
	}
}

func TestCEA708_StyledRegions_Justify(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x97, 0x00, 0x00, 0x02, 0x00})
	svc.ProcessBlock([]byte("Centered"))

	regions := svc.StyledRegions()
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	if regions[0].Justify != 2 {
		t.Errorf("justify: got %d, want 2", regions[0].Justify)
	}
}

func TestCEA708_RST(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Text"))
	svc.DisplayText()

	svc.ProcessBlock([]byte{0x8F})
	for i := 0; i < 8; i++ {
		if svc.windows[i].defined {
			t.Errorf("window %d should not be defined after RST", i)
		}
	}
}

func TestCEA708_ToggleWindows(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte("Toggle"))
	text := svc.DisplayText()
	if text != "Toggle" {
		t.Errorf("initial: got %q, want %q", text, "Toggle")
	}

	svc.ProcessBlock([]byte{0x8B, 0x01})
	if svc.windows[0].visible {
		t.Error("window should be hidden after toggle")
	}

	svc.ProcessBlock([]byte{0x8B, 0x01})
	if !svc.windows[0].visible {
		t.Error("window should be visible after second toggle")
	}
}

func TestCEA708_G2_FractionChars(t *testing.T) {
	tests := []struct {
		code byte
		want rune
	}{
		{0x76, '\u215b'}, {0x77, '\u215c'},
		{0x78, '\u215d'}, {0x79, '\u215e'},
	}
	for _, tt := range tests {
		svc := NewCEA708Service()
		svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
		svc.ProcessBlock([]byte{0x10, tt.code})
		text := svc.DisplayText()
		if text != string(tt.want) {
			t.Errorf("G2 0x%02x: got %q, want %q", tt.code, text, string(tt.want))
		}
	}
}

func TestCEA708_G3_CCIcon(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x10, 0xA0})
	text := svc.DisplayText()
	if text != "\U0001F4AC" {
		t.Errorf("G3 0xA0: got %q, want speech bubble", text)
	}
}

func TestCEA708_C2_VariableLength(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x10, 0x00, 'A'})
	text := svc.DisplayText()
	if text != "A" {
		t.Errorf("C2 short: got %q, want %q", text, "A")
	}
}

func TestCEA708_WindowCurrentAfterDefine(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x99, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	if svc.currentWin != 1 {
		t.Errorf("currentWin: got %d, want 1", svc.currentWin)
	}
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	if svc.currentWin != 0 {
		t.Errorf("currentWin: got %d, want 0", svc.currentWin)
	}
}

func TestCEA708_SWA_BitFields(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})
	svc.ProcessBlock([]byte{0x97, 0x00, 0x40, 0x00, 0x52})

	w := &svc.windows[svc.currentWin]
	if w.displayEffect != DisplayEffect(1) {
		t.Errorf("displayEffect: got %d, want 1 (fade)", w.displayEffect)
	}
	if w.effectDirection != 2 {
		t.Errorf("effectDirection: got %d, want 2", w.effectDirection)
	}
	if w.effectSpeed != 1 {
		t.Errorf("effectSpeed: got %d, want 1", w.effectSpeed)
	}
	if w.borderType != BorderType(1) {
		t.Errorf("borderType: got %d, want 1", w.borderType)
	}
}

func TestCEA708_TGW_SetsWasHidden(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x00, 0x1F, 0x11})

	w := &svc.windows[0]
	if !w.visible {
		t.Fatal("window should be visible after define")
	}

	svc.ProcessBlock([]byte{0x8B, 0x01})
	if w.visible {
		t.Error("window should be hidden after TGW")
	}
	if !w.wasHidden {
		t.Error("wasHidden should be true after TGW hides a visible window")
	}
}

func TestCEA708_TextPreservesEmptyRows(t *testing.T) {
	svc := NewCEA708Service()
	svc.ProcessBlock([]byte{0x98, 0x20, 0x00, 0x00, 0x03, 0x1F, 0x11})

	w := &svc.windows[0]
	w.memory[0][0] = &cea708StyledChar{ch: 'H'}
	w.memory[0][1] = &cea708StyledChar{ch: 'i'}
	w.memory[2][0] = &cea708StyledChar{ch: 'B'}
	w.memory[2][1] = &cea708StyledChar{ch: 'y'}

	got := w.text()
	want := "Hi\n\nBy"
	if got != want {
		t.Errorf("text() with empty row gap: got %q, want %q", got, want)
	}
}
