package section

import (
	"math"

	"brepsection/geom"
)

// snappedPoint stores the representative 2D position of an endpoint cluster
// together with the spread (closure residual) of every raw endpoint in it.
type snappedPoint struct {
	pos geom.Vec2
	err float64
	n   int
}

// endpointSnapper merges near-coincident section endpoints using a spatial
// hash whose cell size equals the closure tolerance.
type endpointSnapper struct {
	tol   float64
	cells map[uint64][]int
	pts   []snappedPoint
}

func newSnapper(tol float64) *endpointSnapper {
	return &endpointSnapper{tol: tol, cells: map[uint64][]int{}}
}

func cellKey(cx, cy int64) uint64 {
	// Offset signed indices so different cells never collide.
	return (uint64(uint32(cx)) << 32) | uint64(uint32(cy))
}

func (s *endpointSnapper) add(p geom.Vec2) int {
	cx := int64(math.Floor(p.X / s.tol))
	cy := int64(math.Floor(p.Y / s.tol))
	for dx := int64(-1); dx <= 1; dx++ {
		for dy := int64(-1); dy <= 1; dy++ {
			for _, idx := range s.cells[cellKey(cx+dx, cy+dy)] {
				if geom.Distance2(p, s.pts[idx].pos) <= s.tol {
					sp := &s.pts[idx]
					sp.n++
					sp.pos = sp.pos.Add(p.Sub(sp.pos).Scale(1 / float64(sp.n)))
					if d := geom.Distance2(p, sp.pos); d > sp.err {
						sp.err = d
					}
					return idx
				}
			}
		}
	}
	idx := len(s.pts)
	s.pts = append(s.pts, snappedPoint{pos: p, n: 1})
	s.cells[cellKey(cx, cy)] = append(s.cells[cellKey(cx, cy)], idx)
	return idx
}

// graphEdge joins two clusters and remembers the raw 3D section segments.
type graphEdge struct {
	a, b   int
	segs   []Seg3
	dir    geom.Vec2
	length float64
}

// stitchedGraph holds the undirected edge network built from raw segments.
type stitchedGraph struct {
	snapper *endpointSnapper
	edges   []*graphEdge
	adj     map[int][]*graphEdge
}

// buildGraph snaps all segment endpoints, removes duplicates and returns the
// undirected section network. In stress mode the in-plane projection is also
// rounded through float32 so junction residuals remain visible.
func buildGraph(raw []Seg3, pl geom.Plane, tol float64, use32 bool) *stitchedGraph {
	sn := newSnapper(tol)
	g := &stitchedGraph{snapper: sn, adj: map[int][]*graphEdge{}}
	edgeKey := map[uint64]*graphEdge{}
	for _, seg := range raw {
		a := sn.add(project2D(pl, seg.A, use32))
		b := sn.add(project2D(pl, seg.B, use32))
		if a == b {
			continue
		}
		lo, hi := a, b
		if lo > hi {
			lo, hi = hi, lo
		}
		key := (uint64(lo) << 32) | uint64(hi)
		if e, ok := edgeKey[key]; ok {
			e.segs = append(e.segs, seg)
			continue
		}
		e := &graphEdge{a: a, b: b, segs: []Seg3{seg}}
		edgeKey[key] = e
		g.edges = append(g.edges, e)
		g.adj[a] = append(g.adj[a], e)
		g.adj[b] = append(g.adj[b], e)
	}
	for _, e := range g.edges {
		d := sn.pts[e.b].pos.Sub(sn.pts[e.a].pos)
		e.dir = d.Normalized()
		e.length = d.Len()
	}
	return g
}

func project2D(pl geom.Plane, q geom.Vec3, use32 bool) geom.Vec2 {
	v := pl.To2D(q)
	if !use32 {
		return v
	}
	return geom.V2(float64(float32(v.X)), float64(float32(v.Y)))
}
