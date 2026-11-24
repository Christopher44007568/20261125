package main

import "math"

// Vec3 is a point/direction in world or camera space.
type Vec3 struct{ X, Y, Z float64 }

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64         { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Norm() Vec3           { return a.Scale(1 / a.Len()) }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

// AngleDeg returns the angle between two directions in degrees.
// It uses atan2(|a x b|, a . b) because acos loses all precision
// (sqrt(eps) ~ 1e-8 rad) for nearly parallel vectors.
func AngleDeg(a, b Vec3) float64 {
	an, bn := a.Norm(), b.Norm()
	return math.Atan2(an.Cross(bn).Len(), an.Dot(bn)) * 180 / math.Pi
}

// Vec4 is a homogeneous coordinate.
type Vec4 struct{ X, Y, Z, W float64 }

// Mat4 is a column-major 4x4 matrix (OpenGL layout).
type Mat4 struct{ M [16]float64 }

func Mul4(a, b Mat4) Mat4 {
	var r Mat4
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			s := 0.0
			for k := 0; k < 4; k++ {
				s += a.M[k*4+row] * b.M[col*4+k]
			}
			r.M[col*4+row] = s
		}
	}
	return r
}

func (m Mat4) MulV(v Vec4) Vec4 {
	return Vec4{
		X: m.M[0]*v.X + m.M[4]*v.Y + m.M[8]*v.Z + m.M[12]*v.W,
		Y: m.M[1]*v.X + m.M[5]*v.Y + m.M[9]*v.Z + m.M[13]*v.W,
		Z: m.M[2]*v.X + m.M[6]*v.Y + m.M[10]*v.Z + m.M[14]*v.W,
		W: m.M[3]*v.X + m.M[7]*v.Y + m.M[11]*v.Z + m.M[15]*v.W,
	}
}

// Translation builds a matrix translating by (x, y, z).
func Translation(x, y, z float64) Mat4 {
	return Mat4{M: [16]float64{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		x, y, z, 1,
	}}
}
