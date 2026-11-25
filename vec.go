package main

import (
	"math"
)

type Vec3 struct {
	X, Y, Z float64
}

func Vec(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

func (a Vec3) MulVec(b Vec3) Vec3 {
	return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z}
}

func (a Vec3) Div(s float64) Vec3 { return a.Mul(1 / s) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) LenSqr() float64 { return a.Dot(a) }

func (a Vec3) Len() float64 { return math.Sqrt(a.LenSqr()) }

func (a Vec3) Normalized() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Div(l)
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func mixVec(a, b Vec3, t float64) Vec3 {
	return Vec3{lerp(a.X, b.X, t), lerp(a.Y, b.Y, t), lerp(a.Z, b.Z, t)}
}

func luminance(c Vec3) float64 {
	return 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z
}

func srgbToLinear(c Vec3) Vec3 {
	return Vec3{srgbChannelToLinear(c.X), srgbChannelToLinear(c.Y), srgbChannelToLinear(c.Z)}
}

func srgbChannelToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToSRGB(c Vec3) Vec3 {
	return Vec3{linearChannelToSRGB(c.X), linearChannelToSRGB(c.Y), linearChannelToSRGB(c.Z)}
}

func linearChannelToSRGB(v float64) float64 {
	v = clamp01(v)
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}
