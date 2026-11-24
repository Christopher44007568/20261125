package main

import "math"

type Rect struct {
	X, Y, W, H int
}

type Camera struct {
	Target  Vec3
	Eye     Vec3
	Forward Vec3
	Right   Vec3
	Up      Vec3

	CenterX float64
	CenterY float64
	Scale   float64
	Rect    Rect
}

type Projected struct {
	X, Y  float64
	Depth float64
}

func NewCamera(target, eye Vec3) Camera {
	forward := target.Sub(eye).Unit()
	right := forward.Cross(Vec3{Z: 1}).Unit()
	up := right.Cross(forward).Unit()
	return Camera{Target: target, Eye: eye, Forward: forward, Right: right, Up: up}
}

func (c Camera) Raw(p Vec3) (float64, float64, float64) {
	d := p.Sub(c.Target)
	return d.Dot(c.Right), d.Dot(c.Up), p.Sub(c.Eye).Dot(c.Forward)
}

func (c Camera) Project(p Vec3) Projected {
	x, y, depth := c.Raw(p)
	return Projected{
		X:     float64(c.Rect.X+c.Rect.W/2) + (x-c.CenterX)*c.Scale,
		Y:     float64(c.Rect.Y+c.Rect.H/2) - (y-c.CenterY)*c.Scale,
		Depth: depth,
	}
}

func (c *Camera) Fit(mesh Mesh, rect Rect, padX, padY float64) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, vertex := range mesh.Vertices {
		x, y, _ := c.Raw(vertex.P)
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
	}
	w := math.Max(1e-6, maxX-minX)
	h := math.Max(1e-6, maxY-minY)
	c.Scale = math.Min(
		(float64(rect.W)-2*padX)/w,
		(float64(rect.H)-2*padY)/h,
	)
	c.CenterX = 0.5 * (minX + maxX)
	c.CenterY = 0.5 * (minY + maxY)
	c.Rect = rect
}
