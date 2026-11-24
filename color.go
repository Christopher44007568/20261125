package main

import "math"

type Transfer int

const (
	Linear Transfer = iota
	SRGB
)

type Sample struct {
	Space   Transfer
	Premul  bool
	R, G, B float64
	A       float64
}

type PremulLinear struct {
	R, G, B, A float64
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

func clampAlpha(v float64) float64 {
	return clamp01(v)
}

func sRGBToLinear(c float64) float64 {
	c = clamp01(c)
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearTosRGB(c float64) float64 {
	c = clamp01(c)
	if c <= 0.0031308 {
		return c * 12.92
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

func decodePremultiplied(s Sample) PremulLinear {
	alpha := clampAlpha(s.A)
	if alpha == 0 {
		return PremulLinear{A: 0}
	}

	if s.Premul {
		if s.Space == SRGB {
			rawR := clamp01(s.R / alpha)
			rawG := clamp01(s.G / alpha)
			rawB := clamp01(s.B / alpha)
			return PremulLinear{
				R: sRGBToLinear(rawR) * alpha,
				G: sRGBToLinear(rawG) * alpha,
				B: sRGBToLinear(rawB) * alpha,
				A: alpha,
			}
		}
		return PremulLinear{R: s.R, G: s.G, B: s.B, A: alpha}
	}

	if s.Space == SRGB {
		return PremulLinear{
			R: sRGBToLinear(s.R) * alpha,
			G: sRGBToLinear(s.G) * alpha,
			B: sRGBToLinear(s.B) * alpha,
			A: alpha,
		}
	}
	return PremulLinear{R: s.R * alpha, G: s.G * alpha, B: s.B * alpha, A: alpha}
}

func (p PremulLinear) Straight() (r, g, b, a float64) {
	if p.A == 0 {
		return 0, 0, 0, 0
	}
	return p.R / p.A, p.G / p.A, p.B / p.A, p.A
}

func toneMap(r, g, b float64) (float64, float64, float64) {
	luminance := 0.2126*r + 0.7152*g + 0.0722*b
	if luminance <= 0 {
		return 0, 0, 0
	}
	mapped := luminance / (1 + luminance)
	scale := mapped / luminance
	return clamp01(r * scale), clamp01(g * scale), clamp01(b * scale)
}
