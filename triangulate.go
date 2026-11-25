package main

import (
	"fmt"
	"math"
	"sort"
)

type Triangle [3]int

type Mesh2 struct {
	Vertices  []Vec2     `json:"vertices"`
	Triangles []Triangle `json:"triangles"`
}

type InvalidTriangle struct {
	Index    int      `json:"index"`
	Triangle Triangle `json:"triangle"`
	Centroid Vec2     `json:"centroid"`
	Reasons  []string `json:"reasons"`
}

func TriangulateSimple(poly []Vec2) (*Mesh2, error) {
	poly = removeCollinear(dedupeAdjacent(poly, attachEps), attachEps)
	poly = ensureOrientation(poly, true)
	mesh := &Mesh2{Vertices: append([]Vec2(nil), poly...)}
	ring := make([]int, len(poly))
	for i := range ring {
		ring[i] = i
	}
	tris, err := earClipRing(mesh.Vertices, ring)
	if err != nil {
		return nil, err
	}
	mesh.Triangles = tris
	return mesh, nil
}

func TriangulateDomain(domain TrimDomain) (*Mesh2, error) {
	outer := removeCollinear(dedupeAdjacent(domain.Outer, attachEps), attachEps)
	outer = ensureOrientation(outer, true)
	mesh := &Mesh2{}
	ring := make([]int, len(outer))
	for i, p := range outer {
		ring[i] = mesh.internVertex(p)
	}

	holes := make([][]Vec2, 0, len(domain.Holes))
	for _, h := range domain.Holes {
		h = removeCollinear(dedupeAdjacent(h, attachEps), attachEps)
		h = ensureOrientation(h, false)
		if len(h) >= 3 {
			holes = append(holes, h)
		}
	}
	holes = mergeTouchingHoleRings(holes)
	sort.Slice(holes, func(i, j int) bool {
		return rightmostX(holes[i]) > rightmostX(holes[j])
	})

	for hi, hole := range holes {
		holeIDs := make([]int, len(hole))
		for i, p := range hole {
			holeIDs[i] = mesh.internVertex(p)
		}
		var err error
		ring, err = bridgeHole(mesh, ring, holeIDs)
		if err != nil {
			return nil, fmt.Errorf("bridge hole %d: %w", hi, err)
		}
	}

	tris, err := earClipRing(mesh.Vertices, ring)
	if err != nil {
		return nil, err
	}
	mesh.Triangles = tris
	return mesh, nil
}

func mergeTouchingHoleRings(holes [][]Vec2) [][]Vec2 {
	merged := true
	for merged && len(holes) > 1 {
		merged = false
	outer:
		for i := 0; i < len(holes); i++ {
			for j := i + 1; j < len(holes); j++ {
				for ai, p := range holes[i] {
					for bj, q := range holes[j] {
						if !approx2(p, q, attachEps*4) {
							continue
						}
						a := rotatePolygon(holes[i], ai)
						b := rotatePolygon(holes[j], bj)
						combined := make([]Vec2, 0, len(a)+len(b))
						combined = append(combined, a...)
						combined = append(combined, b[0])
						combined = append(combined, b[1:]...)
						next := make([][]Vec2, 0, len(holes)-1)
						for k := range holes {
							if k != i && k != j {
								next = append(next, holes[k])
							}
						}
						next = append(next, combined)
						holes = next
						merged = true
						break outer
					}
				}
			}
		}
	}
	return holes
}

func rotatePolygon(poly []Vec2, start int) []Vec2 {
	n := len(poly)
	out := make([]Vec2, 0, n)
	for k := 0; k < n; k++ {
		out = append(out, poly[(start+k)%n])
	}
	return out
}

func (m *Mesh2) internVertex(p Vec2) int {
	for i, q := range m.Vertices {
		if approx2(p, q, attachEps) {
			return i
		}
	}
	m.Vertices = append(m.Vertices, p)
	return len(m.Vertices) - 1
}

func rightmostX(poly []Vec2) float64 {
	x := poly[0].X
	for _, p := range poly[1:] {
		x = math.Max(x, p.X)
	}
	return x
}

type bridgeHit struct {
	point       Vec2
	edgeIndex   int
	segmentT    float64
	existingID  int
	existingPos int
	hasExisting bool
}

func horizontalBridgeHit(verts []Vec2, ring []int, from Vec2) (bridgeHit, bool) {
	best := bridgeHit{}
	found := false
	bestX := math.Inf(1)
	n := len(ring)
	for i := 0; i < n; i++ {
		a := verts[ring[i]]
		b := verts[ring[(i+1)%n]]
		dy := b.Y - a.Y
		if math.Abs(dy) < bridgeEps {
			continue
		}
		t := (from.Y - a.Y) / dy
		if t < -pointEps || t > 1+pointEps {
			continue
		}
		t = math.Max(0, math.Min(1, t))
		x := a.X + (b.X-a.X)*t
		if x <= from.X+bridgeEps {
			continue
		}
		if x < bestX {
			bestX = x
			existing := -1
			pos := -1
			if t <= pointEps {
				existing = ring[i]
				pos = i
			} else if t >= 1-pointEps {
				existing = ring[(i+1)%n]
				pos = (i + 1) % n
			}
			best = bridgeHit{
				point:       v2(x, from.Y),
				edgeIndex:   i,
				segmentT:    t,
				existingID:  existing,
				existingPos: pos,
				hasExisting: existing >= 0,
			}
			found = true
		}
	}
	return best, found
}

func bridgeHole(mesh *Mesh2, ring []int, holeIDs []int) ([]int, error) {
	verts := mesh.Vertices
	hIndex := 0
	for i := 1; i < len(holeIDs); i++ {
		current := verts[holeIDs[i]]
		best := verts[holeIDs[hIndex]]
		if current.X > best.X || (math.Abs(current.X-best.X) <= pointEps && current.Y > best.Y) {
			hIndex = i
		}
	}
	from := verts[holeIDs[hIndex]]
	hit, ok := horizontalBridgeHit(mesh.Vertices, ring, from)
	if !ok {
		return nil, fmt.Errorf("no outer boundary hit to the right of %v", from)
	}

	pRingPos := -1
	pID := -1
	if hit.hasExisting {
		pID = hit.existingID
		pRingPos = hit.existingPos
	} else {
		pID = mesh.internVertex(hit.point)
		verts = mesh.Vertices
		insertAt := hit.edgeIndex + 1
		expanded := make([]int, 0, len(ring)+1)
		expanded = append(expanded, ring[:insertAt]...)
		expanded = append(expanded, pID)
		expanded = append(expanded, ring[insertAt:]...)
		ring = expanded
		pRingPos = insertAt
	}
	if pRingPos < 0 || pRingPos >= len(ring) || ring[pRingPos] != pID {
		return nil, fmt.Errorf("internal bridge position error")
	}

	rotated := make([]int, 0, len(holeIDs))
	rotated = append(rotated, holeIDs[hIndex:]...)
	rotated = append(rotated, holeIDs[:hIndex]...)

	merged := make([]int, 0, len(ring)+len(rotated)+2)
	merged = append(merged, ring[:pRingPos+1]...)
	merged = append(merged, rotated...)
	merged = append(merged, rotated[0])
	merged = append(merged, pID)
	merged = append(merged, ring[pRingPos+1:]...)
	if math.Abs(signedArea(positionsForRing(verts, merged))) < areaEps {
		return nil, fmt.Errorf("bridged polygon is degenerate")
	}
	return merged, nil
}

func positionsForRing(verts []Vec2, ring []int) []Vec2 {
	out := make([]Vec2, len(ring))
	for i, id := range ring {
		out[i] = verts[id]
	}
	return out
}

func pointInTriangleValues(p, a, b, c Vec2, eps float64) (inclusive bool, strict bool) {
	o1 := orient2(a, b, p)
	o2 := orient2(b, c, p)
	o3 := orient2(c, a, p)
	hasNeg := o1 < -eps || o2 < -eps || o3 < -eps
	hasPos := o1 > eps || o2 > eps || o3 > eps
	inclusive = !(hasNeg && hasPos)
	strict = o1 > eps && o2 > eps && o3 > eps
	return inclusive, strict
}

func earClipRing(verts []Vec2, ring []int) ([]Triangle, error) {
	n := len(ring)
	if n < 3 {
		return nil, fmt.Errorf("polygon ring has %d nodes", n)
	}
	prev := make([]int, n)
	next := make([]int, n)
	for i := 0; i < n; i++ {
		prev[i] = (i - 1 + n) % n
		next[i] = (i + 1) % n
	}
	active := make([]bool, n)
	for i := range active {
		active[i] = true
	}
	count := n
	var tris []Triangle

	isEar := func(node int, strictInteriorOnly bool) bool {
		aNode, bNode, cNode := prev[node], node, next[node]
		a := verts[ring[aNode]]
		b := verts[ring[node]]
		c := verts[ring[cNode]]
		if orient2(a, b, c) <= areaEps {
			return false
		}
		for eStart := 0; eStart < n; eStart++ {
			if !active[eStart] {
				continue
			}
			eEnd := next[eStart]
			if !active[eEnd] {
				continue
			}
			hit, ok := segmentIntersection(a, c, verts[ring[eStart]], verts[ring[eEnd]])
			if ok && hit.Kind == segProper {
				return false
			}
		}
		for pNode := 0; pNode < n; pNode++ {
			if !active[pNode] || pNode == aNode || pNode == bNode || pNode == cNode {
				continue
			}
			p := verts[ring[pNode]]
			inclusive, strict := pointInTriangleValues(p, a, b, c, pointEps)
			if strictInteriorOnly {
				if strict {
					return false
				}
			} else if inclusive {
				return false
			}
		}
		return true
	}

	steps := 0
	for count > 3 {
		candidate := -1
		bestScore := -1.0
		for i := 0; i < n; i++ {
			if !active[i] || !isEar(i, false) {
				continue
			}
			a := verts[ring[prev[i]]]
			b := verts[ring[i]]
			c := verts[ring[next[i]]]
			score := earQuality(a, b, c)
			if score > bestScore+1e-12 {
				bestScore = score
				candidate = i
			}
		}
		if candidate < 0 {
			bestScore = -1.0
			for i := 0; i < n; i++ {
				if !active[i] || !isEar(i, true) {
					continue
				}
				a := verts[ring[prev[i]]]
				b := verts[ring[i]]
				c := verts[ring[next[i]]]
				score := earQuality(a, b, c)
				if score > bestScore+1e-12 {
					bestScore = score
					candidate = i
				}
			}
		}
		if candidate < 0 {
			activePoints := make([]Vec2, 0, count)
			for i := 0; i < n; i++ {
				if active[i] {
					activePoints = append(activePoints, verts[ring[i]])
				}
			}
			return nil, fmt.Errorf("ear clipping stalled with %d active nodes near %v", count, activePoints[:min(5, len(activePoints))])
		}
		aNode, cNode := prev[candidate], next[candidate]
		tris = append(tris, Triangle{ring[aNode], ring[candidate], ring[cNode]})
		active[candidate] = false
		next[aNode] = cNode
		prev[cNode] = aNode
		count--
		steps++
		if steps > n+5 {
			return nil, fmt.Errorf("ear clipping exceeded expected step count")
		}
	}

	remaining := make([]int, 0, 3)
	for i := 0; i < n; i++ {
		if active[i] {
			remaining = append(remaining, i)
		}
	}
	if len(remaining) != 3 {
		return nil, fmt.Errorf("ear clipping ended with %d nodes", len(remaining))
	}
	a, b, c := remaining[0], remaining[1], remaining[2]
	tri := Triangle{ring[a], ring[b], ring[c]}
	pa, pb, pc := verts[tri[0]], verts[tri[1]], verts[tri[2]]
	if orient2(pa, pb, pc) < 0 {
		tri[1], tri[2] = tri[2], tri[1]
	}
	tris = append(tris, tri)
	return tris, nil
}

func earQuality(a, b, c Vec2) float64 {
	doubleArea := orient2(a, b, c)
	if doubleArea <= areaEps {
		return -1
	}
	sumSq := dist2(a, b) + dist2(b, c) + dist2(c, a)
	if sumSq < geomEps {
		return -1
	}
	score := 2 * math.Sqrt(3) * doubleArea / sumSq
	return math.Max(0, math.Min(1, score))
}

func trianglePoints(verts []Vec2, tri Triangle) (Vec2, Vec2, Vec2) {
	return verts[tri[0]], verts[tri[1]], verts[tri[2]]
}

func centroid(verts []Vec2, tri Triangle) Vec2 {
	a, b, c := trianglePoints(verts, tri)
	return v2((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3)
}

func appendReasonOnce(reasons []string, reason string) []string {
	for _, r := range reasons {
		if r == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func properEdgesCross(verts []Vec2, tri Triangle, poly []Vec2) bool {
	a, b, c := trianglePoints(verts, tri)
	triEdges := [][2]Vec2{{a, b}, {b, c}, {c, a}}
	for _, te := range triEdges {
		for i := 0; i < len(poly); i++ {
			j := (i + 1) % len(poly)
			hit, ok := segmentIntersection(te[0], te[1], poly[i], poly[j])
			if ok && hit.Kind == segProper {
				return true
			}
		}
	}
	return false
}

func polygonVertexStrictInsideTriangle(poly []Vec2, a, b, c Vec2) bool {
	for _, p := range poly {
		_, strict := pointInTriangleValues(p, a, b, c, pointEps*10)
		if strict {
			return true
		}
	}
	return false
}

func ValidateTriangles(mesh *Mesh2, domain TrimDomain) []InvalidTriangle {
	var invalid []InvalidTriangle
	for ti, tri := range mesh.Triangles {
		a, b, c := trianglePoints(mesh.Vertices, tri)
		center := centroid(mesh.Vertices, tri)
		var reasons []string

		if !PointInPolygon(center, domain.Outer) {
			reasons = appendReasonOnce(reasons, "centroid_outside_outer")
		}
		for _, p := range []Vec2{a, b, c} {
			if !PointInPolygon(p, domain.Outer) {
				reasons = appendReasonOnce(reasons, "vertex_outside_outer")
			}
		}
		if properEdgesCross(mesh.Vertices, tri, domain.Outer) {
			reasons = appendReasonOnce(reasons, "crosses_outer_boundary")
		}
		if polygonVertexStrictInsideTriangle(domain.Outer, a, b, c) {
			reasons = appendReasonOnce(reasons, "outer_vertex_inside_triangle")
		}

		for hi, hole := range domain.Holes {
			if PointInPolygon(center, hole) {
				reasons = appendReasonOnce(reasons, fmt.Sprintf("centroid_inside_hole_%d", hi+1))
			}
			for _, p := range []Vec2{a, b, c} {
				if PointInPolygonStrict(p, hole) {
					reasons = appendReasonOnce(reasons, fmt.Sprintf("vertex_inside_hole_%d", hi+1))
				}
			}
			if polygonVertexStrictInsideTriangle(hole, a, b, c) {
				reasons = appendReasonOnce(reasons, fmt.Sprintf("covers_hole_%d_vertex", hi+1))
			}
			if properEdgesCross(mesh.Vertices, tri, hole) {
				reasons = appendReasonOnce(reasons, fmt.Sprintf("crosses_hole_%d", hi+1))
			}
		}
		if len(reasons) > 0 {
			invalid = append(invalid, InvalidTriangle{Index: ti, Triangle: tri, Centroid: center, Reasons: reasons})
		}
	}
	return invalid
}
