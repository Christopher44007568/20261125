// Package brep defines a faceted boundary representation: vertices plus
// planar faces, each face carrying one outer loop and zero or more inner
// (hole) loops. All geometry is exact straight-line geometry, which keeps
// section extraction deterministic and free of third-party dependencies.
package brep

import (
	"brepsection/geom"
)

// Face is one planar polygon with holes. Loops store vertex indices.
// The outer loop is oriented so its geometric normal points outward;
// inner loops use the opposite winding.
type Face struct {
	Outer []int
	Holes [][]int
	// Normal is the unit outward normal of the planar face.
	Normal geom.Vec3
	// Interior is a solid-side point used to orient and test the face.
	Interior geom.Vec3
}

// BRep is a closed solid boundary.
type BRep struct {
	Name      string
	Points    []geom.Vec3
	Faces     []Face
	edgeCache edgeFaces
}

// AABB computes the solid's axis-aligned bounding box.
func (b *BRep) AABB() geom.AABB {
	return geom.NewAABB(b.Points)
}

// NewellNormal computes the geometric normal of an arbitrary 3D polygon loop.
func NewellNormal(points []geom.Vec3, loop []int) geom.Vec3 {
	var n geom.Vec3
	m := len(loop)
	for i := 0; i < m; i++ {
		a := points[loop[i]]
		c := points[loop[(i+1)%m]]
		n.X += (a.Y - c.Y) * (a.Z + c.Z)
		n.Y += (a.Z - c.Z) * (a.X + c.X)
		n.Z += (a.X - c.X) * (a.Y + c.Y)
	}
	if n.IsZero() {
		return n
	}
	return n.Normalized()
}

// loopCentroid returns the average point of a loop.
func loopCentroid(points []geom.Vec3, loop []int) geom.Vec3 {
	var c geom.Vec3
	for _, idx := range loop {
		c = c.Add(points[idx])
	}
	return c.Scale(1 / float64(len(loop)))
}

// signedArea3 returns the signed loop area relative to a unit normal.
func signedArea3(points []geom.Vec3, loop []int, n geom.Vec3) float64 {
	var s float64
	m := len(loop)
	for i := 0; i < m; i++ {
		a := points[loop[i]]
		b := points[loop[(i+1)%m]]
		s += n.Dot(a.Cross(b))
	}
	return s * 0.5
}

// orientFace flips loops until the outer loop faces away from the solid
// interior point and inner loops use the opposite winding.
func orientFace(points []geom.Vec3, f *Face) {
	n := NewellNormal(points, f.Outer)
	c := loopCentroid(points, f.Outer)
	if n.Dot(c.Sub(f.Interior)) < 0 {
		reverse(f.Outer)
		n = n.Neg()
	}
	f.Normal = n
	for _, hole := range f.Holes {
		if signedArea3(points, hole, n) > 0 {
			reverse(hole)
		}
	}
}

func reverse(loop []int) {
	for i, j := 0, len(loop)-1; i < j; i, j = i+1, j-1 {
		loop[i], loop[j] = loop[j], loop[i]
	}
}
