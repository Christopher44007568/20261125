package glass

const nearOpaque = 1 - 1e-9
const extremelyFaint = 1e-12

func BuiltinScenes() []Scene {
	return []Scene{
		intersectionScene(),
		nearCoplanarScene(),
		sharedMaterialScene(),
	}
}

func DefaultBudget() Budget {
	return Budget{Left: 3, Right: 2}
}

func intersectionScene() Scene {
	return Scene{
		Kind: SceneIntersection,
		Materials: []Material{
			{ID: "ruby", Color: RGB{R: 0.92, G: 0.08, B: 0.12}},
			{ID: "azure", Color: RGB{R: 0.08, G: 0.35, B: 0.95}},
			{ID: "jade", Color: RGB{R: 0.08, G: 0.88, B: 0.42}},
			{ID: "amber", Color: RGB{R: 0.98, G: 0.72, B: 0.08}},
			{ID: "smoke", Color: RGB{R: 0.55, G: 0.58, B: 0.62}},
		},
		Layers: []Layer{
			{
				ID:         "cross-left",
				MaterialID: "ruby",
				Depth:      map[Eye]float64{EyeLeft: 1.0, EyeRight: 4.0},
				Opacity:    0.65,
			},
			{
				ID:         "cross-right",
				MaterialID: "azure",
				Depth:      map[Eye]float64{EyeLeft: 4.0, EyeRight: 1.0},
				Opacity:    0.35,
			},
			{
				ID:         "near-opaque",
				MaterialID: "jade",
				Depth:      map[Eye]float64{EyeLeft: 2.0, EyeRight: 2.5},
				Opacity:    nearOpaque,
			},
			{
				ID:         "middle",
				MaterialID: "amber",
				Depth:      map[Eye]float64{EyeLeft: 3.0, EyeRight: 3.0},
				Opacity:    0.25,
			},
			{
				ID:         "faint",
				MaterialID: "smoke",
				Depth:      map[Eye]float64{EyeLeft: 5.0, EyeRight: 5.0},
				Opacity:    extremelyFaint,
			},
		},
	}
}

func nearCoplanarScene() Scene {
	return Scene{
		Kind: SceneNearCoplanar,
		Materials: []Material{
			{ID: "cp-blue", Color: RGB{R: 0.16, G: 0.42, B: 0.94}},
			{ID: "cp-green", Color: RGB{R: 0.18, G: 0.86, B: 0.48}},
			{ID: "cp-rose", Color: RGB{R: 0.94, G: 0.3, B: 0.55}},
			{ID: "cp-white", Color: RGB{R: 0.86, G: 0.9, B: 0.96}},
			{ID: "cp-amber", Color: RGB{R: 0.98, G: 0.7, B: 0.16}},
		},
		Layers: []Layer{
			{
				ID:         "cp-a-near-opaque",
				MaterialID: "cp-white",
				Depth:      map[Eye]float64{EyeLeft: 2.0, EyeRight: 2.00000005},
				Opacity:    nearOpaque,
			},
			{
				ID:         "cp-b-faint",
				MaterialID: "cp-blue",
				Depth:      map[Eye]float64{EyeLeft: 2.0000002, EyeRight: 2.0},
				Opacity:    extremelyFaint,
			},
			{
				ID:         "cp-c",
				MaterialID: "cp-green",
				Depth:      map[Eye]float64{EyeLeft: 1.9999999, EyeRight: 2.00000015},
				Opacity:    0.42,
			},
			{
				ID:         "cp-d",
				MaterialID: "cp-rose",
				Depth:      map[Eye]float64{EyeLeft: 2.0000001, EyeRight: 2.0000001},
				Opacity:    0.58,
			},
			{
				ID:         "cp-e",
				MaterialID: "cp-amber",
				Depth:      map[Eye]float64{EyeLeft: 2.00000015, EyeRight: 1.99999995},
				Opacity:    0.21,
			},
		},
	}
}

func sharedMaterialScene() Scene {
	return Scene{
		Kind: SceneSharedMaterial,
		Materials: []Material{
			{ID: "frosted", Color: RGB{R: 0.72, G: 0.83, B: 0.9}},
		},
		Layers: []Layer{
			{
				ID:         "shared-a",
				MaterialID: "frosted",
				Depth:      map[Eye]float64{EyeLeft: 0.8, EyeRight: 1.2},
				Opacity:    0.31,
			},
			{
				ID:         "shared-b",
				MaterialID: "frosted",
				Depth:      map[Eye]float64{EyeLeft: 1.6, EyeRight: 0.7},
				Opacity:    0.31,
			},
			{
				ID:         "shared-c",
				MaterialID: "frosted",
				Depth:      map[Eye]float64{EyeLeft: 2.4, EyeRight: 2.5},
				Opacity:    0.31,
			},
			{
				ID:         "shared-d",
				MaterialID: "frosted",
				Depth:      map[Eye]float64{EyeLeft: 3.3, EyeRight: 3.1},
				Opacity:    0.31,
			},
			{
				ID:         "shared-e",
				MaterialID: "frosted",
				Depth:      map[Eye]float64{EyeLeft: 4.9, EyeRight: 4.8},
				Opacity:    0.31,
			},
		},
	}
}
