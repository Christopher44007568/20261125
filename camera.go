package main

import "math"

type Camera struct {
	Width  int
	Height int
	Eye    Vec3
	Target Vec3

	Forward Vec3
	Right   Vec3
	Up      Vec3
	Focal   float64
}

func NewCamera(width, height int) Camera {
	eye := Vec(0, 0.82, 5.05)
	target := Vec(0, 0.56, 0.34)
	forward := target.Sub(eye).Normalized()
	right := forward.Cross(Vec(0, 1, 0)).Normalized()
	up := right.Cross(forward).Normalized()
	fov := 36.0 * math.Pi / 180
	focal := 0.5 * float64(height) / math.Tan(fov*0.5)

	return Camera{
		Width:   width,
		Height:  height,
		Eye:     eye,
		Target:  target,
		Forward: forward,
		Right:   right,
		Up:      up,
		Focal:   focal,
	}
}

func (c Camera) Ray(x, y int) Vec3 {
	u := (float64(x) + 0.5 - float64(c.Width)*0.5) / c.Focal
	v := (float64(c.Height)*0.5 - (float64(y) + 0.5)) / c.Focal
	return c.Forward.Add(c.Right.Mul(u)).Add(c.Up.Mul(v)).Normalized()
}

func (c Camera) Project(point Vec3) (float64, float64, float64, bool) {
	rel := point.Sub(c.Eye)
	z := rel.Dot(c.Forward)
	if z <= 0.05 {
		return 0, 0, z, false
	}
	x := rel.Dot(c.Right)
	y := rel.Dot(c.Up)
	sx := float64(c.Width)*0.5 + c.Focal*x/z
	sy := float64(c.Height)*0.5 - c.Focal*y/z
	return sx, sy, z, true
}

func perspectiveMix(a0, a1, z0, z1, t float64) float64 {
	w0 := 1 / z0
	w1 := 1 / z1
	w := lerp(w0, w1, t)
	if math.Abs(w) < 1e-12 {
		return lerp(a0, a1, t)
	}
	return lerp(a0*w0, a1*w1, t) / w
}
