package main

import "fmt"

// Intrinsics describes a pinhole camera with a possibly off-center
// principal point: equal focal lengths do NOT imply a symmetric frustum.
type Intrinsics struct {
	Fx, Fy    float64 // focal lengths in pixels
	Cx, Cy    float64 // principal point in pixels (image coords, y down)
	W, H      int     // sensor resolution in pixels
	Near, Far float64 // clip distances in meters
}

// Frustum holds the off-center frustum bounds at the near plane (camera space).
type Frustum struct{ L, R, B, T, N, F float64 }

// EyeFrustum derives the asymmetric frustum for one eye of a parallel-camera
// stereo rig. eyeX is the eye's world X offset (meters); convergence is the
// zero-parallax distance (meters). The frustum is horizontally shifted by
// -eyeX*n/convergence so both eyes share the same virtual screen at the
// convergence distance, without any toe-in rotation.
func EyeFrustum(in Intrinsics, eyeX, convergence float64) Frustum {
	n := in.Near
	shift := -eyeX * n / convergence
	return Frustum{
		L: -in.Cx*n/in.Fx + shift,
		R: (float64(in.W)-in.Cx)*n/in.Fx + shift,
		B: -(float64(in.H) - in.Cy) * n / in.Fy, // image y is down, camera y is up
		T: in.Cy * n / in.Fy,
		N: n,
		F: in.Far,
	}
}

// PerspectiveOffCenter builds the OpenGL-style asymmetric projection matrix.
func PerspectiveOffCenter(f Frustum) Mat4 {
	n, ff := f.N, f.F
	return Mat4{M: [16]float64{
		2 * n / (f.R - f.L), 0, 0, 0,
		0, 2 * n / (f.T - f.B), 0, 0,
		(f.R + f.L) / (f.R - f.L), (f.T + f.B) / (f.T - f.B), -(ff + n) / (ff - n), -1,
		0, 0, -2 * ff * n / (ff - n), 0,
	}}
}

// Invert4 inverts a general 4x4 matrix via Gauss-Jordan elimination
// with partial pivoting. ok is false for singular matrices.
func Invert4(m Mat4) (inv Mat4, ok bool) {
	var a [4][8]float64
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			a[r][c] = m.M[c*4+r]
		}
		a[r][4+r] = 1
	}
	for col := 0; col < 4; col++ {
		piv := col
		for r := col + 1; r < 4; r++ {
			if abs(a[r][col]) > abs(a[piv][col]) {
				piv = r
			}
		}
		if abs(a[piv][col]) < 1e-15 {
			return Mat4{}, false
		}
		a[col], a[piv] = a[piv], a[col]
		d := a[col][col]
		for c := 0; c < 8; c++ {
			a[col][c] /= d
		}
		for r := 0; r < 4; r++ {
			if r == col {
				continue
			}
			f := a[r][col]
			for c := 0; c < 8; c++ {
				a[r][c] -= f * a[col][c]
			}
		}
	}
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			inv.M[c*4+r] = a[r][4+c]
		}
	}
	return inv, true
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (m Mat4) String() string {
	s := ""
	for r := 0; r < 4; r++ {
		s += fmt.Sprintf("    | %+.7e %+.7e %+.7e %+.7e |\n",
			m.M[r], m.M[4+r], m.M[8+r], m.M[12+r])
	}
	return s
}
