package main

import "math"

type Vec3 struct{ X, Y, Z float64 }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64       { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Norm() Vec3         { return a.Scale(1 / a.Len()) }

// yaw rotates a direction around the Y axis (positive = turn right).
func yaw(d Vec3, ang float64) Vec3 {
	c, s := math.Cos(ang), math.Sin(ang)
	return Vec3{d.X*c + d.Z*s, d.Y, -d.X*s + d.Z*c}
}

// Camera is a pinhole eye model looking down +Z with a yaw offset.
type Camera struct {
	Origin Vec3
	Yaw    float64 // vergence / head-turn offset in radians
	FovY   float64 // vertical field of view in radians
}

// Ray returns the world-space ray for pixel (px,py) in a WxH image.
func (c Camera) Ray(px, py, w, h int) Vec3 {
	ndcX := (float64(px)+0.5)/float64(w)*2 - 1
	ndcY := 1 - (float64(py)+0.5)/float64(h)*2
	t := math.Tan(c.FovY / 2)
	aspect := float64(w) / float64(h)
	d := Vec3{ndcX * t * aspect, ndcY * t, 1}.Norm()
	return yaw(d, c.Yaw)
}
