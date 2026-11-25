package scalp

import "math"

// Vec3 is a point or direction in scalp/world space.
type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(k float64) Vec3 { return Vec3{a.X * k, a.Y * k, a.Z * k} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Length() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalize() Vec3 {
	n := a.Length()
	if n < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / n)
}

// WrapAngle maps an angle to [-pi, pi).
func WrapAngle(a float64) float64 {
	for a >= math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

func smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// gaussAngle is a periodic Gaussian bump centered at center with the given
// angular width (standard deviation in radians).
func gaussAngle(phi, center, width float64) float64 {
	d := WrapAngle(phi - center)
	return math.Exp(-(d * d) / (2 * width * width))
}
