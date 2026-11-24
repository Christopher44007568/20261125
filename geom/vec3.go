package geom

import "math"

// Vec3 is a point or vector in three-dimensional space.
type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

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
	return a.Mul(1 / s)
}

func (a Vec3) Dot(b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Len2() float64 { return a.Dot(a) }

func (a Vec3) Len() float64 { return math.Sqrt(a.Len2()) }

func (a Vec3) At(i int) float64 {
	switch i {
	case 1:
		return a.Y
	case 2:
		return a.Z
	default:
		return a.X
	}
}

func (a Vec3) Dist(b Vec3) float64 { return a.Sub(b).Len() }

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Mul(t))
}

func (a Vec3) Unit() Vec3 {
	n := a.Len()
	if n <= 1e-15 {
		return Vec3{}
	}
	return a.Div(n)
}

func (a Vec3) IsFinite() bool {
	return !math.IsNaN(a.X) && !math.IsInf(a.X, 0) &&
		!math.IsNaN(a.Y) && !math.IsInf(a.Y, 0) &&
		!math.IsNaN(a.Z) && !math.IsInf(a.Z, 0)
}

func Clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func Bounds(points []Vec3) (Vec3, Vec3) {
	if len(points) == 0 {
		return Vec3{}, Vec3{}
	}
	lo, hi := points[0], points[0]
	for _, p := range points[1:] {
		lo.X = math.Min(lo.X, p.X)
		lo.Y = math.Min(lo.Y, p.Y)
		lo.Z = math.Min(lo.Z, p.Z)
		hi.X = math.Max(hi.X, p.X)
		hi.Y = math.Max(hi.Y, p.Y)
		hi.Z = math.Max(hi.Z, p.Z)
	}
	return lo, hi
}
