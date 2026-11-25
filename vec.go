package main

import "math"

// Vec3 is a minimal double-precision 3D vector used for geometry and color.
type Vec3 struct {
	X, Y, Z float64
}

func v(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}
func (a Vec3) Length() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / l)
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

func clampf(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// wrapPi maps an angle to [-pi, pi].
func wrapPi(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// gauss is a non-normalized Gaussian bump: gauss(0)=1.
func gauss(x, sigma float64) float64 {
	return math.Exp(-0.5 * (x / sigma) * (x / sigma))
}

// periodicGauss evaluates a Gaussian on the circle (period 2*pi).
func periodicGauss(delta, center, sigma float64) float64 {
	d := wrapPi(delta - center)
	g := gauss(d, sigma)
	d2 := d + 2*math.Pi
	g += gauss(d2, sigma)
	d3 := d - 2*math.Pi
	g += gauss(d3, sigma)
	return g
}
