package hair

import "math"

type Vec2 struct {
	X float64
	Y float64
}

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (a Vec3) Add(b Vec3) Vec3 {
	return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z}
}

func (a Vec3) Sub(b Vec3) Vec3 {
	return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z}
}

func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

func (a Vec3) Dot(b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func (a Vec3) Length() float64 {
	return math.Sqrt(a.Dot(a))
}

func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / l)
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

type Bounds struct {
	Min Vec3 `json:"min"`
	Max Vec3 `json:"max"`
}

func boundsOf(strands []Strand) Bounds {
	b := Bounds{
		Min: Vec3{math.Inf(1), math.Inf(1), math.Inf(1)},
		Max: Vec3{math.Inf(-1), math.Inf(-1), math.Inf(-1)},
	}
	for _, s := range strands {
		for _, p := range s.Points {
			b.Min.X = math.Min(b.Min.X, p.X)
			b.Min.Y = math.Min(b.Min.Y, p.Y)
			b.Min.Z = math.Min(b.Min.Z, p.Z)
			b.Max.X = math.Max(b.Max.X, p.X)
			b.Max.Y = math.Max(b.Max.Y, p.Y)
			b.Max.Z = math.Max(b.Max.Z, p.Z)
		}
	}
	return b
}
