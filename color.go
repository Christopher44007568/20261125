package main

import "math"

type RGB struct{ R, G, B float64 }

func C(r, g, b float64) RGB { return RGB{r, g, b} }

func (a RGB) Add(b RGB) RGB             { return RGB{a.R + b.R, a.G + b.G, a.B + b.B} }
func (a RGB) Sub(b RGB) RGB             { return RGB{a.R - b.R, a.G - b.G, a.B - b.B} }
func (a RGB) Mul(s float64) RGB         { return RGB{a.R * s, a.G * s, a.B * s} }
func (a RGB) MulV(b RGB) RGB            { return RGB{a.R * b.R, a.G * b.G, a.B * b.B} }
func (a RGB) Lerp(b RGB, t float64) RGB { return a.Add(b.Sub(a).Mul(t)) }

func (a RGB) Luma() float64 { return 0.2126*a.R + 0.7152*a.G + 0.0722*a.B }

type HDR struct {
	R, G, B, A float64
}

func (p HDR) Premul() RGB { return C(p.R, p.G, p.B) }

func premul(c RGB, a float64) HDR {
	return HDR{R: c.R * a, G: c.G * a, B: c.B * a, A: a}
}

func over(dst HDR, src HDR) HDR {
	a := src.A + dst.A*(1-src.A)
	if a < 1e-12 {
		return HDR{}
	}
	return HDR{
		R: src.R + dst.R*(1-src.A),
		G: src.G + dst.G*(1-src.A),
		B: src.B + dst.B*(1-src.A),
		A: a,
	}
}

func toneMap(x float64) float64 {
	x = math.Max(0, x)
	return 1 - math.Exp(-x*1.08)
}

func toSRGB(x float64) float64 {
	x = sat(x)
	if x <= 0.0031308 {
		return 12.92 * x
	}
	return 1.055*math.Pow(x, 1/2.4) - 0.055
}
