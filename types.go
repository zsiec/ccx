package ccx

// CaptionFrame holds the decoded caption data for a single presentation
// timestamp. It contains either structured region data (from the CEA-608/708
// decoders) or legacy plain text.
type CaptionFrame struct {
	PTS     int64
	Text    string
	Channel int
	Regions []CaptionRegion
}

// CaptionRegion represents a positioned, styled caption region. For CEA-608
// there is typically one region; for CEA-708 there can be up to 8 (one per
// window).
type CaptionRegion struct {
	ID              int
	Justify         int
	ScrollDirection int
	PrintDirection  int
	WordWrap        bool
	FillColor       string
	FillOpacity     int
	BorderColor     string
	BorderType      int
	AnchorV         int
	AnchorH         int
	AnchorID        int
	RelativeToggle  bool
	Priority        int
	Rows            []CaptionRow
}

// CaptionRow is a single row of styled text within a region.
type CaptionRow struct {
	Row   int
	Spans []CaptionSpan
}

// CaptionSpan is a contiguous run of text sharing the same style attributes.
type CaptionSpan struct {
	Text      string
	FgColor   string
	BgColor   string
	FgOpacity int
	BgOpacity int
	Italic    bool
	Underline bool
	Flash     bool
	PenSize   int
	FontTag   int
	Offset    int
	EdgeType  int
	EdgeColor string
}

// PlainText returns the plain text content from regions, or the legacy Text
// field if no regions are present.
func (f *CaptionFrame) PlainText() string {
	if len(f.Regions) == 0 {
		return f.Text
	}
	var result string
	for _, reg := range f.Regions {
		for _, row := range reg.Rows {
			line := ""
			for _, span := range row.Spans {
				line += span.Text
			}
			if line != "" {
				if result != "" {
					result += "\n"
				}
				result += line
			}
		}
	}
	return result
}

// CCPair represents a CEA-608 byte pair with its channel identifier.
type CCPair struct {
	Data    [2]byte
	Channel int
	Field   byte
}

// DTVCCPair holds two DTVCC bytes and whether this is a packet start.
type DTVCCPair struct {
	Data  [2]byte
	Start bool
}

// CaptionData holds both CEA-608 pairs and DTVCC (CEA-708) byte data
// extracted from a single SEI NAL.
type CaptionData struct {
	CC608Pairs []CCPair
	DTVCC      []DTVCCPair
}

// ServiceBlock is a single service block parsed from a DTVCC packet.
type ServiceBlock struct {
	ServiceNum int
	Data       []byte
}
