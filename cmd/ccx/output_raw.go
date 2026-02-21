package main

import (
	"fmt"
	"io"
	"strings"
)

type rawOutput struct {
	w     io.Writer
	count int
}

func newRawOutput(w io.Writer) *rawOutput {
	return &rawOutput{w: w}
}

func (o *rawOutput) event(ev captionEvent) {
	if ev.Data == nil {
		return
	}
	o.count++

	fmt.Fprintf(o.w, "── NAL #%d (%s) ", ev.NALIndex, ev.Codec)
	fmt.Fprintln(o.w, strings.Repeat("─", 40))

	if len(ev.Data.CC608Pairs) > 0 {
		fmt.Fprintf(o.w, "  CEA-608 (%d pairs):\n", len(ev.Data.CC608Pairs))
		for i, pair := range ev.Data.CC608Pairs {
			cc1 := pair.Data[0]
			cc2 := pair.Data[1]

			desc := describe608(cc1, cc2)
			fmt.Fprintf(o.w, "    [%d] %02X %02X  CC%d field=%d  %s\n",
				i, cc1, cc2, pair.Channel, pair.Field, desc)
		}
	}

	if len(ev.Data.DTVCC) > 0 {
		fmt.Fprintf(o.w, "  DTVCC (%d pairs):\n", len(ev.Data.DTVCC))
		for i, pair := range ev.Data.DTVCC {
			marker := "cont"
			if pair.Start {
				marker = "START"
			}
			fmt.Fprintf(o.w, "    [%d] %02X %02X  %s\n",
				i, pair.Data[0], pair.Data[1], marker)
		}
	}
	fmt.Fprintln(o.w)
}

func describe608(cc1, cc2 byte) string {
	if cc1 >= 0x20 && cc1 <= 0x7F {
		s := printableChar(cc1)
		if cc2 >= 0x20 && cc2 <= 0x7F {
			s += printableChar(cc2)
		}
		return fmt.Sprintf("text %q", s)
	}

	if cc1 < 0x10 || cc1 > 0x1F {
		return ""
	}

	cmdClass := cc1 & 0x07

	if cmdClass == 4 || cmdClass == 5 {
		if cc2 < 0x40 {
			return describeMisc(cc2)
		}
	}

	if cmdClass == 7 {
		switch {
		case cc2 == 0x21:
			return "TO1 (tab offset 1)"
		case cc2 == 0x22:
			return "TO2 (tab offset 2)"
		case cc2 == 0x23:
			return "TO3 (tab offset 3)"
		case cc2 == 0x2D:
			return "BT (background transparent)"
		}
	}

	if cmdClass == 1 && cc2 >= 0x30 && cc2 <= 0x3F {
		return "special char"
	}
	if (cmdClass == 2 || cmdClass == 3) && cc2 >= 0x20 && cc2 <= 0x3F {
		return "extended char"
	}
	if cmdClass == 1 && cc2 >= 0x20 && cc2 <= 0x2F {
		return "midrow style"
	}
	if cmdClass == 0 && cc2 >= 0x20 && cc2 <= 0x2F {
		return "background attr"
	}

	if cc2 >= 0x40 && cc2 <= 0x7F {
		return "PAC (preamble address code)"
	}

	return fmt.Sprintf("control %02X %02X", cc1, cc2)
}

func describeMisc(cc2 byte) string {
	switch cc2 {
	case 0x20:
		return "RCL (resume caption loading)"
	case 0x21:
		return "BS (backspace)"
	case 0x22:
		return "AOF"
	case 0x23:
		return "AON"
	case 0x24:
		return "DER (delete to end of row)"
	case 0x25:
		return "RU2 (roll-up 2 rows)"
	case 0x26:
		return "RU3 (roll-up 3 rows)"
	case 0x27:
		return "RU4 (roll-up 4 rows)"
	case 0x28:
		return "FON (flash on)"
	case 0x29:
		return "RDC (resume direct captioning)"
	case 0x2A:
		return "TR (text restart)"
	case 0x2B:
		return "RTD (resume text display)"
	case 0x2C:
		return "EDM (erase displayed memory)"
	case 0x2D:
		return "CR (carriage return)"
	case 0x2E:
		return "ENM (erase non-displayed memory)"
	case 0x2F:
		return "EOC (end of caption / flip)"
	default:
		return fmt.Sprintf("misc %02X", cc2)
	}
}

func printableChar(b byte) string {
	if b >= 0x20 && b < 0x7F {
		return string(rune(b))
	}
	return fmt.Sprintf("\\x%02x", b)
}

func (o *rawOutput) summary() {
	fmt.Fprintf(o.w, "%d NAL unit(s) with caption data\n", o.count)
}
