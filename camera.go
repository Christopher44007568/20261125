package main

import "math"

type Camera struct {
	Pos, Target, Up   Vec3
	Right, Fwd, CamUp Vec3
	TanHalf, Aspect   float64
}

func NewCamera(pos, target Vec3, fovDeg, aspect float64) Camera {
	fwd := target.Sub(pos).Norm()
	right := fwd.Cross(V(0, 1, 0)).Norm()
	up := right.Cross(fwd).Norm()
	return Camera{
		Pos: pos, Target: target, Up: V(0, 1, 0),
		Right: right, Fwd: fwd, CamUp: up,
		TanHalf: math.Tan(fovDeg * math.Pi / 360),
		Aspect:  aspect,
	}
}

func (c Camera) Ray(nx, ny float64) (Vec3, Vec3) {
	d := c.Fwd.Add(c.Right.Mul(nx * c.TanHalf * c.Aspect)).
		Add(c.CamUp.Mul(ny * c.TanHalf)).Norm()
	return c.Pos, d
}

func (c Camera) Project(p Vec3) (float64, float64, float64, bool) {
	rel := p.Sub(c.Pos)
	z := rel.Dot(c.Fwd)
	if z <= 0.02 {
		return 0, 0, z, false
	}
	x := rel.Dot(c.Right) / (z * c.TanHalf * c.Aspect)
	y := rel.Dot(c.CamUp) / (z * c.TanHalf)
	return x, y, z, true
}
