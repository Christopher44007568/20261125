package decallab

import "runtime"

const SchemaVersion = "stereodecal/v1"

// Run executes the deterministic validation lab. The returned report always
// contains the measured negative controls; Pass describes whether the intended
// invariants and detection checks held.
func Run() *Report {
	scenarios := BuildScenarios()
	reports := make([]ScenarioReport, 0, len(scenarios))
	for _, scenario := range scenarios {
		reports = append(reports, runScenario(scenario))
	}

	checks := evaluateChecks(reports)
	passed := 0
	failed := 0
	requiredPass := true
	for _, check := range checks {
		if check.Pass {
			passed++
		} else {
			failed++
			if check.Required {
				requiredPass = false
			}
		}
	}

	report := &Report{
		SchemaVersion: SchemaVersion,
		GoVersion:     runtime.Version(),
		Question:      "Do oriented decals, contact shadows, and detail normals retain world scale in low-sample stereo regions?",
		Scenarios:     reports,
		Checks:        checks,
		Pass:          requiredPass,
		Summary: ReportSummary{
			ScenarioCount:    len(reports),
			CheckCount:       len(checks),
			PassedCheckCount: passed,
			FailedCheckCount: failed,
		},
	}
	report.Conclusion = buildConclusion(report)
	fillSummaryExtrema(report)
	return report
}

func buildConclusion(report *Report) string {
	if report.Pass {
		return "PASS: authoritative dimensions are meters and correct shared-world reprojection preserves scale; direct monocular tangent reuse is measured and rejected."
	}
	return "FAIL: at least one required world-scale, stereo-reprojection, LOD, or negative-control check failed."
}

func fillSummaryExtrema(report *Report) {
	summary := &report.Summary
	for _, scenario := range report.Scenarios {
		summary.ProbeCount += scenario.Stats.ProbeCount
		summary.ObservationCount += len(scenario.Observations)
		summary.ShadowEdgeCount += len(scenario.ShadowEdges)
		summary.MaxObliquityDeg = maxFloat(summary.MaxObliquityDeg, scenario.Stats.MaxViewObliquityDeg)
		summary.MaxBinocularPixelDx = maxFloat(summary.MaxBinocularPixelDx, scenario.Stats.MaxPixelDisparityPx)
		summary.MaxLightingDiff = maxFloat(summary.MaxLightingDiff, scenario.Stats.MaxLightingDifference)
		for _, copyComparison := range scenario.CopyComparisons {
			summary.MaxCopyNormalDeg = maxFloat(summary.MaxCopyNormalDeg, copyComparison.NormalCopy.TangentReuseErrorDeg)
			summary.MaxCopyDecalErrorM = maxFloat(summary.MaxCopyDecalErrorM, copyComparison.DecalCopy.ReprojectionM)
		}
		for _, edge := range scenario.ShadowEdges {
			summary.MaxShadowEdgeBiasM = maxFloat(summary.MaxShadowEdgeBiasM, edge.EdgeAbsBiasM)
		}
	}
}

func maxFloat(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

func surfaceInfo(surface Surface) SurfaceInfo {
	return SurfaceInfo{ID: surface.ID(), Kind: surface.Kind(), Material: surface.GetMaterial()}
}

func alphaForSample(material Material, decal Decal, pattern float64) float64 {
	alpha := material.Opacity * decal.Opacity
	if decal.Pattern == "arrow" || decal.Pattern == "stripe" {
		alpha *= 0.18 + 0.82*pattern
	}
	return clamp01(alpha)
}
