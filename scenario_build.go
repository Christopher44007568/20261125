package decallab

import "math"

func buildCurvedObliqueScenario() Scenario {
	material := Material{
		ID: "satin_ceramic", Albedo: RGB{R: 0.70, G: 0.73, B: 0.78},
		Opacity: 1, Roughness: 0.38,
	}
	surface := NewSphere("curved_ceramic", SurfaceCurved, Vec3{}, 0.68, material)
	anchor := Vec3{X: 0, Y: -0.02, Z: math.Sqrt(0.68*0.68 - 0.02*0.02)}
	decal := NewDecal("curved_directional_decal", surface, anchor, 0.34, 0.205, -18, 0.96, "arrow")
	detail := DefaultDetailNormal(anchor, decal.AxisU, decal.AxisV, decal.Normal)
	shadowCenter := Vec3{X: 0.015, Y: -0.225, Z: math.Sqrt(0.68*0.68 - 0.225*0.225 - 0.015*0.015)}
	shadow := NewContactShadow("curved_contact_edge", surface, shadowCenter, 0.112, 0.052, 0.014, 0.46, -18)
	rig := NewStereoRig(
		Vec3{X: 0.72, Y: 0.16, Z: 1.42},
		Vec3{X: 0, Y: -0.03, Z: 0.24},
		0.064, 46, 1280, 960,
	)
	return Scenario{
		ID: "curved_oblique_shared", Title: "曲面、斜视角与共享世界采样",
		Tags:    []string{"curved", "oblique", "shared_sampling", "normal_probes", "shadow_edges"},
		Surface: surface, Decal: decal, Detail: detail, Shadow: shadow, Rig: rig,
		Lighting:       sharedWorldLight(),
		Probes:         defaultProbes(),
		Densities:      defaultDensities(9, 0.020, 0.026),
		EdgeDirections: []float64{0, 35, 90, 145},
	}
}

func buildTransparentObliqueScenario() Scenario {
	material := Material{
		ID: "slightly_blue_glass", Albedo: RGB{R: 0.46, G: 0.72, B: 0.92},
		Transparent: true, Opacity: 0.24, Roughness: 0.08,
	}
	normal := Vec3{X: 0.47, Y: 0, Z: 0.883}.Normalize()
	surface := NewPlane(
		"transparent_glass_panel", SurfaceTransparent,
		Vec3{X: 0, Y: 0, Z: 0.02}, normal, Vec3{X: 1},
		1.28, 0.96, material,
	)
	anchor := surface.Center
	decal := NewDecal("glass_oriented_decal", surface, anchor, 0.31, 0.185, 27, 0.88, "stripe")
	detail := DefaultDetailNormal(anchor, decal.AxisU, decal.AxisV, decal.Normal)
	shadow := NewContactShadow("glass_contact_edge", surface,
		anchor.Add(Vec3{Y: -0.19}), 0.096, 0.043, 0.012, 0.34, 27)
	rig := NewStereoRig(
		Vec3{X: -0.64, Y: 0.11, Z: 1.38},
		Vec3{X: 0, Y: 0, Z: 0.02},
		0.064, 46, 1280, 960,
	)
	return Scenario{
		ID: "transparent_oblique_two_sided", Title: "透明玻璃、斜视与双面贴花",
		Tags:    []string{"transparent", "oblique", "two_sided", "shared_sampling", "shadow_edges"},
		Surface: surface, Decal: decal, Detail: detail, Shadow: shadow, Rig: rig,
		Lighting:       sharedWorldLight(),
		Probes:         defaultProbes(),
		Densities:      defaultDensities(8, 0.018, 0.024),
		EdgeDirections: []float64{0, 45, 90, 135},
		BacksideCheck:  true,
	}
}

func buildSplitLightLODScenario() Scenario {
	material := Material{
		ID: "painted_curved_lod", Albedo: RGB{R: 0.82, G: 0.76, B: 0.62},
		Opacity: 1, Roughness: 0.56,
	}
	surface := NewSphere("lod_curved_surface", SurfaceCurved, Vec3{}, 0.82, material)
	anchor := Vec3{X: 0, Y: 0.015, Z: math.Sqrt(0.82*0.82 - 0.015*0.015)}
	decal := NewDecal("lod_directional_decal", surface, anchor, 0.39, 0.235, 12, 0.97, "arrow")
	detail := DefaultDetailNormal(anchor, decal.AxisU, decal.AxisV, decal.Normal)
	shadowCenter := Vec3{X: -0.02, Y: -0.27, Z: math.Sqrt(0.82*0.82 - 0.27*0.27 - 0.02*0.02)}
	shadow := NewContactShadow("lod_contact_shadow", surface, shadowCenter, 0.132, 0.061, 0.017, 0.48, 12)
	rig := NewStereoRig(
		Vec3{X: 0.38, Y: 0.13, Z: 1.72},
		Vec3{X: 0, Y: 0, Z: 0.20},
		0.064, 45, 1280, 960,
	)
	return Scenario{
		ID: "split_eye_lighting_material_lod", Title: "两眼光照不同与材质 LOD 切换",
		Tags:    []string{"curved", "eye_lighting_different", "material_lod_switch", "normal_probes"},
		Surface: surface, Decal: decal, Detail: detail, Shadow: shadow, Rig: rig,
		Lighting:       splitEyeLights(),
		Probes:         defaultProbes(),
		Densities:      defaultDensities(8, 0.024, 0.030),
		EdgeDirections: []float64{0, 45, 90},
	}
}

func buildGrazingLowSampleScenario() Scenario {
	material := Material{
		ID: "dark_anodized_curve", Albedo: RGB{R: 0.24, G: 0.29, B: 0.34},
		Opacity: 1, Roughness: 0.31,
	}
	surface := NewSphere("grazing_curved_surface", SurfaceCurved, Vec3{}, 0.56, material)
	anchor := Vec3{X: 0, Y: 0.01, Z: math.Sqrt(0.56*0.56 - 0.01*0.01)}
	decal := NewDecal("grazing_world_scale_decal", surface, anchor, 0.285, 0.172, 31, 0.95, "stripe")
	detail := DefaultDetailNormal(anchor, decal.AxisU, decal.AxisV, decal.Normal)
	shadowCenter := Vec3{X: 0.01, Y: -0.18, Z: math.Sqrt(0.56*0.56 - 0.18*0.18 - 0.01*0.01)}
	shadow := NewContactShadow("grazing_shadow_edge", surface, shadowCenter, 0.092, 0.043, 0.013, 0.50, 31)
	rig := NewStereoRig(
		Vec3{X: 1.04, Y: 0.08, Z: 1.06},
		Vec3{X: 0, Y: 0, Z: 0.16},
		0.064, 52, 1280, 960,
	)
	probes := []ProbeSpec{
		{ID: "center", UOffsetM: 0, VOffsetM: 0},
		{ID: "near_edge", UOffsetM: 0.108, VOffsetM: 0.02},
		{ID: "grazing_corner", UOffsetM: 0.096, VOffsetM: -0.052},
		{ID: "wrap_sample", UOffsetM: -0.128, VOffsetM: 0.040},
	}
	return Scenario{
		ID: "grazing_very_low_sampling", Title: "掠射角与极低采样区",
		Tags:    []string{"curved", "oblique", "grazing", "low_sample_zone", "shared_sampling"},
		Surface: surface, Decal: decal, Detail: detail, Shadow: shadow, Rig: rig,
		Lighting:       sharedWorldLight(),
		Probes:         probes,
		Densities:      defaultDensities(16, 0.052, 0.060),
		EdgeDirections: []float64{0, 30, 70, 110},
	}
}
