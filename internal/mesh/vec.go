package mesh

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }
func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }
func (a Vec3) Mul(s float64) Vec3 {
	return V(a.X*s, a.Y*s, a.Z*s)
}
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Cross(b Vec3) Vec3 {
	return V(a.Y*b.Z-a.Z*b.Y, a.Z*b.X-a.X*b.Z, a.X*b.Y-a.Y*b.X)
}
func (a Vec3) Len() float64 { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l == 0 {
		return Vec3{}
	}
	return a.Mul(1 / l)
}
func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Mul(t))
}
func (a Vec3) Dist(b Vec3) float64 { return a.Sub(b).Len() }

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func smoothstep(a, b, x float64) float64 {
	t := clamp((x-a)/(b-a), 0, 1)
	return t * t * (3 - 2*t)
}
