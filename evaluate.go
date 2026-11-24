package decallab

import (
	"fmt"
	"math"
)

func runScenario(scenario Scenario) ScenarioReport {
	report := ScenarioReport{
		ID:       scenario.ID,
		Title:    scenario.Title,
		Tags:     append([]string{}, scenario.Tags...),
		Surface:  surfaceInfo(scenario.Surface),
		Decal:    scenario.Decal,
		Detail:   scenario.Detail,
		Rig:      scenario.Rig,
		Lighting: scenario.Lighting,
	}

	for _, probe := range scenario.Probes {
		point := scenario.ProbePoint(probe)
		report.Projections = append(report.Projections, ProbeProjectionRecord{
			ProbeID:        probe.ID,
			CanonicalPoint: point,
			Projection:     scenario.Rig.ProjectStereo(point),
		})
	}

	for _, density := range scenario.Densities {
		cache := NewWorldSampleCache(density.SharedCellM, scenario.Decal)
		independent := make(map[string]ObservationRecord)
		var maxCachePointError, maxCacheNormalError float64
		var maxCacheArcDifference, maxCacheShadowDifference float64

		for _, probe := range scenario.Probes {
			point := scenario.ProbePoint(probe)

			for _, eye := range Eyes {
				observation := evaluateEye(scenario, probe.ID, point, density, eye, ModeIndependent)
				report.Observations = append(report.Observations, observation)
				independent[observationKey(probe.ID, eye)] = observation
			}

			for _, eye := range Eyes {
				key := observationKey(probe.ID, eye)
				base := independent[key]
				cached, cacheKey, shared := cache.GetOrEvaluatePoint(
					eye, base.HitPoint, scenario.Surface, scenario.Decal,
					scenario.Detail, scenario.Shadow, base.SpacingM, -1,
				)
				sharedObservation := observationFromCached(base, scenario, cached, cacheKey, shared)
				report.Observations = append(report.Observations, sharedObservation)

				pointError, normalError, arcWidthDifference, arcHeightDifference, shadowDifference := MaterialDifference(cached, base.Sample)
				maxCachePointError = maxFloat(maxCachePointError, pointError)
				maxCacheNormalError = maxFloat(maxCacheNormalError, normalError)
				maxCacheArcDifference = maxFloat(maxCacheArcDifference, maxFloat(arcWidthDifference, arcHeightDifference))
				maxCacheShadowDifference = maxFloat(maxCacheShadowDifference, shadowDifference)
			}

			comparison := compareCopiedMonocular(scenario, probe.ID, density, independent)
			report.CopyComparisons = append(report.CopyComparisons, comparison)
		}

		total := cache.Hits + cache.Misses
		reuse := 0.0
		if total > 0 {
			reuse = 100 * float64(cache.Hits) / float64(total)
		}
		summary := CacheSummary{
			DensityLabel:       density.Label,
			CellSizeM:          cache.CellSizeM,
			Entries:            len(cache.Entries),
			Hits:               cache.Hits,
			Misses:             cache.Misses,
			ReusePct:           reuse,
			MaxSharedPointErrM: maxCachePointError,
			MaxSharedNormalDeg: maxCacheNormalError,
			MaxArcScaleDiffPct: maxCacheArcDifference,
			MaxShadowAlphaDiff: maxCacheShadowDifference,
		}
		report.Caches = append(report.Caches, summary)
	}

	report.Cache = aggregateCaches(report.Caches)
	report.LODStates, report.LODSwitches = buildLODRecords(scenario)
	report.ShadowEdges = buildShadowEdges(scenario)
	if scenario.BacksideCheck {
		report.Backside = buildBacksideCheck(scenario)
	}
	report.Stats = computeScenarioStats(&report)
	return report
}

func observationKey(probeID string, eye Eye) string {
	return fmt.Sprintf("%s/%s", probeID, eye)
}

func evaluateEye(scenario Scenario, probeID string, canonical Vec3, density DensityConfig, eye Eye, mode SamplingMode) ObservationRecord {
	camera := scenario.Rig.Cameras[eye]
	pixel, inFrustum := camera.Project(canonical)
	footX, footY, footprintOK := SurfacePixelFootprintM(camera, pixel, scenario.Surface)
	spacing := density.FallbackSpacingM
	if footprintOK {
		averageFootprint := (footX + footY) * 0.5
		spacing = math.Max(density.MinSpacingM, averageFootprint*density.PixelStride)
	}

	ray := camera.RayForPixel(pixel.X, pixel.Y)
	hit, _, rayHit := IntersectNearest(ray, []Surface{scenario.Surface})
	point := canonical
	hitError := 0.0
	if rayHit {
		point = hit.Point
		hitError = point.Distance(canonical)
	}

	sample := EvaluateMaterial(point, scenario.Surface, scenario.Decal, scenario.Detail, scenario.Shadow, spacing, -1)
	material := scenario.Surface.GetMaterial()
	alpha := alphaForSample(material, scenario.Decal, sample.Decal.PatternValue)
	lighting := ShadeSurface(
		sample.Normal.WorldNormal,
		camera.ViewDirection(point),
		material,
		scenario.Lighting.ForEye(eye),
		alpha,
	)
	lighting.Eye = eye
	viewToCamera := camera.Position.Sub(point).Normalize()
	obliquity := angleBetweenDeg(viewToCamera, sample.Frame.Normal)

	return ObservationRecord{
		ProbeID:          probeID,
		Density:          density.Label,
		Mode:             mode,
		Eye:              eye,
		Pixel:            pixel,
		PixelInFrustum:   inFrustum,
		RayHit:           rayHit,
		CanonicalPoint:   canonical,
		HitPoint:         point,
		HitErrorM:        hitError,
		SpacingM:         spacing,
		LOD:              sample.LOD,
		ViewObliquityDeg: obliquity,
		FootprintXM:      footX,
		FootprintYM:      footY,
		Sample:           sample,
		Lighting:         lighting,
	}
}

func observationFromCached(base ObservationRecord, scenario Scenario, cached MaterialSample, cacheKey string, shared bool) ObservationRecord {
	result := base
	result.Mode = ModeSharedWorld
	result.Sample = cached
	result.HitPoint = cached.Point
	result.HitErrorM = cached.Point.Distance(base.CanonicalPoint)
	result.CacheKey = cacheKey
	result.CacheShared = shared

	camera := scenario.Rig.Cameras[base.Eye]
	material := scenario.Surface.GetMaterial()
	alpha := alphaForSample(material, scenario.Decal, cached.Decal.PatternValue)
	lighting := ShadeSurface(
		cached.Normal.WorldNormal,
		camera.ViewDirection(cached.Point),
		material,
		scenario.Lighting.ForEye(base.Eye),
		alpha,
	)
	lighting.Eye = base.Eye
	result.Lighting = lighting
	viewToCamera := camera.Position.Sub(cached.Point).Normalize()
	result.ViewObliquityDeg = angleBetweenDeg(viewToCamera, cached.Frame.Normal)
	return result
}

func aggregateCaches(caches []CacheSummary) CacheSummary {
	if len(caches) == 0 {
		return CacheSummary{}
	}
	aggregate := CacheSummary{DensityLabel: "all_densities"}
	reuseWeight := 0
	for _, cache := range caches {
		aggregate.Entries += cache.Entries
		aggregate.Hits += cache.Hits
		aggregate.Misses += cache.Misses
		aggregate.MaxSharedPointErrM = maxFloat(aggregate.MaxSharedPointErrM, cache.MaxSharedPointErrM)
		aggregate.MaxSharedNormalDeg = maxFloat(aggregate.MaxSharedNormalDeg, cache.MaxSharedNormalDeg)
		aggregate.MaxArcScaleDiffPct = maxFloat(aggregate.MaxArcScaleDiffPct, cache.MaxArcScaleDiffPct)
		aggregate.MaxShadowAlphaDiff = maxFloat(aggregate.MaxShadowAlphaDiff, cache.MaxShadowAlphaDiff)
		if cache.CellSizeM > aggregate.CellSizeM {
			aggregate.CellSizeM = cache.CellSizeM
		}
		reuseWeight += cache.Hits + cache.Misses
	}
	if reuseWeight > 0 {
		aggregate.ReusePct = 100 * float64(aggregate.Hits) / float64(reuseWeight)
	}
	return aggregate
}
