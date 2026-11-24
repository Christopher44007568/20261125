package glass

import (
	"math/big"
	"strings"
)

func encodeMask(bitOrder []string, included map[string]bool) string {
	bits := new(big.Int)
	for index, id := range bitOrder {
		if included[id] {
			bits.SetBit(bits, index, 1)
		}
	}

	width := (len(bitOrder) + 3) / 4
	if width == 0 {
		width = 1
	}
	raw := bits.Text(16)
	if len(raw) < width {
		raw = strings.Repeat("0", width-len(raw)) + raw
	}
	return "0x" + raw
}

func selectedSet(order []Layer, count int) map[string]bool {
	selected := make(map[string]bool, len(order))
	if count > len(order) {
		count = len(order)
	}
	for index := 0; index < count; index++ {
		selected[order[index].ID] = true
	}
	return selected
}

func complementSet(bitOrder []string, selected map[string]bool) map[string]bool {
	complement := make(map[string]bool, len(bitOrder))
	for _, id := range bitOrder {
		if !selected[id] {
			complement[id] = true
		}
	}
	return complement
}
