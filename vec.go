package main

import "math"

type Vec3 struct{ X, Y, Z float64 }

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3    { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }
func (a Vec3) Sub(b Vec3) Vec3    { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }
func (a Vec3) Mul(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }
func (a Vec3) MulV(b Vec3) Vec3   { return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z) }
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(a.Y*b.Z-a.Z*b.Y, a.Z*b.X-a.X*b.Z, a.X*b.Y-a.Y*b.X)
}

func (a Vec3) Len() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Mul(1 / l)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Mul(t))
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

func mix(a, b, t float64) float64 { return a + (b-a)*t }

func smoothstep(a, b, x float64) float64 {
	t := clamp((x-a)/(b-a), 0, 1)
	return t * t * (3 - 2*t)
}

func sat(x float64) float64 { return clamp(x, 0, 1) }

func reflect(d, n Vec3) Vec3 { return d.Sub(n.Mul(2 * d.Dot(n))) }
