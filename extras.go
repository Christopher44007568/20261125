package decallab

import "math"

func buildLODRecords(scenario Scenario) ([]LODStateRecord, []LODSwitchRecord) {
	states := []LODStateRecord{}
	switches := []LODSwitchRecord{}
	fixedSpacing := 0.05

	for _, probe := range scenario.Probes {
		point := scenario.ProbePoint(probe)
		for lod := 0; lod <= scenario.Detail.MaxLOD; lod++ {
			sample := EvaluateMaterial(point, scenario.Surface, scenario.Decal, scenario.Detail, scenario.Shadow, fixedSpacing, lod)
			state := LODStateRecord{
				ProbeID:             probe.ID,
				LOD:                 lod,
				Point:               point,
				ArcWidthM:           sample.Decal.SurfaceArcWidthM,
				ArcHeightM:          sample.Decal.SurfaceArcHeightM,
				DominantWavelengthM: sample.Normal.DominantWavelengthM,
				PerturbationDeg:     sample.Normal.PerturbationDeg,
				Normal:              sample.Normal,
			}
			states = append(states, state)
			if lod > 0 {
				previous := states[len(states)-2]
				normalDelta := angleBetweenDeg(previous.Normal.WorldNormal, state.Normal.WorldNormal)
				arcDifference := math.Abs(percentError(state.ArcWidthM, previous.ArcWidthM))
				switches = append(switches, LODSwitchRecord{
					ProbeID:                probe.ID,
					FromLOD:                lod - 1,
					ToLOD:                  lod,
					NormalAngleDeltaDeg:    normalDelta,
					PerturbationDeltaDeg:   math.Abs(state.PerturbationDeg - previous.PerturbationDeg),
					DominantWavelengthSame: previous.DominantWavelengthM == state.DominantWavelengthM,
					ArcScaleDifferencePct:  arcDifference,
				})
			}
		}
	}
	return states, switches
}

func buildShadowEdges(scenario Scenario) []ShadowEdgeRecord {
	edges := []ShadowEdgeRecord{}
	for _, density := range scenario.Densities {
		spacing := density.SharedCellM
		if spacing <= 0 {
			spacing = density.FallbackSpacingM
		}
		for _, direction := range scenario.EdgeDirections {
			edge := MeasureShadowEdge(scenario.Surface, scenario.Shadow, direction, spacing)
			edge.DensityLabel = density.Label
			edges = append(edges, edge)
		}
	}
	return edges
}

func buildBacksideCheck(scenario Scenario) *BacksideRecord {
	normal := scenario.Decal.Normal
	ray := Ray{
		Origin:    scenario.Decal.Center.Add(normal.Scale(-0.70)),
		Direction: normal,
	}
	hit, _, ok := IntersectNearest(ray, []Surface{scenario.Surface})
	record := &BacksideRecord{Ray: ray, Hit: ok, WidthM: scenario.Decal.WidthM, HeightM: scenario.Decal.HeightM}
	if !ok {
		return record
	}
	u, v := scenario.Decal.UV(hit.Point)
	record.Side = hit.Side
	record.Point = hit.Point
	record.U = u
	record.V = v
	record.HitErrorM = hit.Point.Distance(scenario.Decal.Center)
	return record
}

func computeScenarioStats(report *ScenarioReport) ScenarioStats {
	stats := ScenarioStats{ProbeCount: len(report.Projections)}
	for _, observation := range report.Observations {
		if observation.Mode != ModeIndependent {
			continue
		}
		stats.MaxViewObliquityDeg = maxFloat(stats.MaxViewObliquityDeg, observation.ViewObliquityDeg)
		stats.MaxFootprintM = maxFloat(stats.MaxFootprintM, observation.SpacingM)
		if !math.IsInf(observation.Sample.Decal.CurvatureStretchPct, 1) && !math.IsNaN(observation.Sample.Decal.CurvatureStretchPct) {
			stats.MaxCurvatureStretchPct = maxFloat(stats.MaxCurvatureStretchPct, observation.Sample.Decal.CurvatureStretchPct)
		}
	}
	for _, projection := range report.Projections {
		stats.MaxBinocularAngleDeg = maxFloat(stats.MaxBinocularAngleDeg, projection.Projection.BinocularAngleDeg)
		stats.MaxPixelDisparityPx = maxFloat(stats.MaxPixelDisparityPx, math.Abs(projection.Projection.HorizontalDisparityPx))
	}
	for _, comparison := range report.CopyComparisons {
		stats.MaxLightingDifference = maxFloat(stats.MaxLightingDifference, comparison.LightingDifference.RadianceDistance)
	}
	stats.SharedCacheReusePct = report.Cache.ReusePct
	return stats
}
