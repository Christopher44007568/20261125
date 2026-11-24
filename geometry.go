package main

import "math"

type Vec2 struct {
	X, Y float64
}

func (v Vec2) Add(w Vec2) Vec2      { return Vec2{v.X + w.X, v.Y + w.Y} }
func (v Vec2) Sub(w Vec2) Vec2      { return Vec2{v.X - w.X, v.Y - w.Y} }
func (v Vec2) Scale(s float64) Vec2 { return Vec2{v.X * s, v.Y * s} }
func (v Vec2) Dot(w Vec2) float64   { return v.X*w.X + v.Y*w.Y }
func (v Vec2) Cross(w Vec2) float64 { return v.X*w.Y - v.Y*w.X }
func (v Vec2) Len() float64         { return math.Hypot(v.X, v.Y) }

type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Add(w Vec3) Vec3      { return Vec3{v.X + w.X, v.Y + w.Y, v.Z + w.Z} }
func (v Vec3) Sub(w Vec3) Vec3      { return Vec3{v.X - w.X, v.Y - w.Y, v.Z - w.Z} }
func (v Vec3) Scale(s float64) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) Dot(w Vec3) float64   { return v.X*w.X + v.Y*w.Y + v.Z*w.Z }
func (v Vec3) Len() float64         { return math.Sqrt(v.Dot(v)) }

func (v Vec3) Cross(w Vec3) Vec3 {
	return Vec3{
		v.Y*w.Z - v.Z*w.Y,
		v.Z*w.X - v.X*w.Z,
		v.X*w.Y - v.Y*w.X,
	}
}

func (v Vec3) Unit() Vec3 {
	n := v.Len()
	if n < 1e-15 {
		return Vec3{}
	}
	return v.Scale(1 / n)
}

func (v Vec2) XY() Vec3 { return Vec3{v.X, v.Y, 0} }

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func signedArea2(poly []Vec2) float64 {
	if len(poly) < 3 {
		return 0
	}
	var sum float64
	for i, p := range poly {
		q := poly[(i+1)%len(poly)]
		sum += p.Cross(q)
	}
	return 0.5 * sum
}

func cross2(a, b, c Vec2) float64 {
	return b.Sub(a).Cross(c.Sub(a))
}
