package ccx

const cea608MaxCols = 32

type cea608Pen struct {
	fgColor   string
	bgColor   string
	underline bool
	italic    bool
	flash     bool
}

var cea608DefaultPen = cea608Pen{fgColor: "ffffff", bgColor: "000000"}

type cea608StyledChar struct {
	ch        rune
	fgColor   string
	bgColor   string
	underline bool
	italic    bool
	flash     bool
}

// CEA608Decoder implements the CTA-608-E closed caption state machine.
// It supports pop-on, roll-up (2/3/4), and paint-on modes with full
// character set and styling support.
type CEA608Decoder struct {
	displayRows [15][cea608MaxCols]*cea608StyledChar
	bufferRows  [15][cea608MaxCols]*cea608StyledChar

	pen        cea608Pen
	currentRow int
	currentCol int
	rollUpRows int
	popOn      bool
	paintOn    bool

	lastControlPair [2]byte
	lastWasControl  bool

	lastDisplayText string
}

// NewCEA608Decoder creates a new CEA-608 decoder with default state.
func NewCEA608Decoder() *CEA608Decoder {
	return &CEA608Decoder{
		currentRow: 14,
		pen:        cea608DefaultPen,
	}
}

// Decode processes a CEA-608 byte pair and returns the current display text
// if it changed, or an empty string otherwise.
func (d *CEA608Decoder) Decode(cc1, cc2 byte) string {
	if cc1 >= 0x10 && cc1 <= 0x1F {
		return d.handleControl(cc1, cc2)
	}

	if cc1 >= 0x20 && cc1 <= 0x7F {
		d.lastWasControl = false
		rows := d.targetRowBuf()
		d.putChar(rows, d.currentRow, cea608Char(cc1))
		if cc2 >= 0x20 && cc2 <= 0x7F {
			d.putChar(rows, d.currentRow, cea608Char(cc2))
		}
		if d.rollUpRows > 0 || d.paintOn {
			return d.displayText()
		}
	}

	return ""
}

func (d *CEA608Decoder) putChar(rows *[15][cea608MaxCols]*cea608StyledChar, row int, ch rune) {
	if d.currentCol >= cea608MaxCols {
		return
	}
	rows[row][d.currentCol] = &cea608StyledChar{
		ch:        ch,
		fgColor:   d.pen.fgColor,
		bgColor:   d.pen.bgColor,
		underline: d.pen.underline,
		italic:    d.pen.italic,
		flash:     d.pen.flash,
	}
	d.currentCol++
}

func (d *CEA608Decoder) handleControl(cc1, cc2 byte) string {
	pair := [2]byte{cc1, cc2}
	if d.lastWasControl && pair == d.lastControlPair {
		d.lastWasControl = false
		return ""
	}
	d.lastControlPair = pair
	d.lastWasControl = true

	cmdClass := cc1 & 0x07

	if cmdClass == 4 || cmdClass == 5 {
		if cc2 < 0x40 {
			return d.handleMiscControl(cc2)
		}
	}

	if cmdClass == 7 {
		if cc2 >= 0x21 && cc2 <= 0x23 {
			offset := int(cc2 - 0x20)
			d.currentCol += offset
			if d.currentCol >= cea608MaxCols {
				d.currentCol = cea608MaxCols - 1
			}
			return ""
		}
	}

	if cmdClass == 1 && cc2 >= 0x30 && cc2 <= 0x3F {
		rows := d.targetRowBuf()
		if d.currentCol > 0 {
			d.currentCol--
		}
		d.putChar(rows, d.currentRow, cea608SpecialChar(cc2))
		if d.rollUpRows > 0 || d.paintOn {
			return d.displayText()
		}
		return ""
	}

	if (cmdClass == 2 || cmdClass == 3) && cc2 >= 0x20 && cc2 <= 0x3F {
		rows := d.targetRowBuf()
		if d.currentCol > 0 {
			d.currentCol--
		}
		ch := cea608ExtendedChar(cc1, cc2)
		d.putChar(rows, d.currentRow, ch)
		if d.rollUpRows > 0 || d.paintOn {
			return d.displayText()
		}
		return ""
	}

	if cmdClass == 1 && cc2 >= 0x20 && cc2 <= 0x2F {
		d.applyMidrowStyle(cc2)
		rows := d.targetRowBuf()
		d.putChar(rows, d.currentRow, ' ')
		if d.rollUpRows > 0 || d.paintOn {
			return d.displayText()
		}
		return ""
	}

	if cmdClass == 0 && cc2 >= 0x20 && cc2 <= 0x2F {
		d.applyBackgroundAttribute(cc2)
		return ""
	}
	if cmdClass == 7 && cc2 == 0x2D {
		d.pen.bgColor = "000000"
		return ""
	}

	if cc2 >= 0x40 && cc2 <= 0x7F {
		return d.handlePAC(cc1, cc2)
	}

	return ""
}

func (d *CEA608Decoder) handleMiscControl(cc2 byte) string {
	switch cc2 {
	case 0x20: // RCL
		d.popOn = true
		d.paintOn = false
		d.rollUpRows = 0
		return ""
	case 0x21: // BS
		rows := d.targetRowBuf()
		if d.currentCol > 0 {
			d.currentCol--
			rows[d.currentRow][d.currentCol] = nil
		}
		if d.rollUpRows > 0 || d.paintOn {
			return d.displayText()
		}
		return ""
	case 0x22, 0x23:
		return ""
	case 0x24: // DER
		rows := d.targetRowBuf()
		for c := d.currentCol; c < cea608MaxCols; c++ {
			rows[d.currentRow][c] = nil
		}
		if d.rollUpRows > 0 || d.paintOn {
			return d.displayText()
		}
		return ""
	case 0x25: // RU2
		return d.switchToRollUp(2)
	case 0x26: // RU3
		return d.switchToRollUp(3)
	case 0x27: // RU4
		return d.switchToRollUp(4)
	case 0x28: // FON
		d.pen.flash = true
		return ""
	case 0x29: // RDC
		d.popOn = false
		d.paintOn = true
		d.rollUpRows = 0
		return ""
	case 0x2A: // TR
		return ""
	case 0x2B: // RTD
		return ""
	case 0x2C: // EDM
		d.displayRows = [15][cea608MaxCols]*cea608StyledChar{}
		if d.paintOn {
			d.currentRow = 0
			d.currentCol = 0
		}
		return d.displayText()
	case 0x2D: // CR
		if d.rollUpRows > 0 {
			d.scrollUp()
			d.currentCol = 0
			return d.displayText()
		}
		if d.paintOn {
			if d.currentRow < 14 {
				d.currentRow++
			}
			d.currentCol = 0
			return d.displayText()
		}
		return ""
	case 0x2E: // ENM
		d.bufferRows = [15][cea608MaxCols]*cea608StyledChar{}
		return ""
	case 0x2F: // EOC
		oldDisplay := d.displayRows
		d.displayRows = d.bufferRows
		d.bufferRows = oldDisplay
		d.popOn = true
		d.paintOn = false
		d.currentCol = 0
		return d.displayText()
	}
	return ""
}

func (d *CEA608Decoder) switchToRollUp(rows int) string {
	wasRollUp := d.rollUpRows > 0
	result := ""

	if !wasRollUp {
		d.displayRows = [15][cea608MaxCols]*cea608StyledChar{}
		d.bufferRows = [15][cea608MaxCols]*cea608StyledChar{}
		d.currentRow = 14
		d.currentCol = 0
		result = d.displayText()
	}

	d.rollUpRows = rows
	d.popOn = false
	d.paintOn = false
	return result
}

var cea608PACColors = [8]string{
	"ffffff", "00ff00", "0000ff", "00ffff",
	"ff0000", "ffff00", "ff00ff", "ffffff",
}

func (d *CEA608Decoder) handlePAC(cc1, cc2 byte) string {
	row := decodePACRow(cc1, cc2)
	if row < 0 || row >= 15 {
		return ""
	}

	if d.rollUpRows > 0 && row != d.currentRow {
		oldBase := d.currentRow
		oldTop := oldBase - d.rollUpRows + 1
		if oldTop < 0 {
			oldTop = 0
		}
		newTop := row - d.rollUpRows + 1
		if newTop < 0 {
			newTop = 0
		}

		var newDisplay [15][cea608MaxCols]*cea608StyledChar
		for i := 0; i < d.rollUpRows; i++ {
			srcRow := oldTop + i
			dstRow := newTop + i
			if srcRow >= 0 && srcRow < 15 && dstRow >= 0 && dstRow < 15 {
				newDisplay[dstRow] = d.displayRows[srcRow]
			}
		}
		d.displayRows = newDisplay
	}

	d.currentRow = row

	d.pen.underline = cc2&0x01 != 0

	if cc2&0x10 != 0 {
		indent := int((cc2>>1)&0x07) * 4
		d.currentCol = indent
		d.pen.fgColor = "ffffff"
		d.pen.italic = false
	} else {
		d.currentCol = 0
		styleIdx := (cc2 >> 1) & 0x07
		if styleIdx == 7 {
			d.pen.fgColor = "ffffff"
			d.pen.italic = true
		} else {
			d.pen.fgColor = cea608PACColors[styleIdx]
			d.pen.italic = false
		}
	}

	d.pen.bgColor = "000000"
	d.pen.flash = false
	return ""
}

var cea608BGColors = [8]string{
	"000000", "00ff00", "0000ff", "00ffff",
	"ff0000", "ffff00", "ff00ff", "ffffff",
}

func (d *CEA608Decoder) applyBackgroundAttribute(cc2 byte) {
	colorIdx := (cc2 >> 1) & 0x07
	d.pen.bgColor = cea608BGColors[colorIdx]
}

func (d *CEA608Decoder) applyMidrowStyle(cc2 byte) {
	d.pen.underline = cc2&0x01 != 0
	styleIdx := (cc2 >> 1) & 0x07
	if styleIdx == 7 {
		d.pen.fgColor = "ffffff"
		d.pen.italic = true
	} else {
		d.pen.fgColor = cea608PACColors[styleIdx]
		d.pen.italic = false
	}
	d.pen.flash = false
}

func (d *CEA608Decoder) targetRowBuf() *[15][cea608MaxCols]*cea608StyledChar {
	if d.popOn {
		return &d.bufferRows
	}
	return &d.displayRows
}

func (d *CEA608Decoder) scrollUp() {
	base := d.currentRow
	top := base - d.rollUpRows + 1
	if top < 0 {
		top = 0
	}
	for i := top; i < base; i++ {
		d.displayRows[i] = d.displayRows[i+1]
	}
	d.displayRows[base] = [cea608MaxCols]*cea608StyledChar{}
}

func (d *CEA608Decoder) displayText() string {
	var text string
	for i := 0; i < 15; i++ {
		rowStr := d.rowToString(i)
		if rowStr != "" {
			if text != "" {
				text += "\n"
			}
			text += rowStr
		}
	}
	if text != d.lastDisplayText {
		d.lastDisplayText = text
		return text
	}
	return ""
}

func (d *CEA608Decoder) rowToString(row int) string {
	last := -1
	for c := cea608MaxCols - 1; c >= 0; c-- {
		if d.displayRows[row][c] != nil {
			last = c
			break
		}
	}
	if last < 0 {
		return ""
	}
	runes := make([]rune, 0, last+1)
	for c := 0; c <= last; c++ {
		ch := d.displayRows[row][c]
		if ch == nil {
			runes = append(runes, ' ')
		} else {
			runes = append(runes, ch.ch)
		}
	}
	return string(runes)
}

func cea608Char(b byte) rune {
	switch b {
	case 0x27:
		return '\u2019'
	case 0x2A:
		return '\u00e1'
	case 0x5C:
		return '\u00e9'
	case 0x5E:
		return '\u00ed'
	case 0x5F:
		return '\u00f3'
	case 0x60:
		return '\u00fa'
	case 0x7B:
		return '\u00e7'
	case 0x7C:
		return '\u00f7'
	case 0x7D:
		return '\u00d1'
	case 0x7E:
		return '\u00f1'
	case 0x7F:
		return '\u2588'
	default:
		return rune(b)
	}
}

func cea608SpecialChar(b byte) rune {
	chars := [16]rune{
		'\u00ae', '\u00b0', '\u00bd', '\u00bf',
		'\u2122', '\u00a2', '\u00a3', '\u266a',
		'\u00e0', ' ', '\u00e8', '\u00e2',
		'\u00ea', '\u00ee', '\u00f4', '\u00fb',
	}
	if b >= 0x30 && b <= 0x3F {
		return chars[b-0x30]
	}
	return ' '
}

func cea608ExtendedChar(cc1, cc2 byte) rune {
	set := cc1 & 0x07

	if set == 2 {
		spanishFrench := map[byte]rune{
			0x20: '\u00c1', 0x21: '\u00c9', 0x22: '\u00d3', 0x23: '\u00da',
			0x24: '\u00dc', 0x25: '\u00fc', 0x26: '\u2018', 0x27: '\u00a1',
			0x28: '*', 0x29: '\u2019', 0x2A: '\u2014', 0x2B: '\u00a9',
			0x2C: '\u2120', 0x2D: '\u2022', 0x2E: '\u201c', 0x2F: '\u201d',
			0x30: '\u00c0', 0x31: '\u00c2', 0x32: '\u00c7', 0x33: '\u00c8',
			0x34: '\u00ca', 0x35: '\u00cb', 0x36: '\u00eb', 0x37: '\u00ce',
			0x38: '\u00cf', 0x39: '\u00ef', 0x3A: '\u00d4', 0x3B: '\u00d9',
			0x3C: '\u00f9', 0x3D: '\u00db', 0x3E: '\u00ab', 0x3F: '\u00bb',
		}
		if ch, ok := spanishFrench[cc2]; ok {
			return ch
		}
	}

	if set == 3 {
		portugueseGerman := map[byte]rune{
			0x20: '\u00c3', 0x21: '\u00e3', 0x22: '\u00cd', 0x23: '\u00cc',
			0x24: '\u00ec', 0x25: '\u00d2', 0x26: '\u00f2', 0x27: '\u00d5',
			0x28: '\u00f5', 0x29: '{', 0x2A: '}', 0x2B: '\\',
			0x2C: '^', 0x2D: '_', 0x2E: '|', 0x2F: '~',
			0x30: '\u00c4', 0x31: '\u00e4', 0x32: '\u00d6', 0x33: '\u00f6',
			0x34: '\u00df', 0x35: '\u00a5', 0x36: '\u00a4', 0x37: '\u2502',
			0x38: '\u00c5', 0x39: '\u00e5', 0x3A: '\u00d8', 0x3B: '\u00f8',
			0x3C: '\u250c', 0x3D: '\u2510', 0x3E: '\u2514', 0x3F: '\u2518',
		}
		if ch, ok := portugueseGerman[cc2]; ok {
			return ch
		}
	}

	return ' '
}

// decodePACRow decodes the row number from a Preamble Address Code.
// Returns 0-indexed row (0-14), or -1 if invalid.
// Per CTA-608-E Table 2.
func decodePACRow(cc1, cc2 byte) int {
	rowCode := int(cc1&0x07)<<1 | int((cc2>>5)&0x01)
	pacRowTable := [16]int{
		10, 10, 0, 1, 2, 3, 11, 12,
		13, 14, 4, 5, 6, 7, 8, 9,
	}
	if rowCode < 0 || rowCode > 15 {
		return -1
	}
	return pacRowTable[rowCode]
}

// StyledRegions returns the current display as structured caption regions
// with per-character styling preserved.
func (d *CEA608Decoder) StyledRegions() []CaptionRegion {
	var rows []CaptionRow
	for r := 0; r < 15; r++ {
		spans := d.rowSpans(r)
		if len(spans) == 0 {
			continue
		}
		rows = append(rows, CaptionRow{Row: r, Spans: spans})
	}
	if len(rows) == 0 {
		return nil
	}
	return []CaptionRegion{{Rows: rows}}
}

func (d *CEA608Decoder) rowSpans(row int) []CaptionSpan {
	last := -1
	for c := cea608MaxCols - 1; c >= 0; c-- {
		if d.displayRows[row][c] != nil {
			last = c
			break
		}
	}
	if last < 0 {
		return nil
	}

	var spans []CaptionSpan
	var cur CaptionSpan
	started := false

	for c := 0; c <= last; c++ {
		cell := d.displayRows[row][c]
		var ch rune
		var fg, bg string
		var italic, underline, flash bool
		if cell != nil {
			ch = cell.ch
			fg = cell.fgColor
			bg = cell.bgColor
			if bg == "" {
				bg = "000000"
			}
			italic = cell.italic
			underline = cell.underline
			flash = cell.flash
		} else {
			ch = ' '
			fg = "ffffff"
			bg = "000000"
		}

		sameStyle := started && cur.FgColor == fg && cur.BgColor == bg &&
			cur.Italic == italic && cur.Underline == underline && cur.Flash == flash

		if sameStyle {
			cur.Text += string(ch)
		} else {
			if started && cur.Text != "" {
				spans = append(spans, cur)
			}
			cur = CaptionSpan{
				Text:      string(ch),
				FgColor:   fg,
				BgColor:   bg,
				Italic:    italic,
				Underline: underline,
				Flash:     flash,
				EdgeColor: "000000",
			}
			started = true
		}
	}
	if started && cur.Text != "" {
		spans = append(spans, cur)
	}
	return spans
}
