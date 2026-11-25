package main

import (
	"math"
	"sort"
)

func (r *Renderer) shadePixel(x, y, idx int) (Vec3, Vec3, Vec3) {
	fragments := uniqueGuideFragments(r.fragments[idx], r.Scene.Segments)

	// Multiplicative camera-depth accumulation is the equivalent multilayer
	// transmission slice. It is independent of fragment draw order.
	cameraTransmission := Vec(1, 1, 1)
	for _, fragment := range fragments {
		segment := r.Scene.Segments[fragment.Segment]
		layerAlpha := fragment.Alpha * guideLayerAlpha(segment.StrandID)
		cameraTransmission = cameraTransmission.MulVec(layerAttenuation(layerAlpha, segment.Tint))
	}

	beauty, receiverShadow := r.shadeBase(x, y, idx)
	shadowGray := receiverShadow

	if len(fragments) == 0 {
		shadow := Vec(shadowGray, shadowGray, shadowGray)
		return beauty, shadow, cameraTransmission
	}

	// Beauty composites camera fragments back-to-front. The deep-opacity map
	// separately supplies physically ordered light-space layers.
	sort.SliceStable(fragments, func(i, j int) bool {
		return fragments[i].Depth > fragments[j].Depth
	})

	for _, fragment := range fragments {
		hairColor := r.shadeHair(fragment)
		blendAlpha := fragment.Alpha * 0.985
		beauty = mixVec(beauty, hairColor, blendAlpha)
	}

	shadow := Vec(shadowGray, shadowGray, shadowGray)

	return beauty, shadow, cameraTransmission
}

func uniqueGuideFragments(fragments []HairFragment, segments []Segment) []HairFragment {
	unique := fragments[:0]
	for _, fragment := range fragments {
		strandID := segments[fragment.Segment].StrandID
		duplicate := -1
		for i := range unique {
			if segments[unique[i].Segment].StrandID == strandID {
				duplicate = i
				break
			}
		}
		if duplicate < 0 {
			unique = append(unique, fragment)
			continue
		}
		if fragment.Alpha > unique[duplicate].Alpha {
			unique[duplicate] = fragment
		}
	}
	return unique
}

func (r *Renderer) shadeBase(x, y, idx int) (Vec3, float64) {
	if math.IsInf(r.skinDepth[idx], 1) {
		t := float64(y) / float64(r.Camera.Height-1)
		top := srgbToLinear(Vec(0.47, 0.55, 0.66))
		bottom := srgbToLinear(Vec(0.86, 0.78, 0.68))
		background := mixVec(top, bottom, t)
		vignette := 1 - 0.10*math.Hypot(float64(x)/float64(r.Camera.Width)-0.5, float64(y)/float64(r.Camera.Height)-0.5)
		return background.Mul(vignette), 1
	}

	point := r.skinPoint[idx]
	normal := r.skinNormal[idx]
	lightTransmission := r.Deep.TransmittanceAt(point)

	hemisphere := r.Scene.AmbientGround.Mul(0.5 - 0.5*normal.Y).
		Add(r.Scene.AmbientSky.Mul(0.5 + 0.5*normal.Y))
	ambient := r.Scene.Skin.Albedo.MulVec(hemisphere).
		Add(r.Scene.Skin.Albedo.Mul(0.20))

	ndl := maxFloat(0, normal.Dot(r.Scene.LightDir))
	wrapped := clamp((ndl+0.34)/1.34, 0, 1)
	direct := r.Scene.Skin.Albedo.
		MulVec(r.Scene.LightColor).
		Mul(wrapped * 0.82).
		MulVec(lightTransmission)

	// A small warm bounce keeps deep strand shadows on skin readable without
	// flattening the multilayer shadow variation.
	bounce := Vec(1.0, 0.34, 0.16).Mul(0.055 * wrapped)
	color := ambient.Add(direct).Add(bounce)
	return color, clamp01(luminance(lightTransmission))
}

func (r *Renderer) shadeHair(fragment HairFragment) Vec3 {
	segment := r.Scene.Segments[fragment.Segment]
	point := segmentPoint(segment, fragment.T)
	view := r.Camera.Eye.Sub(point).Normalized()
	tangent := mixVec(segment.Tangent0, segment.Tangent1, fragment.T).Normalized()
	lightTransmission := r.Deep.TransmittanceAtExcluding(point, segment.StrandID)

	viewPerpendicular := view.Sub(tangent.Mul(view.Dot(tangent)))
	if viewPerpendicular.LenSqr() < 1e-8 {
		viewPerpendicular = r.Camera.Up
	}
	viewPerpendicular = viewPerpendicular.Normalized()
	radial := tangent.Cross(viewPerpendicular)
	if radial.LenSqr() < 1e-8 {
		radial = r.Camera.Right
	}
	radial = radial.Normalized()
	normal := viewPerpendicular.Mul(math.Sqrt(math.Max(0, 1-fragment.Side*fragment.Side))).
		Add(radial.Mul(fragment.Side)).
		Normalized()

	light := r.Scene.LightDir
	ndl := maxFloat(0, normal.Dot(light))
	anisotropicDiffuse := math.Sqrt(math.Max(0, 1-tangent.Dot(light)*tangent.Dot(light)))
	diffuse := 0.30 + 0.48*ndl + 0.34*anisotropicDiffuse

	strandVariation := 0.72 + 0.56*fractFine(float64(segment.StrandID)*12.9898)
	albedo := segment.Albedo.Mul(2.35 * strandVariation)
	ambient := albedo.MulVec(r.Scene.AmbientSky).Mul(2.2).
		Add(Vec(0.035, 0.024, 0.020))
	direct := albedo.
		Mul(diffuse).
		MulVec(r.Scene.LightColor).
		Mul(0.72).
		MulVec(lightTransmission)

	halfVector := light.Add(view).Normalized()
	normalHalf := maxFloat(0, normal.Dot(halfVector))
	normalSpec := math.Pow(normalHalf, 96)
	sheenSpec := math.Pow(normalHalf, 22)
	tangentHalf := tangent.Dot(halfVector)
	anisotropy := math.Sqrt(math.Max(0, 1-tangentHalf*tangentHalf))
	specular := Vec(1.0, 0.84, 0.62).
		Mul((0.46*normalSpec + 0.075*sheenSpec) * (0.65 + 0.35*anisotropy)).
		MulVec(lightTransmission.Mul(0.62).Add(Vec(0.14, 0.12, 0.10)))

	rim := math.Pow(1-maxFloat(0, normal.Dot(view)), 3)
	rimLight := Vec(0.30, 0.38, 0.55).Mul(0.20 * rim)
	transmissionGlow := segment.Tint.Mul(0.10 * (0.35 + 0.65*(1-math.Abs(fragment.Side))))

	return ambient.Add(direct).Add(specular).Add(rimLight).Add(transmissionGlow)
}

func segmentPoint(segment Segment, t float64) Vec3 {
	return segment.A.Mul(1 - t).Add(segment.B.Mul(t))
}

func guideLayerAlpha(strandID int) float64 {
	return clamp(0.50+0.07*math.Sin(float64(strandID)*12.9898), 0.43, 0.57)
}

func layerAttenuation(alpha float64, tint Vec3) Vec3 {
	return Vec3{
		X: (1 - alpha) + alpha*tint.X,
		Y: (1 - alpha) + alpha*tint.Y,
		Z: (1 - alpha) + alpha*tint.Z,
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func fractFine(v float64) float64 {
	return v - math.Floor(v)
}
