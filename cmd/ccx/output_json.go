package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/zsiec/ccx"
)

type jsonSpan struct {
	Text      string `json:"text"`
	FgColor   string `json:"fg_color,omitempty"`
	BgColor   string `json:"bg_color,omitempty"`
	FgOpacity int    `json:"fg_opacity,omitempty"`
	BgOpacity int    `json:"bg_opacity,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Underline bool   `json:"underline,omitempty"`
	Flash     bool   `json:"flash,omitempty"`
	PenSize   string `json:"pen_size,omitempty"`
	Font      string `json:"font,omitempty"`
	EdgeType  string `json:"edge_type,omitempty"`
	EdgeColor string `json:"edge_color,omitempty"`
}

type jsonRow struct {
	Row   int        `json:"row"`
	Spans []jsonSpan `json:"spans"`
}

type jsonRegion struct {
	ID              int       `json:"id"`
	Justify         string    `json:"justify,omitempty"`
	ScrollDirection string    `json:"scroll_direction,omitempty"`
	PrintDirection  string    `json:"print_direction,omitempty"`
	WordWrap        bool      `json:"word_wrap,omitempty"`
	FillColor       string    `json:"fill_color,omitempty"`
	FillOpacity     string    `json:"fill_opacity,omitempty"`
	BorderColor     string    `json:"border_color,omitempty"`
	BorderType      string    `json:"border_type,omitempty"`
	AnchorV         int       `json:"anchor_v,omitempty"`
	AnchorH         int       `json:"anchor_h,omitempty"`
	AnchorPoint     string    `json:"anchor_point,omitempty"`
	Priority        int       `json:"priority,omitempty"`
	Rows            []jsonRow `json:"rows"`
}

type jsonFrame struct {
	NALIndex int          `json:"nal_index"`
	Standard string       `json:"standard"`
	Codec    string       `json:"codec"`
	Channel  int          `json:"channel,omitempty"`
	Text     string       `json:"text"`
	Regions  []jsonRegion `json:"regions,omitempty"`
}

type jsonCC608Pair struct {
	Data    string `json:"data"`
	Channel int    `json:"channel"`
	Field   int    `json:"field"`
}

type jsonDTVCCPair struct {
	Data  string `json:"data"`
	Start bool   `json:"start"`
}

type jsonRawEvent struct {
	NALIndex int             `json:"nal_index"`
	Codec    string          `json:"codec"`
	CC608    []jsonCC608Pair `json:"cc608,omitempty"`
	DTVCC    []jsonDTVCCPair `json:"dtvcc,omitempty"`
}

type jsonOutputEvent struct {
	Frame *jsonFrame    `json:"frame,omitempty"`
	Raw   *jsonRawEvent `json:"raw,omitempty"`
}

type jsonOutput struct {
	w       io.Writer
	raw     bool
	events  []jsonOutputEvent
	encoder *json.Encoder
	stream  bool
}

func newJSONOutput(w io.Writer, raw bool, stream bool) *jsonOutput {
	o := &jsonOutput{w: w, raw: raw, stream: stream}
	if stream {
		o.encoder = json.NewEncoder(w)
	}
	return o
}

func (o *jsonOutput) event(ev captionEvent) {
	var jev jsonOutputEvent

	if o.raw && ev.Data != nil {
		raw := jsonRawEvent{
			NALIndex: ev.NALIndex,
			Codec:    ev.Codec.String(),
		}
		for _, p := range ev.Data.CC608Pairs {
			raw.CC608 = append(raw.CC608, jsonCC608Pair{
				Data:    fmt.Sprintf("%02x%02x", p.Data[0], p.Data[1]),
				Channel: p.Channel,
				Field:   int(p.Field),
			})
		}
		for _, p := range ev.Data.DTVCC {
			raw.DTVCC = append(raw.DTVCC, jsonDTVCCPair{
				Data:  fmt.Sprintf("%02x%02x", p.Data[0], p.Data[1]),
				Start: p.Start,
			})
		}
		jev.Raw = &raw
	}

	if ev.Frame608 != nil {
		jev.Frame = buildJSONFrame("608", ev, ev.Frame608)
	} else if ev.Frame708 != nil {
		jev.Frame = buildJSONFrame("708", ev, ev.Frame708)
	}

	if jev.Frame == nil && jev.Raw == nil {
		return
	}

	if o.stream {
		o.encoder.Encode(jev)
	} else {
		o.events = append(o.events, jev)
	}
}

func (o *jsonOutput) summary() {
	if o.stream {
		return
	}
	enc := json.NewEncoder(o.w)
	enc.SetIndent("", "  ")
	enc.Encode(o.events)
}

func buildJSONFrame(standard string, ev captionEvent, frame *ccx.CaptionFrame) *jsonFrame {
	jf := &jsonFrame{
		NALIndex: ev.NALIndex,
		Standard: standard,
		Codec:    ev.Codec.String(),
		Channel:  frame.Channel,
		Text:     frame.PlainText(),
	}

	for _, reg := range frame.Regions {
		jr := jsonRegion{
			ID:              reg.ID,
			Justify:         justifyName(reg.Justify),
			ScrollDirection: directionName(reg.ScrollDirection),
			PrintDirection:  directionName(reg.PrintDirection),
			WordWrap:        reg.WordWrap,
			FillColor:       colorStr(reg.FillColor),
			FillOpacity:     opacityName(reg.FillOpacity),
			BorderColor:     colorStr(reg.BorderColor),
			BorderType:      borderName(reg.BorderType),
			AnchorV:         reg.AnchorV,
			AnchorH:         reg.AnchorH,
			AnchorPoint:     anchorName(reg.AnchorID),
			Priority:        reg.Priority,
		}
		for _, row := range reg.Rows {
			jrow := jsonRow{Row: row.Row}
			for _, span := range row.Spans {
				jrow.Spans = append(jrow.Spans, jsonSpan{
					Text:      span.Text,
					FgColor:   colorStr(span.FgColor),
					BgColor:   colorStr(span.BgColor),
					FgOpacity: span.FgOpacity,
					BgOpacity: span.BgOpacity,
					Italic:    span.Italic,
					Underline: span.Underline,
					Flash:     span.Flash,
					PenSize:   penSizeName(span.PenSize),
					Font:      fontName(span.FontTag),
					EdgeType:  edgeName(span.EdgeType),
					EdgeColor: colorStr(span.EdgeColor),
				})
			}
			jr.Rows = append(jr.Rows, jrow)
		}
		jf.Regions = append(jf.Regions, jr)
	}
	return jf
}

func colorStr(c string) string {
	if c == "" {
		return ""
	}
	return "#" + c
}

func justifyName(j int) string {
	switch ccx.TextJustification(j) {
	case ccx.JustifyLeft:
		return "left"
	case ccx.JustifyRight:
		return "right"
	case ccx.JustifyCenter:
		return "center"
	case ccx.JustifyFull:
		return "full"
	default:
		return "left"
	}
}

func directionName(d int) string {
	names := [4]string{"left-to-right", "right-to-left", "top-to-bottom", "bottom-to-top"}
	if d >= 0 && d < len(names) {
		return names[d]
	}
	return "left-to-right"
}

func opacityName(o int) string {
	names := [4]string{"solid", "flash", "translucent", "transparent"}
	if o >= 0 && o < len(names) {
		return names[o]
	}
	return "solid"
}

func borderName(b int) string {
	names := [6]string{"none", "raised", "depressed", "uniform", "left-drop-shadow", "right-drop-shadow"}
	if b >= 0 && b < len(names) {
		return names[b]
	}
	return "none"
}

func anchorName(a int) string {
	names := [9]string{
		"upper-left", "upper-center", "upper-right",
		"middle-left", "middle-center", "middle-right",
		"lower-left", "lower-center", "lower-right",
	}
	if a >= 0 && a < len(names) {
		return names[a]
	}
	return "upper-left"
}

func penSizeName(s int) string {
	names := [3]string{"small", "standard", "large"}
	if s >= 0 && s < len(names) {
		return names[s]
	}
	return "standard"
}

func fontName(f int) string {
	names := [8]string{
		"default", "mono-serif", "prop-serif", "mono-sans",
		"prop-sans", "casual", "cursive", "small-caps",
	}
	if f >= 0 && f < len(names) {
		return names[f]
	}
	return "default"
}

func edgeName(e int) string {
	names := [6]string{"none", "raised", "depressed", "uniform", "left-drop-shadow", "right-drop-shadow"}
	if e >= 0 && e < len(names) {
		return names[e]
	}
	return "none"
}
