package camera

import (
	"math"

	"handlate/internal/vec3"
)

const (
	FocalPx   = 260.0
	DepthBias = 0.01 // metres; hand must clear occluders conservatively
	NearPlane = 0.05
	FarPlane  = 8.0
)

// Project maps a world point to camera pixels and view-space depth.
func Project(head vec3.Pose, p vec3.V, width, height int) (x, y, depth float64, ok bool) {
	local := head.Rotate(vec3.Sub(p, head.Pos))
	if local.Z <= NearPlane || local.Z >= FarPlane {
		return 0, 0, 0, false
	}
	x = float64(width)/2 + FocalPx*local.X/local.Z
	y = float64(height)/2 - FocalPx*local.Y/local.Z
	return x, y, local.Z, true
}

// RayBox returns the nearest hit distance along ray o->dir for an AABB.
// Returns ok=false when there is no forward intersection.
func RayBox(o, dir vec3.V, b vec3.AABB) (dist float64, ok bool) {
	tmin := 0.0
	tmax := math.Inf(1)
	axes := []struct{ o, lo, hi float64 }{
		{o.X, b.Min.X, b.Max.X},
		{o.Y, b.Min.Y, b.Max.Y},
		{o.Z, b.Min.Z, b.Max.Z},
	}
	d := []float64{dir.X, dir.Y, dir.Z}
	for i, a := range axes {
		if math.Abs(d[i]) < 1e-9 {
			if a.o < a.lo || a.o > a.hi {
				return 0, false
			}
			continue
		}
		t1 := (a.lo - a.o) / d[i]
		t2 := (a.hi - a.o) / d[i]
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tmin = math.Max(tmin, t1)
		tmax = math.Min(tmax, t2)
		if tmin > tmax {
			return 0, false
		}
	}
	if tmin < 0 {
		tmin = 0
	}
	return tmin, true
}

// Occluded performs a conservative depth test against current world occluders.
// A hand point is hidden when an occluder surface is at least DepthBias closer.
func Occluded(head vec3.Pose, p vec3.V, boxes []vec3.AABB) bool {
	dir := vec3.Sub(p, head.Pos)
	handDist := vec3.Len(dir)
	for _, b := range boxes {
		if hit, ok := RayBox(head.Pos, dir, b); ok && hit < handDist-DepthBias {
			return true
		}
	}
	return false
}
