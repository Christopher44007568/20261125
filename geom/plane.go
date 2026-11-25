package geom

import "math"

// Plane is an oriented cutting plane: points p satisfy (p-Origin)·Normal == 0.
// U and V form an orthonormal in-plane basis, so every 3D point on the plane
// maps to a unique 2D coordinate and back without distortion.
type Plane struct {
	Origin Vec3
	Normal Vec3
	U      Vec3
	V      Vec3
	Name   string
}

// NewPlane builds a plane through origin with the given (non-zero) normal.
func NewPlane(origin, normal Vec3, name string) Plane {
	n := normal.Normalized()
	ref := V3(0, 0, 1)
	if math.Abs(n.Dot(ref)) > 0.9 {
		ref = V3(1, 0, 0)
	}
	u := ref.Sub(n.Scale(ref.Dot(n))).Normalized()
	v := n.Cross(u).Normalized()
	return Plane{Origin: origin, Normal: n, U: u, V: v, Name: name}
}

// SignedDistance is positive on the side the normal points to.
func (p Plane) SignedDistance(q Vec3) float64 {
	return q.Sub(p.Origin).Dot(p.Normal)
}

// Contains reports whether q lies on the plane within eps.
func (p Plane) Contains(q Vec3, eps float64) bool {
	return math.Abs(p.SignedDistance(q)) <= eps
}

// To2D projects an on-plane 3D point to the plane's 2D frame.
func (p Plane) To2D(q Vec3) Vec2 {
	d := q.Sub(p.Origin)
	return V2(d.Dot(p.U), d.Dot(p.V))
}

// To3D lifts a 2D plane coordinate back to 3D.
func (p Plane) To3D(q Vec2) Vec3 {
	return p.Origin.Add(p.U.Scale(q.X)).Add(p.V.Scale(q.Y))
}

// SegmentCross computes the intersection parameter t in [0,1] of segment ab
// with the plane. ok is false when the segment is (numerically) parallel.
func (p Plane) SegmentCross(a, b Vec3) (t float64, q Vec3, ok bool) {
	da := p.SignedDistance(a)
	db := p.SignedDistance(b)
	den := db - da
	if math.Abs(den) < 1e-14 {
		return 0, Vec3{}, false
	}
	t = -da / den
	if t < -1e-10 || t > 1+1e-10 {
		return 0, Vec3{}, false
	}
	t = Clamp(t, 0, 1)
	return t, a.Add(b.Sub(a).Scale(t)), true
}

// AABB is an axis-aligned bounding box.
type AABB struct {
	Min, Max Vec3
}

// NewAABB returns the bounding box of a point set.
func NewAABB(points []Vec3) AABB {
	lo := points[0]
	hi := lo
	for _, p := range points[1:] {
		lo = Min3(lo, p)
		hi = Max3(hi, p)
	}
	return AABB{Min: lo, Max: hi}
}

// Center of the box.
func (b AABB) Center() Vec3 { return b.Min.Add(b.Max).Scale(0.5) }

// Diagonal length of the box.
func (b AABB) Diagonal() float64 { return b.Max.Sub(b.Min).Len() }

// Expand grows the box by d on every side.
func (b AABB) Expand(d float64) AABB {
	e := V3(d, d, d)
	return AABB{Min: b.Min.Sub(e), Max: b.Max.Add(e)}
}
