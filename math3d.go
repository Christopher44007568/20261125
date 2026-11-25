package main

import "math"

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func vadd(a, b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func vsub(a, b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func vmul(a Vec3, s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

func vlerp(a, b Vec3, t float64) Vec3 {
	return Vec3{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
	}
}

func vdot(a, b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func vcross(a, b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func vlen(a Vec3) float64 { return math.Sqrt(vdot(a, a)) }

func vnorm(a Vec3) Vec3 {
	l := vlen(a)
	if l < 1e-12 {
		return Vec3{}
	}
	return vmul(a, 1/l)
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := clampFloat((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func lerpFloat(a, b, t float64) float64 { return a + (b-a)*t }

type Camera struct {
	Pos     Vec3
	Forward Vec3
	Right   Vec3
	Up      Vec3
	Focal   float64
	Near    float64
}

type CameraPath struct {
	Target   Vec3
	StartZ   float64
	EndZ     float64
	Arc      float64
	Height   float64
	FovY     float64
	Width    int
	HeightPx int
}

func DefaultCameraPath(width, height int) CameraPath {
	return CameraPath{
		Target:   Vec3{X: 0, Y: 1.43, Z: 0},
		StartZ:   1.75,
		EndZ:     5.55,
		Arc:      0.22,
		Height:   1.72,
		FovY:     34 * math.Pi / 180,
		Width:    width,
		HeightPx: height,
	}
}

func (p CameraPath) At(t float64) Camera {
	t = clampFloat(t, 0, 1)
	distanceZ := lerpFloat(p.StartZ, p.EndZ, t)
	angle := p.Arc * math.Sin(math.Pi*t)
	pos := Vec3{
		X: math.Sin(angle) * 0.55,
		Y: p.Height + 0.08*math.Sin(math.Pi*t),
		Z: distanceZ,
	}
	forward := vnorm(vsub(p.Target, pos))
	right := vnorm(vcross(forward, Vec3{Y: 1}))
	up := vnorm(vcross(right, forward))
	focal := float64(p.HeightPx) / (2 * math.Tan(p.FovY/2))
	return Camera{
		Pos:     pos,
		Forward: forward,
		Right:   right,
		Up:      up,
		Focal:   focal,
		Near:    0.05,
	}
}

func (c Camera) ViewPoint(p Vec3) (x, y, z float64, ok bool) {
	q := vsub(p, c.Pos)
	z = vdot(q, c.Forward)
	if z <= c.Near {
		return 0, 0, z, false
	}
	x = vdot(q, c.Right)
	y = vdot(q, c.Up)
	return x, y, z, true
}

func (c Camera) Project(p Vec3, width, height int) (sx, sy, viewZ, invZ float64, ok bool) {
	x, y, z, visible := c.ViewPoint(p)
	if !visible {
		return 0, 0, 0, 0, false
	}
	sx = float64(width)/2 + x/z*c.Focal
	sy = float64(height)/2 - y/z*c.Focal
	return sx, sy, z, 1 / z, true
}
