package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const version = "0.1.0"

const usage = `ccx - extract and decode closed captions from H.264/H.265 video streams

Usage:
  ccx [flags] [file]

  Reads an Annex B bitstream from a file or stdin and extracts CEA-608/708
  closed captions. The codec (H.264 vs H.265) is auto-detected from NAL
  unit headers, or can be forced with --codec.

  To extract from an MP4 or other container, pipe through ffmpeg:
    ffmpeg -i video.mp4 -c:v copy -bsf:v h264_mp4toannexb -f h264 - | ccx
    ffmpeg -i video.mp4 -c:v copy -bsf:v hevc_mp4toannexb -f hevc - | ccx

Output Formats:
  text     Caption text with NAL indices (default)
  json     Structured JSON with full styling/positioning data
  render   Visual terminal rendering with ANSI colors
  raw      Hex dump of caption byte pairs with protocol decoding

Channel Filter:
  By default, 708 captions are shown when present (higher fidelity),
  falling back to 608 if no 708 data is found. Use -ch to override:
    708      CEA-708 only
    608      CEA-608 only
    cc1-cc4  specific CEA-608 channel
    all      show all channels

Flags:
`

func main() {
	var (
		format    string
		codecStr  string
		channel   string
		verbose   bool
		jsonRaw   bool
		ndjson    bool
		showVer   bool
	)

	flag.StringVar(&format, "format", "text", "output format: text, json, render, raw")
	flag.StringVar(&format, "f", "text", "output format (shorthand)")
	flag.StringVar(&codecStr, "codec", "", "force codec: h264, h265 (default: auto-detect)")
	flag.StringVar(&channel, "channel", "", "filter to channel: 608, 708, cc1-cc4, all (default: 708 if present, then 608)")
	flag.StringVar(&channel, "ch", "", "filter to channel (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "show extra detail in text mode")
	flag.BoolVar(&verbose, "v", false, "verbose (shorthand)")
	flag.BoolVar(&jsonRaw, "json-raw", false, "include raw byte pairs in JSON output")
	flag.BoolVar(&ndjson, "ndjson", false, "stream JSON as newline-delimited objects")
	flag.BoolVar(&showVer, "version", false, "print version and exit")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  ccx testdata/h264_608_rollup2.h264")
		fmt.Fprintln(os.Stderr, "  ccx -f render testdata/h264_708_multiwindow.h264")
		fmt.Fprintln(os.Stderr, "  ccx -f json testdata/h265_608_pac_colors.h265")
		fmt.Fprintln(os.Stderr, "  ccx -f raw testdata/h264_608_popon.h264")
		fmt.Fprintln(os.Stderr, "  ffmpeg -i video.mp4 -c:v copy -bsf:v h264_mp4toannexb -f h264 - 2>/dev/null | ccx")
		fmt.Fprintln(os.Stderr, "  ffmpeg -i video.mp4 -c:v copy -bsf:v h264_mp4toannexb -f h264 - 2>/dev/null | ccx -f render")
		fmt.Fprintln(os.Stderr)
	}
	flag.Parse()

	if showVer {
		fmt.Printf("ccx %s\n", version)
		os.Exit(0)
	}

	forceCodec := codecUnknown
	switch strings.ToLower(codecStr) {
	case "h264", "avc":
		forceCodec = codecH264
	case "h265", "hevc":
		forceCodec = codecH265
	case "":
	default:
		fatal("unknown codec %q (use h264 or h265)", codecStr)
	}

	var r io.Reader
	var inputName string

	if flag.NArg() > 0 {
		path := flag.Arg(0)
		f, err := os.Open(path)
		if err != nil {
			fatal("%v", err)
		}
		defer f.Close()
		r = f
		inputName = filepath.Base(path)

		if forceCodec == codecUnknown {
			forceCodec = codecFromExtension(path)
		}
	} else {
		stat, _ := os.Stdin.Stat()
		if stat.Mode()&os.ModeCharDevice != 0 {
			flag.Usage()
			os.Exit(1)
		}
		r = os.Stdin
		inputName = "stdin"
	}

	w := os.Stdout

	type outputSink interface {
		event(captionEvent)
		summary()
	}

	var sink outputSink
	switch strings.ToLower(format) {
	case "text", "t":
		sink = newTextOutput(w, verbose)
	case "json", "j":
		sink = newJSONOutput(w, jsonRaw, ndjson)
	case "render", "r":
		sink = newRenderOutput(w)
	case "raw", "d":
		sink = newRawOutput(w)
	default:
		fatal("unknown format %q (use text, json, render, or raw)", format)
	}

	cf := parseChannelFilter(channel)

	if format != "json" {
		printBanner(w, inputName, forceCodec)
	}

	var events []captionEvent
	err := parseStream(r, forceCodec, func(ev captionEvent) {
		events = append(events, ev)
	})
	if err != nil {
		fatal("parse error: %v", err)
	}

	if cf == channelFilterAuto {
		cf = autoSelectChannel(events)
	}

	for i := range events {
		filterEvent(&events[i], cf)
		ev := events[i]
		hasFrames := ev.Frame608 != nil || ev.Frame708 != nil
		hasRawData := ev.Data != nil && (len(ev.Data.CC608Pairs) > 0 || len(ev.Data.DTVCC) > 0)
		if hasFrames || hasRawData {
			sink.event(ev)
		}
	}

	sink.summary()
}

func printBanner(w io.Writer, input string, c codec) {
	codecStr := "auto-detect"
	if c != codecUnknown {
		codecStr = c.String()
	}
	fmt.Fprintf(w, "ccx %s │ %s │ codec: %s\n", version, input, codecStr)
	fmt.Fprintln(w, strings.Repeat("─", 60))
}

func codecFromExtension(path string) codec {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".h264", ".264", ".avc":
		return codecH264
	case ".h265", ".265", ".hevc":
		return codecH265
	default:
		return codecUnknown
	}
}

type channelFilter int

const (
	channelFilterAuto channelFilter = iota
	channelFilterAll
	channelFilter608
	channelFilter708
	channelFilterCC1
	channelFilterCC2
	channelFilterCC3
	channelFilterCC4
)

func parseChannelFilter(s string) channelFilter {
	switch strings.ToLower(s) {
	case "":
		return channelFilterAuto
	case "all":
		return channelFilterAll
	case "608":
		return channelFilter608
	case "708":
		return channelFilter708
	case "cc1", "1":
		return channelFilterCC1
	case "cc2", "2":
		return channelFilterCC2
	case "cc3", "3":
		return channelFilterCC3
	case "cc4", "4":
		return channelFilterCC4
	default:
		fatal("unknown channel %q (use 608, 708, cc1-cc4, or all)", s)
		return channelFilterAll
	}
}

func autoSelectChannel(events []captionEvent) channelFilter {
	has708 := false
	has608 := false
	for _, ev := range events {
		if ev.Frame708 != nil {
			has708 = true
		}
		if ev.Frame608 != nil {
			has608 = true
		}
		if has708 && has608 {
			break
		}
	}
	if has708 {
		return channelFilter708
	}
	if has608 {
		return channelFilter608
	}
	return channelFilterAll
}

func filterEvent(ev *captionEvent, cf channelFilter) {
	switch cf {
	case channelFilterAll:
		return
	case channelFilter708:
		ev.Frame608 = nil
	case channelFilter608:
		ev.Frame708 = nil
	case channelFilterCC1:
		ev.Frame708 = nil
		if ev.Frame608 != nil && ev.Frame608.Channel != 1 {
			ev.Frame608 = nil
		}
	case channelFilterCC2:
		ev.Frame708 = nil
		if ev.Frame608 != nil && ev.Frame608.Channel != 2 {
			ev.Frame608 = nil
		}
	case channelFilterCC3:
		ev.Frame708 = nil
		if ev.Frame608 != nil && ev.Frame608.Channel != 3 {
			ev.Frame608 = nil
		}
	case channelFilterCC4:
		ev.Frame708 = nil
		if ev.Frame608 != nil && ev.Frame608.Channel != 4 {
			ev.Frame608 = nil
		}
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ccx: "+format+"\n", args...)
	os.Exit(1)
}
