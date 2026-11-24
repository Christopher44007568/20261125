package main

import (
	"fmt"
	"sort"
)

const (
	StatusBoundary    = "boundary"
	StatusManifold    = "manifold"
	StatusNonManifold = "non-manifold"
)

type Incidence struct {
	FaceIndex int
	FaceID    string
	From      int
	To        int
}

type Edge struct {
	ID                    string      `json:"id"`
	U                     int         `json:"u"`
	V                     int         `json:"v"`
	FaceIndices           []int       `json:"face_indices"`
	Faces                 []string    `json:"faces"`
	Incidence             []Incidence `json:"incidence"`
	Count                 int         `json:"count"`
	Status                string      `json:"status"`
	Length                float64     `json:"length"`
	GeometryKey           string      `json:"geometry_key"`
	OrientationConsistent bool        `json:"orientation_consistent"`
}

type FaceCoverage struct {
	FaceID                 string   `json:"face_id"`
	SourceID               string   `json:"source_id"`
	Sheet                  string   `json:"sheet"`
	DirectNonManifoldEdges []string `json:"direct_non_manifold_edges"`
	SheetNonManifoldEdges  []string `json:"sheet_non_manifold_edges"`
	Coverage               string   `json:"coverage"` // direct | same-sheet | clear
	SameSheetAffectedFaces []string `json:"same_sheet_affected_faces"`
}

type SheetCoverage struct {
	Sheet              string   `json:"sheet"`
	FaceIDs            []string `json:"face_ids"`
	PrimaryFaceIDs     []string `json:"primary_face_ids"`
	NonManifoldEdgeIDs []string `json:"non_manifold_edge_ids"`
}

type Topology struct {
	Edges            []*Edge
	BoundaryCount    int
	ManifoldCount    int
	NonManifoldCount int
	NonManifoldEdges []*Edge
	FaceCoverage     []FaceCoverage
	SheetCoverage    []SheetCoverage
	// faceEdges[i] 为面 i 的 3 条拓扑边（按顶点顺序）
	FaceEdges [][]*Edge
}

func vertexKey(m *Mesh, vi int) string {
	v := m.Vertices[vi]
	return fmt.Sprintf("%f,%f,%f", v.X, v.Y, v.Z)
}

func geometryKey(m *Mesh, u, v int) string {
	a, b := vertexKey(m, u), vertexKey(m, v)
	va, vb := m.Vertices[u], m.Vertices[v]
	less := va.X < vb.X ||
		(va.X == vb.X && va.Y < vb.Y) ||
		(va.X == vb.X && va.Y == vb.Y && va.Z < vb.Z)
	if less {
		return a + "|" + b
	}
	return b + "|" + a
}

// Analyze 构建半边索引并分类每条拓扑边。
func Analyze(m *Mesh) *Topology {
	type edgeKey struct{ u, v int }
	byKey := map[edgeKey]*Edge{}
	for fi, f := range m.Faces {
		n := len(f.Vertices)
		for i := 0; i < n; i++ {
			from := f.Vertices[i]
			to := f.Vertices[(i+1)%n]
			k := edgeKey{min(from, to), max(from, to)}
			e := byKey[k]
			if e == nil {
				e = &Edge{U: k.u, V: k.v}
				byKey[k] = e
			}
			e.FaceIndices = append(e.FaceIndices, fi)
			e.Faces = append(e.Faces, f.ID)
			e.Incidence = append(e.Incidence, Incidence{fi, f.ID, from, to})
		}
	}
	t := &Topology{FaceEdges: make([][]*Edge, len(m.Faces))}
	keys := make([]edgeKey, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].u != keys[j].u {
			return keys[i].u < keys[j].u
		}
		return keys[i].v < keys[j].v
	})
	for i, k := range keys {
		e := byKey[k]
		e.ID = fmt.Sprintf("E%02d", i)
		e.Count = len(e.Faces)
		switch {
		case e.Count == 1:
			e.Status = StatusBoundary
			t.BoundaryCount++
		case e.Count == 2:
			e.Status = StatusManifold
			t.ManifoldCount++
		default:
			e.Status = StatusNonManifold
			t.NonManifoldCount++
			t.NonManifoldEdges = append(t.NonManifoldEdges, e)
		}
		e.Length = m.Vertices[e.U].Sub(m.Vertices[e.V]).Len()
		e.GeometryKey = geometryKey(m, e.U, e.V)
		e.OrientationConsistent = true
		if e.Count == 2 {
			a, b := e.Incidence[0], e.Incidence[1]
			e.OrientationConsistent = a.From == b.To && a.To == b.From
		}
		t.Edges = append(t.Edges, e)
	}
	edgeByPair := func(u, v int) *Edge {
		return byKey[edgeKey{min(u, v), max(u, v)}]
	}
	for fi, f := range m.Faces {
		n := len(f.Vertices)
		for i := 0; i < n; i++ {
			t.FaceEdges[fi] = append(t.FaceEdges[fi], edgeByPair(f.Vertices[i], f.Vertices[(i+1)%n]))
		}
	}
	t.computeCoverage(m)
	return t
}

func (t *Topology) computeCoverage(m *Mesh) {
	nmEdges := map[*Edge]bool{}
	for _, e := range t.NonManifoldEdges {
		nmEdges[e] = true
	}
	// 薄片 -> 直接受影响的非流形边 / 面
	sheetEdges := map[string][]string{}
	sheetPrimary := map[string][]string{}
	sheetFaces := map[string][]string{}
	var sheetOrder []string
	for fi, f := range m.Faces {
		sheet := f.Provenance.Sheet
		if _, ok := sheetFaces[sheet]; !ok {
			sheetOrder = append(sheetOrder, sheet)
		}
		sheetFaces[sheet] = append(sheetFaces[sheet], f.ID)
		for _, e := range t.FaceEdges[fi] {
			if nmEdges[e] && !contains(sheetEdges[sheet], e.ID) {
				sheetEdges[sheet] = append(sheetEdges[sheet], e.ID)
			}
		}
	}
	for _, e := range t.NonManifoldEdges {
		for _, fi := range e.FaceIndices {
			sheet := m.Faces[fi].Provenance.Sheet
			id := m.Faces[fi].ID
			if !contains(sheetPrimary[sheet], id) {
				sheetPrimary[sheet] = append(sheetPrimary[sheet], id)
			}
		}
	}
	for fi, f := range m.Faces {
		fc := FaceCoverage{
			FaceID:   f.ID,
			SourceID: f.Provenance.SourceID,
			Sheet:    f.Provenance.Sheet,
		}
		for _, e := range t.FaceEdges[fi] {
			if nmEdges[e] {
				fc.DirectNonManifoldEdges = append(fc.DirectNonManifoldEdges, e.ID)
			}
		}
		sheet := f.Provenance.Sheet
		fc.SheetNonManifoldEdges = append(fc.SheetNonManifoldEdges, sheetEdges[sheet]...)
		switch {
		case len(fc.DirectNonManifoldEdges) > 0:
			fc.Coverage = "direct"
			fc.SameSheetAffectedFaces = []string{f.ID}
		case len(fc.SheetNonManifoldEdges) > 0:
			fc.Coverage = "same-sheet"
			fc.SameSheetAffectedFaces = append(fc.SameSheetAffectedFaces, sheetPrimary[sheet]...)
		default:
			fc.Coverage = "clear"
		}
		t.FaceCoverage = append(t.FaceCoverage, fc)
	}
	for _, sheet := range sheetOrder {
		t.SheetCoverage = append(t.SheetCoverage, SheetCoverage{
			Sheet:              sheet,
			FaceIDs:            sheetFaces[sheet],
			PrimaryFaceIDs:     sheetPrimary[sheet],
			NonManifoldEdgeIDs: sheetEdges[sheet],
		})
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
