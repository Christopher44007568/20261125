package section

import (
	"brepsection/brep"
	"brepsection/geom"
)

// Extract runs the full section pipeline for one BRep and one cutting plane.
// The tolerance is derived from the solid's bounding box and scaled up for the
// float32 stress mode, matching its rounding magnitude.
func Extract(b *brep.BRep, pl geom.Plane, mode Mode) *Report {
	diag := b.AABB().Diagonal()
	eps := diag * 1e-9
	if mode.Use32 {
		eps = diag * 1e-6
	}
	// Snap tolerance dominates every active predicate error scale.
	tol := eps * 10

	var raw []Seg3
	for idx := range b.Faces {
		raw = append(raw, faceSegments(b, idx, pl, eps, mode.Use32)...)
	}

	g := buildGraph(raw, pl, tol, mode.Use32)
	g.mergeCollinear()
	loops := g.walkLoops()
	assignLevels(loops)

	rep := &Report{
		PlaneName:   pl.Name,
		ModeName:    mode.Name,
		RawSegments: len(raw),
		Junctions:   len(g.snapper.pts),
		Loops:       loops,
		Tolerance:   tol,
	}

	rep.Closed = true
	for _, l := range loops {
		l.ClosureGap = closureResidual(l, g)
		if l.ClosureGap > tol {
			rep.Closed = false
		}
		if l.ClosureGap > rep.MaxGap {
			rep.MaxGap = l.ClosureGap
		}
		l.Crossings = selfIntersections(l)
		if len(l.Crossings) > 0 {
			rep.SelfIntersect = true
		}
	}

	// A closed section network has even degree at every junction; dangling
	// endpoints mark an open, therefore invalid, contour.
	for _, edges := range g.adj {
		if len(edges)%2 != 0 {
			rep.Closed = false
		}
	}

	rep.OrderConsistent = checkOrder(loops)
	rep.WindingConsistent = checkWinding(loops)
	rep.NodeErrors = collectNodeErrors(g)
	return rep
}

// checkWinding verifies that every contour at the same nesting level shares
// one orientation. Absolute CCW/CW depends on the cutting-plane basis, so the
// stable invariant is within-level uniformity, not strict alternation.
func checkWinding(loops []*Loop) bool {
	if len(loops) == 0 {
		return true
	}
	levelCCW := map[int]bool{}
	levelSeen := map[int]bool{}
	for _, l := range loops {
		if !levelSeen[l.Level] {
			levelSeen[l.Level] = true
			levelCCW[l.Level] = l.CCW
			continue
		}
		if l.CCW != levelCCW[l.Level] {
			return false
		}
	}
	return true
}

// checkOrder enforces the layer sequence contract:
//   - every non-outer loop sits inside exactly one loop of the level below;
//   - same-level loops never nest inside each other.
func checkOrder(loops []*Loop) bool {
	for _, l := range loops {
		if l.Level < 0 {
			return false
		}
		var otherLoops [][]geom.Vec2
		for _, o := range loops {
			if o != l {
				otherLoops = append(otherLoops, o.Nodes)
			}
		}
		c := RepresentativePoint(l.Nodes, otherLoops...)
		parents := 0
		for _, other := range loops {
			if other == l || !pointStrictlyInside(c, other.Nodes) {
				continue
			}
			if other.Level >= l.Level {
				return false
			}
			if other.Level == l.Level-1 {
				parents++
			}
		}
		if l.Level > 0 && parents != 1 {
			return false
		}
	}
	return true
}

func collectNodeErrors(g *stitchedGraph) []NodeErr {
	out := make([]NodeErr, 0, len(g.snapper.pts))
	for _, sp := range g.snapper.pts {
		out = append(out, NodeErr{Pos: sp.pos, Err: sp.err})
	}
	return out
}
