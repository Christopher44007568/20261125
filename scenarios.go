package decallab

type ProbeSpec struct {
	ID       string  `json:"id"`
	UOffsetM float64 `json:"u_offset_m"`
	VOffsetM float64 `json:"v_offset_m"`
}

type Scenario struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Tags           []string        `json:"tags"`
	Surface        Surface         `json:"-"`
	Decal          Decal           `json:"decal"`
	Detail         DetailNormal    `json:"detail"`
	Shadow         ContactShadow   `json:"shadow"`
	Rig            Rig             `json:"rig"`
	Lighting       LightingRig     `json:"lighting"`
	Probes         []ProbeSpec     `json:"probes"`
	Densities      []DensityConfig `json:"densities"`
	EdgeDirections []float64       `json:"edge_directions"`
	BacksideCheck  bool            `json:"backside_check"`
}

func (s Scenario) ProbePoint(probe ProbeSpec) Vec3 {
	directionVector := s.Decal.AxisU.Scale(probe.UOffsetM).Add(s.Decal.AxisV.Scale(probe.VOffsetM))
	distance := directionVector.Norm()
	if distance < 1e-9 {
		return s.Decal.Center
	}
	return PointOnSurfaceAtDistance(s.Surface, s.Decal.Center, directionVector.Normalize(), distance)
}

func defaultProbes() []ProbeSpec {
	return []ProbeSpec{
		{ID: "center", UOffsetM: 0, VOffsetM: 0},
		{ID: "u_directional_edge", UOffsetM: 0.132, VOffsetM: 0.018},
		{ID: "v_edge", UOffsetM: -0.035, VOffsetM: 0.071},
		{ID: "oblique_corner", UOffsetM: 0.118, VOffsetM: -0.060},
		{ID: "curvature_wrap", UOffsetM: -0.158, VOffsetM: 0.052},
	}
}

func defaultDensities(lowStride float64, lowMinSpacing, lowCell float64) []DensityConfig {
	return []DensityConfig{
		{
			Label:            "reference",
			PixelStride:      1,
			MinSpacingM:      0.0015,
			FallbackSpacingM: 0.004,
			SharedCellM:      0.004,
		},
		{
			Label:            "low_sample",
			PixelStride:      lowStride,
			MinSpacingM:      lowMinSpacing,
			FallbackSpacingM: lowMinSpacing * 1.4,
			SharedCellM:      lowCell,
		},
	}
}

func sharedWorldLight() LightingRig {
	return LightingRig{
		Mode: "shared_world_light",
		Shared: Light{
			ID:        "world_key",
			Direction: Vec3{X: -0.42, Y: 0.58, Z: 0.70}.Normalize(),
			Color:     RGB{R: 1.0, G: 0.96, B: 0.90},
			Intensity: 1.15,
			Shininess: 42,
		},
	}
}

func splitEyeLights() LightingRig {
	shared := Light{
		ID:        "unused_in_eye_specific_mode",
		Direction: Vec3{X: 0, Y: 1, Z: 1}.Normalize(),
		Color:     RGB{R: 1, G: 1, B: 1},
		Intensity: 0.8,
		Shininess: 32,
	}
	return LightingRig{
		Mode:   "eye_specific",
		Shared: shared,
		LeftOverride: Light{
			ID:        "left_warm_raking",
			Direction: Vec3{X: -0.82, Y: 0.24, Z: 0.52}.Normalize(),
			Color:     RGB{R: 1.0, G: 0.72, B: 0.48},
			Intensity: 1.35,
			Shininess: 64,
		},
		RightOverride: Light{
			ID:        "right_cool_opposed",
			Direction: Vec3{X: 0.74, Y: 0.48, Z: 0.46}.Normalize(),
			Color:     RGB{R: 0.58, G: 0.76, B: 1.0},
			Intensity: 1.05,
			Shininess: 28,
		},
	}
}

func BuildScenarios() []Scenario {
	return []Scenario{
		buildCurvedObliqueScenario(),
		buildTransparentObliqueScenario(),
		buildSplitLightLODScenario(),
		buildGrazingLowSampleScenario(),
	}
}
