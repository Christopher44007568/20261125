package analysis

import (
	"math"

	"meshscope/internal/mesh"
)

func absorbCurvedFringe(t topology, ids []int, regions []Region) {
	for i := range ids {
		if ids[i] >= 0 {
			continue
		}
		counts := map[int]int{}
		for _, nb := range t.adj[i] {
			id := ids[nb.face]
			if id >= 0 && regions[id].Kind != mesh.Plane && nb.angle < 35 {
				counts[id]++
			}
		}
		best, bestCount := -1, 0
		for id, count := range counts {
			if count > bestCount {
				best, bestCount = id, count
			}
		}
		if best >= 0 && bestCount >= 2 {
			ids[i] = best
			regions[best].Faces = append(regions[best].Faces, i)
		}
	}
	for i := range ids {
		if ids[i] >= 0 {
			continue
		}
		bestID, bestScore := -1, math.Inf(1)
		for _, nb := range t.adj[i] {
			id := ids[nb.face]
			if id < 0 || regions[id].Kind == mesh.Plane {
				continue
			}
			cf := regions[id].Cylinder
			if !cf.valid {
				continue
			}
			d := t.center[i].Sub(cf.center)
			d = d.Sub(cf.axis.Mul(d.Dot(cf.axis)))
			radial := d.Norm()
			residual := math.Abs(d.Len() - cf.radius)
			expected := radial
			if cf.signed < 0 {
				expected = radial.Mul(-1)
			}
			normalErr := angleDeg(t.normal[i], expected)
			score := residual/math.Max(0.15, cf.radius*0.04) + normalErr/14
			if residual < math.Max(0.28, cf.radius*0.055) && normalErr < 19 && score < bestScore {
				bestID, bestScore = id, score
			}
		}
		if bestID >= 0 {
			ids[i] = bestID
			regions[bestID].Faces = append(regions[bestID].Faces, i)
		}
	}
}

func mergePlaneRegions(t topology, ids []int, regions []Region) ([]Region, []int) {
	n := len(regions)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	find := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for _, faces := range t.edges {
		if len(faces) != 2 {
			continue
		}
		a, b := ids[faces[0]], ids[faces[1]]
		if a < 0 || b < 0 || a == b {
			continue
		}
		ra, rb := regions[a], regions[b]
		if ra.Kind != mesh.Plane || rb.Kind != mesh.Plane {
			continue
		}
		if angleDeg(ra.Plane.normal, rb.Plane.normal) > 1.35 {
			continue
		}
		ca := regionCenter(t, ra.Faces)
		cb := regionCenter(t, rb.Faces)
		if math.Abs(ra.Plane.normal.Dot(cb)-ra.Plane.offset) > 0.24 ||
			math.Abs(rb.Plane.normal.Dot(ca)-rb.Plane.offset) > 0.24 {
			continue
		}
		union(a, b)
	}
	roots := map[int]int{}
	var merged []Region
	for i := range regions {
		if find(i) != i || len(regions[i].Faces) == 0 {
			continue
		}
		if regions[i].Kind != mesh.Plane {
			roots[i] = len(merged)
			merged = append(merged, regions[i])
		}
	}
	for i := range regions {
		r := find(i)
		if r == i && regions[i].Kind == mesh.Plane && len(regions[i].Faces) > 0 {
			if _, ok := roots[r]; !ok {
				roots[r] = len(merged)
				kept := regions[i]
				kept.Faces = append([]int(nil), regions[i].Faces...)
				merged = append(merged, kept)
				regions[i].Faces = nil
			}
		}
	}
	for i := range regions {
		if regions[i].Kind != mesh.Plane {
			continue
		}
		root := find(i)
		if root == i {
			continue
		}
		dst := roots[root]
		merged[dst].Faces = append(merged[dst].Faces, regions[i].Faces...)
	}
	newIDs := make([]int, len(ids))
	for i, id := range ids {
		if id < 0 {
			newIDs[i] = -1
			continue
		}
		newIDs[i] = roots[find(id)]
	}
	for i := range merged {
		merged[i].ID = i
		if len(merged[i].Faces) == 0 {
			continue
		}
		if merged[i].Kind == mesh.Plane {
			merged[i].Plane = fitPlane(t, merged[i].Faces)
			merged[i].Confidence = planeConfidence(t, merged[i].Plane, merged[i].Faces)
		}
	}
	return merged, newIDs
}

func regionCenter(t topology, faces []int) mesh.Vec3 {
	var c mesh.Vec3
	var a float64
	for _, f := range faces {
		w := math.Max(t.area[f], 1e-6)
		c = c.Add(t.center[f].Mul(w))
		a += w
	}
	if a > 0 {
		c = c.Mul(1 / a)
	}
	return c
}
