package main

import "math"

type normalDSU struct {
	parent []int
}

func newNormalDSU(n int) *normalDSU {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &normalDSU{parent: p}
}

func (d *normalDSU) find(x int) int {
	for d.parent[x] != x {
		d.parent[x] = d.parent[d.parent[x]]
		x = d.parent[x]
	}
	return x
}

func (d *normalDSU) union(a, b int) {
	ra, rb := d.find(a), d.find(b)
	if ra != rb {
		d.parent[rb] = ra
	}
}

// CornerNormals returns one normal per triangle corner. Positions may be
// welded across a crease; normals are still split whenever the dihedral angle
// is hard or the triangles belong to different smoothing groups.
func ComputeCornerNormals(m *Mesh, hardAngleDeg float64) []Vec3 {
	cornerCount := len(m.Tri) * 3
	dsu := newNormalDSU(cornerCount)
	uses := BuildEdgeUses(m.Tri)

	for _, eu := range uses {
		if len(eu) != 2 {
			continue
		}
		u0, u1 := eu[0], eu[1]
		t0, t1 := m.Tri[u0.Triangle], m.Tri[u1.Triangle]
		n0 := m.FaceNormal(u0.Triangle)
		n1 := m.FaceNormal(u1.Triangle)
		angle := degFromRad(math.Acos(clampFloat(n0.Dot(n1), -1, 1)))
		smooth := t0.Group == t1.Group && angle < hardAngleDeg
		if !smooth {
			continue
		}

		verts0 := [3]int{t0.A, t0.B, t0.C}
		verts1 := [3]int{t1.A, t1.B, t1.C}
		for _, v := range [2]int{u0.A, u0.B} {
			c0 := cornerIndexForVertex(verts0, v)
			c1 := cornerIndexForVertex(verts1, v)
			if c0 >= 0 && c1 >= 0 {
				dsu.union(u0.Triangle*3+c0, u1.Triangle*3+c1)
			}
		}
	}

	sums := make(map[int]Vec3, cornerCount)
	for ti := range m.Tri {
		n := m.FaceNormal(ti)
		for c := 0; c < 3; c++ {
			node := dsu.find(ti*3 + c)
			sums[node] = sums[node].Add(n)
		}
	}

	out := make([]Vec3, cornerCount)
	for ti, t := range m.Tri {
		verts := [3]int{t.A, t.B, t.C}
		n := m.FaceNormal(ti)
		for c := 0; c < 3; c++ {
			node := dsu.find(ti*3 + c)
			if summed, ok := sums[node]; ok && summed.LenSqr() > 1e-20 {
				out[ti*3+c] = summed.Unit()
			} else {
				out[ti*3+c] = n
			}
			_ = verts
		}
	}
	return out
}

func cornerIndexForVertex(verts [3]int, v int) int {
	for i, x := range verts {
		if x == v {
			return i
		}
	}
	return -1
}

// SharedPositionNormalJump compares the two smoothing sides at a welded seam
// vertex. It is useful in the JSON report to prove the sharp normal was not
// averaged away.
func SeamNormalJump(m *Mesh, cornerNormals []Vec3, seamVertex int) (float64, Vec3, Vec3, bool) {
	var first Vec3
	var second Vec3
	foundFirst := false
	groups := map[int]Vec3{}
	for ti, t := range m.Tri {
		verts := [3]int{t.A, t.B, t.C}
		for c, v := range verts {
			if v != seamVertex {
				continue
			}
			n := cornerNormals[ti*3+c]
			if existing, ok := groups[t.Group]; ok {
				groups[t.Group] = existing.Add(n).Scale(0.5).Unit()
			} else {
				groups[t.Group] = n
			}
		}
	}
	for _, n := range groups {
		if !foundFirst {
			first, foundFirst = n, true
			continue
		}
		second = n
		angle := degFromRad(math.Acos(clampFloat(first.Dot(second), -1, 1)))
		return angle, first, second, true
	}
	return 0, first, second, false
}
