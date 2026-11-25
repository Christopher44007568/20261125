package main

import "math"

// Vec3 is a 3D vector in groom/model space (1 unit ~ 1 cm).
type Vec3 struct{ X, Y, Z float64 }

func v3(x, y, z float64) Vec3 { return Vec3{x, y, z} }

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

func (a Vec3) Len() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Scale(1 - t).Add(b.Scale(t))
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

func clampi(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// randSeed is a tiny deterministic LCG so the groom is reproducible.
type randSeed struct{ s uint64 }

func (r *randSeed) float() float64 {
	r.s = r.s*6364136223846793005 + 1442695040888963407
	return float64(r.s>>11) / float64(uint64(1)<<53)
}

func (r *randSeed) rangeF(lo, hi float64) float64 { return lo + (hi-lo)*r.float() }
