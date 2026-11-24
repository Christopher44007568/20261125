package decallab

import (
	"fmt"
	"math"
)

func checkNormalWavelength(reports []ScenarioReport) Check {
	maxDeviation := 0.0
	probeCount := 0
	for _, scenario := range reports {
		expected := scenario.Detail.Octaves[0].WavelengthM
		for _, observation := range scenario.Observations {
			if observation.Mode != ModeIndependent {
				continue
			}
			probeCount++
			maxDeviation = maxFloat(maxDeviation,
				math.Abs(observation.Sample.Normal.DominantWavelengthM-expected))
		}
		for _, state := range scenario.LODStates {
			maxDeviation = maxFloat(maxDeviation, math.Abs(state.DominantWavelengthM-expected))
		}
	}
	return makeCheck("normal.world_wavelength_preserved",
		"细节法线的主波长以 cycles/meter 定义，不随眼别或 LOD 改变", maxDeviation < 1e-12,
		maxDeviation, 1e-12, "m", fmt.Sprintf("%d eye/density normal probes", probeCount))
}

func checkLODScale(reports []ScenarioReport) Check {
	maxScaleDifference := 0.0
	badWavelength := 0
	maxNormalDelta := 0.0
	for _, scenario := range reports {
		for _, change := range scenario.LODSwitches {
			maxScaleDifference = maxFloat(maxScaleDifference, change.ArcScaleDifferencePct)
			maxNormalDelta = maxFloat(maxNormalDelta, change.NormalAngleDeltaDeg)
			if !change.DominantWavelengthSame {
				badWavelength++
			}
		}
	}
	pass := maxScaleDifference < 1e-8 && badWavelength == 0
	check := makeCheck("lod.switch_keeps_world_scale",
		"材质 LOD 切换只移除高频层，不重标定保留法线/贴花尺度", pass,
		maxScaleDifference, 1e-8, "percent",
		fmt.Sprintf("bad wavelength switches=%d, maximum normal change=%.4f deg", badWavelength, maxNormalDelta))
	return check
}

func checkStereoPresence(reports []ScenarioReport) Check {
	minimumMaxDisparity := math.Inf(1)
	maxAngle := 0.0
	for _, scenario := range reports {
		minimumMaxDisparity = math.Min(minimumMaxDisparity, scenario.Stats.MaxPixelDisparityPx)
		maxAngle = maxFloat(maxAngle, scenario.Stats.MaxBinocularAngleDeg)
	}
	pass := minimumMaxDisparity > 2
	return makeCheck("stereo.binocular_differences_recorded",
		"每个场景都记录到双眼投影像素差和会聚角", pass,
		minimumMaxDisparity, 2, "px",
		fmt.Sprintf("smallest per-scenario maximum disparity; global max angle=%.3f deg", maxAngle))
}

func checkCopyNegativeControl(reports []ScenarioReport) Check {
	scenariosWithoutDetection := []string{}
	maxNormalError := 0.0
	maxDecalError := 0.0
	totalComparisons := 0
	detected := 0
	for _, scenario := range reports {
		scenarioDetected := false
		for _, comparison := range scenario.CopyComparisons {
			totalComparisons++
			if !comparison.RayHit {
				scenariosWithoutDetection = append(scenariosWithoutDetection, scenario.ID+":ray-miss")
			}
			maxNormalError = maxFloat(maxNormalError, comparison.NormalCopy.TangentReuseErrorDeg)
			maxDecalError = maxFloat(maxDecalError, comparison.DecalCopy.ReprojectionM)
			if comparison.Detected {
				detected++
				scenarioDetected = true
			}
		}
		if !scenarioDetected {
			scenariosWithoutDetection = append(scenariosWithoutDetection, scenario.ID)
		}
	}
	pass := len(scenariosWithoutDetection) == 0
	check := makeCheck("negative_control.monocular_tangent_copy_rejected",
		"直接复制单眼切线/贴花结果会产生可测双眼误差并被阈值捕获", pass,
		maxNormalError, copyNormalErrorThreshold, "deg",
		fmt.Sprintf("detected %d/%d same-pixel copies; max decal error=%.4fm; misses=%v",
			detected, totalComparisons, maxDecalError, scenariosWithoutDetection))
	check.NegativeControl = true
	check.Threshold = copyNormalErrorThreshold
	return check
}

func checkRequiredConditions(reports []ScenarioReport) []Check {
	checks := []Check{}
	highObliquityScenarios := 0
	extremeObliquity := 0.0
	maxLightingDifference := 0.0
	lowLODCount := 0
	highLODCount := 0
	maxLowSpacing := 0.0

	for _, scenario := range reports {
		if scenario.Stats.MaxViewObliquityDeg >= 45 {
			highObliquityScenarios++
		}
		if scenario.Stats.MaxViewObliquityDeg >= 60 {
			extremeObliquity++
		}
		for _, observation := range scenario.Observations {
			if observation.Mode != ModeIndependent || observation.Density != "low_sample" {
				continue
			}
			maxLowSpacing = maxFloat(maxLowSpacing, observation.SpacingM)
			if observation.LOD >= 1 {
				lowLODCount++
			}
			if observation.LOD >= 2 {
				highLODCount++
			}
		}
		if scenario.ID == "split_eye_lighting_material_lod" {
			maxLightingDifference = scenario.Stats.MaxLightingDifference
		}
	}

	obliquePass := highObliquityScenarios >= 2 && extremeObliquity >= 1
	checks = append(checks, makeCheck("conditions.oblique_angles",
		"至少两个场景达到 45° 斜视，其中一个达到 60° 掠射", obliquePass,
		float64(highObliquityScenarios), 2, "scenarios>=45deg",
		fmt.Sprintf("%.0f scenario(s) reach 60deg; global max=%.1fdeg", extremeObliquity, maxObliquity(reports))))

	lightingPass := maxLightingDifference >= 0.15
	checks = append(checks, makeCheck("conditions.different_eye_lighting",
		"两眼光照不同场景产生显著辐射差，且不复制左眼着色", lightingPass,
		maxLightingDifference, 0.15, "RGB distance", "split warm/cool eye-light rig"))

	lodPass := lowLODCount > 0 && highLODCount > 0
	checks = append(checks, makeCheck("conditions.low_sample_drives_lod",
		"低采样区确实触发 LOD1/LOD2，而不是只改变报告标签", lodPass,
		float64(highLODCount), 1, "LOD2 probes",
		fmt.Sprintf("%d low-sample probes use LOD>=1, %d use LOD2, max spacing=%.3fm", lowLODCount, highLODCount, maxLowSpacing)))

	transparentCheck := checkTransparentBackside(reports)
	checks = append(checks, transparentCheck)
	return checks
}

func maxObliquity(reports []ScenarioReport) float64 {
	maximum := 0.0
	for _, scenario := range reports {
		maximum = maxFloat(maximum, scenario.Stats.MaxViewObliquityDeg)
	}
	return maximum
}

func checkTransparentBackside(reports []ScenarioReport) Check {
	for _, scenario := range reports {
		if scenario.Surface.Kind != SurfaceTransparent {
			continue
		}
		backside := scenario.Backside
		pass := backside != nil && backside.Hit && backside.Side == "back" &&
			backside.U >= 0 && backside.U <= 1 && backside.V >= 0 && backside.V <= 1 &&
			backside.HitErrorM < 1e-7
		observed := 0.0
		side := "missing"
		if backside != nil {
			observed = backside.HitErrorM
			side = backside.Side
		}
		return makeCheck("conditions.transparent_two_sided_decal",
			"透明表面可从背面命中，贴花仍使用同一米制世界映射", pass,
			observed, 1e-7, "m", fmt.Sprintf("side=%s u/v recorded=%.3f/%.3f", side, valueOrZero(backside, true), valueOrZero(backside, false)))
	}
	return makeCheck("conditions.transparent_two_sided_decal", "存在透明表面背面检查", false, 0, 0, "", "transparent scenario not found")
}

func valueOrZero(record *BacksideRecord, u bool) float64 {
	if record == nil {
		return 0
	}
	if u {
		return record.U
	}
	return record.V
}
