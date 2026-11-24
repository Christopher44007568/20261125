package decallab

import (
	"fmt"
	"math"
)

func evaluateChecks(reports []ScenarioReport) []Check {
	checks := []Check{}
	checks = append(checks, checkRayHits(reports))
	checks = append(checks, checkDecalWorldDimensions(reports))
	checks = append(checks, checkDecalScaleAcrossDensities(reports))
	checks = append(checks, checkSharedCacheBounds(reports))
	checks = append(checks, checkShadowEdges(reports))
	checks = append(checks, checkNormalWavelength(reports))
	checks = append(checks, checkLODScale(reports))
	checks = append(checks, checkStereoPresence(reports))
	checks = append(checks, checkCopyNegativeControl(reports))
	checks = append(checks, checkRequiredConditions(reports)...)
	return checks
}

func makeCheck(id, title string, pass bool, observed, threshold float64, unit, evidence string) Check {
	return Check{
		ID: id, Title: title, Pass: pass, Required: true,
		Observed: observed, Threshold: threshold, Unit: unit, Evidence: evidence,
	}
}

func checkRayHits(reports []ScenarioReport) Check {
	total := 0
	failed := 0
	maxError := 0.0
	maxFrustum := 0.0
	for _, scenario := range reports {
		for _, observation := range scenario.Observations {
			if observation.Mode != ModeIndependent {
				continue
			}
			total++
			if !observation.RayHit {
				failed++
			}
			if !observation.PixelInFrustum {
				maxFrustum++
			}
			maxError = maxFloat(maxError, observation.HitErrorM)
		}
	}
	pass := failed == 0 && maxError < 1e-7
	return makeCheck("rays.reproject_to_probes",
		"每眼投影像素反射线命中同一世界探针", pass,
		maxError, 1e-7, "m",
		fmt.Sprintf("%d observations, %d misses, %d outside primary frustum", total, failed, int(maxFrustum)))
}

func checkDecalWorldDimensions(reports []ScenarioReport) Check {
	maxWidthError := 0.0
	maxHeightError := 0.0
	for _, scenario := range reports {
		for _, observation := range scenario.Observations {
			maxWidthError = maxFloat(maxWidthError, math.Abs(observation.Sample.Decal.ProjectedWidthM-scenario.Decal.WidthM))
			maxHeightError = maxFloat(maxHeightError, math.Abs(observation.Sample.Decal.ProjectedHeightM-scenario.Decal.HeightM))
		}
	}
	observed := maxFloat(maxWidthError, maxHeightError)
	return makeCheck("decal.nominal_world_dimensions",
		"贴花宽高始终以米保存，不随眼别/密度/LOD改变", observed < 1e-12,
		observed, 1e-12, "m", "maximum nominal dimension reconstruction error")
}

func checkDecalScaleAcrossDensities(reports []ScenarioReport) Check {
	maxDifference := 0.0
	comparisons := 0
	for _, scenario := range reports {
		byKey := make(map[string][]ObservationRecord)
		for _, observation := range scenario.Observations {
			if observation.Mode == ModeIndependent {
				key := observationKey(observation.ProbeID, observation.Eye)
				byKey[key] = append(byKey[key], observation)
			}
		}
		for _, observations := range byKey {
			if len(observations) < 2 {
				continue
			}
			reference := observations[0]
			for _, candidate := range observations[1:] {
				maxDifference = maxFloat(maxDifference,
					math.Abs(percentError(candidate.Sample.Decal.SurfaceArcWidthM, reference.Sample.Decal.SurfaceArcWidthM)))
				maxDifference = maxFloat(maxDifference,
					math.Abs(percentError(candidate.Sample.Decal.SurfaceArcHeightM, reference.Sample.Decal.SurfaceArcHeightM)))
				comparisons++
			}
		}
	}
	return makeCheck("decal.scale_invariant_to_sampling_density",
		"高/低采样密度在同一世界点测得相同曲面贴花尺度", maxDifference < 1e-7,
		maxDifference, 1e-7, "percent", fmt.Sprintf("%d density comparisons", comparisons))
}

func checkSharedCacheBounds(reports []ScenarioReport) Check {
	maxRatio := 0.0
	maxPointError := 0.0
	evidence := "no shared caches"
	for _, scenario := range reports {
		for _, cache := range scenario.Caches {
			cellBound := math.Sqrt(3) * cache.CellSizeM
			ratio := cache.MaxSharedPointErrM / cellBound
			if ratio > maxRatio {
				maxRatio = ratio
				maxPointError = cache.MaxSharedPointErrM
				evidence = fmt.Sprintf("%s/%s point=%.5fm bound=%.5fm normal=%.3fdeg reuse=%.1f%%",
					scenario.ID, cache.DensityLabel, cache.MaxSharedPointErrM, cellBound,
					cache.MaxSharedNormalDeg, cache.ReusePct)
			}
		}
	}
	pass := maxRatio <= 1+1e-9
	check := makeCheck("shared_world.cell_contract",
		"共享米制格点的独立/共享位置差不超过单元对角线", pass,
		maxRatio, 1.0, "cell_diagonal_ratio",
		fmt.Sprintf("%s; worst point error=%.5fm", evidence, maxPointError))
	return check
}

func checkShadowEdges(reports []ScenarioReport) Check {
	maxBias := 0.0
	maxBiasRatio := 0.0
	missingCrossings := 0
	scaleDrift := 0.0
	for _, scenario := range reports {
		expectedByDirection := make(map[float64]float64)
		for _, edge := range scenario.ShadowEdges {
			maxBias = maxFloat(maxBias, edge.EdgeAbsBiasM)
			if edge.SampleSpacingM > 0 {
				maxBiasRatio = maxFloat(maxBiasRatio, edge.EdgeAbsBiasM/edge.SampleSpacingM)
			}
			if edge.CrossingSamples != 1 {
				missingCrossings++
			}
			if expected, exists := expectedByDirection[edge.DirectionDeg]; exists {
				scaleDrift = maxFloat(scaleDrift, math.Abs(edge.ExpectedRadiusM-expected))
			} else {
				expectedByDirection[edge.DirectionDeg] = edge.ExpectedRadiusM
			}
		}
	}
	pass := missingCrossings == 0 && scaleDrift < 1e-12 && maxBiasRatio <= 1.0+1e-9
	check := makeCheck("shadow.edge_world_scale",
		"接触阴影边半径不随采样密度漂移，插值偏差不超过一个采样步长", pass,
		maxBiasRatio, 1.0, "sample_spacing_ratio",
		fmt.Sprintf("max interpolation bias=%.5fm, missing crossings=%d, scale drift=%.2g", maxBias, missingCrossings, scaleDrift))
	return check
}
