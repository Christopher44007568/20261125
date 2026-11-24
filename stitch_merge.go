package main

import "fmt"

func StitchPatches(cfg Config, asm PatchAssembly) (StitchResult, error) {
	samples := UnifyBoundarySamples(cfg, asm.ChainA, asm.ChainB)

	meshA, _, infoA, err := RefineBoundaryChain(asm.MeshA, asm.ChainA, samples)
	if err != nil {
		return StitchResult{}, fmt.Errorf("refine A: %w", err)
	}
	meshB, _, infoB, err := RefineBoundaryChain(asm.MeshB, asm.ChainB, samples)
	if err != nil {
		return StitchResult{}, fmt.Errorf("refine B: %w", err)
	}

	merged := &Mesh{Name: "stitched-watertight-seam"}
	seamIDs := make([]int, len(samples))
	for i, s := range samples {
		seamIDs[i] = merged.AddVertex(s.P)
	}

	mapA := mergeVertexMap(meshA, infoA.SampleIndexByV, seamIDs, merged)
	mapB := mergeVertexMap(meshB, infoB.SampleIndexByV, seamIDs, merged)

	for _, t := range meshA.Tri {
		merged.Tri = append(merged.Tri, Triangle{
			A: mapA[t.A], B: mapA[t.B], C: mapA[t.C],
			Group: t.Group, Filled: t.Filled,
		})
	}
	for _, t := range meshB.Tri {
		merged.Tri = append(merged.Tri, Triangle{
			A: mapB[t.A], B: mapB[t.B], C: mapB[t.C],
			Group: t.Group, Filled: t.Filled,
		})
	}

	seamEdges := make([][2]int, 0, len(seamIDs)-1)
	for i := 0; i+1 < len(seamIDs); i++ {
		seamEdges = append(seamEdges, [2]int{seamIDs[i], seamIDs[i+1]})
	}

	report := CheckWatertight(merged, seamEdges)
	normals := ComputeCornerNormals(merged, cfg.HardAngleDeg)
	sharp := DetectSharpEdges(merged, seamEdges, cfg.HardAngleDeg)

	res := StitchResult{
		Mesh:          merged,
		Samples:       samples,
		SeamIDs:       seamIDs,
		SeamEdges:     seamEdges,
		RefineA:       infoA,
		RefineB:       infoB,
		Watertight:    report,
		SharpEdges:    sharp,
		CornerNormals: normals,
	}

	mid := seamIDs[len(seamIDs)/2]
	if angle, na, nb, ok := SeamNormalJump(merged, normals, mid); ok {
		res.SharpNormalAngle = angle
		res.NormalA = na
		res.NormalB = nb
	}

	if !report.SeamWatertight {
		return res, fmt.Errorf("seam is not watertight after stitch: %v", report.Issues)
	}
	return res, nil
}

func mergeVertexMap(src *Mesh, seamSampleByV map[int]int, seamIDs []int, dst *Mesh) map[int]int {
	idMap := make(map[int]int, len(src.Pos))
	for vid, p := range src.Pos {
		if sampleIdx, ok := seamSampleByV[vid]; ok && sampleIdx >= 0 {
			idMap[vid] = seamIDs[sampleIdx]
			continue
		}
		idMap[vid] = dst.AddVertex(p)
	}
	return idMap
}
