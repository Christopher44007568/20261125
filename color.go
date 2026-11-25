package main

import "math"

type RGB [3]float64

func (c RGB) mul(k float64) RGB {
	return RGB{c[0] * k, c[1] * k, c[2] * k}
}

func (c RGB) add(d RGB) RGB {
	return RGB{c[0] + d[0], c[1] + d[1], c[2] + d[2]}
}

func (c RGB) maxChannel() float64 {
	return math.Max(c[0], math.Max(c[1], c[2]))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clamp255(v float64) uint8 {
	v = math.Round(clamp01(v) * 255)
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func sRGBEncodeLinear(v float64) float64 {
	v = clamp01(v)
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

func sRGBEncode(c RGB) RGB {
	return RGB{sRGBEncodeLinear(c[0]), sRGBEncodeLinear(c[1]), sRGBEncodeLinear(c[2])}
}

const (
	pqM1 = 2610.0 / 16384.0
	pqM2 = 2523.0 * 128.0 / 4096.0
	pqC1 = 3424.0 / 4096.0
	pqC2 = 2413.0 / 128.0
	pqC3 = 2392.0 / 128.0
)

// pqEncodeNits implements SMPTE ST 2084 OETF for absolute scene/display nits.
func pqEncodeNits(nits float64) float64 {
	y := clamp01(nits / 10000.0)
	yp := math.Pow(y, pqM1)
	return math.Pow((pqC1+pqC2*yp)/(1.0+pqC3*yp), pqM2)
}

// pqDecodeNits is the inverse ST 2084 EOTF helper used by round-trip checks.
func pqDecodeNits(code float64) float64 {
	np := math.Pow(clamp01(code), 1.0/pqM2)
	y := math.Pow(math.Max(0, np-pqC1)/(pqC2-pqC3*np), 1.0/pqM1)
	return 10000.0 * y
}

const (
	hlgA = 0.17883277
	hlgB = 1.0 - 4.0*hlgA
)

var hlgC = 0.5 - hlgA*math.Log(4.0*hlgA)

// hlgEncodeScene implements ARIB STD-B67 / BT.2100 HLG OETF.
func hlgEncodeScene(e float64) float64 {
	if e <= 1.0/12.0 {
		return math.Sqrt(3.0 * math.Max(0, e))
	}
	return hlgA*math.Log(12.0*math.Max(e, 0)-hlgB) + hlgC
}

func hlgDecodeScene(code float64) float64 {
	code = clamp01(code)
	if code <= 0.5 {
		return code * code / 3.0
	}
	return (math.Exp((code-hlgC)/hlgA) + hlgB) / 12.0
}

func luma709(c RGB) float64 {
	return 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func mix(a, b float64, t float64) float64 {
	return a + (b-a)*clamp01(t)
}

func mixRGB(a, b RGB, t float64) RGB {
	t = clamp01(t)
	return RGB{mix(a[0], b[0], t), mix(a[1], b[1], t), mix(a[2], b[2], t)}
}
