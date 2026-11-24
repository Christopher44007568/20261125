package main

import "math"

// Vec3 is a meter-space point. The viewer looks toward +Z, +X is right and +Y is up.
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{X: v.X + o.X, Y: v.Y + o.Y, Z: v.Z + o.Z}
}

func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{X: v.X - o.X, Y: v.Y - o.Y, Z: v.Z - o.Z}
}

func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

// RotateY applies the engine's right-handed yaw transform.
func RotateY(v Vec3, yaw float64) Vec3 {
	c, s := math.Cos(yaw), math.Sin(yaw)
	return Vec3{
		X: c*v.X + s*v.Z,
		Y: v.Y,
		Z: -s*v.X + c*v.Z,
	}
}

// Pose is a head/eye pose in world coordinates. Only yaw is needed by the
// head-turn cases, but positions remain full 3D.
type Pose struct {
	Pos Vec3    `json:"position"`
	Yaw float64 `json:"yawRadians"`
}

// ToEye transforms a world point into this pose's eye coordinates.
func (p Pose) ToEye(world Vec3) Vec3 {
	d := world.Sub(p.Pos)
	c, s := math.Cos(p.Yaw), math.Sin(p.Yaw)
	return Vec3{
		X: c*d.X - s*d.Z,
		Y: d.Y,
		Z: s*d.X + c*d.Z,
	}
}

type ScreenPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type FloatRect struct {
	MinX float64 `json:"minX"`
	MinY float64 `json:"minY"`
	MaxX float64 `json:"maxX"`
	MaxY float64 `json:"maxY"`
}

func RectFromPoints(points ...ScreenPoint) FloatRect {
	if len(points) == 0 {
		return FloatRect{}
	}
	r := FloatRect{MinX: points[0].X, MaxX: points[0].X, MinY: points[0].Y, MaxY: points[0].Y}
	for _, p := range points[1:] {
		r.MinX = math.Min(r.MinX, p.X)
		r.MaxX = math.Max(r.MaxX, p.X)
		r.MinY = math.Min(r.MinY, p.Y)
		r.MaxY = math.Max(r.MaxY, p.Y)
	}
	return r
}

func (r FloatRect) Width() float64  { return math.Max(0, r.MaxX-r.MinX) }
func (r FloatRect) Height() float64 { return math.Max(0, r.MaxY-r.MinY) }
func (r FloatRect) Area() float64   { return r.Width() * r.Height() }

func (r FloatRect) Inset(x, y float64) FloatRect {
	return FloatRect{MinX: r.MinX + x, MinY: r.MinY + y, MaxX: r.MaxX - x, MaxY: r.MaxY - y}
}

func (r FloatRect) Intersection(other FloatRect) FloatRect {
	out := FloatRect{
		MinX: math.Max(r.MinX, other.MinX),
		MinY: math.Max(r.MinY, other.MinY),
		MaxX: math.Min(r.MaxX, other.MaxX),
		MaxY: math.Min(r.MaxY, other.MaxY),
	}
	if out.MaxX < out.MinX || out.MaxY < out.MinY {
		return FloatRect{}
	}
	return out
}

func (r FloatRect) Intersects(other FloatRect) bool {
	i := r.Intersection(other)
	return i.Width() > 0 && i.Height() > 0
}

func (r FloatRect) ContainsPoint(p ScreenPoint) bool {
	return p.X >= r.MinX && p.X <= r.MaxX && p.Y >= r.MinY && p.Y <= r.MaxY
}
