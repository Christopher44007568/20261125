package main

import "math"

type Camera struct {
	Azimuth   float64
	Elevation float64
	Center    Vec3
	Distance  float64
	Scale     float64
	Width     float64
	Height    float64
	PanX      float64
	PanY      float64
}

func NewCamera(azDeg, elDeg, width, height, scale float64) Camera {
	return Camera{
		Azimuth:   azDeg * math.Pi / 180,
		Elevation: elDeg * math.Pi / 180,
		Center:    Vec3{0, 0, 16},
		Distance:  520,
		Scale:     scale,
		Width:     width,
		Height:    height,
	}
}

func (c Camera) Basis() (right, up, view Vec3) {
	ca, sa := math.Cos(c.Azimuth), math.Sin(c.Azimuth)
	ce, se := math.Cos(c.Elevation), math.Sin(c.Elevation)
	view = Vec3{ce * ca, ce * sa, se}
	right = Vec3{-sa, ca, 0}
	up = view.Cross(right).Unit()
	return
}

// Project returns screen pixels and a painter's depth (larger is farther).
func (c Camera) Project(p Vec3) (x, y, depth float64) {
	right, up, view := c.Basis()
	rel := p.Sub(c.Center)
	xw := rel.Dot(right)
	yw := rel.Dot(up)
	depth = c.Distance - rel.Dot(view)
	persp := c.Distance / math.Max(depth, 40)
	x = c.Width*0.5 + c.PanX + xw*c.Scale*persp
	y = c.Height*0.5 + c.PanY - yw*c.Scale*persp
	return
}

// ProjectMM is Project without the pixel offset or scale.
func (c Camera) ProjectMM(p Vec3) (x, y float64) {
	right, up, view := c.Basis()
	rel := p.Sub(c.Center)
	xw := rel.Dot(right)
	yw := rel.Dot(up)
	depth := c.Distance - rel.Dot(view)
	persp := c.Distance / math.Max(depth, 40)
	return xw * persp, -yw * persp
}

func cameraFor(azDeg, elDeg float64) Camera {
	return NewCamera(azDeg, elDeg, 1600, 1100, 4.55)
}
