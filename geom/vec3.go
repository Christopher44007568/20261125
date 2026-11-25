package geom

import "math"

// Vec3 is a 3D double-precision vector.
type Vec3 struct {
	X, Y, Z float64
}

func V3(x, y, z float64) Vec3 { return Vec3{x, y, z} }

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

func (a Vec3) Len() float64  { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Len2() float64 { return a.Dot(a) }
func (a Vec3) Neg() Vec3     { return Vec3{-a.X, -a.Y, -a.Z} }
func (a Vec3) IsZero() bool  { return a.Len2() < 1e-24 }

func (a Vec3) Approx(b Vec3, eps float64) bool {
	return a.Sub(b).Len() <= eps
}

// Normalized returns the unit vector. It panics for near-zero vectors.
func (a Vec3) Normalized() Vec3 {
	l := a.Len()
	if l < 1e-15 {
		panic("geom: normalize of near-zero vector")
	}
	return a.Scale(1 / l)
}

// Distance returns the Euclidean distance between two points.
func Distance(a, b Vec3) float64 { return a.Sub(b).Len() }

const deg = math.Pi / 180

// Deg converts degrees to radians.
func Deg(d float64) float64 { return d * deg }

// Clamp constrains v to [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Min3 / Max3 component-wise helpers.
func Min3(a, b Vec3) Vec3 {
	return Vec3{math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z)}
}

func Max3(a, b Vec3) Vec3 {
	return Vec3{math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z)}
}
