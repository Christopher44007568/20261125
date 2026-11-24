package main

import "math"

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }

func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }

func (a Vec3) Scale(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }

func (a Vec3) Mul(b Vec3) Vec3 { return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) LenSqr() float64 { return a.Dot(a) }

func (a Vec3) Len() float64 { return math.Sqrt(a.LenSqr()) }

func (a Vec3) Unit() Vec3 {
	l := a.Len()
	if l < 1e-15 {
		return V(0, 0, 0)
	}
	return a.Scale(1 / l)
}

func (a Vec3) Distance(b Vec3) float64 { return a.Sub(b).Len() }

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return V(
		a.X+(b.X-a.X)*t,
		a.Y+(b.Y-a.Y)*t,
		a.Z+(b.Z-a.Z)*t,
	)
}

func clampFloat(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func degFromRad(r float64) float64 { return r * 180 / math.Pi }
