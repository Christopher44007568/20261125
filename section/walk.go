package section

import (
	"math"

	"brepsection/geom"
)

// angleTol decides when two edges count as collinear. Facet joins that model
// tangent surfaces are exactly collinear algebraically; 1e-7 rad keeps the
// merge sensitive to the float32 stress run without touching real corners.
const angleTol = 1e-7

// mergeCollinear repeatedly removes degree-2 nodes whose two edges form a
// straight line, so tangent facet chains become one polygon edge. Junctions
// of the real network are never removed.
func (g *stitchedGraph) mergeCollinear() {
	for {
		changed := false
		for node, edges := range g.adj {
			if len(edges) != 2 {
				continue
			}
			e0, e1 := edges[0], edges[1]
			d0 := g.directedDir(node, e0)
			d1 := g.directedDir(node, e1).Neg()
			if math.Acos(geom.Clamp(d0.Dot(d1), -1, 1)) > angleTol {
				continue
			}
			other0 := e0.other(node)
			other1 := e1.other(node)
			if other0 == other1 {
				continue // degenerate two-edge ring
			}
			merged := &graphEdge{
				a:      other0,
				b:      other1,
				segs:   append(append([]Seg3{}, e0.segs...), e1.segs...),
				dir:    g.snapper.pts[other1].pos.Sub(g.snapper.pts[other0].pos).Normalized(),
				length: geom.Distance2(g.snapper.pts[other0].pos, g.snapper.pts[other1].pos),
			}
			g.replaceEdge(node, e0, e1, merged)
			changed = true
			break
		}
		if !changed {
			return
		}
	}
}

func (e *graphEdge) other(node int) int {
	if e.a == node {
		return e.b
	}
	return e.a
}

// directedDir returns the unit direction leaving node along edge e.
func (g *stitchedGraph) directedDir(node int, e *graphEdge) geom.Vec2 {
	d := g.snapper.pts[e.other(node)].pos.Sub(g.snapper.pts[node].pos)
	return d.Normalized()
}

func (g *stitchedGraph) replaceEdge(node int, e0, e1, merged *graphEdge) {
	o0, o1 := e0.other(node), e1.other(node)
	var adj0, adj1 []*graphEdge
	for _, e := range g.adj[o0] {
		if e != e0 {
			adj0 = append(adj0, e)
		}
	}
	for _, e := range g.adj[o1] {
		if e != e1 {
			adj1 = append(adj1, e)
		}
	}
	adj0 = append(adj0, merged)
	adj1 = append(adj1, merged)
	g.adj[o0], g.adj[o1] = adj0, adj1
	delete(g.adj, node)

	for i, e := range g.edges {
		if e == e0 || e == e1 {
			g.edges[i] = merged
		}
	}
	seen := map[*graphEdge]bool{}
	var unique []*graphEdge
	for _, e := range g.edges {
		if !seen[e] {
			seen[e] = true
			unique = append(unique, e)
		}
	}
	g.edges = unique
}

// walkLoops traverses the network into closed loops. At each node it takes
// the edge that continues most straight ahead (smallest turn), which yields
// the correct separate outer and hole contours even at degree-4 crossings of
// collinear edge pairs.
func (g *stitchedGraph) walkLoops() []*Loop {
	used := map[*graphEdge]bool{}
	var loops []*Loop
	for {
		var start *graphEdge
		for _, e := range g.edges {
			if !used[e] {
				start = e
				break
			}
		}
		if start == nil {
			return loops
		}

		var nodePath []int
		var edgePath []*graphEdge
		cur := start
		node := cur.a
		origin := node
		for {
			used[cur] = true
			edgePath = append(edgePath, cur)
			nodePath = append(nodePath, node)
			node = cur.other(node)
			if node == origin {
				break
			}
			next := g.pickNext(node, cur, used)
			if next == nil {
				break // open network; closure validation flags it
			}
			cur = next
		}

		nodes := make([]geom.Vec2, len(nodePath))
		ids := make([]int, len(nodePath))
		var segs []Seg3
		for i, id := range nodePath {
			nodes[i] = g.snapper.pts[id].pos
			ids[i] = id
		}
		for _, e := range edgePath {
			segs = append(segs, e.segs...)
		}
		loops = append(loops, &Loop{Nodes: nodes, NodeIDs: ids, Segments3D: segs})
	}
}

// pickNext chooses the unused edge leaving node, excluding the edge just
// traversed. Degree-2 nodes take their only continuation; junctions use the
// smallest turning angle so collinear chains stay on their own contour.
func (g *stitchedGraph) pickNext(node int, incoming *graphEdge, used map[*graphEdge]bool) *graphEdge {
	var candidates []*graphEdge
	for _, e := range g.adj[node] {
		if e != incoming && !used[e] {
			candidates = append(candidates, e)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	best := candidates[0]
	bestTurn := math.Pi * 2
	dirIn := g.directedDir(node, incoming).Neg()
	for _, e := range candidates {
		turn := math.Acos(geom.Clamp(dirIn.Dot(g.directedDir(node, e)), -1, 1))
		if turn < bestTurn {
			bestTurn, best = turn, e
		}
	}
	return best
}
