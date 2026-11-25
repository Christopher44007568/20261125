package analysis

import (
	"math"

	"meshscope/internal/mesh"
)

type neighbor struct {
	face  int
	angle float64
}

type topology struct {
	adj    [][]neighbor
	edges  map[mesh.EdgeKey][]int
	center []mesh.Vec3
	normal []mesh.Vec3
	area   []float64
}

func buildTopology(m *mesh.Mesh) topology {
	n := len(m.Tris)
	t := topology{
		adj:    make([][]neighbor, n),
		edges:  mesh.EdgeMap(m),
		center: make([]mesh.Vec3, n),
		normal: make([]mesh.Vec3, n),
		area:   make([]float64, n),
	}
	for i, tri := range m.Tris {
		t.center[i] = tri.Center
		t.normal[i] = tri.Normal
		t.area[i] = tri.Area
	}
	for _, faces := range t.edges {
		if len(faces) != 2 {
			continue
		}
		a, b := faces[0], faces[1]
		angle := angleDeg(t.normal[a], t.normal[b])
		t.adj[a] = append(t.adj[a], neighbor{face: b, angle: angle})
		t.adj[b] = append(t.adj[b], neighbor{face: a, angle: angle})
	}
	for i := range t.adj {
		if len(t.adj[i]) > 0 {
			continue
		}
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			d := t.center[i].Dist(t.center[j])
			if d < 1e-5 {
				angle := angleDeg(t.normal[i], t.normal[j])
				t.adj[i] = append(t.adj[i], neighbor{face: j, angle: angle})
			}
		}
	}
	return t
}

func angleDeg(a, b mesh.Vec3) float64 {
	d := math.Max(-1, math.Min(1, a.Norm().Dot(b.Norm())))
	return math.Acos(d) * 180 / math.Pi
}

func (t topology) smoothPatch(seed, depth int) []int {
	seen := map[int]bool{seed: true}
	out := []int{seed}
	frontier := []int{seed}
	for d := 0; d < depth; d++ {
		var next []int
		for _, f := range frontier {
			for _, n := range t.adj[f] {
				if seen[n.face] || n.angle > 55 {
					continue
				}
				seen[n.face] = true
				out = append(out, n.face)
				next = append(next, n.face)
			}
		}
		frontier = next
	}
	return out
}

func (t topology) averageNormal(faces []int) mesh.Vec3 {
	var sum mesh.Vec3
	for _, f := range faces {
		sum = sum.Add(t.normal[f].Mul(t.area[f]))
	}
	return sum.Norm()
}
