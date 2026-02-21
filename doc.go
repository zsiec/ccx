// Package ccx decodes CEA-608 and CEA-708 closed captions from H.264 and
// H.265/HEVC video streams and provides a compact binary codec for transporting
// styled caption data over the wire.
//
// The package has three layers that can be used independently:
//
// # Extraction
//
// [ExtractCaptions] pulls CEA-608 byte pairs and DTVCC (CEA-708) data out of
// H.264 SEI NAL units containing ATSC A/53 user data. [ExtractCaptionsHEVC]
// does the same for H.265/HEVC streams, which use a 2-byte NAL header and
// SEI NAL types 39/40 instead of H.264's type 6.
//
// Both functions handle emulation prevention byte removal, parity stripping,
// field routing, and channel identification. The inner SEI message structure
// and A/53 caption payload are identical across both codecs.
//
//	// H.264
//	cd := ccx.ExtractCaptions(nalData)
//
//	// H.265/HEVC
//	cd := ccx.ExtractCaptionsHEVC(nalData)
//
//	for _, pair := range cd.CC608Pairs {
//	    text := decoder.Decode(pair.Data[0], pair.Data[1])
//	}
//
// Helper functions [IsH264SEI] and [IsHEVCSEI] identify SEI NAL units by
// their header bytes, useful when scanning Annex B or MPEG-TS bitstreams.
//
// # Decoding
//
// [CEA608Decoder] implements the CTA-608-E state machine with full support
// for pop-on, roll-up, and paint-on modes; all character sets (G0, special,
// extended Spanish/French, extended Portuguese/German); PAC row positioning
// and styling; midrow style changes; and background color attributes.
//
// [CEA708Service] implements the CTA-708-E windowed caption model with up to
// 8 independent windows, full SPA/SPC/SWA/DefineWindow attribute parsing,
// G0/G1/G2/G3 character sets, and all C0/C1 control codes.
//
// Both decoders produce structured output via their StyledRegions() methods,
// returning [CaptionRegion] slices that preserve all styling, positioning, and
// window attributes from the original caption stream.
//
// # Codec
//
// [CaptionFrame.Serialize] encodes a complete caption frame — including all
// regions, rows, spans, colors, opacity, font, edge effects, and positioning —
// into a compact binary format suitable for WebTransport, WebSocket, or any
// byte-oriented transport. [DeserializeCaptionFrame] decodes it back.
//
// The wire format is versioned (currently v2) and backward-compatible. A
// reference TypeScript parser is included in the js/ directory for browser-side
// decoding.
package ccx
