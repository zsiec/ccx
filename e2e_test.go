package ccx

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

type e2eExpect608 struct {
	name       string
	wantText   string
	wantRows   int
	wantSpans  int
	checkStyle func(t *testing.T, regions []CaptionRegion)
}

type e2eExpect708 struct {
	name       string
	wantText   string
	wantRows   int
	checkStyle func(t *testing.T, regions []CaptionRegion)
}

func decode608FromFile(t *testing.T, path string, codec string) (*CEA608Decoder, []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	nals := splitAnnexBNALs(data)
	dec := NewCEA608Decoder()
	var texts []string
	for _, nal := range nals {
		if len(nal) == 0 {
			continue
		}
		var cd *CaptionData
		if codec == "h264" {
			if !IsH264SEI(nal[0]) {
				continue
			}
			cd = ExtractCaptions(nal)
		} else {
			if len(nal) < 2 || !IsHEVCSEI(nal[0]) {
				continue
			}
			cd = ExtractCaptionsHEVC(nal)
		}
		if cd == nil {
			continue
		}
		for _, pair := range cd.CC608Pairs {
			text := dec.Decode(pair.Data[0], pair.Data[1])
			if text != "" {
				texts = append(texts, text)
			}
		}
	}
	return dec, texts
}

func decode708FromFile(t *testing.T, path string, codec string) (*CEA708Service, []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	nals := splitAnnexBNALs(data)
	dec708 := NewCEA708Decoder()
	var texts []string
	for _, nal := range nals {
		if len(nal) == 0 {
			continue
		}
		var cd *CaptionData
		if codec == "h264" {
			if !IsH264SEI(nal[0]) {
				continue
			}
			cd = ExtractCaptions(nal)
		} else {
			if len(nal) < 2 || !IsHEVCSEI(nal[0]) {
				continue
			}
			cd = ExtractCaptionsHEVC(nal)
		}
		if cd == nil {
			continue
		}
		for _, pair := range cd.DTVCC {
			text := dec708.AddTriplet(pair)
			if text != "" {
				texts = append(texts, text)
			}
		}
	}
	return &dec708.service, texts
}

func lastText(texts []string) string {
	if len(texts) == 0 {
		return ""
	}
	return texts[len(texts)-1]
}

func countRows(regions []CaptionRegion) int {
	n := 0
	for _, r := range regions {
		n += len(r.Rows)
	}
	return n
}

func countSpans(regions []CaptionRegion) int {
	n := 0
	for _, r := range regions {
		for _, row := range r.Rows {
			n += len(row.Spans)
		}
	}
	return n
}

func regionText(regions []CaptionRegion) string {
	var lines []string
	for _, r := range regions {
		for _, row := range r.Rows {
			var line string
			for _, span := range row.Spans {
				line += span.Text
			}
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func roundTrip(t *testing.T, regions []CaptionRegion) {
	t.Helper()
	frame := &CaptionFrame{Channel: 1, Regions: regions}
	wire := frame.Serialize()
	decoded := DeserializeCaptionFrame(wire)
	if decoded == nil {
		t.Fatal("round-trip: DeserializeCaptionFrame returned nil")
	}
	got := decoded.PlainText()
	want := regionText(regions)
	if got != want {
		t.Errorf("round-trip text: got %q, want %q", got, want)
	}
}

var e2e608Tests = []e2eExpect608{
	{
		name:     "608_rollup2",
		wantText: "L1\nL2",
		wantRows: 2,
	},
	{
		name:     "608_rollup3",
		wantText: "AA\nBB\nCC",
		wantRows: 3,
	},
	{
		name:     "608_rollup4",
		wantText: "R1\nR2\nR3\nR4",
		wantRows: 4,
	},
	{
		name:     "608_popon",
		wantText: "Top\nBot",
		wantRows: 2,
	},
	{
		name:     "608_painton",
		wantText: "Two",
		wantRows: 1,
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			text := regionText(regions)
			if !strings.Contains(text, "Two") {
				t.Errorf("paint-on should show latest line: %q", text)
			}
		},
	},
	{
		name:     "608_mode_switch",
		wantText: "Nw",
		wantRows: 1,
	},
	{
		name:     "608_special_chars",
		wantText: "®°½¿™¢£♪à è\u00e2\u00ea\u00ee\u00f4\u00fb",
		wantRows: 1,
	},
	{
		name: "608_extended_spanish",
		wantText: "\u00c1\u00c9\u00d3\u00da\u00dc\u00fc\u2018\u00a1*\u2019\u2014\u00a9\u2120\u2022\u201c\u201d" +
			"\u00c0\u00c2\u00c7\u00c8\u00ca\u00cb\u00eb\u00ce\u00cf\u00ef\u00d4\u00d9\u00f9\u00db\u00ab\u00bb",
		wantRows: 1,
	},
	{
		name: "608_extended_portuguese",
		wantText: "\u00c3\u00e3\u00cd\u00cc\u00ec\u00d2\u00f2\u00d5\u00f5{}\\" +
			"^_|~\u00c4\u00e4\u00d6\u00f6\u00df\u00a5\u00a4\u2502\u00c5\u00e5\u00d8\u00f8\u250c\u2510\u2514\u2518",
		wantRows: 1,
	},
	{
		name:     "608_g0_overrides",
		wantText: "\u2019\u00e1\u00e9\u00ed\u00f3\u00fa\u00e7\u00f7\u00d1\u00f1\u2588",
		wantRows: 1,
	},
	{
		name:     "608_backspace",
		wantText: "ABX",
		wantRows: 1,
	},
	{
		name:     "608_tab_offsets",
		wantText: "A B  C   D",
		wantRows: 1,
	},
	{
		name:     "608_erase_displayed",
		wantText: "",
		wantRows: 0,
	},
	{
		name:     "608_erase_nondisplayed",
		wantText: "",
		wantRows: 0,
	},
	{
		name: "608_pac_colors",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			wantColors := []string{
				"ffffff", "00ff00", "0000ff", "00ffff",
				"ff0000", "ffff00", "ff00ff",
			}
			rows := countRows(regions)
			if rows < len(wantColors) {
				t.Fatalf("expected at least %d rows for PAC colors, got %d", len(wantColors), rows)
			}
			colorIdx := 0
			for _, r := range regions {
				for _, row := range r.Rows {
					if colorIdx >= len(wantColors) {
						break
					}
					if len(row.Spans) == 0 {
						continue
					}
					got := row.Spans[0].FgColor
					if got != wantColors[colorIdx] {
						t.Errorf("row %d: fg=%s, want %s", colorIdx, got, wantColors[colorIdx])
					}
					colorIdx++
				}
			}
		},
	},
	{
		name: "608_pac_indent",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			text := regionText(regions)
			if !strings.Contains(text, "I4") {
				t.Errorf("indent-4 text not found: %q", text)
			}
			if !strings.Contains(text, "IC") {
				t.Errorf("indent-12 text not found: %q", text)
			}
		},
	},
	{
		name:     "608_pac_underline",
		wantText: "UL",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 || len(regions[0].Rows) == 0 || len(regions[0].Rows[0].Spans) == 0 {
				t.Fatal("no spans")
			}
			if !regions[0].Rows[0].Spans[0].Underline {
				t.Error("expected underline=true")
			}
		},
	},
	{
		name:     "608_pac_italic",
		wantText: "IT",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 || len(regions[0].Rows) == 0 || len(regions[0].Rows[0].Spans) == 0 {
				t.Fatal("no spans")
			}
			if !regions[0].Rows[0].Spans[0].Italic {
				t.Error("expected italic=true")
			}
		},
	},
	{
		name: "608_midrow_colors",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 || len(regions[0].Rows) == 0 {
				t.Fatal("no rows")
			}
			spans := regions[0].Rows[0].Spans
			if len(spans) < 2 {
				t.Fatalf("expected multiple spans from midrow, got %d", len(spans))
			}
			uniqueColors := map[string]bool{}
			for _, s := range spans {
				uniqueColors[s.FgColor] = true
			}
			if len(uniqueColors) < 3 {
				t.Errorf("expected multiple distinct colors, got %d", len(uniqueColors))
			}
		},
	},
	{
		name: "608_midrow_italic_underline",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 || len(regions[0].Rows) == 0 {
				t.Fatal("no rows")
			}
			spans := regions[0].Rows[0].Spans
			hasItalic := false
			hasUnderline := false
			for _, s := range spans {
				if s.Italic {
					hasItalic = true
				}
				if s.Underline {
					hasUnderline = true
				}
			}
			if !hasItalic {
				t.Error("expected at least one italic span")
			}
			if !hasUnderline {
				t.Error("expected at least one underline span")
			}
		},
	},
	{
		name: "608_background_colors",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 || len(regions[0].Rows) == 0 {
				t.Fatal("no rows")
			}
			spans := regions[0].Rows[0].Spans
			uniqueBg := map[string]bool{}
			for _, s := range spans {
				uniqueBg[s.BgColor] = true
			}
			if len(uniqueBg) < 4 {
				t.Errorf("expected multiple distinct bg colors, got %d: %v", len(uniqueBg), uniqueBg)
			}
		},
	},
	{
		name: "608_flash",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 || len(regions[0].Rows) == 0 {
				t.Fatal("no rows")
			}
			spans := regions[0].Rows[0].Spans
			hasFlash := false
			hasNonFlash := false
			for _, s := range spans {
				if s.Flash {
					hasFlash = true
				} else {
					hasNonFlash = true
				}
			}
			if !hasFlash {
				t.Error("expected flash span")
			}
			if !hasNonFlash {
				t.Error("expected non-flash span")
			}
		},
	},
	{
		name: "608_pac_row_positioning",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			rows := countRows(regions)
			if rows < 3 {
				t.Errorf("expected at least 3 positioned rows, got %d", rows)
			}
			text := regionText(regions)
			if !strings.Contains(text, "R0") || !strings.Contains(text, "R3") || !strings.Contains(text, "R7") {
				t.Errorf("positioned text not found: %q", text)
			}
		},
	},
	{
		name:     "608_rollup_pac_relocate",
		wantText: "MV",
		wantRows: 1,
	},
	{
		name: "608_column_overflow",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			text := regionText(regions)
			if len([]rune(text)) > 32 {
				t.Errorf("text exceeds 32 columns: len=%d", len([]rune(text)))
			}
		},
	},
	{
		name: "608_delete_to_end",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			text := regionText(regions)
			if strings.Contains(text, "E") || strings.Contains(text, "F") {
				t.Errorf("DER should have removed chars at/after col 4, got %q", text)
			}
			if !strings.Contains(text, "A") {
				t.Errorf("DER should preserve chars before col 4, got %q", text)
			}
		},
	},
}

var e2e708Tests = []e2eExpect708{
	{
		name:     "708_g0_music_note",
		wantText: "\u266a",
	},
	{
		name:     "708_g1_latin1",
		wantText: "\u00c0\u00c9\u00d1\u00e9\u00fc",
	},
	{
		name: "708_g2_full",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			text := regionText(regions)
			if !strings.ContainsRune(text, '\u2026') {
				t.Error("missing ellipsis from G2")
			}
			if !strings.ContainsRune(text, '\u215b') {
				t.Error("missing 1/8 fraction from G2")
			}
			if !strings.ContainsRune(text, '\u250c') {
				t.Error("missing box-drawing from G2")
			}
			if len([]rune(text)) < 20 {
				t.Errorf("expected 20+ G2 chars, got %d", len([]rune(text)))
			}
		},
	},
	{
		name:     "708_g3_icon",
		wantText: "\U0001F4AC",
	},
	{
		name:     "708_p16",
		wantText: "A",
	},
	{
		name: "708_multiwindow",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) < 2 {
				t.Fatalf("expected 2 regions, got %d", len(regions))
			}
			text := regionText(regions)
			if !strings.Contains(text, "W0") || !strings.Contains(text, "W1") {
				t.Errorf("multi-window text: %q", text)
			}
		},
	},
	{
		name: "708_hide_show",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			text := regionText(regions)
			if !strings.Contains(text, "!") {
				t.Errorf("expected text after hide/show cycle: %q", text)
			}
		},
	},
	{
		name: "708_toggle",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) != 0 {
				t.Errorf("window should be hidden after toggle, got %d regions", len(regions))
			}
		},
	},
	{
		name: "708_delete_window",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) != 0 {
				t.Errorf("window should be deleted, got %d regions", len(regions))
			}
		},
	},
	{
		name:     "708_clear_window",
		wantText: "Nw",
	},
	{
		name: "708_reset",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) != 0 {
				t.Errorf("all windows should be reset, got %d regions", len(regions))
			}
		},
	},
	{
		name: "708_spa_full",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 || len(regions[0].Rows) == 0 || len(regions[0].Rows[0].Spans) == 0 {
				t.Fatal("no spans")
			}
			s := regions[0].Rows[0].Spans[0]
			if !s.Italic {
				t.Error("expected italic from SPA")
			}
			if !s.Underline {
				t.Error("expected underline from SPA")
			}
			if s.PenSize != int(PenSizeLarge) {
				t.Errorf("penSize=%d, want %d (Large)", s.PenSize, PenSizeLarge)
			}
			if s.FontTag != int(FontMonoSans) {
				t.Errorf("fontTag=%d, want %d (MonoSans)", s.FontTag, FontMonoSans)
			}
			if s.Offset != int(PenOffsetNormal) {
				t.Errorf("offset=%d, want %d (Normal)", s.Offset, PenOffsetNormal)
			}
			if s.EdgeType != int(EdgeDepressed) {
				t.Errorf("edgeType=%d, want %d (Depressed)", s.EdgeType, EdgeDepressed)
			}
		},
	},
	{
		name: "708_spc_full",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 || len(regions[0].Rows) == 0 || len(regions[0].Rows[0].Spans) == 0 {
				t.Fatal("no spans")
			}
			s := regions[0].Rows[0].Spans[0]
			if s.FgColor != "ff0000" {
				t.Errorf("fgColor=%s, want ff0000", s.FgColor)
			}
			if s.FgOpacity != int(OpacitySolid) {
				t.Errorf("fgOpacity=%d, want %d (Solid)", s.FgOpacity, OpacitySolid)
			}
			if s.EdgeColor != "ffffff" {
				t.Errorf("edgeColor=%s, want ffffff", s.EdgeColor)
			}
		},
	},
	{
		name: "708_swa_full",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) == 0 {
				t.Fatal("no regions")
			}
			r := regions[0]
			if r.Justify != int(JustifyCenter) {
				t.Errorf("justify=%d, want %d (Center)", r.Justify, JustifyCenter)
			}
			if !r.WordWrap {
				t.Error("expected wordWrap=true")
			}
			if r.FillColor != "ff0000" {
				t.Errorf("fillColor=%s, want ff0000", r.FillColor)
			}
			if r.FillOpacity != int(OpacityTranslucent) {
				t.Errorf("fillOpacity=%d, want %d (Translucent)", r.FillOpacity, OpacityTranslucent)
			}
		},
	},
	{
		name: "708_predefined_pen_styles",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) < 7 {
				t.Fatalf("expected 7 regions for pen styles, got %d", len(regions))
			}
			wantFonts := []FontTag{FontDefault, FontMonoSerif, FontPropSerif, FontMonoSans, FontPropSans, FontMonoSans, FontPropSans}
			for i, want := range wantFonts {
				if len(regions[i].Rows) == 0 || len(regions[i].Rows[0].Spans) == 0 {
					t.Errorf("region %d: no spans", i)
					continue
				}
				got := FontTag(regions[i].Rows[0].Spans[0].FontTag)
				if got != want {
					t.Errorf("pen style %d: fontTag=%d, want %d", i+1, got, want)
				}
			}
		},
	},
	{
		name: "708_predefined_window_styles",
		checkStyle: func(t *testing.T, regions []CaptionRegion) {
			t.Helper()
			if len(regions) < 7 {
				t.Fatalf("expected 7 regions for window styles, got %d", len(regions))
			}
			if regions[0].Justify != int(JustifyLeft) {
				t.Errorf("style 1: justify=%d, want Left", regions[0].Justify)
			}
			if regions[1].Justify != int(JustifyCenter) {
				t.Errorf("style 2: justify=%d, want Center", regions[1].Justify)
			}
			if !regions[2].WordWrap {
				t.Error("style 3: expected wordWrap=true")
			}
		},
	},
	{
		name:     "708_backspace",
		wantText: "ABX",
	},
	{
		name:     "708_carriage_return",
		wantText: "L1\nL2\nL3",
		wantRows: 3,
	},
	{
		name:     "708_formfeed",
		wantText: "New",
	},
	{
		name:     "708_hcr",
		wantText: "New",
	},
	{
		name:     "708_etx",
		wantText: "AB",
	},
}

func TestE2E_608_Comprehensive(t *testing.T) {
	for _, tc := range e2e608Tests {
		for _, codec := range []string{"h264", "h265"} {
			name := fmt.Sprintf("%s/%s", tc.name, codec)
			t.Run(name, func(t *testing.T) {
				path := fmt.Sprintf("testdata/%s_%s.%s", codec, tc.name, codec)

				dec, texts := decode608FromFile(t, path, codec)
				regions := dec.StyledRegions()

				if tc.wantText != "" {
					got := lastText(texts)
					if got != tc.wantText {
						t.Errorf("text: got %q, want %q", got, tc.wantText)
					}
				}

				if tc.wantRows > 0 {
					got := countRows(regions)
					if got != tc.wantRows {
						t.Errorf("rows: got %d, want %d", got, tc.wantRows)
					}
				}

				if tc.wantSpans > 0 {
					got := countSpans(regions)
					if got < tc.wantSpans {
						t.Errorf("spans: got %d, want >= %d", got, tc.wantSpans)
					}
				}

				if tc.checkStyle != nil {
					tc.checkStyle(t, regions)
				}

				if len(regions) > 0 {
					roundTrip(t, regions)
				}
			})
		}
	}
}

func TestE2E_708_Comprehensive(t *testing.T) {
	for _, tc := range e2e708Tests {
		for _, codec := range []string{"h264", "h265"} {
			name := fmt.Sprintf("%s/%s", tc.name, codec)
			t.Run(name, func(t *testing.T) {
				path := fmt.Sprintf("testdata/%s_%s.%s", codec, tc.name, codec)

				svc, texts := decode708FromFile(t, path, codec)
				regions := svc.StyledRegions()

				if tc.wantText != "" {
					got := lastText(texts)
					if got != tc.wantText {
						t.Errorf("text: got %q, want %q", got, tc.wantText)
					}
				}

				if tc.wantRows > 0 {
					got := countRows(regions)
					if got != tc.wantRows {
						t.Errorf("rows: got %d, want %d", got, tc.wantRows)
					}
				}

				if tc.checkStyle != nil {
					tc.checkStyle(t, regions)
				}

				if len(regions) > 0 {
					roundTrip(t, regions)
				}
			})
		}
	}
}
