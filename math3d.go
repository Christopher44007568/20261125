package xrpath

import (
	"math"
)

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }

func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }

func (a Vec3) Scale(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) Length() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalize() Vec3 {
	n := a.Length()
	if n < 1e-12 {
		return V(0, 0, 0)
	}
	return a.Scale(1 / n)
}

func (a Vec3) Approx(b Vec3, eps float64) bool {
	return math.Abs(a.X-b.X) <= eps &&
		math.Abs(a.Y-b.Y) <= eps &&
		math.Abs(a.Z-b.Z) <= eps
}

type Quat struct {
	W float64 `json:"w"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func QuatFromYawPitch(yaw, pitch float64) Quat {
	hy, hp := yaw*0.5, pitch*0.5
	sy, cy := math.Sin(hy), math.Cos(hy)
	sp, cp := math.Sin(hp), math.Cos(hp)
	return Quat{
		W: cy*cp + sy*sp*0,
		X: cy * sp,
		Y: sy * cp,
		Z: -sy * sp,
	}
}

func (q Quat) Rotate(v Vec3) Vec3 {
	qx, qy, qz, qw := q.X, q.Y, q.Z, q.W
	ux, uy, uz := qx, qy, qz
	s := qw
	uvx := uy*v.Z - uz*v.Y
	uvy := uz*v.X - ux*v.Z
	uvz := ux*v.Y - uy*v.X
	uuvx := uy*uvz - uz*uvy
	uuvy := uz*uvx - ux*uvz
	uuvz := ux*uvy - uy*uvx
	return V(
		v.X+2*(s*uvx+uuvx),
		v.Y+2*(s*uvy+uuvy),
		v.Z+2*(s*uvz+uuvz),
	)
}

func angleDelta(a, b float64) float64 {
	d := math.Mod(b-a+math.Pi, 2*math.Pi)
	if d < 0 {
		d += 2 * math.Pi
	}
	return d - math.Pi
}

func smoothStep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func lerpVec(a, b Vec3, t float64) Vec3 {
	return V(lerp(a.X, b.X, t), lerp(a.Y, b.Y, t), lerp(a.Z, b.Z, t))
}

func clampFloat(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
