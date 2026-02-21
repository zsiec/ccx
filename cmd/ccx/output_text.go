package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/zsiec/ccx"
)

type textOutput struct {
	w       io.Writer
	verbose bool
	count   int
}

func newTextOutput(w io.Writer, verbose bool) *textOutput {
	return &textOutput{w: w, verbose: verbose}
}

func (o *textOutput) event(ev captionEvent) {
	if ev.Frame608 != nil {
		o.count++
		o.printFrame("608", ev, ev.Frame608)
	}
	if ev.Frame708 != nil {
		o.count++
		o.printFrame("708", ev, ev.Frame708)
	}
}

func (o *textOutput) printFrame(standard string, ev captionEvent, frame *ccx.CaptionFrame) {
	header := fmt.Sprintf("[%s] NAL #%d", standard, ev.NALIndex)
	if frame.Channel > 0 {
		header += fmt.Sprintf(" CC%d", frame.Channel)
	}

	text := frame.PlainText()
	if text == "" {
		text = "(cleared)"
	}

	if o.verbose {
		fmt.Fprintf(o.w, "%s\n", header)
		for _, line := range strings.Split(text, "\n") {
			fmt.Fprintf(o.w, "  %s\n", line)
		}
		if len(frame.Regions) > 0 {
			for _, reg := range frame.Regions {
				for _, row := range reg.Rows {
					for _, span := range row.Spans {
						attrs := spanAttrs(span)
						if attrs != "" {
							fmt.Fprintf(o.w, "  style: %s\n", attrs)
						}
					}
				}
			}
		}
		fmt.Fprintln(o.w)
	} else {
		lines := strings.Split(text, "\n")
		fmt.Fprintf(o.w, "%s  %s\n", header, strings.Join(lines, " | "))
	}
}

func spanAttrs(span ccx.CaptionSpan) string {
	var parts []string
	if span.FgColor != "" && span.FgColor != "ffffff" {
		parts = append(parts, "fg=#"+span.FgColor)
	}
	if span.BgColor != "" && span.BgColor != "000000" {
		parts = append(parts, "bg=#"+span.BgColor)
	}
	if span.Italic {
		parts = append(parts, "italic")
	}
	if span.Underline {
		parts = append(parts, "underline")
	}
	if span.Flash {
		parts = append(parts, "flash")
	}
	return strings.Join(parts, " ")
}

func (o *textOutput) summary() {
	fmt.Fprintf(o.w, "\n%d caption update(s)\n", o.count)
}
