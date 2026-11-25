package render

import (
	"math"

	"gamutmap/gamut"
)

// rec2020ToHSV treats the encoded Rec.2020 triplet as an HSV color. This is
// only a demo generator; it does not alter neutral RGB values.
func rec2020ToHSV(c gamut.RGB) (h, s, v float64) {
	mx := math.Max(c.R, math.Max(c.G, c.B))
	mn := math.Min(c.R, math.Min(c.G, c.B))
	delta := mx - mn
	v = mx
	if mx == 0 {
		return 0, 0, 0
	}
	s = delta / mx
	if delta == 0 {
		return 0, s, v
	}
	switch mx {
	case c.R:
		h = math.Mod((c.G-c.B)/delta, 6)
	case c.G:
		h = (c.B-c.R)/delta + 2
	default:
		h = (c.R-c.G)/delta + 4
	}
	h /= 6
	if h < 0 {
		h++
	}
	return h, s, v
}
