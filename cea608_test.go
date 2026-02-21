package ccx

import (
	"testing"
)

func TestCEA608_PrintableChars(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode(0x14, 0x25)

	text := dec.Decode('H', 'e')
	if text != "He" {
		t.Errorf("got %q, want %q", text, "He")
	}

	text = dec.Decode('l', 'l')
	if text != "Hell" {
		t.Errorf("got %q, want %q", text, "Hell")
	}

	text = dec.Decode('o', 0)
	if text != "Hello" {
		t.Errorf("got %q, want %q", text, "Hello")
	}
}

func TestCEA608_SpecialChars(t *testing.T) {
	tests := []struct {
		input byte
		want  rune
	}{
		{0x27, '\u2019'}, {0x2A, '\u00e1'}, {0x5C, '\u00e9'},
		{0x5E, '\u00ed'}, {0x5F, '\u00f3'}, {0x60, '\u00fa'},
		{0x7B, '\u00e7'}, {0x7C, '\u00f7'}, {0x7D, '\u00d1'},
		{0x7E, '\u00f1'},
	}

	for _, tt := range tests {
		dec := NewCEA608Decoder()
		dec.Decode(0x14, 0x25)
		text := dec.Decode(tt.input, 0)
		want := string(tt.want)
		if text != want {
			t.Errorf("char 0x%02x: got %q, want %q", tt.input, text, want)
		}
	}
}

func TestCEA608_ControlCodeDedup(t *testing.T) {
	dec := NewCEA608Decoder()

	text := dec.Decode(0x14, 0x25)
	if text != "" {
		t.Errorf("first control should return empty, got %q", text)
	}

	text = dec.Decode(0x14, 0x25)
	if text != "" {
		t.Errorf("duplicate control should return empty, got %q", text)
	}

	dec.Decode('A', 0)
	dec.Decode(0x14, 0x25)
}

func TestCEA608_PopOnMode(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x20)

	text := dec.Decode('T', 'e')
	if text != "" {
		t.Errorf("buffered text should return empty, got %q", text)
	}
	dec.Decode('s', 't')

	text = dec.Decode(0x14, 0x2F)
	if text != "Test" {
		t.Errorf("got %q, want %q", text, "Test")
	}
}

func TestCEA608_PopOnMultiFlip(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x20)
	dec.Decode('O', 'n')
	dec.Decode('e', 0)
	text := dec.Decode(0x14, 0x2F)
	if text != "One" {
		t.Errorf("first flip: got %q, want %q", text, "One")
	}

	dec.Decode(0x14, 0x20)
	dec.Decode('T', 'w')
	dec.Decode('o', 0)
	text = dec.Decode(0x14, 0x2F)
	if text != "Two" {
		t.Errorf("second flip: got %q, want %q", text, "Two")
	}
}

func TestCEA608_EOC_Swaps(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x20)
	dec.Decode('A', 'B')
	dec.Decode(0x14, 0x2F)

	dec.Decode(0x14, 0x20)
	dec.Decode('C', 'D')
	text := dec.Decode(0x14, 0x2F)
	if text != "CD" {
		t.Errorf("second EOC: got %q, want %q", text, "CD")
	}
}

func TestCEA608_RollUp(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)

	dec.Decode('L', 'i')
	dec.Decode('n', 'e')
	text := dec.Decode(' ', '1')
	if text != "Line 1" {
		t.Errorf("got %q, want %q", text, "Line 1")
	}

	dec.Decode(0x14, 0x2D)
	dec.Decode('L', 'i')
	dec.Decode('n', 'e')
	text = dec.Decode(' ', '2')
	if text != "Line 1\nLine 2" {
		t.Errorf("got %q, want %q", text, "Line 1\nLine 2")
	}
}

func TestCEA608_EraseDisplayed(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode('H', 'i')

	text := dec.Decode(0x14, 0x2C)
	if text != "" {
		t.Errorf("erase should clear, got %q", text)
	}
}

func TestCEA608Char(t *testing.T) {
	if cea608Char('A') != 'A' {
		t.Error("'A' should map to 'A'")
	}
	if cea608Char(' ') != ' ' {
		t.Error("space should map to space")
	}
	if cea608Char(0x2A) != '\u00e1' {
		t.Error("0x2A should map to 'á'")
	}
	if cea608Char(0x27) != '\u2019' {
		t.Error("0x27 should map to '''")
	}
	if cea608Char(0x5C) != '\u00e9' {
		t.Error("0x5C should map to 'é'")
	}
	if cea608Char(0x7E) != '\u00f1' {
		t.Error("0x7E should map to 'ñ'")
	}
}

func TestCEA608_Backspace(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode('A', 'B')
	dec.Decode('C', 0)

	text := dec.Decode(0x14, 0x21)
	if text != "AB" {
		t.Errorf("after backspace: got %q, want %q", text, "AB")
	}
}

func TestCEA608_MidrowSpace(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode('H', 'i')

	text := dec.Decode(0x11, 0x20)
	if text != "Hi " {
		t.Errorf("after midrow: got %q, want %q", text, "Hi ")
	}

	text = dec.Decode('O', 'K')
	if text != "Hi OK" {
		t.Errorf("after text: got %q, want %q", text, "Hi OK")
	}
}

func TestCEA608_SpecialCharReplace(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode(' ', 0)
	text := dec.Decode(0x11, 0x37)
	if text != "♪" {
		t.Errorf("special char: got %q, want %q", text, "♪")
	}
}

func TestDecodePACRow(t *testing.T) {
	for cc1 := byte(0x11); cc1 <= 0x17; cc1++ {
		for cc2 := byte(0x40); cc2 <= 0x7F; cc2++ {
			row := decodePACRow(cc1, cc2)
			if row < 0 || row > 14 {
				t.Errorf("decodePACRow(0x%02x, 0x%02x) = %d, want 0-14", cc1, cc2, row)
			}
		}
	}
}

func TestDecodePACRow_KnownValues(t *testing.T) {
	tests := []struct {
		cc1, cc2 byte
		wantRow  int
	}{
		{0x11, 0x40, 0}, {0x11, 0x60, 1}, {0x12, 0x40, 2},
		{0x12, 0x60, 3}, {0x15, 0x40, 4}, {0x15, 0x60, 5},
		{0x16, 0x40, 6}, {0x16, 0x60, 7}, {0x17, 0x40, 8},
		{0x17, 0x60, 9}, {0x10, 0x40, 10}, {0x13, 0x40, 11},
		{0x13, 0x60, 12}, {0x14, 0x40, 13}, {0x14, 0x60, 14},
	}
	for _, tt := range tests {
		got := decodePACRow(tt.cc1, tt.cc2)
		if got != tt.wantRow {
			t.Errorf("decodePACRow(0x%02x, 0x%02x) = %d, want %d", tt.cc1, tt.cc2, got, tt.wantRow)
		}
	}
}

func TestCEA608_PACStyle(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)

	dec.Decode(0x11, 0x42)
	if dec.pen.fgColor != "00ff00" {
		t.Errorf("PAC green: got %q, want %q", dec.pen.fgColor, "00ff00")
	}
	if dec.pen.italic {
		t.Error("PAC green should not set italic")
	}

	dec.Decode(0x11, 0x4E)
	if !dec.pen.italic {
		t.Error("PAC italic should set italic")
	}
	if dec.pen.fgColor != "ffffff" {
		t.Errorf("PAC italic: got color %q, want %q", dec.pen.fgColor, "ffffff")
	}

	dec.Decode(0x11, 0x41)
	if !dec.pen.underline {
		t.Error("PAC should set underline when bit 0 is set")
	}
}

func TestCEA608_MidrowStyle(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode('H', 'i')

	dec.Decode(0x11, 0x28)
	if dec.pen.fgColor != "ff0000" {
		t.Errorf("midrow red: got %q, want %q", dec.pen.fgColor, "ff0000")
	}

	dec.Decode(0x11, 0x2E)
	if !dec.pen.italic {
		t.Error("midrow italic should set italic")
	}
}

func TestCEA608_RollUpModeTransitionClearsDisplay(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x20)
	dec.Decode('O', 'l')
	dec.Decode('d', 0)
	dec.Decode(0x14, 0x2F)

	text := dec.Decode(0x14, 0x25)
	if text != "" {
		t.Errorf("transition to rollup should clear display, got %q", text)
	}

	text = dec.Decode('N', 'e')
	if text != "Ne" {
		t.Errorf("after rollup transition: got %q, want %q", text, "Ne")
	}
}

func TestCEA608_RCL_PreservesCursor(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x20)
	dec.Decode('A', 'B')

	dec.Decode(0x14, 0x20)
	dec.Decode(0x14, 0x20)
	dec.Decode('C', 0)

	text := dec.Decode(0x14, 0x2F)
	if text != "ABC" {
		t.Errorf("RCL should preserve cursor, got %q, want %q", text, "ABC")
	}
}

func TestCEA608_FlashOn(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)

	dec.Decode(0x14, 0x28)
	dec.Decode(0x14, 0x28)
	if !dec.pen.flash {
		t.Error("FON should set flash on pen")
	}
}

func TestCEA608_StyledCharStorage(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)

	dec.Decode(0x11, 0x42)
	dec.Decode(0x11, 0x42)
	dec.Decode('A', 0)

	row := dec.currentRow
	cell := dec.displayRows[row][0]
	if cell == nil {
		t.Fatal("expected styled char at [row][0]")
	}
	if cell.ch != 'A' {
		t.Errorf("char: got %c, want A", cell.ch)
	}
	if cell.fgColor != "00ff00" {
		t.Errorf("color: got %q, want %q", cell.fgColor, "00ff00")
	}
}

func TestCEA608_StyledRegions_Basic(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode('H', 'i')

	regions := dec.StyledRegions()
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	if len(regions[0].Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(regions[0].Rows))
	}
	row := regions[0].Rows[0]
	totalText := ""
	for _, span := range row.Spans {
		totalText += span.Text
	}
	if totalText != "Hi" {
		t.Errorf("text: got %q, want %q", totalText, "Hi")
	}
}

func TestCEA608_StyledRegions_MultiSpan(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode('H', 'i')

	dec.Decode(0x11, 0x28)
	dec.Decode(0x11, 0x28)
	dec.Decode('R', 'D')

	regions := dec.StyledRegions()
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	row := regions[0].Rows[0]
	if len(row.Spans) < 2 {
		t.Fatalf("expected at least 2 spans, got %d", len(row.Spans))
	}
	if row.Spans[0].FgColor != "ffffff" {
		t.Errorf("span 0 color: got %q, want %q", row.Spans[0].FgColor, "ffffff")
	}
}

func TestCEA608_StyledRegions_Empty(t *testing.T) {
	dec := NewCEA608Decoder()
	regions := dec.StyledRegions()
	if regions != nil {
		t.Errorf("empty decoder should return nil regions, got %d", len(regions))
	}
}

func TestCEA608_ExtendedChars(t *testing.T) {
	tests := []struct {
		cc1, cc2 byte
		want     rune
	}{
		{0x12, 0x20, '\u00c1'}, {0x12, 0x21, '\u00c9'},
		{0x12, 0x2E, '\u201c'}, {0x12, 0x2F, '\u201d'},
		{0x13, 0x20, '\u00c3'}, {0x13, 0x34, '\u00df'},
	}
	for _, tt := range tests {
		got := cea608ExtendedChar(tt.cc1, tt.cc2)
		if got != tt.want {
			t.Errorf("extChar(0x%02x, 0x%02x): got %U, want %U", tt.cc1, tt.cc2, got, tt.want)
		}
	}
}

func TestCEA608_TabOffset(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)

	dec.Decode(0x17, 0x22)
	dec.Decode(0x17, 0x22)
	if dec.currentCol != 2 {
		t.Errorf("after tab 2: col=%d, want 2", dec.currentCol)
	}

	dec.Decode(0x17, 0x21)
	dec.Decode(0x17, 0x21)
	if dec.currentCol != 3 {
		t.Errorf("after tab 1: col=%d, want 3", dec.currentCol)
	}
}

func TestCEA608_DER(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode('A', 'B')
	dec.Decode('C', 'D')
	dec.Decode('E', 0)

	dec.currentCol = 2
	text := dec.Decode(0x14, 0x24)
	if text != "AB" {
		t.Errorf("after DER: got %q, want %q", text, "AB")
	}
}

func TestCEA608_ENM(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x20)
	dec.Decode('A', 'B')

	dec.Decode(0x14, 0x2E)

	text := dec.Decode(0x14, 0x2F)
	if text != "" {
		t.Errorf("after ENM+EOC: got %q, want %q", text, "")
	}
}

func TestCEA608_RollUpScrollPreservesWindow(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x26)

	dec.Decode('L', '1')
	dec.Decode(0x14, 0x2D)
	dec.Decode(0x14, 0x2D)

	dec.Decode('L', '2')
	dec.Decode(0x14, 0x2D)

	dec.Decode('L', '3')
	dec.Decode(0x14, 0x2D)

	text := dec.Decode('L', '4')
	if text == "" {
		t.Fatal("expected text after scroll")
	}
	lines := 0
	for _, ch := range text {
		if ch == '\n' {
			lines++
		}
	}
	if lines > 2 {
		t.Errorf("RU3 should show at most 3 lines, got %d newlines", lines)
	}
}

func TestCEA608_CR_PaintOnMode(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x29)
	dec.Decode(0x14, 0x29)

	dec.Decode(0x11, 0x40)
	dec.Decode(0x11, 0x40)

	dec.Decode('A', 'B')
	dec.Decode(0x14, 0x2D)
	dec.Decode(0x14, 0x2D)

	text := dec.Decode('C', 'D')
	if text == "" {
		t.Fatal("expected text after CR in paint-on")
	}
	if text != "AB\nCD" {
		t.Errorf("got %q, want %q", text, "AB\nCD")
	}
}

func TestCEA608_EDM_ResetsCursorInPaintOn(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x29)
	dec.Decode(0x14, 0x29)

	dec.Decode(0x14, 0x70)
	dec.Decode(0x14, 0x70)
	dec.Decode('X', 'Y')

	dec.Decode(0x14, 0x2C)
	dec.Decode(0x14, 0x2C)

	text := dec.Decode('A', 'B')
	if text == "" {
		t.Fatal("expected text after EDM + new chars")
	}
	if text != "AB" {
		t.Errorf("got %q, want %q (cursor should be at row 0)", text, "AB")
	}
}

func TestCEA608_BackgroundColorAttribute(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode(0x14, 0x25)

	dec.Decode(0x10, 0x28)
	dec.Decode(0x10, 0x28)

	dec.Decode('H', 'i')

	regions := dec.StyledRegions()
	if len(regions) == 0 {
		t.Fatal("expected regions")
	}
	spans := regions[0].Rows[0].Spans
	if len(spans) == 0 {
		t.Fatal("expected spans")
	}
	if spans[0].BgColor != "ff0000" {
		t.Errorf("bgColor: got %q, want %q", spans[0].BgColor, "ff0000")
	}
}

func TestCEA608_BackgroundBlackTransparent(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode(0x14, 0x25)

	dec.Decode(0x17, 0x2D)
	dec.Decode(0x17, 0x2D)

	dec.Decode('O', 'K')

	regions := dec.StyledRegions()
	if len(regions) == 0 {
		t.Fatal("expected regions")
	}
	spans := regions[0].Rows[0].Spans
	if spans[0].BgColor != "000000" {
		t.Errorf("bgColor: got %q, want %q", spans[0].BgColor, "000000")
	}
}

func TestCEA608_PACResetsBgColor(t *testing.T) {
	dec := NewCEA608Decoder()
	dec.Decode(0x14, 0x25)
	dec.Decode(0x14, 0x25)

	dec.Decode(0x10, 0x2C)
	dec.Decode(0x10, 0x2C)

	dec.Decode(0x11, 0x40)
	dec.Decode(0x11, 0x40)

	dec.Decode('A', 'B')

	regions := dec.StyledRegions()
	if len(regions) == 0 {
		t.Fatal("expected regions")
	}
	spans := regions[0].Rows[0].Spans
	if spans[0].BgColor != "000000" {
		t.Errorf("bgColor after PAC: got %q, want %q", spans[0].BgColor, "000000")
	}
}
