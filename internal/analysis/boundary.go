package analysis

import (
	"fmt"
	"math"
	"sort"

	"meshscope/internal/mesh"
)

func absolveUnknown(t topology, ids []int, regions *[]Region) {
	for pass := 0; pass < 8; pass++ {
		changed := false
		for i := range ids {
			if ids[i] >= 0 {
				continue
			}
			type choice struct {
				id    int
				score float64
			}
			scores := map[int]float64{}
			for _, nb := range t.adj[i] {
				if ids[nb.face] >= 0 {
					scores[ids[nb.face]] += math.Max(t.area[nb.face], 0.01) / (1 + nb.angle/40)
				}
			}
			if len(scores) == 0 {
				continue
			}
			var best int
			var bestScore float64
			for id, score := range scores {
				if score > bestScore {
					best, bestScore = id, score
				}
			}
			if best >= len(*regions) {
				continue
			}
			ids[i] = best
			(*regions)[best].Faces = append((*regions)[best].Faces, i)
			changed = true
		}
		if !changed {
			return
		}
	}
}

func finalizeRegions(t topology, ids []int, regions []Region) {
	for i := range regions {
		r := &regions[i]
		r.Area = areaSum(t, r.Faces)
		if r.Kind == mesh.Plane {
			r.Plane = fitPlane(t, r.Faces)
			r.Confidence = planeConfidence(t, r.Plane, r.Faces)
			continue
		}
		r.Cylinder = fitCylinder(t, r.Faces)
		if !r.Cylinder.valid {
			r.Confidence = 0.3
			continue
		}
		r.Axis = r.Cylinder.axis
		r.Radius = r.Cylinder.radius
		r.Span = r.Cylinder.span
		r.Signed = r.Cylinder.signed
		r.RadialRMS = r.Cylinder.radialRMS
		if r.Cylinder.span < 150 {
			r.Kind = mesh.Fillet
		} else if r.Cylinder.signed < -0.08 {
			r.Kind = mesh.Hole
		} else {
			r.Kind = mesh.Cylinder
		}
		r.Confidence = curvedConfidence(r.Cylinder)
	}
}

func findBoundaries(t topology, m *mesh.Mesh, regionIDs []int, regions []Region, faces []FaceResult) []Boundary {
	seen := map[mesh.EdgeKey]bool{}
	var out []Boundary
	for key, faceIDs := range t.edges {
		if len(faceIDs) != 2 || seen[key] {
			continue
		}
		seen[key] = true
		a, b := faceIDs[0], faceIDs[1]
		fa, fb := faces[a], faces[b]
		if fa.Kind == mesh.Unknown || fb.Kind == mesh.Unknown {
			continue
		}
		if fa.RegionID == fb.RegionID && fa.Kind == fb.Kind {
			continue
		}
		if fa.Kind == fb.Kind && fa.RegionID != fb.RegionID {
			small := len(regions[fa.RegionID].Faces)
			if len(regions[fb.RegionID].Faces) < small {
				small = len(regions[fb.RegionID].Faces)
			}
			if small < 5 {
				continue
			}
		}
		pa, pb := edgePositions(m.Tris[a], key)
		if pa.Dist(pb) < 1e-7 {
			continue
		}
		base := math.Min(fa.Confidence, fb.Confidence)
		dihedral := angleDeg(m.Tris[a].Normal, m.Tris[b].Normal)
		var conf float64
		if fa.Kind != fb.Kind {
			conf = base * (0.67 + 0.33*math.Min(1, dihedral/48))
		} else {
			conf = base * (0.42 + 0.22*math.Min(1, dihedral/55))
		}
		conf = math.Max(0.18, math.Min(0.99, conf))
		out = append(out, Boundary{
			A: pa, B: pb, KindA: fa.Kind, KindB: fb.Kind,
			N:       m.Tris[a].Normal.Add(m.Tris[b].Normal).Norm(),
			RegionA: fa.RegionID, RegionB: fb.RegionID,
			Confidence: conf, Length: pa.Dist(pb),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence == out[j].Confidence {
			return out[i].Length > out[j].Length
		}
		return out[i].Confidence < out[j].Confidence
	})
	_ = regionIDs
	_ = regions
	return out
}

func edgePositions(tri mesh.Triangle, key mesh.EdgeKey) (mesh.Vec3, mesh.Vec3) {
	for e := 0; e < 3; e++ {
		if mesh.MakeEdgeKey(tri.P[e], tri.P[(e+1)%3]) == key {
			return tri.P[e], tri.P[(e+1)%3]
		}
	}
	return tri.P[0], tri.P[1]
}

type boundaryGroup struct {
	kindA, kindB mesh.Feature
	a, b         int
	best         Boundary
	count        int
	totalLen     float64
}

func makeCallouts(bounds []Boundary) []Callout {
	gmap := map[string]*boundaryGroup{}
	for _, b := range bounds {
		ka, kb := b.KindA, b.KindB
		ra, rb := b.RegionA, b.RegionB
		if ra > rb || (ra == rb && ka > kb) {
			ka, kb, ra, rb = kb, ka, rb, ra
		}
		key := fmt.Sprintf("%d:%d:%d:%d", ka, kb, ra, rb)
		g := gmap[key]
		if g == nil {
			g = &boundaryGroup{kindA: ka, kindB: kb, a: ra, b: rb, best: b}
			gmap[key] = g
		}
		g.count++
		g.totalLen += b.Length
		if b.Length > g.best.Length || (b.Length == g.best.Length && b.Confidence < g.best.Confidence) {
			g.best = b
		}
	}
	var groups []*boundaryGroup
	for _, g := range gmap {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		pi := boundaryPairPriority(groups[i].kindA, groups[i].kindB)
		pj := boundaryPairPriority(groups[j].kindA, groups[j].kindB)
		if pi != pj {
			return pi < pj
		}
		if groups[i].totalLen != groups[j].totalLen {
			return groups[i].totalLen > groups[j].totalLen
		}
		return groups[i].best.Confidence < groups[j].best.Confidence
	})
	var out []Callout
	used := map[string]bool{}
	for _, g := range groups {
		pair := kindPair(g.kindA, g.kindB)
		if used[pair] {
			continue
		}
		used[pair] = true
		p := g.best.A.Add(g.best.B).Mul(0.5)
		label := featureLabel(g.kindA) + " / " + featureLabel(g.kindB)
		out = append(out, Callout{
			Position: p, N: g.best.N, KindA: g.kindA, KindB: g.kindB,
			Confidence: g.best.Confidence, Label: label,
		})
		if len(out) >= 7 {
			break
		}
	}
	return out
}

func errorProbes(m *mesh.Mesh, faces []FaceResult) []ErrorProbe {
	type item struct {
		face int
		abs  float64
	}
	var items []item
	for i, f := range faces {
		if f.Kind == mesh.Unknown {
			continue
		}
		items = append(items, item{face: i, abs: math.Abs(f.Deviation)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].abs > items[j].abs })
	var out []ErrorProbe
	for _, it := range items {
		if it.abs < 0.025 {
			continue
		}
		p := m.Tris[it.face].Center
		ok := true
		for _, e := range out {
			if p.Dist(e.Position) < 18 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		out = append(out, ErrorProbe{
			Position: p, N: m.Tris[it.face].Normal,
			Value: faces[it.face].Deviation, Kind: faces[it.face].Kind,
		})
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func computeBoundaryAverage(res *Result) {
	if len(res.Boundaries) == 0 {
		return
	}
	sum := 0.0
	for _, b := range res.Boundaries {
		sum += b.Confidence * math.Max(b.Length, 0.01)
	}
	length := 0.0
	for _, b := range res.Boundaries {
		length += math.Max(b.Length, 0.01)
	}
	if length > 0 {
		res.BoundaryAvg = sum / length
	}
}

func featureLabel(f mesh.Feature) string {
	switch f {
	case mesh.Plane:
		return "PLANE"
	case mesh.Cylinder:
		return "CYLINDER"
	case mesh.Hole:
		return "HOLE"
	case mesh.Fillet:
		return "FILLET"
	default:
		return "UNKNOWN"
	}
}

func kindPair(a, b mesh.Feature) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("%d-%d", a, b)
}

func boundaryPairPriority(a, b mesh.Feature) int {
	if a > b {
		a, b = b, a
	}
	switch {
	case a == mesh.Fillet && b == mesh.Fillet:
		return 0
	case a == mesh.Hole || b == mesh.Hole:
		return 1
	case a == mesh.Fillet || b == mesh.Fillet:
		return 2
	case a == mesh.Cylinder && b == mesh.Cylinder:
		return 3
	case a == mesh.Cylinder || b == mesh.Cylinder:
		return 4
	case a == mesh.Plane && b == mesh.Plane:
		return 5
	default:
		return 6
	}
}
