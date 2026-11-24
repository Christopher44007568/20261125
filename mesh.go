package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}
func (a Vec3) Len() float64 { return math.Sqrt(a.X*a.X + a.Y*a.Y + a.Z*a.Z) }
func (a Vec3) Normalized() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

type Provenance struct {
	SourceID   string `json:"source_id"`
	SourceFace string `json:"source_face"`
	Sheet      string `json:"sheet"`
	Role       string `json:"role"`
}

type Face struct {
	ID         string     `json:"id"`
	Vertices   []int      `json:"vertices"`
	Provenance Provenance `json:"provenance"`
}

type Mesh struct {
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Units       string `json:"units"`
	Vertices    []Vec3 `json:"vertices"`
	Faces       []Face `json:"faces"`
}

func (m *Mesh) FaceNormal(f Face) Vec3 {
	a := m.Vertices[f.Vertices[0]]
	b := m.Vertices[f.Vertices[1]]
	c := m.Vertices[f.Vertices[2]]
	return b.Sub(a).Cross(c.Sub(a)).Normalized()
}

func (m *Mesh) FaceCentroid(f Face) Vec3 {
	var c Vec3
	for _, vi := range f.Vertices {
		c = c.Add(m.Vertices[vi])
	}
	return c.Scale(1 / float64(len(f.Vertices)))
}

func SaveMesh(m *Mesh, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func LoadMesh(path string) (*Mesh, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Mesh
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("解析网格 %s: %w", path, err)
	}
	return &m, nil
}

// GenerateFanMesh 生成教学用缺陷零件：三片薄板（alpha/beta/gamma）
// 各由两个三角形构成，共享同一条几何脊边 v0--v1。
// 脊边因此挂接 3 个面，是一处典型非流形边；删除任一面都会破坏零件语义。
func GenerateFanMesh() *Mesh {
	return &Mesh{
		Schema:      "meshfix.mesh.v1",
		Name:        "three_sheet_fan",
		Description: "三片薄板共享同一几何脊边 AB；每片由两个三角形组成，任一面都承载零件语义。",
		Units:       "mm",
		Vertices: []Vec3{
			{X: -0.2, Y: 0, Z: -2.2},    // 0: 脊边端点 A
			{X: 0.2, Y: 0, Z: 2.2},      // 1: 脊边端点 B
			{X: 3.1, Y: 0.4, Z: 1},      // 2: alpha
			{X: 3.8, Y: -0.2, Z: -1.4},  // 3: alpha
			{X: -1.4, Y: 3, Z: 0.9},     // 4: beta
			{X: -3.2, Y: 2.1, Z: -1.4},  // 5: beta
			{X: -1.2, Y: -2.8, Z: -0.2}, // 6: gamma
			{X: -3, Y: -1.6, Z: 1.5},    // 7: gamma
		},
		Faces: []Face{
			{ID: "F00", Vertices: []int{0, 1, 2}, Provenance: Provenance{"sheet-alpha", "alpha/root", "alpha", "root patch on shared spine"}},
			{ID: "F01", Vertices: []int{0, 2, 3}, Provenance: Provenance{"sheet-alpha", "alpha/outer", "alpha", "outer patch"}},
			{ID: "F02", Vertices: []int{0, 1, 4}, Provenance: Provenance{"sheet-beta", "beta/root", "beta", "root patch on shared spine"}},
			{ID: "F03", Vertices: []int{0, 4, 5}, Provenance: Provenance{"sheet-beta", "beta/outer", "beta", "outer patch"}},
			{ID: "F04", Vertices: []int{0, 1, 6}, Provenance: Provenance{"sheet-gamma", "gamma/root", "gamma", "root patch on shared spine"}},
			{ID: "F05", Vertices: []int{0, 6, 7}, Provenance: Provenance{"sheet-gamma", "gamma/outer", "gamma", "outer patch"}},
		},
	}
}
