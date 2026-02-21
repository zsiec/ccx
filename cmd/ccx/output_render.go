package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/zsiec/ccx"
)

const (
	renderWidth  = 42
	renderHeight = 15
	framePadH    = 1
	framePadV    = 0
)

type renderOutput struct {
	w     io.Writer
	count int
}

func newRenderOutput(w io.Writer) *renderOutput {
	return &renderOutput{w: w}
}

func (o *renderOutput) event(ev captionEvent) {
	if ev.Frame608 != nil {
		o.count++
		o.renderFrame("608", ev, ev.Frame608)
	}
	if ev.Frame708 != nil {
		o.count++
		o.renderFrame("708", ev, ev.Frame708)
	}
}

func (o *renderOutput) renderFrame(standard string, ev captionEvent, frame *ccx.CaptionFrame) {
	header := fmt.Sprintf(" CEA-%s ", standard)
	if frame.Channel > 0 {
		header += fmt.Sprintf("CC%d ", frame.Channel)
	}
	header += fmt.Sprintf("│ NAL #%d ", ev.NALIndex)

	totalW := renderWidth + framePadH*2 + 2

	// Top border with header
	topLeft := "╭"
	topRight := "╮"
	headerLen := visibleLen(header)
	remaining := totalW - 2 - headerLen
	if remaining < 0 {
		remaining = 0
	}
	topLine := topLeft + header + strings.Repeat("─", remaining) + topRight

	fmt.Fprintln(o.w, topLine)

	grid := o.buildGrid(frame)
	for _, row := range grid {
		pad := strings.Repeat(" ", framePadH)
		fmt.Fprintf(o.w, "│%s%s%s│\n", pad, row, pad)
	}

	// Bottom border
	fmt.Fprintf(o.w, "╰%s╯\n", strings.Repeat("─", totalW-2))
}

func (o *renderOutput) buildGrid(frame *ccx.CaptionFrame) []string {
	grid := make([]string, renderHeight)
	for i := range grid {
		grid[i] = strings.Repeat(" ", renderWidth)
	}

	if frame.PlainText() == "" {
		center := renderHeight / 2
		msg := dimText("(empty)")
		padLeft := (renderWidth - 7) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		grid[center] = strings.Repeat(" ", padLeft) + msg + strings.Repeat(" ", renderWidth-padLeft-7)
		return grid
	}

	if len(frame.Regions) > 1 && frame.Regions[0].AnchorV == 0 && frame.Regions[1].AnchorV == 0 {
		// Multiple 708 windows with default anchors: stack them vertically
		// so they don't overlap on the same grid rows.
		nextRow := 0
		for _, reg := range frame.Regions {
			for _, row := range reg.Rows {
				targetRow := nextRow
				if targetRow >= renderHeight {
					break
				}
				grid[targetRow] = o.renderRow(row, reg)
				nextRow++
			}
			nextRow++
		}
		return grid
	}

	for _, reg := range frame.Regions {
		for _, row := range reg.Rows {
			if row.Row < 0 || row.Row >= renderHeight {
				continue
			}
			grid[row.Row] = o.renderRow(row, reg)
		}
	}
	return grid
}

func (o *renderOutput) renderRow(row ccx.CaptionRow, reg ccx.CaptionRegion) string {
	var b strings.Builder
	visLen := 0

	for _, span := range row.Spans {
		styled := renderSpan(span)
		b.WriteString(styled)
		for _, r := range span.Text {
			_ = r
			visLen++
		}
	}

	if visLen < renderWidth {
		justified := applyJustify(b.String(), visLen, reg.Justify)
		return justified
	}
	return b.String()
}

func applyJustify(rendered string, visLen int, justify int) string {
	remaining := renderWidth - visLen
	if remaining <= 0 {
		return rendered
	}

	switch ccx.TextJustification(justify) {
	case ccx.JustifyCenter:
		left := remaining / 2
		right := remaining - left
		return strings.Repeat(" ", left) + rendered + strings.Repeat(" ", right)
	case ccx.JustifyRight:
		return strings.Repeat(" ", remaining) + rendered
	default:
		return rendered + strings.Repeat(" ", remaining)
	}
}

func renderSpan(span ccx.CaptionSpan) string {
	text := span.Text

	var codes []string

	if fg := hexToANSI256(span.FgColor); fg != "" {
		codes = append(codes, fg)
	}
	if bg := hexToBGANSI256(span.BgColor); bg != "" {
		codes = append(codes, bg)
	}
	if span.Italic {
		codes = append(codes, "3")
	}
	if span.Underline {
		codes = append(codes, "4")
	}
	if span.Flash {
		codes = append(codes, "5")
	}

	if len(codes) == 0 {
		return text
	}

	return "\033[" + strings.Join(codes, ";") + "m" + text + "\033[0m"
}

func hexToANSI256(hex string) string {
	if hex == "" || hex == "ffffff" {
		return ""
	}
	r, g, b := parseHex(hex)
	idx := ansi256(r, g, b)
	return "38;5;" + strconv.Itoa(idx)
}

func hexToBGANSI256(hex string) string {
	if hex == "" || hex == "000000" {
		return ""
	}
	r, g, b := parseHex(hex)
	idx := ansi256(r, g, b)
	return "48;5;" + strconv.Itoa(idx)
}

func parseHex(hex string) (uint8, uint8, uint8) {
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return uint8(r), uint8(g), uint8(b)
}

// ansi256 maps an RGB color to the nearest xterm-256 color index.
func ansi256(r, g, b uint8) int {
	if r == g && g == b {
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return int(((float64(r)-8)/247)*24) + 232
	}

	ri := colorCube(r)
	gi := colorCube(g)
	bi := colorCube(b)
	return 16 + 36*ri + 6*gi + bi
}

func colorCube(v uint8) int {
	if v < 48 {
		return 0
	}
	if v < 115 {
		return 1
	}
	return int((v-35)/40) - 1
}

func dimText(s string) string {
	return "\033[2m" + s + "\033[0m"
}

func visibleLen(s string) int {
	inEscape := false
	count := 0
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		count++
	}
	return count
}

func (o *renderOutput) summary() {
	fmt.Fprintf(o.w, "\n%d caption frame(s) rendered\n", o.count)
}
