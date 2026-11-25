package main

import "math"

type Vec2 struct {
	X float64
	Y float64
}

func (a Vec2) Add(b Vec2) Vec2    { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2    { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Dot(b Vec2) float64 { return a.X*b.X + a.Y*b.Y }
func (a Vec2) Len() float64       { return math.Hypot(a.X, a.Y) }
func (a Vec2) Lerp(b Vec2, t float64) Vec2 {
	return Vec2{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}
}

func rotateAround(p, pivot Vec2, angle float64) Vec2 {
	c, s := math.Cos(angle), math.Sin(angle)
	d := p.Sub(pivot)
	return Vec2{
		X: pivot.X + d.X*c - d.Y*s,
		Y: pivot.Y + d.X*s + d.Y*c,
	}
}

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return Vec3{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t, a.Z + (b.Z-a.Z)*t}
}

type Color struct {
	R float64
	G float64
	B float64
}

func RGB(r, g, b float64) Color { return Color{r, g, b} }

func (c Color) Add(d Color) Color {
	return Color{c.R + d.R, c.G + d.G, c.B + d.B}
}

func (c Color) Mul(s float64) Color {
	return Color{c.R * s, c.G * s, c.B * s}
}

func (c Color) MulColor(d Color) Color {
	return Color{c.R * d.R, c.G * d.G, c.B * d.B}
}

func (c Color) Lerp(d Color, t float64) Color {
	return Color{
		R: c.R + (d.R-c.R)*t,
		G: c.G + (d.G-c.G)*t,
		B: c.B + (d.B-c.B)*t,
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clamp01(v float64) float64 { return clamp(v, 0, 1) }

func smoothstep(a, b, x float64) float64 {
	t := clamp01((x - a) / (b - a))
	return t * t * (3 - 2*t)
}

func cubicBezier(p0, p1, p2, p3 Vec2, t float64) Vec2 {
	mt := 1 - t
	a := mt * mt * mt
	b := 3 * mt * mt * t
	c := 3 * mt * t * t
	d := t * t * t
	return Vec2{
		X: a*p0.X + b*p1.X + c*p2.X + d*p3.X,
		Y: a*p0.Y + b*p1.Y + c*p2.Y + d*p3.Y,
	}
}

func catmullRom(p0, p1, p2, p3 Vec2, t float64) Vec2 {
	t2, t3 := t*t, t*t*t
	return Vec2{
		X: 0.5 * ((2 * p1.X) + (-p0.X+p2.X)*t +
			(2*p0.X-5*p1.X+4*p2.X-p3.X)*t2 +
			(-p0.X+3*p1.X-3*p2.X+p3.X)*t3),
		Y: 0.5 * ((2 * p1.Y) + (-p0.Y+p2.Y)*t +
			(2*p0.Y-5*p1.Y+4*p2.Y-p3.Y)*t2 +
			(-p0.Y+3*p1.Y-3*p2.Y+p3.Y)*t3),
	}
}

func hash01(n uint32) float64 {
	n ^= n >> 16
	n *= 0x7feb352d
	n ^= n >> 15
	n *= 0x846ca68b
	n ^= n >> 16
	return float64(n) / 4294967295.0
}

func chaikinClosed(pts []Vec2, iterations int) []Vec2 {
	out := append([]Vec2(nil), pts...)
	for it := 0; it < iterations; it++ {
		next := make([]Vec2, 0, len(out)*2)
		for i, a := range out {
			b := out[(i+1)%len(out)]
			next = append(next, a.Mul(0.75).Add(b.Mul(0.25)))
			next = append(next, a.Mul(0.25).Add(b.Mul(0.75)))
		}
		out = next
	}
	return out
}
