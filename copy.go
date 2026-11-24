package decallab

import "math"

const (
	copyDecalErrorThresholdM = 0.004
	copyNormalErrorThreshold = 0.75
	copyShadowAlphaThreshold = 0.030
	copyRadianceThreshold    = 0.080
)

func compareCopiedMonocular(scenario Scenario, probeID string, density DensityConfig, independent map[string]ObservationRecord) CopyComparisonRecord {
	left := independent[observationKey(probeID, EyeLeft)]
	rightCamera := scenario.Rig.Cameras[EyeRight]
	ray := rightCamera.RayForPixel(left.Pixel.X, left.Pixel.Y)
	hit, _, rayHit := IntersectNearest(ray, []Surface{scenario.Surface})

	rightPoint := left.HitPoint
	if rayHit {
		rightPoint = hit.Point
	}

	footX, footY, footprintOK := SurfacePixelFootprintM(rightCamera, left.Pixel, scenario.Surface)
	spacing := density.FallbackSpacingM
	if footprintOK {
		spacing = math.Max(density.MinSpacingM, (footX+footY)*0.5*density.PixelStride)
	}

	rightSample := EvaluateMaterial(
		rightPoint, scenario.Surface, scenario.Decal, scenario.Detail,
		scenario.Shadow, spacing, -1,
	)
	material := scenario.Surface.GetMaterial()
	rightAlpha := alphaForSample(material, scenario.Decal, rightSample.Decal.PatternValue)
	rightLighting := ShadeSurface(
		rightSample.Normal.WorldNormal,
		rightCamera.ViewDirection(rightPoint),
		material,
		scenario.Lighting.ForEye(EyeRight),
		rightAlpha,
	)
	rightLighting.Eye = EyeRight

	decalError := scenario.Decal.MeasureCopiedTangent(
		left.HitPoint, rightPoint, left.Sample.Decal, rightSample.Decal,
	)
	normalError := ReuseTangentAtDestination(
		left.Sample.Normal, rightSample.Frame, rightSample.Normal,
	)
	shadowDelta := math.Abs(left.Sample.Shadow.Alpha - rightSample.Shadow.Alpha)
	lightingDifference := CompareLighting(left.Lighting, rightLighting)
	copiedRadianceError := radianceDistance(left.Lighting.Radiance, rightLighting.Radiance)
	parallaxShift := left.HitPoint.Distance(rightPoint)

	detected := decalError.ReprojectionM > copyDecalErrorThresholdM ||
		normalError.TangentReuseErrorDeg > copyNormalErrorThreshold ||
		shadowDelta > copyShadowAlphaThreshold ||
		copiedRadianceError > copyRadianceThreshold

	return CopyComparisonRecord{
		ProbeID:               probeID,
		Density:               density.Label,
		RayHit:                rayHit,
		LeftPixel:             left.Pixel,
		RightSamePixel:        left.Pixel,
		LeftPoint:             left.HitPoint,
		RightSamePixelPoint:   rightPoint,
		ParallaxShiftM:        parallaxShift,
		SourceLOD:             left.LOD,
		DestinationLOD:        rightSample.LOD,
		DecalCopy:             decalError,
		NormalCopy:            normalError,
		SourceShadowAlpha:     left.Sample.Shadow.Alpha,
		CorrectShadowAlpha:    rightSample.Shadow.Alpha,
		CopiedShadowAlphaDiff: shadowDelta,
		LightingDifference:    lightingDifference,
		CopiedRadianceError:   copiedRadianceError,
		Detected:              detected,
	}
}

func radianceDistance(a, b RGB) float64 {
	dr := a.R - b.R
	dg := a.G - b.G
	db := a.B - b.B
	return math.Sqrt(dr*dr + dg*dg + db*db)
}
