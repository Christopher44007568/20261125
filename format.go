package eyestage

import (
	"fmt"
	"strconv"
)

func formatNS(ns int64) string {
	negative := ns < 0
	if negative {
		ns = -ns
	}
	var text string
	switch {
	case ns < 1_000:
		text = strconv.FormatInt(ns, 10) + "ns"
	case ns < 1_000_000:
		text = trimFloat(float64(ns)/1_000, "us")
	case ns < 1_000_000_000:
		text = trimFloat(float64(ns)/1_000_000, "ms")
	default:
		text = trimFloat(float64(ns)/1_000_000_000, "s")
	}
	if negative {
		return "-" + text
	}
	return text
}

func trimFloat(value float64, unit string) string {
	text := fmt.Sprintf("%.3f", value)
	for len(text) > 0 && text[len(text)-1] == '0' {
		text = text[:len(text)-1]
	}
	if len(text) > 0 && text[len(text)-1] == '.' {
		text = text[:len(text)-1]
	}
	return text + unit
}
