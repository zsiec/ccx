package main

import (
	"fmt"

	"github.com/zsiec/ccx"
)

func main() {
	nalData := buildSampleSEI()

	cd := ccx.ExtractCaptions(nalData)
	if cd == nil {
		fmt.Println("No caption data found")
		return
	}

	fmt.Printf("Found %d CEA-608 pairs, %d DTVCC pairs\n",
		len(cd.CC608Pairs), len(cd.DTVCC))

	dec608 := ccx.NewCEA608Decoder()
	dec708 := ccx.NewCEA708Decoder()

	for _, pair := range cd.CC608Pairs {
		text := dec608.Decode(pair.Data[0], pair.Data[1])
		if text != "" {
			fmt.Printf("[608 CC%d] %s\n", pair.Channel, text)
		}
	}

	for _, pair := range cd.DTVCC {
		text := dec708.AddTriplet(pair)
		if text != "" {
			fmt.Printf("[708] %s\n", text)
		}
	}
}

func buildSampleSEI() []byte {
	t35 := []byte{
		0xB5, 0x00, 0x31,
		'G', 'A', '9', '4',
		0x03,
		0x40 | 4,
		0xFF,
		0xFC, 0x14, 0x25, // RU2 command (CC1)
		0xFC, 0x14, 0x25, // RU2 dedup
		0xFC, 'H', 'i',   // printable "Hi"
		0xFC, '!', 0x00,  // printable "!"
	}

	nalData := []byte{0x06, 0x04, byte(len(t35))}
	nalData = append(nalData, t35...)
	return nalData
}
