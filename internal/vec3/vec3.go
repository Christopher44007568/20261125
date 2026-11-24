package vec3

import "math"

// V is a right-handed position vector in world coordinates, metres.
type V struct {
	X, Y, Z float64
}

func Add(a, b V) V           { return V{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func Sub(a, b V) V           { return V{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func Scale(a V, s float64) V { return V{a.X * s, a.Y * s, a.Z * s} }
func Dot(a, b V) float64     { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func Cross(a, b V) V {
	return V{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func Len(a V) float64     { return math.Sqrt(Dot(a, a)) }
func Dist(a, b V) float64 { return Len(Sub(a, b)) }

// Lerp returns a + (b-a)*t.
func Lerp(a, b V, t float64) V {
	return V{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
	}
}

// Pose is a rigid transform: position plus yaw/pitch (radians, camera convention).
type Pose struct {
	Pos   V
	Yaw   float64
	Pitch float64
}

// Rotate applies yaw then pitch to a local-space vector.
func (p Pose) Rotate(local V) V {
	cy, sy := math.Cos(p.Yaw), math.Sin(p.Yaw)
	yawed := V{
		X: cy*local.X + sy*local.Z,
		Y: local.Y,
		Z: -sy*local.X + cy*local.Z,
	}
	cp, sp := math.Cos(p.Pitch), math.Sin(p.Pitch)
	return V{
		X: yawed.X,
		Y: cp*yawed.Y - sp*yawed.Z,
		Z: sp*yawed.Y + cp*yawed.Z,
	}
}

// Transform maps a local point into world space.
func (p Pose) Transform(local V) V { return Add(p.Pos, p.Rotate(local)) }

// AABB is an axis-aligned box in world space.
type AABB struct {
	Min, Max V
}

func NewAABB(center V, half V) AABB {
	return AABB{Min: Sub(center, half), Max: Add(center, half)}
}

// Contains reports whether p is inside the box, optionally inflated by eps.
func (b AABB) Contains(p V, eps float64) bool {
	return p.X >= b.Min.X-eps && p.X <= b.Max.X+eps &&
		p.Y >= b.Min.Y-eps && p.Y <= b.Max.Y+eps &&
		p.Z >= b.Min.Z-eps && p.Z <= b.Max.Z+eps
}
