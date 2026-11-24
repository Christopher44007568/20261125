package main

import (
	"math"
	"sort"
)

type Tri2 struct {
	P        [3]Vec2
	Segment  int
	Part     int
	Inverted bool
}

type Intersection struct {
	I, J int
	P    Vec2
	T, U float64
}

type SectionDiagnostics struct {
	Triangles      []Tri2
	Inverted       int
	Intersections  []Intersection
	LoopCount      int
	LoopComponents []LoopComponent
}

type LoopComponent struct {
	Segments []int
	Points   []Vec2
}

func sectionTriangles(profile Profile) []Tri2 {
	out := make([]Tri2, 0, 2*(len(profile.Outer)-1))
	expected := expectedBandSign(profile)
	for i := 0; i+1 < len(profile.Outer); i++ {
		p0, p1 := profile.Outer[i], profile.Outer[i+1]
		q0, q1 := profile.Inner[i], profile.Inner[i+1]
		parts := [][3]Vec2{
			{p0, q0, p1},
			{p1, q0, q1},
		}
		for part, tri := range parts {
			area := cross2(tri[0], tri[1], tri[2])
			inv := math.Abs(area) > 1e-10 && area*expected < 0
			out = append(out, Tri2{
				P: tri, Segment: i, Part: part, Inverted: inv,
			})
		}
	}
	return out
}

func expectedBandSign(profile Profile) float64 {
	if len(profile.Outer) < 2 {
		return -1
	}
	a := cross2(profile.Outer[0], profile.Inner[0], profile.Outer[1])
	if math.Abs(a) < 1e-14 {
		return -1
	}
	return math.Copysign(1, a)
}

func innerIntersections(inner []Vec2) []Intersection {
	var hits []Intersection
	for i := 0; i+1 < len(inner); i++ {
		for j := i + 1; j+1 < len(inner); j++ {
			if j == i+1 {
				continue
			}
			if p, t, u, ok := segmentIntersection(
				inner[i], inner[i+1], inner[j], inner[j+1]); ok {
				hits = append(hits, Intersection{I: i, J: j, P: p, T: t, U: u})
			}
		}
	}
	return hits
}

func segmentIntersection(a, b, c, d Vec2) (Vec2, float64, float64, bool) {
	r := b.Sub(a)
	s := d.Sub(c)
	den := r.Cross(s)
	if math.Abs(den) < 1e-13 {
		return Vec2{}, 0, 0, false
	}
	qp := c.Sub(a)
	t := qp.Cross(s) / den
	u := qp.Cross(r) / den
	const edgeEpsilon = 1e-9
	if t <= edgeEpsilon || t >= 1-edgeEpsilon ||
		u <= edgeEpsilon || u >= 1-edgeEpsilon {
		return Vec2{}, 0, 0, false
	}
	return a.Add(r.Scale(t)), t, u, true
}

func analyzeSection(profile Profile) SectionDiagnostics {
	tris := sectionTriangles(profile)
	inv := 0
	for _, tri := range tris {
		if tri.Inverted {
			inv++
		}
	}
	hits := innerIntersections(profile.Inner)
	components := groupLoops(hits)
	return SectionDiagnostics{
		Triangles: tris, Inverted: inv, Intersections: hits,
		LoopCount: len(components), LoopComponents: components,
	}
}

func groupLoops(hits []Intersection) []LoopComponent {
	if len(hits) == 0 {
		return nil
	}
	parent := make(map[int]int)
	var find func(int) int
	find = func(x int) int {
		if parent[x] == 0 {
			parent[x] = x
		}
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for _, h := range hits {
		union(h.I, h.J)
	}
	groups := make(map[int]*LoopComponent)
	for _, h := range hits {
		root := find(h.I)
		g := groups[root]
		if g == nil {
			g = &LoopComponent{}
			groups[root] = g
		}
		g.Points = append(g.Points, h.P)
		g.Segments = appendUnique(g.Segments, h.I)
		g.Segments = appendUnique(g.Segments, h.J)
	}
	out := make([]LoopComponent, 0, len(groups))
	for _, g := range groups {
		sort.Ints(g.Segments)
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Segments[0] != out[j].Segments[0] {
			return out[i].Segments[0] < out[j].Segments[0]
		}
		return len(out[i].Segments) > len(out[j].Segments)
	})
	return out
}

func appendUnique(xs []int, x int) []int {
	for _, v := range xs {
		if v == x {
			return xs
		}
	}
	return append(xs, x)
}

func innerWallInverted(profile Profile) []bool {
	out := make([]bool, len(profile.Outer)-1)
	for i := range out {
		outerDir := profile.Outer[i+1].Sub(profile.Outer[i])
		innerDir := profile.Inner[i+1].Sub(profile.Inner[i])
		out[i] = outerDir.Dot(innerDir) < -1e-12
	}
	return out
}
