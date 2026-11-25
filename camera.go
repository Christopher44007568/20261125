package main

import "math"

type Projected struct {
	X float64
	Y float64
	Z float64
}

func projectedVec3(p Projected) Vec3 {
	return Vec3{X: p.X, Y: p.Y, Z: p.Z}
}

func (c *Camera) orient() (Vec3, Vec3, Vec3) {
	cp := math.Cos(c.Pitch)
	offset := Vec3{
		X: c.Distance * math.Cos(c.Yaw) * cp,
		Y: c.Distance * math.Sin(c.Yaw) * cp,
		Z: c.Distance * math.Sin(c.Pitch),
	}
	c.Position = add(c.Target, offset)
	forward := normalize(sub(c.Target, c.Position))
	right := normalize(cross(forward, c.Up))
	trueUp := cross(right, forward)
	return forward, right, trueUp
}

func (c *Camera) Project(point Vec3) Projected {
	forward, right, trueUp := c.orient()
	relative := sub(point, c.Position)
	z := dot(relative, forward)
	x := dot(relative, right)
	y := dot(relative, trueUp)
	scale := c.Focal / z
	return Projected{
		X: float64(c.Width)/2 + x*scale,
		Y: float64(c.Height)/2 - y*scale,
		Z: z,
	}
}

func projectedDistance(a, b Projected) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(dx*dx + dy*dy)
}
