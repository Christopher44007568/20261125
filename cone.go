package normalcone

import (
	"fmt"
	"math"
)

const (
	deg90  = math.Pi / 2
	deg180 = math.Pi
	eps    = 1e-12
)

type Vec3 struct {
	X, Y, Z float64
}

type Cone struct {
	Axis      Vec3
	HalfAngle float64
	Valid     bool
	Reason    string
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

func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		X: v.Y*o.Z - v.Z*o.Y,
		Y: v.Z*o.X - v.X*o.Z,
		Z: v.X*o.Y - v.Y*o.X,
	}
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

func (v Vec3) Normalize() (Vec3, bool) {
	length := v.Length()
	if !isFinite(length) || length <= eps {
		return Vec3{}, false
	}
	return v.Scale(1 / length), true
}

func Normalize(v Vec3) (Vec3, bool) {
	return v.Normalize()
}

func Unit(n Vec3) bool {
	length := n.Length()
	return isFinite(length) && math.Abs(length-1) <= 1e-10
}

func FaceNormal(a, b, c Vec3) (Vec3, bool) {
	return b.Sub(a).Cross(c.Sub(a)).Normalize()
}

func ReferenceCone(normals []Vec3) Cone {
	if len(normals) == 0 {
		return invalidCone("normal set is empty")
	}

	var sum Vec3
	for index, normal := range normals {
		if !isFiniteVec(normal) {
			return invalidCone(fmt.Sprintf("normal %d contains a non-finite component", index))
		}
		length := normal.Length()
		if length <= eps {
			return invalidCone(fmt.Sprintf("normal %d is zero or too short", index))
		}
		if math.Abs(length-1) > 1e-10 {
			return invalidCone(fmt.Sprintf("normal %d is not unit length", index))
		}
		sum = sum.Add(normal)
	}

	axis, ok := sum.Normalize()
	if !ok {
		return Cone{HalfAngle: deg180, Valid: true}
	}

	minDot := 1.0
	for _, normal := range normals {
		minDot = math.Min(minDot, axis.Dot(normal))
	}
	halfAngle := math.Acos(clamp(minDot, -1, 1))
	return Cone{Axis: axis, HalfAngle: halfAngle, Valid: true}
}

func ConservativeCull(cone Cone, surfacePoint, camera Vec3) bool {
	if !cone.Valid {
		return false
	}
	view, ok := camera.Sub(surfacePoint).Normalize()
	if !ok || cone.HalfAngle >= deg180 || cone.Axis.Length() <= eps {
		return false
	}

	dot := view.Dot(cone.Axis)
	limit := math.Cos(deg90 + cone.HalfAngle)
	return dot < limit
}

func AggressiveCull(axis Vec3, surfacePoint, camera Vec3) bool {
	view, ok := camera.Sub(surfacePoint).Normalize()
	if !ok {
		return false
	}
	return view.Dot(axis) <= 0
}

func ActuallyVisible(normal, surfacePoint, camera Vec3) (bool, bool) {
	view, ok := camera.Sub(surfacePoint).Normalize()
	if !ok {
		return false, false
	}
	return view.Dot(normal) > eps, true
}

func invalidCone(reason string) Cone {
	return Cone{Valid: false, Reason: reason}
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func isFiniteVec(v Vec3) bool {
	return isFinite(v.X) && isFinite(v.Y) && isFinite(v.Z)
}

func clamp(v, low, high float64) float64 {
	return math.Max(low, math.Min(high, v))
}
