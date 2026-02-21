package main

import (
	"fmt"

	"github.com/zsiec/ccx"
)

func main() {
	fmt.Println("=== H.264 Extraction ===")
	extractH264()

	fmt.Println("\n=== H.265/HEVC Extraction ===")
	extractH265()
}

func extractH264() {
	nalData := buildH264SEI()

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

func extractH265() {
	nalData := buildH265SEI()

	cd := ccx.ExtractCaptionsHEVC(nalData)
	if cd == nil {
		fmt.Println("No caption data found")
		return
	}

	fmt.Printf("Found %d CEA-608 pairs, %d DTVCC pairs\n",
		len(cd.CC608Pairs), len(cd.DTVCC))

	dec608 := ccx.NewCEA608Decoder()
	for _, pair := range cd.CC608Pairs {
		text := dec608.Decode(pair.Data[0], pair.Data[1])
		if text != "" {
			fmt.Printf("[608 CC%d] %s\n", pair.Channel, text)
		}
	}
}

func buildA53Payload() []byte {
	return []byte{
		0xB5, 0x00, 0x31,
		'G', 'A', '9', '4',
		0x03,
		0x40 | 4,
		0xFF,
		0xFC, 0x14, 0x25, // RU2 command (CC1)
		0xFC, 0x14, 0x25, // RU2 dedup
		0xFC, 'H', 'i',
		0xFC, '!', 0x00,
	}
}

func buildH264SEI() []byte {
	t35 := buildA53Payload()
	nalData := []byte{0x06, 0x04, byte(len(t35))}
	nalData = append(nalData, t35...)
	return nalData
}

func buildH265SEI() []byte {
	t35 := buildA53Payload()
	nalType := byte(39) // PREFIX_SEI
	nalHeader0 := nalType << 1
	nalHeader1 := byte(0x01)
	nalData := []byte{nalHeader0, nalHeader1, 0x04, byte(len(t35))}
	nalData = append(nalData, t35...)
	return nalData
}
