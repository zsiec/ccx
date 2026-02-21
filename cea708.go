package ccx

const (
	cea708MaxRows = 16
	cea708MaxCols = 42
)

type cea708Pen struct {
	row       int
	col       int
	italic    bool
	underline bool
	fgColor   string
	bgColor   string
	fgOpacity Opacity
	bgOpacity Opacity
	edgeColor string
	edgeType  EdgeType
	penSize   PenSize
	fontTag   FontTag
	offset    PenOffset
	textTag   int
}

type cea708StyledChar struct {
	ch        rune
	italic    bool
	underline bool
	fgColor   string
	bgColor   string
	fgOpacity Opacity
	bgOpacity Opacity
	edgeColor string
	edgeType  EdgeType
	penSize   PenSize
	fontTag   FontTag
	offset    PenOffset
}

type cea708Window struct {
	memory   [cea708MaxRows][cea708MaxCols]*cea708StyledChar
	pen      cea708Pen
	visible  bool
	defined  bool
	rowCount int
	colCount int

	anchorV        int
	anchorH        int
	anchorID       AnchorID
	relativeToggle bool
	justify        TextJustification
	priority       int
	rowLock        bool
	colLock        bool

	fillColor       string
	fillOpacity     Opacity
	borderColor     string
	borderType      BorderType
	wordWrap        bool
	printDirection  PrintDirection
	scrollDirection ScrollDirection
	displayEffect   DisplayEffect
	effectDirection int
	effectSpeed     int

	contentID uint64
	wasHidden bool
}

func (w *cea708Window) resetMemory() {
	w.memory = [cea708MaxRows][cea708MaxCols]*cea708StyledChar{}
	w.pen.row = 0
	w.pen.col = 0
}

func (w *cea708Window) resetPen() {
	w.pen = cea708Pen{
		fgColor:   "ffffff",
		bgColor:   "000000",
		fgOpacity: OpacitySolid,
		bgOpacity: OpacitySolid,
		edgeColor: "000000",
		penSize:   PenSizeStandard,
		fontTag:   FontDefault,
		offset:    PenOffsetNormal,
	}
}

func (w *cea708Window) setCharacter(ch rune) {
	if w.wasHidden {
		w.resetMemory()
		w.wasHidden = false
	}
	if w.pen.row < 0 || w.pen.row >= w.rowCount ||
		w.pen.col < 0 || w.pen.col >= w.colCount {
		return
	}
	w.memory[w.pen.row][w.pen.col] = &cea708StyledChar{
		ch:        ch,
		italic:    w.pen.italic,
		underline: w.pen.underline,
		fgColor:   w.pen.fgColor,
		bgColor:   w.pen.bgColor,
		fgOpacity: w.pen.fgOpacity,
		bgOpacity: w.pen.bgOpacity,
		edgeColor: w.pen.edgeColor,
		edgeType:  w.pen.edgeType,
		penSize:   w.pen.penSize,
		fontTag:   w.pen.fontTag,
		offset:    w.pen.offset,
	}
	w.pen.col++
	w.contentID++
}

func (w *cea708Window) backspace() {
	if w.pen.col <= 0 && w.pen.row <= 0 {
		return
	}
	if w.pen.col <= 0 {
		w.pen.col = w.colCount - 1
		w.pen.row--
	} else {
		w.pen.col--
	}
	if w.pen.row >= 0 && w.pen.row < cea708MaxRows &&
		w.pen.col >= 0 && w.pen.col < cea708MaxCols {
		w.memory[w.pen.row][w.pen.col] = nil
	}
}

func (w *cea708Window) carriageReturn() {
	if w.pen.row+1 >= w.rowCount {
		for r := 1; r < w.rowCount; r++ {
			w.memory[r-1] = w.memory[r]
		}
		w.memory[w.rowCount-1] = [cea708MaxCols]*cea708StyledChar{}
		w.pen.col = 0
		return
	}
	w.pen.row++
	w.pen.col = 0
}

func (w *cea708Window) horizontalCarriageReturn() {
	if w.pen.row >= 0 && w.pen.row < cea708MaxRows {
		w.memory[w.pen.row] = [cea708MaxCols]*cea708StyledChar{}
	}
	w.pen.col = 0
}

func (w *cea708Window) text() string {
	firstRow := -1
	lastRow := -1
	for r := 0; r < w.rowCount; r++ {
		for c := 0; c < w.colCount; c++ {
			if w.memory[r][c] != nil {
				if firstRow < 0 {
					firstRow = r
				}
				lastRow = r
				break
			}
		}
	}
	if firstRow < 0 {
		return ""
	}

	var result string
	for r := firstRow; r <= lastRow; r++ {
		if r > firstRow {
			result += "\n"
		}
		last := -1
		for c := w.colCount - 1; c >= 0; c-- {
			if w.memory[r][c] != nil {
				last = c
				break
			}
		}
		if last < 0 {
			continue
		}
		runes := make([]rune, 0, last+1)
		for c := 0; c <= last; c++ {
			if w.memory[r][c] != nil {
				runes = append(runes, w.memory[r][c].ch)
			} else {
				runes = append(runes, ' ')
			}
		}
		result += string(runes)
	}
	return result
}

// CEA708Service implements the CTA-708-E caption service state machine.
// It manages up to 8 caption windows with full attribute support.
type CEA708Service struct {
	windows       [8]cea708Window
	currentWin    int
	lastText      string
	lastContentID [8]uint64
}

// NewCEA708Service creates a new CEA-708 service decoder.
func NewCEA708Service() *CEA708Service {
	return &CEA708Service{}
}

// ProcessBlock processes a single DTVCC service block and returns true if
// the display content changed.
func (s *CEA708Service) ProcessBlock(data []byte) bool {
	changed := false
	i := 0

	for i < len(data) {
		b := data[i]

		switch {
		case b >= 0x20 && b <= 0x7F:
			s.handleG0(b)
			changed = true
			i++

		case b <= 0x1F:
			n, ch := s.handleC0(data[i:])
			if ch {
				changed = true
			}
			i += n

		case b >= 0x80 && b <= 0x9F:
			n, ch := s.handleC1(data[i:])
			if ch {
				changed = true
			}
			i += n

		default:
			s.handleG1(b)
			changed = true
			i++
		}
	}
	return changed
}

// DisplayText returns the combined visible text from all windows if it changed
// since the last call. Returns empty string if unchanged.
func (s *CEA708Service) DisplayText() string {
	var text string
	for i := 0; i < 8; i++ {
		w := &s.windows[i]
		if !w.visible || !w.defined {
			continue
		}
		if w.contentID == s.lastContentID[i] {
			continue
		}
		wt := w.text()
		if wt != "" {
			if text != "" {
				text += "\n"
			}
			text += wt
		}
	}

	if text != "" && text != s.lastText {
		s.lastText = text
		for i := 0; i < 8; i++ {
			s.lastContentID[i] = s.windows[i].contentID
		}
		return text
	}

	if text == "" && s.lastText != "" {
		anyChanged := false
		for i := 0; i < 8; i++ {
			if s.windows[i].visible && s.windows[i].defined &&
				s.windows[i].contentID != s.lastContentID[i] {
				anyChanged = true
				break
			}
		}
		if anyChanged {
			s.lastText = ""
			for i := 0; i < 8; i++ {
				s.lastContentID[i] = s.windows[i].contentID
			}
			return ""
		}
	}

	return ""
}

func (s *CEA708Service) handleG0(b byte) {
	w := &s.windows[s.currentWin]
	if !w.defined {
		return
	}
	if b == 0x7F {
		w.setCharacter('\u266a')
	} else {
		w.setCharacter(rune(b))
	}
}

func (s *CEA708Service) handleG1(b byte) {
	w := &s.windows[s.currentWin]
	if !w.defined {
		return
	}
	w.setCharacter(rune(b))
}

func (s *CEA708Service) handleC0(data []byte) (int, bool) {
	if len(data) == 0 {
		return 1, false
	}
	b := data[0]
	w := &s.windows[s.currentWin]

	switch {
	case b == 0x00:
		return 1, false
	case b == 0x03:
		return 1, true
	case b == 0x08:
		if w.defined {
			w.backspace()
		}
		return 1, true
	case b == 0x0C:
		if w.defined {
			w.resetMemory()
		}
		return 1, true
	case b == 0x0D:
		if w.defined {
			w.carriageReturn()
		}
		return 1, true
	case b == 0x0E:
		if w.defined {
			w.horizontalCarriageReturn()
		}
		return 1, true
	case b >= 0x01 && b <= 0x0F:
		return 1, false

	case b == 0x10:
		return s.handleEXT1(data)

	case b >= 0x11 && b <= 0x17:
		if len(data) < 2 {
			return len(data), false
		}
		return 2, false

	case b == 0x18:
		if len(data) < 3 {
			return len(data), false
		}
		codePoint := (uint16(data[1]) << 8) | uint16(data[2])
		if codePoint >= 0x20 && w.defined {
			w.setCharacter(rune(codePoint))
			return 3, true
		}
		return 3, false

	case b >= 0x19 && b <= 0x1F:
		if len(data) < 3 {
			return len(data), false
		}
		return 3, false

	default:
		return 1, false
	}
}

func (s *CEA708Service) handleEXT1(data []byte) (int, bool) {
	if len(data) < 2 {
		return len(data), false
	}
	ext := data[1]

	switch {
	case ext <= 0x1F:
		var extra int
		switch {
		case ext <= 0x07:
			extra = 0
		case ext <= 0x0F:
			extra = 1
		case ext <= 0x17:
			extra = 2
		default:
			extra = 3
		}
		total := 2 + extra
		if len(data) < total {
			return len(data), false
		}
		return total, false

	case ext <= 0x7F:
		w := &s.windows[s.currentWin]
		ch := g2Char(ext)
		if ch != 0 && w.defined {
			w.setCharacter(ch)
			return 2, true
		}
		if w.defined {
			w.setCharacter('_')
			return 2, true
		}
		return 2, false

	case ext <= 0x9F:
		var extra int
		if ext <= 0x87 {
			extra = 4
		} else {
			extra = 5
		}
		total := 2 + extra
		if len(data) < total {
			return len(data), false
		}
		return total, false

	default:
		w := &s.windows[s.currentWin]
		if w.defined {
			if ext == 0xA0 {
				w.setCharacter('\U0001F4AC')
			} else {
				w.setCharacter('_')
			}
			return 2, true
		}
		return 2, false
	}
}

func g2Char(code byte) rune {
	g2Map := map[byte]rune{
		0x20: ' ', 0x21: '\u00a0', 0x25: '\u2026',
		0x2a: '\u0160', 0x2c: '\u0152',
		0x30: '\u2588', 0x31: '\u2018', 0x32: '\u2019',
		0x33: '\u201c', 0x34: '\u201d', 0x35: '\u2022',
		0x39: '\u2122', 0x3a: '\u0161', 0x3c: '\u0153',
		0x3d: '\u2120', 0x3f: '\u0178',

		0x40: '\u00c0', 0x41: '\u00c1', 0x42: '\u00c2',
		0x43: '\u00c3', 0x44: '\u00c4', 0x45: '\u00c5',
		0x46: '\u00c6', 0x47: '\u00c7', 0x48: '\u00c8',
		0x49: '\u00c9', 0x4a: '\u00ca', 0x4b: '\u00cb',
		0x4c: '\u00cc', 0x4d: '\u00cd', 0x4e: '\u00ce',
		0x4f: '\u00cf', 0x50: '\u00d0', 0x51: '\u00d1',
		0x52: '\u00d2', 0x53: '\u00d3', 0x54: '\u00d4',
		0x55: '\u00d5', 0x56: '\u00d6', 0x57: '\u00d7',
		0x58: '\u00d8', 0x59: '\u00d9', 0x5a: '\u00da',
		0x5b: '\u00db', 0x5c: '\u00dc', 0x5d: '\u00dd',
		0x5e: '\u00de', 0x5f: '\u00df', 0x60: '\u00e0',
		0x61: '\u00e1', 0x62: '\u00e2', 0x63: '\u00e3',
		0x64: '\u00e4', 0x65: '\u00e5', 0x66: '\u00e6',
		0x67: '\u00e7', 0x68: '\u00e8', 0x69: '\u00e9',
		0x6a: '\u00ea', 0x6b: '\u00eb', 0x6c: '\u00ec',
		0x6d: '\u00ed', 0x6e: '\u00ee', 0x6f: '\u00ef',
		0x70: '\u00f0', 0x71: '\u00f1', 0x72: '\u00f2',
		0x73: '\u00f3', 0x74: '\u00f4', 0x75: '\u00f5',

		0x76: '\u215b', 0x77: '\u215c', 0x78: '\u215d',
		0x79: '\u215e', 0x7a: '\u2502', 0x7b: '\u2510',
		0x7c: '\u2514', 0x7d: '\u2500', 0x7e: '\u2518',
		0x7f: '\u250c',
	}
	if ch, ok := g2Map[code]; ok {
		return ch
	}
	return 0
}

func (s *CEA708Service) handleC1(data []byte) (int, bool) {
	if len(data) == 0 {
		return 1, false
	}
	b := data[0]
	switch {
	case b >= 0x80 && b <= 0x87:
		winID := int(b - 0x80)
		if s.windows[winID].defined {
			s.currentWin = winID
		}
		return 1, false

	case b == 0x88: // CLW
		if len(data) < 2 {
			return len(data), false
		}
		bm := data[1]
		for i := 0; i < 8; i++ {
			if bm&(1<<uint(i)) != 0 && s.windows[i].defined {
				s.windows[i].resetMemory()
			}
		}
		return 2, true

	case b == 0x89: // DSW
		if len(data) < 2 {
			return len(data), false
		}
		bm := data[1]
		changed := false
		for i := 0; i < 8; i++ {
			if bm&(1<<uint(i)) != 0 && s.windows[i].defined {
				if !s.windows[i].visible {
					changed = true
				}
				s.windows[i].visible = true
			}
		}
		return 2, changed

	case b == 0x8A: // HDW
		if len(data) < 2 {
			return len(data), false
		}
		bm := data[1]
		changed := false
		for i := 0; i < 8; i++ {
			if bm&(1<<uint(i)) != 0 && s.windows[i].defined {
				s.windows[i].wasHidden = true
				if s.windows[i].visible {
					changed = true
				}
				s.windows[i].visible = false
			}
		}
		return 2, changed

	case b == 0x8B: // TGW
		if len(data) < 2 {
			return len(data), false
		}
		bm := data[1]
		for i := 0; i < 8; i++ {
			if bm&(1<<uint(i)) != 0 && s.windows[i].defined {
				if s.windows[i].visible {
					s.windows[i].wasHidden = true
				}
				s.windows[i].visible = !s.windows[i].visible
			}
		}
		return 2, true

	case b == 0x8C: // DLW
		if len(data) < 2 {
			return len(data), false
		}
		bm := data[1]
		for i := 0; i < 8; i++ {
			if bm&(1<<uint(i)) != 0 {
				s.windows[i] = cea708Window{}
			}
		}
		return 2, true

	case b == 0x8D: // DLY
		if len(data) < 2 {
			return len(data), false
		}
		return 2, false

	case b == 0x8E: // DLC
		return 1, false

	case b == 0x8F: // RST
		for i := range s.windows {
			s.windows[i] = cea708Window{}
		}
		s.currentWin = 0
		return 1, true

	case b == 0x90: // SPA
		if len(data) < 3 {
			return len(data), false
		}
		s.handleSPA(data[1], data[2])
		return 3, false

	case b == 0x91: // SPC
		if len(data) < 4 {
			return len(data), false
		}
		s.handleSPC(data[1], data[2], data[3])
		return 4, false

	case b == 0x92: // SPL
		if len(data) < 3 {
			return len(data), false
		}
		s.handleSPL(data[1], data[2])
		return 3, false

	case b >= 0x93 && b <= 0x96:
		return 1, false

	case b == 0x97: // SWA
		if len(data) < 5 {
			return len(data), false
		}
		s.handleSWA(data[1], data[2], data[3], data[4])
		return 5, false

	case b >= 0x98 && b <= 0x9F: // DF0-DF7
		if len(data) < 7 {
			return len(data), false
		}
		winID := int(b - 0x98)
		s.handleDefineWindow(winID, data[1:7])
		return 7, true

	default:
		return 1, false
	}
}

func (s *CEA708Service) handleSPA(b1, b2 byte) {
	w := &s.windows[s.currentWin]
	if !w.defined {
		return
	}
	w.pen.penSize = PenSize(b1 & 0x03)
	w.pen.offset = PenOffset((b1 >> 2) & 0x03)
	w.pen.textTag = int((b1 >> 4) & 0x0F)
	w.pen.fontTag = FontTag(b2 & 0x07)
	w.pen.edgeType = EdgeType((b2 >> 3) & 0x07)
	w.pen.underline = b2&0x40 != 0
	w.pen.italic = b2&0x80 != 0
}

func (s *CEA708Service) handleSPC(b1, b2, b3 byte) {
	w := &s.windows[s.currentWin]
	if !w.defined {
		return
	}
	w.pen.fgOpacity = Opacity((b1 >> 6) & 0x03)
	w.pen.fgColor = cea708ColorToHex(b1)
	w.pen.bgOpacity = Opacity((b2 >> 6) & 0x03)
	w.pen.bgColor = cea708ColorToHex(b2)
	w.pen.edgeColor = cea708ColorToHex(b3)
}

var cea708ColorLevels = [4]byte{0x00, 0x55, 0xAA, 0xFF}

func cea708ColorToHex(b byte) string {
	rr := (b >> 4) & 0x03
	gg := (b >> 2) & 0x03
	bb := b & 0x03

	r := cea708ColorLevels[rr]
	g := cea708ColorLevels[gg]
	bv := cea708ColorLevels[bb]

	hex := [6]byte{
		hexDigit(r >> 4), hexDigit(r & 0x0F),
		hexDigit(g >> 4), hexDigit(g & 0x0F),
		hexDigit(bv >> 4), hexDigit(bv & 0x0F),
	}
	return string(hex[:])
}

func hexDigit(v byte) byte {
	if v < 10 {
		return '0' + v
	}
	return 'a' + v - 10
}

func (s *CEA708Service) handleSPL(b1, b2 byte) {
	w := &s.windows[s.currentWin]
	if !w.defined {
		return
	}
	row := int(b1 & 0x0F)
	col := int(b2 & 0x3F)
	if row >= w.rowCount {
		row = w.rowCount - 1
	}
	if col >= w.colCount {
		col = w.colCount - 1
	}
	w.pen.row = row
	w.pen.col = col
}

func (s *CEA708Service) handleSWA(b1, b2, b3, b4 byte) {
	w := &s.windows[s.currentWin]
	if !w.defined {
		return
	}
	w.fillOpacity = Opacity((b1 >> 6) & 0x03)
	w.fillColor = cea708ColorToHex(b1)
	bt0 := (b2 >> 6) & 0x03
	w.borderColor = cea708ColorToHex(b2)
	w.justify = TextJustification(b3 & 0x03)
	w.scrollDirection = ScrollDirection((b3 >> 2) & 0x03)
	w.printDirection = PrintDirection((b3 >> 4) & 0x03)
	w.wordWrap = b3&0x40 != 0
	bt1 := b4 & 0x01
	w.effectSpeed = int((b4 >> 1) & 0x03)
	w.effectDirection = int((b4 >> 3) & 0x07)
	w.displayEffect = DisplayEffect((b4 >> 6) & 0x03)
	w.borderType = BorderType((bt1 << 2) | bt0)
}

func (s *CEA708Service) handleDefineWindow(winID int, b []byte) {
	alreadyDefined := s.windows[winID].defined

	priority := int(b[0] & 0x07)
	colLock := b[0]&0x08 != 0
	rowLock := b[0]&0x10 != 0
	visible := b[0]&0x20 != 0
	relToggle := b[1]&0x80 != 0
	vertAnchor := int(b[1] & 0x7F)
	horAnchor := int(b[2])
	anchorID := AnchorID((b[3] >> 4) & 0x0F)
	rowCount := int(b[3]&0x0F) + 1
	colCount := int(b[4]&0x3F) + 1
	penStyle := b[5] & 0x07
	winStyle := (b[5] >> 3) & 0x07

	if rowCount > cea708MaxRows {
		rowCount = cea708MaxRows
	}
	if colCount > cea708MaxCols {
		colCount = cea708MaxCols
	}

	w := &s.windows[winID]

	if !alreadyDefined {
		w.resetMemory()
	}

	if !alreadyDefined || penStyle != 0 {
		w.resetPen()
		applyPredefinedPenStyle(w, penStyle)
	}

	w.visible = visible
	w.defined = true
	w.rowCount = rowCount
	w.colCount = colCount
	w.anchorV = vertAnchor
	w.anchorH = horAnchor
	w.anchorID = anchorID
	w.relativeToggle = relToggle
	w.priority = priority
	w.rowLock = rowLock
	w.colLock = colLock

	if !alreadyDefined || winStyle != 0 {
		applyPredefinedWindowStyle(w, winStyle)
	}

	s.currentWin = winID
}

func applyPredefinedPenStyle(w *cea708Window, style byte) {
	w.pen.fgColor = "ffffff"
	w.pen.bgColor = "000000"
	w.pen.italic = false
	w.pen.underline = false
	w.pen.fgOpacity = OpacitySolid
	w.pen.edgeType = EdgeNone
	w.pen.edgeColor = "000000"
	w.pen.penSize = PenSizeStandard
	w.pen.offset = PenOffsetNormal

	switch style {
	case 1:
		w.pen.fontTag = FontDefault
		w.pen.bgOpacity = OpacitySolid
	case 2:
		w.pen.fontTag = FontMonoSerif
		w.pen.bgOpacity = OpacitySolid
	case 3:
		w.pen.fontTag = FontPropSerif
		w.pen.bgOpacity = OpacitySolid
	case 4:
		w.pen.fontTag = FontMonoSans
		w.pen.bgOpacity = OpacityTransparent
	case 5:
		w.pen.fontTag = FontPropSans
		w.pen.bgOpacity = OpacityTransparent
	case 6:
		w.pen.fontTag = FontMonoSans
		w.pen.bgOpacity = OpacityTransparent
		w.pen.edgeType = EdgeUniform
	case 7:
		w.pen.fontTag = FontPropSans
		w.pen.bgOpacity = OpacityTransparent
		w.pen.edgeType = EdgeUniform
	}
}

func applyPredefinedWindowStyle(w *cea708Window, style byte) {
	w.fillOpacity = OpacitySolid
	w.fillColor = "000000"
	w.borderType = BorderNone
	w.borderColor = "000000"
	w.wordWrap = false
	w.printDirection = PrintDirLeftToRight
	w.scrollDirection = ScrollDirBottomToTop
	w.displayEffect = EffectSnap
	w.effectDirection = 0
	w.effectSpeed = 0

	switch style {
	case 1:
		w.justify = JustifyLeft
		w.fillOpacity = OpacitySolid
		w.printDirection = PrintDirLeftToRight
		w.scrollDirection = ScrollDirBottomToTop
	case 2:
		w.justify = JustifyCenter
		w.fillOpacity = OpacitySolid
		w.printDirection = PrintDirLeftToRight
		w.scrollDirection = ScrollDirBottomToTop
	case 3:
		w.justify = JustifyLeft
		w.fillOpacity = OpacitySolid
		w.printDirection = PrintDirLeftToRight
		w.scrollDirection = ScrollDirBottomToTop
		w.wordWrap = true
	case 4:
		w.justify = JustifyCenter
		w.fillOpacity = OpacitySolid
		w.printDirection = PrintDirLeftToRight
		w.scrollDirection = ScrollDirBottomToTop
		w.wordWrap = true
	case 5:
		w.justify = JustifyLeft
		w.fillOpacity = OpacityTransparent
		w.printDirection = PrintDirBottomToTop
		w.scrollDirection = ScrollDirRightToLeft
	case 6:
		w.justify = JustifyCenter
		w.fillOpacity = OpacityTransparent
		w.printDirection = PrintDirBottomToTop
		w.scrollDirection = ScrollDirRightToLeft
	case 7:
		w.justify = JustifyLeft
		w.fillOpacity = OpacitySolid
		w.printDirection = PrintDirLeftToRight
		w.scrollDirection = ScrollDirBottomToTop
	}
}

// ParseDTVCCPacket parses a raw DTVCC packet into service blocks.
func ParseDTVCCPacket(packet []byte) []ServiceBlock {
	if len(packet) < 2 {
		return nil
	}

	var blocks []ServiceBlock
	i := 1

	for i < len(packet) {
		if packet[i] == 0x00 {
			break
		}

		serviceNum := int(packet[i] >> 5)
		blockSize := int(packet[i] & 0x1F)
		i++

		if serviceNum == 7 && i < len(packet) {
			serviceNum = int(packet[i] & 0x3F)
			i++
		}

		if blockSize == 0 || i+blockSize > len(packet) {
			break
		}

		blocks = append(blocks, ServiceBlock{
			ServiceNum: serviceNum,
			Data:       packet[i : i+blockSize],
		})
		i += blockSize
	}
	return blocks
}

// DTVCCPacketSize returns the total packet size in bytes given the header byte.
func DTVCCPacketSize(header byte) int {
	sizeCode := int(header & 0x3F)
	if sizeCode == 0 {
		return 128
	}
	return sizeCode * 2
}

// CEA708Decoder reassembles DTVCC byte pairs into complete packets and
// decodes them through a [CEA708Service].
type CEA708Decoder struct {
	buf     []byte
	service CEA708Service
}

// NewCEA708Decoder creates a new CEA-708 packet assembler and decoder.
func NewCEA708Decoder() *CEA708Decoder {
	return &CEA708Decoder{}
}

// AddTriplet processes a DTVCC byte pair. Returns the current display text
// if it changed, or empty string otherwise.
func (d *CEA708Decoder) AddTriplet(pair DTVCCPair) string {
	if pair.Start {
		result := d.drainPacket()
		d.buf = d.buf[:0]
		d.buf = append(d.buf, pair.Data[0], pair.Data[1])
		return result
	}
	d.buf = append(d.buf, pair.Data[0], pair.Data[1])
	return d.drainPacket()
}

// StyledRegions returns structured caption regions from the underlying service.
func (d *CEA708Decoder) StyledRegions() []CaptionRegion {
	return d.service.StyledRegions()
}

func (d *CEA708Decoder) drainPacket() string {
	if len(d.buf) < 1 {
		return ""
	}

	packetSize := DTVCCPacketSize(d.buf[0])
	if len(d.buf) < packetSize {
		return ""
	}

	blocks := ParseDTVCCPacket(d.buf[:packetSize])
	d.buf = d.buf[packetSize:]

	changed := false
	for _, block := range blocks {
		if block.ServiceNum == 1 {
			if d.service.ProcessBlock(block.Data) {
				changed = true
			}
		}
	}

	if changed {
		return d.service.DisplayText()
	}
	return ""
}

// StyledRegions returns structured caption regions from all visible windows.
func (s *CEA708Service) StyledRegions() []CaptionRegion {
	var regions []CaptionRegion
	for i := 0; i < 8; i++ {
		w := &s.windows[i]
		if !w.visible || !w.defined {
			continue
		}
		rows := w.styledRows()
		if len(rows) == 0 {
			continue
		}
		reg := CaptionRegion{
			ID:              i,
			Justify:         int(w.justify),
			ScrollDirection: int(w.scrollDirection),
			PrintDirection:  int(w.printDirection),
			WordWrap:        w.wordWrap,
			FillColor:       w.fillColor,
			FillOpacity:     int(w.fillOpacity),
			BorderColor:     w.borderColor,
			BorderType:      int(w.borderType),
			AnchorV:         w.anchorV,
			AnchorH:         w.anchorH,
			AnchorID:        int(w.anchorID),
			RelativeToggle:  w.relativeToggle,
			Priority:        w.priority,
			Rows:            rows,
		}
		regions = append(regions, reg)
	}
	return regions
}

func (w *cea708Window) styledRows() []CaptionRow {
	var rows []CaptionRow
	for r := 0; r < w.rowCount; r++ {
		spans := w.rowSpans(r)
		if len(spans) == 0 {
			continue
		}
		rows = append(rows, CaptionRow{Row: r, Spans: spans})
	}
	return rows
}

func (w *cea708Window) rowSpans(row int) []CaptionSpan {
	last := -1
	for c := w.colCount - 1; c >= 0; c-- {
		if w.memory[row][c] != nil {
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
		cell := w.memory[row][c]
		var ch rune
		var fg, bg, edge string
		var fgOp, bgOp, ps, ft, off, et int
		var italic, underline bool

		if cell != nil {
			ch = cell.ch
			fg = cell.fgColor
			bg = cell.bgColor
			fgOp = int(cell.fgOpacity)
			bgOp = int(cell.bgOpacity)
			italic = cell.italic
			underline = cell.underline
			edge = cell.edgeColor
			et = int(cell.edgeType)
			ps = int(cell.penSize)
			ft = int(cell.fontTag)
			off = int(cell.offset)
		} else {
			ch = ' '
			fg = "ffffff"
			bg = "000000"
			edge = "000000"
		}

		sameStyle := started &&
			cur.FgColor == fg && cur.BgColor == bg &&
			cur.FgOpacity == fgOp && cur.BgOpacity == bgOp &&
			cur.Italic == italic && cur.Underline == underline &&
			cur.EdgeColor == edge && cur.EdgeType == et &&
			cur.PenSize == ps && cur.FontTag == ft && cur.Offset == off

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
				FgOpacity: fgOp,
				BgOpacity: bgOp,
				Italic:    italic,
				Underline: underline,
				EdgeColor: edge,
				EdgeType:  et,
				PenSize:   ps,
				FontTag:   ft,
				Offset:    off,
			}
			started = true
		}
	}
	if started && cur.Text != "" {
		spans = append(spans, cur)
	}
	return spans
}
