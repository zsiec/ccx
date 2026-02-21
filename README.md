# ccx

A Go library for extracting, decoding, and encoding CEA-608/708 closed captions from H.264 video streams. Zero dependencies.

```
go get github.com/zsiec/ccx
```

## What It Does

**ccx** turns raw H.264 NAL units into fully structured, styled caption data — and provides a compact binary codec for sending that data over the wire.

```
H.264 SEI NAL → ExtractCaptions() → CEA608Decoder / CEA708Service → StyledRegions() → Serialize()
```

Three layers, usable independently:

| Layer | What | Entry Points |
|-------|------|-------------|
| **Extract** | Pull caption bytes out of H.264 SEI NALs | `ExtractCaptions()`, `ExtractCEA608()` |
| **Decode** | Full CEA-608/708 state machines | `CEA608Decoder`, `CEA708Service`, `CEA708Decoder` |
| **Codec** | Compact binary serialization | `CaptionFrame.Serialize()`, `DeserializeCaptionFrame()` |

## Quick Start

### Decode CEA-608

```go
dec := ccx.NewCEA608Decoder()

// Feed byte pairs from your caption stream
text := dec.Decode(cc1, cc2)
if text != "" {
    fmt.Println(text) // Plain text output
}

// Or get full structured data with styling
regions := dec.StyledRegions()
for _, reg := range regions {
    for _, row := range reg.Rows {
        for _, span := range row.Spans {
            fmt.Printf("%q fg=%s italic=%v\n", span.Text, span.FgColor, span.Italic)
        }
    }
}
```

### Decode CEA-708

```go
svc := ccx.NewCEA708Service()

// Define a window and write text
svc.ProcessBlock(data)
text := svc.DisplayText()

// Structured output with full window attributes
regions := svc.StyledRegions()
// regions[i].Justify, .FillColor, .AnchorV, .Priority, etc.
```

### Extract from H.264

```go
// Pull captions from an SEI NAL unit
cd := ccx.ExtractCaptions(nalData)

// Decode CEA-608
dec608 := ccx.NewCEA608Decoder()
for _, pair := range cd.CC608Pairs {
    text := dec608.Decode(pair.Data[0], pair.Data[1])
}

// Decode CEA-708
dec708 := ccx.NewCEA708Decoder()
for _, pair := range cd.DTVCC {
    text := dec708.AddTriplet(pair)
}
```

### Serialize for Transport

```go
frame := &ccx.CaptionFrame{
    Channel: 1,
    Regions: regions, // from StyledRegions()
}

// Encode to compact binary (~60 bytes for typical caption)
wire := frame.Serialize()

// Decode on the other end
decoded := ccx.DeserializeCaptionFrame(wire)
```

A TypeScript decoder is included in [`js/decode.ts`](js/decode.ts) for browser-side parsing.

## CEA-608 Coverage

Full CTA-608-E compliance:

- **Modes**: Pop-on, roll-up (2/3/4 rows), paint-on
- **Characters**: G0 (with spec overrides for `á`, `é`, `í`, `ó`, `ú`, `ç`, `÷`, `Ñ`, `ñ`), special characters (®, °, ½, ¿, ™, ¢, £, ♪, etc.), extended Spanish/French, extended Portuguese/German
- **Control codes**: RCL, BS, AOF/AON, DER, RU2/3/4, FON, RDC, TR, RTD, EDM, CR, ENM, EOC
- **Styling**: PAC row positioning with 8 colors + italic, midrow style changes, background color attributes, underline, flash
- **Correctness**: PAC row decode table verified against CTA-608-E Table 2, EOC performs proper memory swap (not clear), mode transitions clear display per spec, CR works in paint-on mode

## CEA-708 Coverage

Full CTA-708-E compliance:

- **Windows**: 8 independent windows with define, show, hide, toggle, delete, clear, reset
- **Character sets**: G0 (ASCII + music note), G1 (Latin-1), G2 (60+ characters including extended Latin, fractions, box-drawing), G3 (CC icon)
- **Control codes**: All C0 codes (NUL, ETX, BS, FF, CR, HCR, P16), all C1 codes (CW, CLW, DSW, HDW, TGW, DLW, DLY, DLC, RST, SPA, SPC, SPL, SWA, DF0-7), correct handling of reserved commands (0x93-0x96 consume 1 byte)
- **Attributes**: Full SPA parsing (pen size, font tag, offset, edge type, italic, underline), full SPC parsing (fg/bg colors, fg/bg opacity, edge color), full SWA parsing (fill color/opacity, border color/type, justify, scroll/print direction, word wrap, display effect, effect speed/direction)
- **Positioning**: Anchor point (9 positions), vertical/horizontal anchors, relative toggle, priority, row/column locks
- **Predefined styles**: 7 pen styles (varying fonts, opacity, edge effects), 7 window styles (varying justification, opacity, direction)
- **DTVCC packets**: Packet assembly, size decoding, service block parsing, extended service numbers

## Wire Format (v2)

The binary codec is designed for real-time transport (WebTransport, WebSocket, etc.). Typical caption frames serialize to 40-120 bytes.

```
Header:
  [2] magic    0xCC02
  [1] version  2
  [1] channel
  [1] region count

Per region:
  [1] id
  [1] flags    justify(2) | scrollDir(2) | printDir(2) | wordWrap(1) | relativeToggle(1)
  [1] fill     fillOpacity(2) | borderType(3) | priority(3)
  [3] fillColor (RGB)
  [3] borderColor (RGB)
  [1] anchorV
  [1] anchorH
  [1] anchorID
  [2] row count (big-endian)

  Per row:
    [1] row index
    [1] span count

    Per span:
      [2]  text length (big-endian)
      [n]  text (UTF-8)
      [3]  fgColor (RGB)
      [3]  bgColor (RGB)
      [1]  attr0   fgOpacity(2) | bgOpacity(2) | italic(1) | underline(1) | flash(1) | penSize[1](1)
      [1]  attr1   penSize[0](1) | fontTag(3) | offset(2) | edgeType[1:0](2)
      [1]  attr2   edgeType[2](1) | reserved(7)
      [3]  edgeColor (RGB)
```

Legacy fallback (no magic): `[1] channel [n] text`

## Project Structure

```
ccx/
├── doc.go           # Package documentation
├── types.go         # CaptionFrame, CaptionRegion, CaptionRow, CaptionSpan
├── extract.go       # H.264 SEI → caption byte extraction
├── cea608.go        # CEA-608 decoder state machine
├── cea708.go        # CEA-708 decoder state machine
├── enums.go         # CEA-708 typed constants
├── codec.go         # Binary serialization/deserialization
├── js/
│   └── decode.ts    # TypeScript reference decoder
└── examples/
    ├── decode/      # Decoder usage example
    └── extract/     # H.264 extraction example
```

## License

MIT
