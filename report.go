package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

type TopoSummary struct {
	VertexCount      int             `json:"vertex_count"`
	FaceCount        int             `json:"face_count"`
	EdgeCount        int             `json:"edge_count"`
	BoundaryCount    int             `json:"boundary_count"`
	ManifoldCount    int             `json:"manifold_count"`
	NonManifoldCount int             `json:"non_manifold_count"`
	Edges            []*Edge         `json:"edges"`
	NonManifoldEdges []*Edge         `json:"non_manifold_edges"`
	FaceCoverage     []FaceCoverage  `json:"face_coverage"`
	SheetCoverage    []SheetCoverage `json:"sheet_coverage"`
}

func summarize(m *Mesh, t *Topology) *TopoSummary {
	return &TopoSummary{
		VertexCount:      len(m.Vertices),
		FaceCount:        len(m.Faces),
		EdgeCount:        len(t.Edges),
		BoundaryCount:    t.BoundaryCount,
		ManifoldCount:    t.ManifoldCount,
		NonManifoldCount: t.NonManifoldCount,
		Edges:            t.Edges,
		NonManifoldEdges: t.NonManifoldEdges,
		FaceCoverage:     t.FaceCoverage,
		SheetCoverage:    t.SheetCoverage,
	}
}

type EdgeRef struct {
	ID          string `json:"id"`
	U           int    `json:"u"`
	V           int    `json:"v"`
	Status      string `json:"status"`
	GeometryKey string `json:"geometry_key"`
}

type FaceMapping struct {
	FaceID            string     `json:"face_id"`
	Provenance        Provenance `json:"provenance"`
	BeforeVertices    []int      `json:"before_vertices"`
	AfterVertices     []int      `json:"after_vertices"`
	BeforeCentroid    Vec3       `json:"before_centroid"`
	AfterCentroid     Vec3       `json:"after_centroid"`
	BeforeNormal      Vec3       `json:"before_normal"`
	AfterNormal       Vec3       `json:"after_normal"`
	BeforeEdges       []EdgeRef  `json:"before_edges"`
	AfterEdges        []EdgeRef  `json:"after_edges"`
	SemanticKeyBefore string     `json:"semantic_key_before"`
	SemanticKeyAfter  string     `json:"semantic_key_after"`
	GeometryPreserved bool       `json:"geometry_preserved"`
	SourcePreserved   bool       `json:"source_preserved"`
	NormalPreserved   bool       `json:"normal_preserved"`
}

type Safety struct {
	FacesDeleted          int  `json:"faces_deleted"`
	FaceCountBefore       int  `json:"face_count_before"`
	FaceCountAfter        int  `json:"face_count_after"`
	AllGeometryPreserved  bool `json:"all_geometry_preserved"`
	AllSourcesPreserved   bool `json:"all_sources_preserved"`
	AllNormalsPreserved   bool `json:"all_normals_preserved"`
	AllSemanticsPreserved bool `json:"all_semantics_preserved"`
}

type Report struct {
	Schema          string        `json:"schema"`
	Status          string        `json:"status"`
	RepairConfirmed bool          `json:"repair_confirmed"`
	SourceMesh      string        `json:"source_mesh"`
	RepairedMesh    string        `json:"repaired_mesh,omitempty"`
	Before          *TopoSummary  `json:"before"`
	After           *TopoSummary  `json:"after,omitempty"`
	Splits          []SplitRecord `json:"splits,omitempty"`
	FaceMappings    []FaceMapping `json:"face_mappings"`
	Safety          Safety        `json:"safety"`
}

type FaceProvenanceDoc struct {
	Schema       string        `json:"schema"`
	Status       string        `json:"status"`
	SourceMesh   string        `json:"source_mesh"`
	RepairedMesh string        `json:"repaired_mesh"`
	FaceMappings []FaceMapping `json:"face_mappings"`
}

func faceEdgeRefs(t *Topology, fi int) []EdgeRef {
	var refs []EdgeRef
	for _, e := range t.FaceEdges[fi] {
		refs = append(refs, EdgeRef{e.ID, e.U, e.V, e.Status, e.GeometryKey})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	return refs
}

// semanticKey 由面来源与排序后的顶点坐标导出，修复前后可逐面对比。
func semanticKey(m *Mesh, f Face) string {
	var coords []string
	for _, vi := range f.Vertices {
		v := m.Vertices[vi]
		coords = append(coords, fmt.Sprintf("%.6f,%.6f,%.6f", v.X, v.Y, v.Z))
	}
	sort.Strings(coords)
	p := f.Provenance
	raw := p.SourceID + "|" + p.SourceFace + "|" + p.Sheet + "|" + p.Role + "|" + strings.Join(coords, ";")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func sortedCoords(m *Mesh, verts []int) []string {
	var out []string
	for _, vi := range verts {
		v := m.Vertices[vi]
		out = append(out, fmt.Sprintf("%.6f,%.6f,%.6f", v.X, v.Y, v.Z))
	}
	sort.Strings(out)
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// BuildFaceMappings 逐面建立修复前后对应：顶点、质心、法线、拓扑边、语义键。
func BuildFaceMappings(before, after *Mesh, tBefore, tAfter *Topology) []FaceMapping {
	var out []FaceMapping
	for fi := range before.Faces {
		fb := before.Faces[fi]
		fa := after.Faces[fi]
		nb := before.FaceNormal(fb)
		na := after.FaceNormal(fa)
		dot := nb.X*na.X + nb.Y*na.Y + nb.Z*na.Z
		m := FaceMapping{
			FaceID:            fb.ID,
			Provenance:        fb.Provenance,
			BeforeVertices:    append([]int(nil), fb.Vertices...),
			AfterVertices:     append([]int(nil), fa.Vertices...),
			BeforeCentroid:    before.FaceCentroid(fb),
			AfterCentroid:     after.FaceCentroid(fa),
			BeforeNormal:      nb,
			AfterNormal:       na,
			BeforeEdges:       faceEdgeRefs(tBefore, fi),
			AfterEdges:        faceEdgeRefs(tAfter, fi),
			SemanticKeyBefore: semanticKey(before, fb),
			SemanticKeyAfter:  semanticKey(after, fa),
		}
		m.GeometryPreserved = equalStrings(sortedCoords(before, fb.Vertices), sortedCoords(after, fa.Vertices))
		m.SourcePreserved = fb.Provenance == fa.Provenance && fb.ID == fa.ID
		m.NormalPreserved = math.Abs(dot-1) < 1e-9
		out = append(out, m)
	}
	return out
}

func buildSafety(before, after *Mesh, maps []FaceMapping) Safety {
	s := Safety{
		FacesDeleted:          len(before.Faces) - len(after.Faces),
		FaceCountBefore:       len(before.Faces),
		FaceCountAfter:        len(after.Faces),
		AllGeometryPreserved:  true,
		AllSourcesPreserved:   true,
		AllNormalsPreserved:   true,
		AllSemanticsPreserved: true,
	}
	for _, m := range maps {
		s.AllGeometryPreserved = s.AllGeometryPreserved && m.GeometryPreserved
		s.AllSourcesPreserved = s.AllSourcesPreserved && m.SourcePreserved
		s.AllNormalsPreserved = s.AllNormalsPreserved && m.NormalPreserved
		s.AllSemanticsPreserved = s.AllSemanticsPreserved &&
			m.SemanticKeyBefore == m.SemanticKeyAfter
	}
	return s
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
