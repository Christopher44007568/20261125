package section

import (
	"fmt"
	"math"

	"brepsection/geom"
)

// assignLevels computes the nesting level of every loop (0 = outer boundary,
// 1 = hole, 2 = island ...) using parity point-in-loop tests, then sorts
// loops into a deterministic layer order: outer contours first, holes next,
// and area-descending within the same level.
func assignLevels(loops []*Loop) {
	for _, l := range loops {
		level := 0
		var otherLoops [][]geom.Vec2
		for _, o := range loops {
			if o != l {
				otherLoops = append(otherLoops, o.Nodes)
			}
		}
		c := RepresentativePoint(l.Nodes, otherLoops...)
		for _, other := range loops {
			if other == l {
				continue
			}
			if pointStrictlyInside(c, other.Nodes) {
				level++
			}
		}
		l.Level = level
		l.CCW = geom.PolygonArea(l.Nodes) > 0
	}

	// Bubble-free deterministic ordering keyed by level and area.
	for i := 1; i < len(loops); i++ {
		for j := i; j > 0; j-- {
			a, b := loops[j-1], loops[j]
			aa := math.Abs(geom.PolygonArea(a.Nodes))
			ba := math.Abs(geom.PolygonArea(b.Nodes))
			if a.Level > b.Level || (a.Level == b.Level && aa < ba) {
				loops[j-1], loops[j] = b, a
			} else {
				break
			}
		}
	}
	for i, l := range loops {
		l.Index = i
	}
}

// pointStrictlyInside is a parity test that ignores the polygon boundary, so
// two disjoint contours that merely touch never count as nested.
func pointStrictlyInside(p geom.Vec2, loop []geom.Vec2) bool {
	inside := false
	n := len(loop)
	for i := 0; i < n; i++ {
		a := loop[i]
		b := loop[(i+1)%n]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			x := a.X + (p.Y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
			if x > p.X {
				inside = !inside
			}
		}
	}
	return inside
}

// RepresentativePoint returns a point strictly inside the given loop. For
// nesting tests it must belong to the loop's own material, so candidates are
// accepted only when they are interior to this loop yet outside every other
// loop (a centroid can otherwise land inside a hole of its own polygon).
func RepresentativePoint(loop []geom.Vec2, others ...[]geom.Vec2) geom.Vec2 {
	candidates := []geom.Vec2{}
	var centroid geom.Vec2
	for _, p := range loop {
		centroid = centroid.Add(p)
	}
	centroid = centroid.Scale(1 / float64(len(loop)))
	candidates = append(candidates, centroid)
	for i, p := range loop {
		q := loop[(i+1)%len(loop)]
		candidates = append(candidates, p.Add(q).Scale(0.5))
	}
	for _, c := range candidates {
		if !pointStrictlyInside(c, loop) {
			continue
		}
		blocked := false
		for _, o := range others {
			if pointStrictlyInside(c, o) {
				blocked = true
				break
			}
		}
		if !blocked {
			return c
		}
	}
	return centroid
}

// selfIntersections checks every non-adjacent edge pair of a loop for proper
// crossings. Shared endpoints (adjacent edges and ring closure) are allowed.
func selfIntersections(l *Loop) []string {
	var out []string
	n := len(l.Nodes)
	for i := 0; i < n; i++ {
		a := l.Nodes[i]
		b := l.Nodes[(i+1)%n]
		for j := i + 1; j < n; j++ {
			if j == i || (j+1)%n == i || i+1 == j {
				continue
			}
			c := l.Nodes[j]
			d := l.Nodes[(j+1)%n]
			if segmentsCross(a, b, c, d) {
				out = append(out, fmt.Sprintf("edge %d x edge %d", i, j))
			}
		}
	}
	return out
}

// segmentsCross reports a strict interior crossing of two 2D segments.
func segmentsCross(a, b, c, d geom.Vec2) bool {
	r := b.Sub(a)
	s := d.Sub(c)
	den := r.Cross(s)
	if math.Abs(den) < 1e-12 {
		return false // collinear overlaps are not proper self-crossings
	}
	t := c.Sub(a).Cross(s) / den
	u := c.Sub(a).Cross(r) / den
	eps := 1e-9
	return t > eps && t < 1-eps && u > eps && u < 1-eps
}

// closureResidual returns the maximum endpoint cluster spread touching a loop.
func closureResidual(l *Loop, g *stitchedGraph) float64 {
	worst := 0.0
	for _, id := range l.NodeIDs {
		if g.snapper.pts[id].err > worst {
			worst = g.snapper.pts[id].err
		}
	}
	return worst
}
