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

	for _, reg := range frame.Regions {
		baseRow := anchorToGridRow(reg, renderHeight)
		for _, row := range reg.Rows {
			targetRow := baseRow + row.Row
			if targetRow < 0 || targetRow >= renderHeight {
				continue
			}
			grid[targetRow] = o.renderRow(row, reg)
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
		visLen += len([]rune(span.Text))
	}

	if visLen < renderWidth {
		justified := applyJustify(b.String(), visLen, reg.Justify)
		return justified
	}
	return b.String()
}

// anchorToGridRow maps a region's anchor position to a starting row on the
// render grid. For CEA-608 regions (AnchorV=0, AnchorID=0), row indices are
// already absolute screen rows so the base is 0. For CEA-708, the anchor
// position is scaled to the grid and adjusted by the anchor point.
func anchorToGridRow(reg ccx.CaptionRegion, gridHeight int) int {
	if reg.AnchorV == 0 && reg.AnchorID == 0 && len(reg.Rows) > 0 && reg.Rows[0].Row > 0 {
		return 0
	}

	rowCount := len(reg.Rows)
	anchorV := reg.AnchorV

	// Scale 708 anchor coordinates (0-74 absolute, 0-99 relative) to grid
	maxAnchor := 74
	if reg.RelativeToggle {
		maxAnchor = 99
	}
	gridRow := anchorV * (gridHeight - 1) / maxAnchor

	anchorID := reg.AnchorID
	switch {
	case anchorID >= 3 && anchorID <= 5: // middle anchors
		gridRow -= rowCount / 2
	case anchorID >= 6: // lower anchors
		gridRow -= rowCount - 1
	}

	if gridRow < 0 {
		gridRow = 0
	}
	if gridRow+rowCount > gridHeight {
		gridRow = gridHeight - rowCount
	}
	if gridRow < 0 {
		gridRow = 0
	}
	return gridRow
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
