package main

type Report struct {
	OK             bool              `json:"ok"`
	Config         Config            `json:"config"`
	CanonicalQ     []float64         `json:"canonical_q"`
	OriginalQA     []float64         `json:"original_a_q_forward"`
	OriginalQB     []float64         `json:"original_b_q_reversed"`
	Cracks         CrackSummary      `json:"cracks"`
	Refinement     RefinementReport  `json:"refinement"`
	Watertight     WatertightReport  `json:"watertight"`
	SharpEdgeCount int               `json:"sharp_edge_count"`
	SharpSeamEdge  *SharpEdge        `json:"sharpest_edge,omitempty"`
	SharpNormalDeg float64           `json:"sharp_normal_jump_deg"`
	NormalA        Vec3              `json:"normal_a"`
	NormalB        Vec3              `json:"normal_b"`
	Outputs        map[string]string `json:"outputs"`
}

type CrackSummary struct {
	SampleCount   int     `json:"sample_count"`
	MaxGapMM      float64 `json:"max_gap_mm"`
	MaxAtQ        float64 `json:"max_at_q"`
	MeanGapMM     float64 `json:"mean_gap_mm"`
	RMSGapMM      float64 `json:"rms_gap_mm"`
	AfterMaxGapMM float64 `json:"after_max_gap_mm"`
	AfterMeanMM   float64 `json:"after_mean_gap_mm"`
}

type RefinementReport struct {
	UnifiedSamples int `json:"unified_samples"`
	Segments       int `json:"seam_segments"`
	ANewTriangles  int `json:"a_new_triangles"`
	BNewTriangles  int `json:"b_new_triangles"`
	ANetAdded      int `json:"a_net_added"`
	BNetAdded      int `json:"b_net_added"`
	TotalNetAdded  int `json:"total_net_added"`
}

func BuildReport(cfg Config, asm PatchAssembly, cracks CrackReport, st StitchResult, outputs map[string]string) Report {
	var sharpest *SharpEdge
	if len(st.SharpEdges) > 0 {
		se := st.SharpEdges[0]
		sharpest = &se
	}
	return Report{
		OK:         st.Watertight.SeamWatertight && cracks.AfterMaxGapMM == 0,
		Config:     cfg,
		CanonicalQ: qFromSamples(st.Samples),
		OriginalQA: append([]float64(nil), asm.ChainA.Q...),
		OriginalQB: append([]float64(nil), asm.ChainB.Q...),
		Cracks: CrackSummary{
			SampleCount:   cracks.SampleCount,
			MaxGapMM:      cracks.MaxGapMM,
			MaxAtQ:        cracks.MaxAtQ,
			MeanGapMM:     cracks.MeanGapMM,
			RMSGapMM:      cracks.RMSGapMM,
			AfterMaxGapMM: cracks.AfterMaxGapMM,
			AfterMeanMM:   cracks.AfterMeanMM,
		},
		Refinement: RefinementReport{
			UnifiedSamples: len(st.Samples),
			Segments:       len(st.Samples) - 1,
			ANewTriangles:  st.RefineA.NewTriangles,
			BNewTriangles:  st.RefineB.NewTriangles,
			ANetAdded:      st.RefineA.NetAdded,
			BNetAdded:      st.RefineB.NetAdded,
			TotalNetAdded:  st.RefineA.NetAdded + st.RefineB.NetAdded,
		},
		Watertight:     st.Watertight,
		SharpEdgeCount: len(st.SharpEdges),
		SharpSeamEdge:  sharpest,
		SharpNormalDeg: st.SharpNormalAngle,
		NormalA:        st.NormalA,
		NormalB:        st.NormalB,
		Outputs:        outputs,
	}
}

func qFromSamples(samples []SeamSample) []float64 {
	out := make([]float64, len(samples))
	for i, s := range samples {
		out[i] = s.Q
	}
	return out
}
