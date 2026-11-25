package main

import "math"

// ComputeMetrics derives FPS, headroom, relative load and the quality
// score from signals plus the effective capability state.
//
// Quality model (fitted to the audit baseline):
//
//	sharpness contribution = 45 + 55*samplingScale   (100 / 86.25 / 72.5)
//	monocular penalty      = -10
//	reprojection off       = -9
func ComputeMetrics(frame int, sig Signals, cfg Config, st State) Metrics {
	fps := 0.0
	if sig.FrameTimeMS > 0 {
		fps = 1000 / sig.FrameTimeMS
	}
	headroom := cfg.BudgetMS - sig.FrameTimeMS
	eyeFactor := float64(st.EyeCount) / 2.0
	load := eyeFactor * st.SamplingScale * st.SamplingScale
	quality := 45 + 55*st.SamplingScale
	if st.EyeCount < 2 {
		quality -= 10
	}
	if !st.Reprojection {
		quality -= 9
	}
	return Metrics{
		Frame:            frame,
		FrameTimeMS:      sig.FrameTimeMS,
		BudgetMS:         cfg.BudgetMS,
		FPS:              fps,
		BudgetHeadroomMS: headroom,
		EyeCount:         st.EyeCount,
		Stereo:           st.Stereo,
		SampleLevel:      st.SampleLevel,
		SamplingScale:    st.SamplingScale,
		Reprojection:     st.Reprojection,
		HistoryCapture:   st.History,
		Interaction:      st.Interaction,
		RelativeLoad:     load,
		QualityScore:     math.Round(quality*100) / 100,
	}
}
