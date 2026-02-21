package main

import (
	"fmt"

	"github.com/zsiec/ccx"
)

func main() {
	dec608 := ccx.NewCEA608Decoder()

	dec608.Decode(0x14, 0x25) // Roll-up 2
	dec608.Decode(0x14, 0x25) // (control dedup)

	text := dec608.Decode('H', 'e')
	fmt.Printf("608 text: %q\n", text)

	text = dec608.Decode('l', 'l')
	fmt.Printf("608 text: %q\n", text)

	text = dec608.Decode('o', '!')
	fmt.Printf("608 text: %q\n", text)

	dec608.Decode(0x14, 0x2D) // CR
	dec608.Decode(0x14, 0x2D) // (control dedup)

	text = dec608.Decode('W', 'o')
	fmt.Printf("608 text after CR: %q\n", text)

	text = dec608.Decode('r', 'l')
	fmt.Printf("608 text: %q\n", text)

	text = dec608.Decode('d', 0)
	fmt.Printf("608 text: %q\n", text)

	regions := dec608.StyledRegions()
	fmt.Printf("\nStructured output: %d region(s)\n", len(regions))
	for _, reg := range regions {
		for _, row := range reg.Rows {
			fmt.Printf("  Row %d:\n", row.Row)
			for _, span := range row.Spans {
				fmt.Printf("    %q fg=%s bg=%s italic=%v\n",
					span.Text, span.FgColor, span.BgColor, span.Italic)
			}
		}
	}

	fmt.Println("\n--- CEA-708 ---")
	svc := ccx.NewCEA708Service()

	svc.ProcessBlock([]byte{
		0x98,
		0x20, 0x00, 0x00, 0x02, 0x1F, 0x11,
	})

	svc.ProcessBlock([]byte("Live captions"))

	text = svc.DisplayText()
	fmt.Printf("708 text: %q\n", text)

	svc.ProcessBlock([]byte{0x0D}) // CR
	svc.ProcessBlock([]byte("Second line"))

	text = svc.DisplayText()
	fmt.Printf("708 text: %q\n", text)

	fmt.Println("\n--- Codec ---")
	frame := &ccx.CaptionFrame{
		Channel: 1,
		Regions: []ccx.CaptionRegion{{
			Justify: 2,
			Rows: []ccx.CaptionRow{{
				Row: 0,
				Spans: []ccx.CaptionSpan{
					{Text: "Hello ", FgColor: "ffffff", BgColor: "000000", EdgeColor: "000000"},
					{Text: "World", FgColor: "ff0000", BgColor: "000000", Italic: true, EdgeColor: "000000"},
				},
			}},
		}},
	}

	wire := frame.Serialize()
	fmt.Printf("Serialized: %d bytes\n", len(wire))

	decoded := ccx.DeserializeCaptionFrame(wire)
	fmt.Printf("Decoded: %q (channel %d)\n", decoded.PlainText(), decoded.Channel)
	fmt.Printf("Regions: %d, Spans: %d\n",
		len(decoded.Regions), len(decoded.Regions[0].Rows[0].Spans))
}
