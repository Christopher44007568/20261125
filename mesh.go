package main

import (
	"fmt"
	"math"
	"sort"
)

type Triangle struct {
	A      int  `json:"a"`
	B      int  `json:"b"`
	C      int  `json:"c"`
	Group  int  `json:"group"`
	Filled bool `json:"filled,omitempty"`
}

type Mesh struct {
	Name string     `json:"name"`
	Pos  []Vec3     `json:"positions"`
	Tri  []Triangle `json:"triangles"`
}

func (m *Mesh) Clone() *Mesh {
	cp := &Mesh{Name: m.Name, Pos: append([]Vec3(nil), m.Pos...), Tri: append([]Triangle(nil), m.Tri...)}
	return cp
}

func (m *Mesh) AddVertex(p Vec3) int {
	m.Pos = append(m.Pos, p)
	return len(m.Pos) - 1
}

func (m *Mesh) AddTriangle(a, b, c, group int, filled bool) int {
	m.Tri = append(m.Tri, Triangle{A: a, B: b, C: c, Group: group, Filled: filled})
	return len(m.Tri) - 1
}

func (m *Mesh) triIndices(t Triangle) [3]int { return [3]int{t.A, t.B, t.C} }

func (m *Mesh) FaceNormal(ti int) Vec3 {
	t := m.Tri[ti]
	ab := m.Pos[t.B].Sub(m.Pos[t.A])
	ac := m.Pos[t.C].Sub(m.Pos[t.A])
	n := ab.Cross(ac)
	if n.LenSqr() < 1e-24 {
		return V(0, 0, 1)
	}
	return n.Unit()
}

type EdgeKey [2]int

func MakeEdgeKey(a, b int) EdgeKey {
	if a > b {
		a, b = b, a
	}
	return EdgeKey{a, b}
}

type EdgeUse struct {
	Triangle int
	A, B     int
}

type EdgeStat struct {
	A             int     `json:"a"`
	B             int     `json:"b"`
	Count         int     `json:"count"`
	LengthMM      float64 `json:"length_mm"`
	Seam          bool    `json:"seam"`
	OrientationOK bool    `json:"orientation_ok,omitempty"`
}

type WatertightReport struct {
	SeamWatertight        bool     `json:"seam_watertight"`
	Watertight            bool     `json:"watertight"`
	ClosedManifold        bool     `json:"closed_manifold"`
	OrientationConsistent bool     `json:"orientation_consistent"`
	EdgeCount             int      `json:"edge_count"`
	TriangleCount         int      `json:"triangle_count"`
	VertexCount           int      `json:"vertex_count"`
	SeamEdgeCount         int      `json:"seam_edge_count"`
	SeamBadEdges          int      `json:"seam_bad_edges"`
	BoundaryEdges         int      `json:"boundary_edges"`
	OuterBoundaryEdges    int      `json:"outer_boundary_edges"`
	NonManifoldEdges      int      `json:"non_manifold_edges"`
	OrientationBadEdges   int      `json:"orientation_bad_edges"`
	MaxSeamEdgeLengthMM   float64  `json:"max_seam_edge_length_mm"`
	Issues                []string `json:"issues"`
}

func BuildEdgeUses(tris []Triangle) map[EdgeKey][]EdgeUse {
	uses := make(map[EdgeKey][]EdgeUse)
	for ti, t := range tris {
		idx := [3]int{t.A, t.B, t.C}
		for e := 0; e < 3; e++ {
			a, b := idx[e], idx[(e+1)%3]
			key := MakeEdgeKey(a, b)
			uses[key] = append(uses[key], EdgeUse{Triangle: ti, A: a, B: b})
		}
	}
	return uses
}

func CheckWatertight(m *Mesh, seamEdges [][2]int) WatertightReport {
	uses := BuildEdgeUses(m.Tri)
	seamSet := make(map[EdgeKey]bool, len(seamEdges))
	for _, e := range seamEdges {
		seamSet[MakeEdgeKey(e[0], e[1])] = true
	}

	rep := WatertightReport{
		SeamWatertight:        true,
		Watertight:            true,
		ClosedManifold:        true,
		OrientationConsistent: true,
		EdgeCount:             len(uses),
		TriangleCount:         len(m.Tri),
		VertexCount:           len(m.Pos),
		SeamEdgeCount:         len(seamEdges),
		Issues:                []string{},
	}

	keys := make([]EdgeKey, 0, len(uses))
	for k := range uses {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})

	for _, key := range keys {
		u := uses[key]
		length := m.Pos[key[0]].Distance(m.Pos[key[1]])
		isSeam := seamSet[key]
		if isSeam && length > rep.MaxSeamEdgeLengthMM {
			rep.MaxSeamEdgeLengthMM = length
		}

		if len(u) != 2 {
			rep.ClosedManifold = false
			if len(u) == 1 {
				rep.BoundaryEdges++
				if isSeam {
					rep.SeamWatertight = false
					rep.SeamBadEdges++
					rep.Issues = append(rep.Issues, fmt.Sprintf("seam edge %d-%d has only one triangle", key[0], key[1]))
				} else {
					rep.OuterBoundaryEdges++
				}
			} else {
				rep.NonManifoldEdges++
				if isSeam {
					rep.SeamWatertight = false
					rep.SeamBadEdges++
				}
				rep.Issues = append(rep.Issues, fmt.Sprintf("edge %d-%d has %d triangles", key[0], key[1], len(u)))
			}
			continue
		}

		if u[0].A == u[1].A && u[0].B == u[1].B {
			rep.OrientationConsistent = false
			rep.OrientationBadEdges++
			if isSeam {
				rep.SeamWatertight = false
				rep.SeamBadEdges++
				rep.Issues = append(rep.Issues, fmt.Sprintf("seam edge %d-%d has equal winding", key[0], key[1]))
			}
		}
	}

	for key := range seamSet {
		if _, ok := uses[key]; !ok {
			rep.SeamWatertight = false
			rep.SeamBadEdges++
			rep.Issues = append(rep.Issues, fmt.Sprintf("expected seam edge %d-%d is absent", key[0], key[1]))
		}
	}

	rep.Watertight = rep.SeamWatertight && rep.SeamBadEdges == 0
	return rep
}

func (r WatertightReport) GlobalWatertight() bool {
	return r.ClosedManifold && r.NonManifoldEdges == 0 && r.OrientationConsistent
}

type SharpEdge struct {
	A        int     `json:"a"`
	B        int     `json:"b"`
	AngleDeg float64 `json:"angle_deg"`
	Seam     bool    `json:"seam"`
}

func DetectSharpEdges(m *Mesh, seamEdges [][2]int, thresholdDeg float64) []SharpEdge {
	uses := BuildEdgeUses(m.Tri)
	seam := make(map[EdgeKey]bool, len(seamEdges))
	for _, e := range seamEdges {
		seam[MakeEdgeKey(e[0], e[1])] = true
	}
	var out []SharpEdge
	for key, eu := range uses {
		if len(eu) != 2 {
			continue
		}
		n0 := m.FaceNormal(eu[0].Triangle)
		n1 := m.FaceNormal(eu[1].Triangle)
		angle := degFromRad(math.Acos(clampFloat(n0.Dot(n1), -1, 1)))
		if angle >= thresholdDeg || m.Tri[eu[0].Triangle].Group != m.Tri[eu[1].Triangle].Group {
			out = append(out, SharpEdge{A: key[0], B: key[1], AngleDeg: angle, Seam: seam[key]})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AngleDeg > out[j].AngleDeg })
	return out
}
