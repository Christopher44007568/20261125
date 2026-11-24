package eyebw

import "fmt"

func highResolutionConfig() FrameConfig {
	cfg := DefaultConfig()
	cfg.Width = 3200
	cfg.Height = 3200
	cfg.MSAASamples = 4
	cfg.Budget.FrameBytes = 512 * MebiByte
	return cfg
}

func DefaultScenarios() []Scenario {
	balanced := DefaultConfig()
	balanced.Budget.FrameBytes = 512 * MebiByte

	ultra := DefaultConfig()
	ultra.Width = 4096
	ultra.Height = 4096
	ultra.MSAASamples = 8
	ultra.Budget.FrameBytes = 512 * MebiByte
	ultra.CacheCapacityBytes = 64 * MebiByte

	trap := highResolutionConfig()
	trap.SingleEye.AggressiveCenterReuse = true
	trap.Budget.FrameBytes = 1 * GibiByte

	return []Scenario{
		{
			Name:        "balanced_2400_msaa4_dynamic_fovea",
			Description: "Typical high-quality eye resolution, 4x MSAA, moving gaze.",
			Config:      balanced,
			Frames:      12,
			DynamicGaze: true,
		},
		{
			Name:        "highres_3200_msaa4_dynamic_fovea",
			Description: "High-resolution per-eye buffers with 4x MSAA and moving gaze.",
			Config:      highResolutionConfig(),
			Frames:      16,
			DynamicGaze: true,
		},
		{
			Name:        "ultra_4096_msaa8_dynamic_fovea_budget",
			Description: "Stress case for oversized eye buffers and 8x MSAA.",
			Config:      ultra,
			Frames:      16,
			DynamicGaze: true,
		},
		{
			Name:                   "quality_trap_aggressive_center_reuse",
			Description:            "Deliberately reuses center pixels across eyes; quality gate must fail.",
			Config:                 trap,
			Frames:                 8,
			DynamicGaze:            true,
			ExpectQualityFailure:   true,
			ExpectedFailedPipeline: PipelineSingleReuse,
		},
	}
}

// MatrixScenarios covers resolution, MSAA and foveation toggles. It is used by
// tests and can also be run by tooling that wants the broad parameter sweep.
func MatrixScenarios(framesEach int) []Scenario {
	if framesEach <= 0 {
		framesEach = 4
	}
	resolutions := []Resolution{
		{Width: 2064, Height: 2208},
		{Width: 3200, Height: 3200},
		{Width: 4096, Height: 4096},
	}
	msaaLevels := []int{1, 2, 4, 8}
	out := make([]Scenario, 0, len(resolutions)*len(msaaLevels)*2)
	for _, resolution := range resolutions {
		for _, msaa := range msaaLevels {
			for _, foveated := range []bool{false, true} {
				cfg := DefaultConfig()
				cfg.Width = resolution.Width
				cfg.Height = resolution.Height
				cfg.MSAASamples = msaa
				cfg.Foveation.Enabled = foveated
				cfg.Budget.FrameBytes = 0
				name := fmt.Sprintf("matrix_%dx%d_msaa%d_fovea%t",
					resolution.Width, resolution.Height, msaa, foveated)
				out = append(out, Scenario{
					Name:        name,
					Description: "Resolution/MSAA/foveation matrix case.",
					Config:      cfg,
					Frames:      framesEach,
					DynamicGaze: foveated,
				})
			}
		}
	}
	return out
}
