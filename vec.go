package main

import "math"

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func V(x, y, z float64) Vec3 {
	return Vec3{X: x, Y: y, Z: z}
}

func (a Vec3) Add(b Vec3) Vec3 {
	return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z}
}

func (a Vec3) Sub(b Vec3) Vec3 {
	return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z}
}

func (a Vec3) Mul(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

func (a Vec3) Div(s float64) Vec3 {
	return Vec3{a.X / s, a.Y / s, a.Z / s}
}

func (a Vec3) Dot(b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Len2() float64 {
	return a.Dot(a)
}

func (a Vec3) Len() float64 {
	return math.Sqrt(a.Len2())
}

func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Div(l)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Mul(1 - t).Add(b.Mul(t))
}

func (a Vec3) Dist(b Vec3) float64 {
	return a.Sub(b).Len()
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

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

type Pose struct {
	Yaw       float64
	Pitch     float64
	Roll      float64
	Translate Vec3
}

func (p Pose) Apply(v Vec3) Vec3 {
	return p.Rotate(v).Add(p.Translate)
}

func (p Pose) Rotate(v Vec3) Vec3 {
	cy, sy := math.Cos(p.Yaw), math.Sin(p.Yaw)
	x, z := cy*v.X+sy*v.Z, -sy*v.X+cy*v.Z
	cp, sp := math.Cos(p.Pitch), math.Sin(p.Pitch)
	y, z := cp*v.Y-sp*z, sp*v.Y+cp*z
	cr, sr := math.Cos(p.Roll), math.Sin(p.Roll)
	x, y = cr*x-sr*y, sr*x+cr*y
	return V(x, y, z)
}

func shortestAngle(a, b float64) float64 {
	d := math.Mod(b-a+3*math.Pi, 2*math.Pi)
	if d < 0 {
		d += 2 * math.Pi
	}
	return d - math.Pi
}

func (p Pose) Lerp(q Pose, t float64) Pose {
	return Pose{
		Yaw:       p.Yaw + shortestAngle(p.Yaw, q.Yaw)*t,
		Pitch:     p.Pitch + shortestAngle(p.Pitch, q.Pitch)*t,
		Roll:      p.Roll + shortestAngle(p.Roll, q.Roll)*t,
		Translate: p.Translate.Lerp(q.Translate, t),
	}
}
