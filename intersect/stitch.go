package intersect

import (
	"math"

	"ruledintersect/geom"
)

type edgeRef struct {
	seg  int
	node int
}

// stitchSegments welds endpoints and traces chains through the resulting
// graph. Directional preference at a branch preserves the smoothest
// continuation; degree-one endpoints are used as seed nodes first.
func stitchSegments(segs []Segment, tolerance float64) []Polyline {
	if len(segs) == 0 {
		return nil
	}
	nodes := make([]geom.Vec3, 2*len(segs))
	parent := make([]int, len(nodes))
	for i := range nodes {
		parent[i] = i
	}
	for i, s := range segs {
		nodes[2*i], nodes[2*i+1] = s.A, s.B
	}
	var find func(int) int
	find = func(x int) int {
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
	tol2 := tolerance * tolerance
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[i].Sub(nodes[j]).Len2() <= tol2 {
				union(i, j)
			}
		}
	}

	rootIDs := map[int]int{}
	nodeAt := make([]int, len(nodes))
	posSum := map[int]geom.Vec3{}
	count := map[int]int{}
	for i, p := range nodes {
		r := find(i)
		id, ok := rootIDs[r]
		if !ok {
			id = len(rootIDs)
			rootIDs[r] = id
		}
		nodeAt[i] = id
		posSum[id] = posSum[id].Add(p)
		count[id]++
	}
	pos := make([]geom.Vec3, len(rootIDs))
	for id, p := range posSum {
		pos[id] = p.Div(float64(count[id]))
	}

	adj := make([][]edgeRef, len(pos))
	valid := make([]bool, len(segs))
	for i := range segs {
		a, b := nodeAt[2*i], nodeAt[2*i+1]
		if a == b {
			continue
		}
		valid[i] = true
		adj[a] = append(adj[a], edgeRef{seg: i, node: a})
		adj[b] = append(adj[b], edgeRef{seg: i, node: b})
	}
	visited := make([]bool, len(segs))
	var curves []Polyline

	order := make([]int, 0, len(segs))
	for i := range segs {
		if !valid[i] {
			continue
		}
		a, b := nodeAt[2*i], nodeAt[2*i+1]
		if len(adj[a]) == 1 || len(adj[b]) == 1 {
			order = append(order, i)
		}
	}
	for i := range segs {
		if valid[i] {
			order = append(order, i)
		}
	}

	for _, seed := range order {
		if visited[seed] || !valid[seed] {
			continue
		}
		a, b := nodeAt[2*seed], nodeAt[2*seed+1]
		visited[seed] = true
		points := []geom.Vec3{pos[a], pos[b]}
		chainNodes := []int{a, b}
		statuses := []SegmentStatus{segs[seed].Status}

		next := b
		for {
			incoming := pos[next].Sub(points[len(points)-2]).Unit()
			ref, ok := chooseNext(adj[next], visited, nodeAt, pos, incoming, false)
			if !ok {
				break
			}
			visited[ref.seg] = true
			other := otherNode(ref, nodeAt)
			points = append(points, pos[other])
			chainNodes = append(chainNodes, other)
			statuses = append(statuses, segs[ref.seg].Status)
			next = other
		}

		front := a
		for {
			outgoing := pos[chainNodes[1]].Sub(pos[front]).Unit()
			ref, ok := chooseNext(adj[front], visited, nodeAt, pos, outgoing, true)
			if !ok {
				break
			}
			visited[ref.seg] = true
			other := otherNode(ref, nodeAt)
			points = append([]geom.Vec3{pos[other]}, points...)
			chainNodes = append([]int{other}, chainNodes...)
			statuses = append([]SegmentStatus{segs[ref.seg].Status}, statuses...)
			front = other
		}
		if polylineLength(points) > tolerance {
			curves = append(curves, Polyline{Points: points, Statuses: statuses})
		}
	}
	return curves
}

func otherNode(ref edgeRef, nodeAt []int) int {
	return nodeAt[2*ref.seg] + nodeAt[2*ref.seg+1] - ref.node
}

func chooseNext(refs []edgeRef, visited []bool, nodeAt []int,
	pos []geom.Vec3, dir geom.Vec3, backward bool) (edgeRef, bool) {
	best, found := edgeRef{}, false
	bestScore := math.Inf(1)
	if !backward {
		bestScore = math.Inf(-1)
	}
	for _, ref := range refs {
		if visited[ref.seg] {
			continue
		}
		other := otherNode(ref, nodeAt)
		cand := pos[other].Sub(pos[ref.node])
		if cand.Len2() < 1e-24 {
			continue
		}
		score := cand.Unit().Dot(dir)
		better := score > bestScore
		if backward {
			better = score < bestScore
		}
		if better || !found {
			best, bestScore, found = ref, score, true
		}
	}
	return best, found
}

func polylineLength(points []geom.Vec3) float64 {
	var total float64
	for i := 1; i < len(points); i++ {
		total += points[i].Dist(points[i-1])
	}
	return total
}
