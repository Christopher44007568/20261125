package section

import (
	"brepsection/brep"
	"brepsection/geom"
)

// faceFrame builds an orthonormal 2D coordinate system on a planar face.
type faceFrame struct {
	origin geom.Vec3
	u, v   geom.Vec3
}

func newFaceFrame(n geom.Vec3, origin geom.Vec3) faceFrame {
	ref := geom.V3(0, 0, 1)
	if abs(n.Dot(ref)) > 0.9 {
		ref = geom.V3(1, 0, 0)
	}
	u := ref.Sub(n.Scale(ref.Dot(n))).Normalized()
	v := n.Cross(u).Normalized()
	return faceFrame{origin: origin, u: u, v: v}
}

func (f faceFrame) project(p geom.Vec3) geom.Vec2 {
	d := p.Sub(f.origin)
	return geom.V2(d.Dot(f.u), d.Dot(f.v))
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// faceSegments intersects one face with the cutting plane and returns the
// section segments that lie inside the face material.
func faceSegments(b *brep.BRep, faceIdx int, pl geom.Plane, eps float64, use32 bool) []Seg3 {
	f := b.Faces[faceIdx]
	pts := b.Points
	if abs(f.Normal.Dot(pl.Normal)) > 1-1e-9 {
		var out []Seg3
		emit := func(a, c geom.Vec3, _, _ bool) {
			if !pl.Contains(a, eps) || !pl.Contains(c, eps) || a.Approx(c, eps) {
				return
			}
			out = append(out, Seg3{A: a, B: c})
		}
		loopSection(pts, f.Outer, pl, eps, use32, true, emit)
		for _, hole := range f.Holes {
			loopSection(pts, hole, pl, eps, use32, false, emit)
		}
		return out
	}

	var out []Seg3
	emit := func(a, c geom.Vec3, boundary, outerLoop bool) {
		if a.Approx(c, eps) {
			return
		}
		// Boundary edges lie on the face loop itself (including hole outlines)
		// and are always section geometry. Crossing chords must pass the
		// midpoint parity test so void-space chords are rejected.
		if boundary {
			out = append(out, Seg3{A: a, B: c})
			return
		}
		// A crossing chord produced by a hole loop is the section outline of
		// that hole itself, so it is kept unconditionally.
		if !outerLoop {
			out = append(out, Seg3{A: a, B: c})
			return
		}
		mid := a.Add(c).Scale(0.5)
		// An outer-loop crossing chord stays inside the outer boundary even
		// when it spans a hole; a hole-loop chord must avoid the hole void.
		if !pointInOuterOnly(mid, f, pts) {
			return
		}
		out = append(out, Seg3{A: a, B: c})
	}

	loopSection(pts, f.Outer, pl, eps, use32, true, emit)
	for _, hole := range f.Holes {
		loopSection(pts, hole, pl, eps, use32, false, emit)
	}
	return out
}

// pointInOuterOnly tests membership against the outer loop alone, ignoring
// holes. It supports outer crossing chords that legitimately span voids.
func pointInOuterOnly(q geom.Vec3, f brep.Face, pts []geom.Vec3) bool {
	frame := newFaceFrame(f.Normal, pts[f.Outer[0]])
	qp := frame.project(q)
	outer := make([]geom.Vec2, len(f.Outer))
	for i, idx := range f.Outer {
		outer[i] = frame.project(pts[idx])
	}
	return geom.PointInLoop(qp, outer) != 0
}
