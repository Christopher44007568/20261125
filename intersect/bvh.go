package intersect

import (
	"math"
	"sort"

	"ruledintersect/geom"
)

type aabb struct {
	lo, hi geom.Vec3
}

func triangleBox(t geom.MeshTriangle) aabb {
	lo, hi := geom.Bounds(t.P[:])
	return aabb{lo: lo.Sub(geom.V(planeEpsilon, planeEpsilon, planeEpsilon)),
		hi: hi.Add(geom.V(planeEpsilon, planeEpsilon, planeEpsilon))}
}

func unionBox(a, b aabb) aabb {
	return aabb{
		lo: geom.V(math.Min(a.lo.X, b.lo.X), math.Min(a.lo.Y, b.lo.Y), math.Min(a.lo.Z, b.lo.Z)),
		hi: geom.V(math.Max(a.hi.X, b.hi.X), math.Max(a.hi.Y, b.hi.Y), math.Max(a.hi.Z, b.hi.Z)),
	}
}

func (a aabb) overlaps(b aabb) bool {
	return a.lo.X <= b.hi.X && a.hi.X >= b.lo.X &&
		a.lo.Y <= b.hi.Y && a.hi.Y >= b.lo.Y &&
		a.lo.Z <= b.hi.Z && a.hi.Z >= b.lo.Z
}

type bvhNode struct {
	box         aabb
	left, right int
	triangles   []int
	leaf        bool
}

type bvh struct {
	nodes []bvhNode
	mesh  *geom.Mesh
}

func buildBVH(mesh *geom.Mesh) *bvh {
	b := &bvh{mesh: mesh}
	ids := make([]int, len(mesh.Triangles))
	for i := range ids {
		ids[i] = i
	}
	b.nodes = append(b.nodes, bvhNode{})
	b.build(0, ids)
	return b
}

func (b *bvh) build(slot int, ids []int) {
	box := triangleBox(b.mesh.Triangles[ids[0]])
	for _, id := range ids[1:] {
		box = unionBox(box, triangleBox(b.mesh.Triangles[id]))
	}
	b.nodes[slot].box = box
	if len(ids) <= 6 {
		b.nodes[slot].leaf = true
		b.nodes[slot].triangles = append([]int(nil), ids...)
		return
	}
	extent := box.hi.Sub(box.lo)
	axis := 0
	if extent.Y > extent.X {
		axis = 1
	}
	if extent.Z > []float64{extent.X, extent.Y, extent.Z}[axis] {
		axis = 2
	}
	sort.Slice(ids, func(i, j int) bool {
		return triangleCenter(b.mesh.Triangles[ids[i]], axis) <
			triangleCenter(b.mesh.Triangles[ids[j]], axis)
	})
	mid := len(ids) / 2
	left, right := len(b.nodes), len(b.nodes)+1
	b.nodes = append(b.nodes, bvhNode{}, bvhNode{})
	b.nodes[slot].left, b.nodes[slot].right = left, right
	b.build(left, ids[:mid])
	b.build(right, ids[mid:])
}

func triangleCenter(t geom.MeshTriangle, axis int) float64 {
	c := t.P[0].Add(t.P[1]).Add(t.P[2]).Div(3)
	switch axis {
	case 1:
		return c.Y
	case 2:
		return c.Z
	default:
		return c.X
	}
}

func (b *bvh) leafTriangles(slot int) []int {
	n := b.nodes[slot]
	if !n.leaf {
		return nil
	}
	return n.triangles
}

func (b *bvh) pairs(other *bvh, fn func(int, int)) {
	var walk func(int, int)
	walk = func(ai, bi int) {
		an, bn := b.nodes[ai], other.nodes[bi]
		if !an.box.overlaps(bn.box) {
			return
		}
		if an.leaf && bn.leaf {
			for _, i := range b.leafTriangles(ai) {
				for _, j := range other.leafTriangles(bi) {
					fn(i, j)
				}
			}
			return
		}
		if !an.leaf && (bn.leaf || boxVolume(an.box) >= boxVolume(bn.box)) {
			walk(an.left, bi)
			walk(an.right, bi)
			return
		}
		walk(ai, bn.left)
		walk(ai, bn.right)
	}
	walk(0, 0)
}

func boxVolume(b aabb) float64 {
	d := b.hi.Sub(b.lo)
	return math.Max(d.X, 1e-12) * math.Max(d.Y, 1e-12) * math.Max(d.Z, 1e-12)
}
