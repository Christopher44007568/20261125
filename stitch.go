package main

import (
	"fmt"
	"math"
	"sort"
)

type BoundaryChain struct {
	IDs []int
	Q   []float64
}

type SeamSample struct {
	Q float64
	P Vec3
}

type RefineInfo struct {
	OldTriangles   int
	NewTriangles   int
	NetAdded       int
	FanTriangles   int
	Segments       int
	SampleIndexByV map[int]int
}

type StitchResult struct {
	Mesh             *Mesh
	Samples          []SeamSample
	SeamIDs          []int
	SeamEdges        [][2]int
	RefineA          RefineInfo
	RefineB          RefineInfo
	Watertight       WatertightReport
	SharpEdges       []SharpEdge
	CornerNormals    []Vec3
	SharpNormalAngle float64
	NormalA          Vec3
	NormalB          Vec3
}

func UnifyBoundarySamples(cfg Config, chains ...BoundaryChain) []SeamSample {
	qs := []float64{0, 1}
	for _, ch := range chains {
		qs = append(qs, ch.Q...)
	}
	sort.Float64s(qs)

	unique := make([]float64, 0, len(qs))
	for _, q := range qs {
		q = clampFloat(q, 0, 1)
		if len(unique) == 0 || math.Abs(q-unique[len(unique)-1]) > 1e-10 {
			unique = append(unique, q)
		}
	}

	for iter := 0; iter < 10; iter++ {
		var extra []float64
		for i := 0; i+1 < len(unique); i++ {
			q0, q1 := unique[i], unique[i+1]
			if q1-q0 < 1e-5 {
				continue
			}
			qm := (q0 + q1) * 0.5
			p0 := SharedEdge(cfg, q0)
			p1 := SharedEdge(cfg, q1)
			pm := SharedEdge(cfg, qm)
			chord := p1.Sub(p0)
			den := chord.LenSqr()
			var err float64
			if den < 1e-20 {
				err = pm.Distance(p0)
			} else {
				t := clampFloat(pm.Sub(p0).Dot(chord)/den, 0, 1)
				err = pm.Distance(p0.Add(chord.Scale(t)))
			}
			if err > cfg.ChordTolMM {
				extra = append(extra, qm)
			}
		}
		if len(extra) == 0 {
			break
		}
		unique = append(unique, extra...)
		sort.Float64s(unique)
		merged := unique[:0]
		for _, q := range unique {
			if len(merged) == 0 || math.Abs(q-merged[len(merged)-1]) > 1e-10 {
				merged = append(merged, q)
			}
		}
		unique = merged
	}

	out := make([]SeamSample, 0, len(unique))
	for _, q := range unique {
		out = append(out, SeamSample{Q: q, P: SharedEdge(cfg, q)})
	}
	return out
}

func sampleIndexForQ(samples []SeamSample, q float64) int {
	best := -1
	bestDist := math.Inf(1)
	for i, s := range samples {
		d := math.Abs(s.Q - q)
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	if bestDist < 1e-8 {
		return best
	}
	return -1
}

func RefineBoundaryChain(in *Mesh, chain BoundaryChain, samples []SeamSample) (*Mesh, BoundaryChain, RefineInfo, error) {
	out := in.Clone()
	info := RefineInfo{OldTriangles: len(in.Tri), SampleIndexByV: map[int]int{}}
	oldUses := BuildEdgeUses(out.Tri)
	qByID := make(map[int]float64, len(chain.IDs))
	for i, id := range chain.IDs {
		qByID[id] = chain.Q[i]
		if idx := sampleIndexForQ(samples, chain.Q[i]); idx >= 0 {
			info.SampleIndexByV[id] = idx
		}
	}

	removed := make(map[int]bool)
	var replacementTris []Triangle
	newChain := BoundaryChain{}

	for e := 0; e+1 < len(chain.IDs); e++ {
		id0 := chain.IDs[e]
		q0 := chain.Q[e]
		if e == 0 {
			newChain.IDs = append(newChain.IDs, id0)
			newChain.Q = append(newChain.Q, q0)
		}
		id1 := chain.IDs[e+1]
		q1 := chain.Q[e+1]
		key := MakeEdgeKey(id0, id1)
		uses := oldUses[key]
		if len(uses) != 1 {
			return nil, BoundaryChain{}, info, fmt.Errorf("boundary edge %v has %d incident triangles, want 1", key, len(uses))
		}
		use := uses[0]
		oldTri := out.Tri[use.Triangle]
		removed[use.Triangle] = true

		opp := -1
		for _, vid := range [3]int{oldTri.A, oldTri.B, oldTri.C} {
			if vid != use.A && vid != use.B {
				opp = vid
			}
		}
		if opp < 0 {
			return nil, BoundaryChain{}, info, fmt.Errorf("cannot find opposite vertex for edge %v", key)
		}

		qStart, qEnd := qByID[use.A], qByID[use.B]
		var mids []SeamSample
		for _, s := range samples {
			if betweenOpen(s.Q, qStart, qEnd, 1e-10) {
				mids = append(mids, s)
			}
		}
		if qStart > qEnd {
			sort.Slice(mids, func(i, j int) bool { return mids[i].Q > mids[j].Q })
		} else {
			sort.Slice(mids, func(i, j int) bool { return mids[i].Q < mids[j].Q })
		}

		directedPath := []int{use.A}
		for _, s := range mids {
			vid := out.AddVertex(s.P)
			info.SampleIndexByV[vid] = sampleIndexForQ(samples, s.Q)
			directedPath = append(directedPath, vid)
		}
		directedPath = append(directedPath, use.B)

		for k := 0; k+1 < len(directedPath); k++ {
			replacementTris = append(replacementTris, Triangle{
				A:      directedPath[k],
				B:      directedPath[k+1],
				C:      opp,
				Group:  oldTri.Group,
				Filled: true,
			})
			info.FanTriangles++
		}

		chainPath := append([]int(nil), directedPath...)
		if chainPath[0] != id0 {
			for i, j := 0, len(chainPath)-1; i < j; i, j = i+1, j-1 {
				chainPath[i], chainPath[j] = chainPath[j], chainPath[i]
			}
		}
		for _, vid := range chainPath[1:] {
			newChain.IDs = append(newChain.IDs, vid)
			if idx, ok := info.SampleIndexByV[vid]; ok {
				newChain.Q = append(newChain.Q, samples[idx].Q)
			} else if vid == id1 {
				newChain.Q = append(newChain.Q, q1)
			} else {
				return nil, BoundaryChain{}, info, fmt.Errorf("missing q for refined vertex %d", vid)
			}
		}
	}

	kept := make([]Triangle, 0, len(out.Tri)-len(removed)+len(replacementTris))
	for ti, tri := range out.Tri {
		if !removed[ti] {
			kept = append(kept, tri)
		}
	}
	kept = append(kept, replacementTris...)
	out.Tri = kept

	info.NewTriangles = len(out.Tri)
	info.NetAdded = info.NewTriangles - info.OldTriangles
	info.Segments = len(newChain.IDs) - 1
	return out, newChain, info, nil
}

func betweenOpen(x, a, b, eps float64) bool {
	lo, hi := a, b
	if lo > hi {
		lo, hi = hi, lo
	}
	return x > lo+eps && x < hi-eps
}
