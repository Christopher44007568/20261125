package main

import "math"

type Vec struct {
	X, Y float64
}

func (v Vec) Add(o Vec) Vec       { return Vec{v.X + o.X, v.Y + o.Y} }
func (v Vec) Sub(o Vec) Vec       { return Vec{v.X - o.X, v.Y - o.Y} }
func (v Vec) Scale(s float64) Vec { return Vec{s * v.X, s * v.Y} }
func (v Vec) Dot(o Vec) float64   { return v.X*o.X + v.Y*o.Y }
func (v Vec) Len() float64        { return math.Hypot(v.X, v.Y) }

func (v Vec) Norm() Vec {
	l := v.Len()
	if l < 1e-12 {
		return Vec{0, 0}
	}
	return Vec{v.X / l, v.Y / l}
}

func (v Vec) Rot(a float64) Vec {
	c, s := math.Cos(a), math.Sin(a)
	return Vec{c*v.X - s*v.Y, s*v.X + c*v.Y}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func quadPoint(p0, p1, p2 Vec, t float64) Vec {
	u := 1 - t
	return p0.Scale(u * u).
		Add(p1.Scale(2 * u * t)).
		Add(p2.Scale(t * t))
}

func quadTangent(p0, p1, p2 Vec, t float64) Vec {
	return p1.Sub(p0).Scale(2 * (1 - t)).
		Add(p2.Sub(p1).Scale(2 * t))
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

type Mat3 struct {
	A [9]float64
}

func (m Mat3) At(r, c int) float64 { return m.A[r*3+c] }

func matMul(a, b Mat3) Mat3 {
	var r Mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			s := 0.0
			for k := 0; k < 3; k++ {
				s += a.At(i, k) * b.At(k, j)
			}
			r.A[i*3+j] = s
		}
	}
	return r
}

func matInv(m Mat3) (Mat3, bool) {
	var a [3][6]float64
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			a[r][c] = m.At(r, c)
			a[r][c+3] = 0
		}
		a[r][r+3] = 1
	}
	for col := 0; col < 3; col++ {
		pivot := col
		big := math.Abs(a[col][col])
		for r := col + 1; r < 3; r++ {
			if v := math.Abs(a[r][col]); v > big {
				big, pivot = v, r
			}
		}
		if big < 1e-12 {
			return Mat3{}, false
		}
		if pivot != col {
			a[col], a[pivot] = a[pivot], a[col]
		}
		d := a[col][col]
		for k := 0; k < 6; k++ {
			a[col][k] /= d
		}
		for r := 0; r < 3; r++ {
			if r == col {
				continue
			}
			f := a[r][col]
			for k := 0; k < 6; k++ {
				a[r][k] -= f * a[col][k]
			}
		}
	}
	var inv Mat3
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			inv.A[r*3+c] = a[r][c+3]
		}
	}
	return inv, true
}

func matApply(m Mat3, x, y float64) (float64, float64) {
	w := m.At(2, 0)*x + m.At(2, 1)*y + m.At(2, 2)
	return (m.At(0, 0)*x + m.At(0, 1)*y + m.At(0, 2)) / w,
		(m.At(1, 0)*x + m.At(1, 1)*y + m.At(1, 2)) / w
}

func findHomography(src, dst [4]Vec) Mat3 {
	var eq [8][9]float64
	for i := 0; i < 4; i++ {
		x, y := src[i].X, src[i].Y
		u, v := dst[i].X, dst[i].Y
		eq[2*i] = [9]float64{x, y, 1, 0, 0, 0, -u * x, -u * y, u}
		eq[2*i+1] = [9]float64{0, 0, 0, x, y, 1, -v * x, -v * y, v}
	}
	for col := 0; col < 8; col++ {
		piv := col
		big := math.Abs(eq[col][col])
		for r := col + 1; r < 8; r++ {
			if v := math.Abs(eq[r][col]); v > big {
				big, piv = v, r
			}
		}
		eq[col], eq[piv] = eq[piv], eq[col]
		d := eq[col][col]
		for k := 0; k < 9; k++ {
			eq[col][k] /= d
		}
		for r := 0; r < 8; r++ {
			if r == col {
				continue
			}
			f := eq[r][col]
			for k := 0; k < 9; k++ {
				eq[r][k] -= f * eq[col][k]
			}
		}
	}
	return Mat3{A: [9]float64{
		eq[0][8], eq[1][8], eq[2][8],
		eq[3][8], eq[4][8], eq[5][8],
		eq[6][8], eq[7][8], 1,
	}}
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
