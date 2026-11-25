package mesh

import (
	"fmt"
	"math"
	"sort"

	"adaptivebezier/internal/geom"
)

func (b *builder) chordSample(t Triangle, n int) float64 {
	if n < 2 {
		n = 2
	}
	a := b.vertices[t.A].World
	bv := b.vertices[t.B].World
	c := b.vertices[t.C].World
	normal := bv.Sub(a).Cross(c.Sub(a)).Normalize()
	maxErr := 0.0
	p0 := b.vertices[t.A].Param
	p1 := b.vertices[t.B].Param
	p2 := b.vertices[t.C].Param
	for i := 0; i <= n; i++ {
		for j := 0; j <= n-i; j++ {
			l0 := float64(i) / float64(n)
			l1 := float64(j) / float64(n)
			l2 := 1 - l0 - l1
			u := p0.X*l0 + p1.X*l1 + p2.X*l2
			v := p0.Y*l0 + p1.Y*l1 + p2.Y*l2
			q := b.patch.Eval(u, v)
			err := math.Abs(q.Sub(a).Dot(normal)) * b.project.Scale
			if err > maxErr {
				maxErr = err
			}
		}
	}
	return maxErr
}

func (b *builder) edgeUsage() map[Edge]int {
	usage := make(map[Edge]int, len(b.triangles)*3)
	for _, t := range b.triangles {
		for i := 0; i < 3; i++ {
			a, c := edgeOf(t, i)
			usage[makeEdge(a, c)]++
		}
	}
	return usage
}

func (b *builder) refine() error {
	target := b.tol * 0.86
	for iter := 0; iter < b.opts.MaxIterations; iter++ {
		b.stats.Iterations = iter + 1
		marked := make(map[Edge]bool)
		for _, t := range b.triangles {
			if b.chordSample(t, b.opts.ErrorSampleN) <= target {
				continue
			}
			for i := 0; i < 3; i++ {
				a, c := edgeOf(t, i)
				e := makeEdge(a, c)
				if !b.trimEdges[e] {
					marked[e] = true
				}
			}
		}
		if len(marked) == 0 {
			return nil
		}
		edges := make([]Edge, 0, len(marked))
		for e := range marked {
			edges = append(edges, e)
		}
		sort.Slice(edges, func(i, j int) bool {
			if edges[i].A != edges[j].A {
				return edges[i].A < edges[j].A
			}
			return edges[i].B < edges[j].B
		})
		mid := make(map[Edge]int, len(marked))
		for _, e := range edges {
			pa := b.vertices[e.A].Param
			pb := b.vertices[e.B].Param
			p := geom.Vec2{X: (pa.X + pb.X) * 0.5, Y: (pa.Y + pb.Y) * 0.5}
			if p.Sub(pa).Len() < 1e-12 || p.Sub(pb).Len() < 1e-12 {
				continue
			}
			mid[e] = b.vertexFor(p)
		}
		if len(mid) == 0 {
			return fmt.Errorf("adaptive refinement stalled")
		}
		next := make([]Triangle, 0, len(b.triangles)*2)
		for _, t := range b.triangles {
			next = append(next, b.conformingSplit(t, mid)...)
		}
		b.triangles = next
		if len(b.triangles) > 250000 {
			return fmt.Errorf("mesh exceeded 250000 triangles")
		}
	}
	return fmt.Errorf("refinement did not converge after %d iterations", b.opts.MaxIterations)
}

func (b *builder) conformingSplit(t Triangle, mids map[Edge]int) []Triangle {
	e := [3]Edge{
		makeEdge(t.A, t.B),
		makeEdge(t.B, t.C),
		makeEdge(t.C, t.A),
	}
	m := [3]int{-1, -1, -1}
	bits := 0
	for i := range e {
		if idx, ok := mids[e[i]]; ok {
			m[i], bits = idx, bits|(1<<i)
		}
	}
	out := make([]Triangle, 0, 4)
	add := func(a, c, d int) {
		out = append(out, Triangle{A: a, B: c, C: d})
	}
	ma, mb, mc := m[0], m[1], m[2]
	switch bits {
	case 0:
		add(t.A, t.B, t.C)
	case 1:
		add(t.A, ma, t.C)
		add(ma, t.B, t.C)
	case 2:
		add(t.A, t.B, mb)
		add(t.A, mb, t.C)
	case 4:
		add(t.A, t.B, mc)
		add(t.B, t.C, mc)
	case 3:
		add(t.A, ma, t.C)
		add(ma, t.B, mb)
		add(ma, mb, t.C)
	case 6:
		add(t.A, t.B, mb)
		add(t.A, mb, mc)
		add(mb, t.C, mc)
	case 5:
		add(t.A, ma, mc)
		add(ma, t.B, t.C)
		add(ma, t.C, mc)
	case 7:
		add(t.A, ma, mc)
		add(ma, t.B, mb)
		add(t.C, mc, mb)
		add(ma, mb, mc)
	}
	return out
}

func (b *builder) finish(sampler *boundarySampler, points []boundaryPoint) *Mesh {
	usage := b.edgeUsage()
	m := &Mesh{
		Vertices: b.vertices, Triangles: b.triangles, Boundary: b.boundary,
		TrimEdges: b.trimEdges, Stats: b.stats, Watertight: true,
		Contour: sampler.contourError(points),
	}
	for _, t := range b.triangles {
		if err := b.chordSample(t, b.opts.AuditSampleN); err > m.MaxChord {
			m.MaxChord = err
		}
	}
	for e, count := range usage {
		expected := 2
		if b.trimEdges[e] {
			expected = 1
		}
		if count == expected {
			continue
		}
		m.CrackEdges++
		m.Watertight = false
		m.Crack += b.vertices[e.A].Screen.Sub(b.vertices[e.B].Screen).Len()
	}
	return m
}
