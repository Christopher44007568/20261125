package main

import "math"

type Surface struct {
	controls [4][4]Vec3
	zScale   float64
}

func newSurface() *Surface {
	controls := [4][4]Vec3{
		{{-6, -4, -0.35}, {-2, -4, 0.85}, {2, -4, -0.55}, {6, -4, 0.35}},
		{{-6, -1.35, 0.55}, {-2, -1.35, -0.75}, {2, -1.35, 0.95}, {6, -1.35, -0.25}},
		{{-6, 1.35, -0.85}, {-2, 1.35, 1.05}, {2, 1.35, -1.1}, {6, 1.35, 0.75}},
		{{-6, 4, 0.45}, {-2, 4, -0.6}, {2, 4, 0.75}, {6, 4, -0.45}},
	}
	return &Surface{controls: controls, zScale: 0.39}
}

func bezier1D(t float64) [4]float64 {
	mt := 1 - t
	return [4]float64{
		mt * mt * mt,
		3 * mt * mt * t,
		3 * mt * t * t,
		t * t * t,
	}
}

func (s *Surface) Point(u, v float64) Vec3 {
	bu := bezier1D(u)
	bv := bezier1D(v)
	var p Vec3
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			w := bu[i] * bv[j]
			p = add(p, scale(s.controls[i][j], w))
		}
	}
	p.Z *= s.zScale
	return p
}

func (s *Surface) Normal(u, v float64) Vec3 {
	const eps = 1e-5
	u0 := math.Max(0, u-eps)
	u1 := math.Min(1, u+eps)
	v0 := math.Max(0, v-eps)
	v1 := math.Min(1, v+eps)
	du := scale(sub(s.Point(u1, v), s.Point(u0, v)), 1/(u1-u0))
	dv := scale(sub(s.Point(u, v1), s.Point(u, v0)), 1/(v1-v0))
	return normalize(cross(du, dv))
}

func (s *Surface) Bounds() (Vec3, Vec3) {
	minPoint := s.Point(0, 0)
	maxPoint := minPoint
	for i := 0; i <= 64; i++ {
		for j := 0; j <= 64; j++ {
			p := s.Point(float64(i)/64, float64(j)/64)
			minPoint.X = math.Min(minPoint.X, p.X)
			minPoint.Y = math.Min(minPoint.Y, p.Y)
			minPoint.Z = math.Min(minPoint.Z, p.Z)
			maxPoint.X = math.Max(maxPoint.X, p.X)
			maxPoint.Y = math.Max(maxPoint.Y, p.Y)
			maxPoint.Z = math.Max(maxPoint.Z, p.Z)
		}
	}
	return minPoint, maxPoint
}
