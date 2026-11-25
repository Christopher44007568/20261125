package main

import "math"

type edgeInfo struct {
	Count  int
	Length float64
	A      int32
	B      int32
}

type transitionSpec struct {
	U        float64
	FineNV   int
	CoarseNV int
}

func edgeKey(a, b int32) [2]int32 {
	if a > b {
		a, b = b, a
	}
	return [2]int32{a, b}
}

func isDomainEdge(mesh *Mesh, a, b int32) bool {
	pa := mesh.Params[a]
	pb := mesh.Params[b]
	const eps = 1e-9
	onU := (pa.U < eps || pa.U > 1-eps) && (pb.U < eps || pb.U > 1-eps)
	onV := (pa.V < eps || pa.V > 1-eps) && (pb.V < eps || pb.V > 1-eps)
	return onU || onV
}

func measureCracks(surface *Surface, mesh *Mesh, grids GridSet) CrackMetrics {
	transitions := []transitionSpec{
		{U: 30.0 / 96.0, FineNV: grids.NearV, CoarseNV: grids.MidV},
		{U: 62.0 / 96.0, FineNV: grids.MidV, CoarseNV: grids.FarV},
	}

	metrics := CrackMetrics{TransitionCount: len(transitions)}
	for _, transition := range transitions {
		fineLength := 0.0
		for j := 0; j < transition.FineNV; j++ {
			a := surface.Point(transition.U, float64(j)/float64(transition.FineNV))
			b := surface.Point(transition.U, float64(j+1)/float64(transition.FineNV))
			fineLength += distance(a, b)
		}

		coarseLength := 0.0
		for j := 0; j < transition.CoarseNV; j++ {
			v0 := float64(j) / float64(transition.CoarseNV)
			v1 := float64(j+1) / float64(transition.CoarseNV)
			a := surface.Point(transition.U, v0)
			b := surface.Point(transition.U, v1)
			coarseLength += distance(a, b)

			midV := (v0 + v1) / 2
			surfaceMid := surface.Point(transition.U, midV)
			chordMid := scale(add(a, b), 0.5)
			opening := distance(surfaceMid, chordMid)
			metrics.NaiveMaxTransitionOpeningMM = math.Max(metrics.NaiveMaxTransitionOpeningMM, opening)
		}

		metrics.LODBoundaryLengthMM += fineLength
		metrics.NaiveCrackCenterlineLengthMM += fineLength
		metrics.NaiveOpenEdgeLengthMM += fineLength + coarseLength
	}
	edges := make(map[[2]int32]edgeInfo)
	for _, triangle := range mesh.Tris {
		for edgeIndex := 0; edgeIndex < 3; edgeIndex++ {
			a := triangle.Vertices[edgeIndex]
			b := triangle.Vertices[(edgeIndex+1)%3]
			key := edgeKey(a, b)
			info := edges[key]
			info.Count++
			info.A = a
			info.B = b
			info.Length = distance(mesh.Vertices[a], mesh.Vertices[b])
			edges[key] = info
		}
	}

	for _, info := range edges {
		if info.Count == 2 {
			continue
		}
		if isDomainEdge(mesh, info.A, info.B) {
			continue
		}
		metrics.UnmatchedStitchedEdgeCount++
		metrics.StitchedOpenEdgeLengthMM += info.Length
	}
	metrics.CrossLODCrackLengthMM = metrics.StitchedOpenEdgeLengthMM
	return metrics
}
