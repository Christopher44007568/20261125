package main

import (
	"fmt"
)

type Mesh struct {
	Vertices  []Vec3
	Triangles [][3]int
}

type Topology struct {
	Vertices      int  `json:"vertices"`
	Triangles     int  `json:"triangles"`
	Edges         int  `json:"edges"`
	BoundaryEdges int  `json:"boundary_edges"`
	ManifoldEdges int  `json:"manifold_edges"`
	AbnormalEdges int  `json:"abnormal_edges"`
	Watertight    bool `json:"watertight"`
}

func buildMesh(profile Profile) Mesh {
	poly := profile.Polygon()
	n := len(poly)
	verts := make([]Vec3, 0, 2*n)
	for _, p := range poly {
		verts = append(verts, Vec3{p.X, p.Y, 0})
	}
	for _, p := range poly {
		verts = append(verts, Vec3{p.X, p.Y, profile.Params.Height})
	}

	positive := signedArea2(poly) >= 0
	tris := make([][3]int, 0, 4*n-4)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		if positive {
			tris = append(tris, [3]int{i, j, n + j}, [3]int{i, n + j, n + i})
		} else {
			tris = append(tris, [3]int{i, n + j, j}, [3]int{i, n + i, n + j})
		}
	}
	for i := 1; i < n-1; i++ {
		if positive {
			tris = append(tris, [3]int{0, i + 1, i})
			tris = append(tris, [3]int{n, n + i, n + i + 1})
		} else {
			tris = append(tris, [3]int{0, i, i + 1})
			tris = append(tris, [3]int{n, n + i + 1, n + i})
		}
	}
	return Mesh{Vertices: verts, Triangles: tris}
}

func (m Mesh) Triangle(t int) [3]Vec3 {
	idx := m.Triangles[t]
	return [3]Vec3{m.Vertices[idx[0]], m.Vertices[idx[1]], m.Vertices[idx[2]]}
}

func (m Mesh) SignedVolume() float64 {
	var sum float64
	for _, idx := range m.Triangles {
		a := m.Vertices[idx[0]]
		b := m.Vertices[idx[1]]
		c := m.Vertices[idx[2]]
		sum += a.Dot(b.Cross(c)) / 6
	}
	return sum
}

func (m Mesh) Volume() float64 {
	return absFloat(m.SignedVolume())
}

func (m Mesh) Topology() Topology {
	type edge struct{ A, B int }
	counts := make(map[edge]int)
	for _, tri := range m.Triangles {
		for k := 0; k < 3; k++ {
			a, b := tri[k], tri[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			counts[edge{a, b}]++
		}
	}
	topo := Topology{
		Vertices:   len(m.Vertices),
		Triangles:  len(m.Triangles),
		Edges:      len(counts),
		Watertight: true,
	}
	for _, count := range counts {
		switch count {
		case 1:
			topo.BoundaryEdges++
		case 2:
			topo.ManifoldEdges++
		default:
			topo.AbnormalEdges++
		}
	}
	topo.Watertight = topo.BoundaryEdges == 0 && topo.AbnormalEdges == 0
	return topo
}

func (m Mesh) Validate() error {
	topo := m.Topology()
	if !topo.Watertight {
		return fmt.Errorf("non-watertight combinatorial mesh: %+v", topo)
	}
	if topo.Vertices-topo.Edges+topo.Triangles != 2 {
		return fmt.Errorf("mesh is not a closed genus-zero shell: %+v", topo)
	}
	return nil
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
