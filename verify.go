package main

import (
	"math"
)

const (
	positionToleranceMM = 1e-9
	tangentToleranceDeg = 1e-7
	normalToleranceDeg  = 1e-7
)

type ModelInfo struct {
	WedgeIncludedAngleDeg float64 `json:"wedge_included_angle_deg"`
	LengthMM              float64 `json:"length_mm"`
	FaceWidthMM           float64 `json:"face_width_mm"`
	MeshU                 int     `json:"mesh_u"`
	MeshV                 int     `json:"mesh_v"`
}

type RadiusInfo struct {
	StartMM float64 `json:"start_mm"`
	EndMM   float64 `json:"end_mm"`
	MinMM   float64 `json:"min_mm"`
	MaxMM   float64 `json:"max_mm"`
}

type SeamMetrics struct {
	Name                     string  `json:"name"`
	Samples                  int     `json:"samples"`
	MaxPositionErrorMM       float64 `json:"max_position_error_mm"`
	MaxTangentAngleDeg       float64 `json:"max_tangent_angle_deg"`
	MaxNormalAngleDeg        float64 `json:"max_normal_angle_deg"`
	EndpointPositionErrorMM  float64 `json:"endpoint_max_position_error_mm"`
	EndpointTangentAngleDeg  float64 `json:"endpoint_max_tangent_angle_deg"`
	EndpointNormalAngleDeg   float64 `json:"endpoint_max_normal_angle_deg"`
	MaxCurvatureJumpPerMM    float64 `json:"max_curvature_jump_1_per_mm"`
	PositionContinuous       bool    `json:"position_continuous"`
	TangentContinuous        bool    `json:"tangent_continuous"`
	NormalOrientationValid   bool    `json:"normal_orientation_valid"`
	EndpointContinuityPassed bool    `json:"endpoint_continuity_passed"`
}

type AcceptanceInfo struct {
	PositionToleranceMM   float64 `json:"position_tolerance_mm"`
	TangentToleranceDeg   float64 `json:"tangent_tolerance_deg"`
	NormalToleranceDeg    float64 `json:"normal_tolerance_deg"`
	ContinuityClass       string  `json:"continuity_class"`
	CurvatureNote         string  `json:"curvature_note"`
	FilletParentSeamsPass bool    `json:"fillet_parent_seams_pass"`
}

type CurvatureInfo struct {
	FilletMinPerMM float64 `json:"fillet_min_1_per_mm"`
	FilletMaxPerMM float64 `json:"fillet_max_1_per_mm"`
	DisplayMax     float64 `json:"display_max_1_per_mm"`
}

type Report struct {
	Units      string         `json:"units"`
	Model      ModelInfo      `json:"model"`
	Radius     RadiusInfo     `json:"radius"`
	Curvature  CurvatureInfo  `json:"curvature"`
	Seams      []SeamMetrics  `json:"seams"`
	Acceptance AcceptanceInfo `json:"acceptance"`
}

func angleBetweenVectorsDeg(a, b Vec3) float64 {
	ua, ub := a.Unit(), b.Unit()
	if ua.Len() == 0 || ub.Len() == 0 {
		return 90
	}
	return math.Atan2(ua.Cross(ub).Len(), ua.Dot(ub)) * radToDeg
}

// angleBetweenLinesDeg ignores orientation and reports the acute angle between
// two tangent directions, which is the correct G1 test for opposite parameter
// directions.
func angleBetweenLinesDeg(a, b Vec3) float64 {
	ua, ub := a.Unit(), b.Unit()
	if ua.Len() == 0 || ub.Len() == 0 {
		return 90
	}
	return math.Atan2(ua.Cross(ub).Len(), math.Abs(ua.Dot(ub))) * radToDeg
}

func checkSeam(cfg Config, name string, fillet FilletSurface, upper bool, samples int) SeamMetrics {
	parent := NewParentSurface(cfg, 1)
	uFillet := 0.0
	if !upper {
		parent = NewParentSurface(cfg, -1)
		uFillet = 1
	}

	m := SeamMetrics{
		Name:                   name,
		Samples:                samples,
		NormalOrientationValid: true,
	}
	for i := 0; i < samples; i++ {
		v := float64(i) / float64(samples-1)
		fs := fillet.Eval(uFillet, v)
		ps := parent.Eval(0, v)

		positionError := fs.P.Sub(ps.P).Len()
		tangentError := math.Max(angleBetweenLinesDeg(fs.Pu, ps.Pu), angleBetweenLinesDeg(fs.Pv, ps.Pv))
		normalError := angleBetweenVectorsDeg(fs.N, ps.N)
		m.MaxPositionErrorMM = math.Max(m.MaxPositionErrorMM, positionError)
		m.MaxTangentAngleDeg = math.Max(m.MaxTangentAngleDeg, tangentError)
		m.MaxNormalAngleDeg = math.Max(m.MaxNormalAngleDeg, normalError)
		fk := fs.Curvatures().MaxAbs()
		pk := ps.Curvatures().MaxAbs()
		m.MaxCurvatureJumpPerMM = math.Max(m.MaxCurvatureJumpPerMM, math.Abs(fk-pk))

		if i == 0 || i == samples-1 {
			m.EndpointPositionErrorMM = math.Max(m.EndpointPositionErrorMM, positionError)
			m.EndpointTangentAngleDeg = math.Max(m.EndpointTangentAngleDeg, tangentError)
			m.EndpointNormalAngleDeg = math.Max(m.EndpointNormalAngleDeg, normalError)
		}
		if fs.N.Dot(fs.Pv.Cross(fs.Pu)) <= 0 {
			m.NormalOrientationValid = false
		}
	}
	m.PositionContinuous = m.MaxPositionErrorMM <= positionToleranceMM
	m.TangentContinuous = m.MaxTangentAngleDeg <= tangentToleranceDeg
	m.NormalOrientationValid = m.NormalOrientationValid && m.MaxNormalAngleDeg <= normalToleranceDeg
	m.EndpointContinuityPassed = m.EndpointPositionErrorMM <= positionToleranceMM &&
		m.EndpointTangentAngleDeg <= tangentToleranceDeg &&
		m.EndpointNormalAngleDeg <= normalToleranceDeg
	return m
}

func buildReport(cfg Config) Report {
	fillet := NewFilletSurface(cfg)
	seams := []SeamMetrics{
		checkSeam(cfg, "upper_contact", fillet, true, 401),
		checkSeam(cfg, "lower_contact", fillet, false, 401),
	}

	minCurv, maxCurv := math.Inf(1), 0.0
	for j := 0; j <= 24; j++ {
		v := float64(j) / 24
		for i := 0; i <= 32; i++ {
			u := float64(i) / 32
			k := fillet.Eval(u, v).Curvatures().MaxAbs()
			minCurv = math.Min(minCurv, k)
			maxCurv = math.Max(maxCurv, k)
		}
	}

	pass := true
	for _, seam := range seams {
		pass = pass && seam.PositionContinuous && seam.TangentContinuous &&
			seam.NormalOrientationValid && seam.EndpointContinuityPassed
	}

	return Report{
		Units: "mm",
		Model: ModelInfo{
			WedgeIncludedAngleDeg: 2 * cfg.HalfAngle,
			LengthMM:              cfg.Length,
			FaceWidthMM:           cfg.FaceWidth,
			MeshU:                 cfg.SamplesU,
			MeshV:                 cfg.SamplesV,
		},
		Radius: RadiusInfo{
			StartMM: cfg.Radius(0),
			EndMM:   cfg.Radius(1),
			MinMM:   math.Min(cfg.RadiusStart, cfg.RadiusEnd),
			MaxMM:   math.Max(cfg.RadiusStart, cfg.RadiusEnd),
		},
		Curvature: CurvatureInfo{
			FilletMinPerMM: minCurv,
			FilletMaxPerMM: maxCurv,
			DisplayMax:     niceCeil(maxCurv * 1.08),
		},
		Seams: seams,
		Acceptance: AcceptanceInfo{
			PositionToleranceMM:   positionToleranceMM,
			TangentToleranceDeg:   tangentToleranceDeg,
			NormalToleranceDeg:    normalToleranceDeg,
			ContinuityClass:       "G0 positions + G1 tangent planes (normals aligned)",
			CurvatureNote:         "A curvature jump at the planar-parent seams is expected and is reported for diagnosis, not used as a G1 failure.",
			FilletParentSeamsPass: pass,
		},
	}
}

func niceCeil(x float64) float64 {
	if x <= 0 {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(x)))
	n := math.Ceil(x/mag*4) / 4
	return n * mag
}
