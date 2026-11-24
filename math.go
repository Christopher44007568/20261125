package decallab

import (
	"math"
)

// Vec3 is a world-space vector in meters unless a field name says otherwise.
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (a Vec3) Add(b Vec3) Vec3 {
	return Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

func (a Vec3) Sub(b Vec3) Vec3 {
	return Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{X: a.X * s, Y: a.Y * s, Z: a.Z * s}
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

func (a Vec3) NormSquared() float64 {
	return a.Dot(a)
}

func (a Vec3) Norm() float64 {
	return math.Sqrt(a.NormSquared())
}

func (a Vec3) Normalize() Vec3 {
	length := a.Norm()
	if length < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / length)
}

func (a Vec3) Distance(b Vec3) float64 {
	return a.Sub(b).Norm()
}

// Frame is a right-handed surface frame: Tangent cross Bitangent = Normal.
type Frame struct {
	Origin    Vec3 `json:"origin"`
	Normal    Vec3 `json:"normal"`
	Tangent   Vec3 `json:"tangent"`
	Bitangent Vec3 `json:"bitangent"`
}

func MakeFrame(origin, normal, hint Vec3) Frame {
	n := normal.Normalize()
	t := hint.Sub(n.Scale(hint.Dot(n)))
	if t.Norm() < 1e-9 {
		if math.Abs(n.Y) < 0.9 {
			t = Vec3{Y: 1}.Sub(n.Scale(n.Y))
		} else {
			t = Vec3{X: 1}.Sub(n.Scale(n.X))
		}
	}
	t = t.Normalize()
	b := n.Cross(t).Normalize()
	return Frame{Origin: origin, Normal: n, Tangent: t, Bitangent: b}
}

func RotateFrameAxes(frame Frame, angleDeg float64) (u, v Vec3) {
	rad := angleDeg * math.Pi / 180
	c := math.Cos(rad)
	s := math.Sin(rad)
	u = frame.Tangent.Scale(c).Add(frame.Bitangent.Scale(s)).Normalize()
	v = frame.Tangent.Scale(-s).Add(frame.Bitangent.Scale(c)).Normalize()
	return u, v
}

type Ray struct {
	Origin    Vec3 `json:"origin"`
	Direction Vec3 `json:"direction"`
}

func (r Ray) At(t float64) Vec3 {
	return r.Origin.Add(r.Direction.Scale(t))
}

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func smoothStep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func deg_from_rad(rad float64) float64 {
	return rad * 180 / math.Pi
}

func angleBetweenDeg(a, b Vec3) float64 {
	d := clamp01(math.Abs(a.Normalize().Dot(b.Normalize())))
	return deg_from_rad(math.Acos(d))
}

func signedAngleBetweenDeg(a, b, axis Vec3) float64 {
	an := a.Normalize()
	bn := b.Normalize()
	angle := math.Atan2(axis.Dot(an.Cross(bn)), an.Dot(bn))
	return deg_from_rad(angle)
}

func approxEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func percentError(measured, expected float64) float64 {
	if math.Abs(expected) < 1e-12 {
		if math.Abs(measured) < 1e-12 {
			return 0
		}
		return math.Inf(1)
	}
	return (measured - expected) / expected * 100
}
