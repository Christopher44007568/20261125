package main

import "math"

func buildMetadata(cfg Config, res *RenderResult) map[string]any {
	return map[string]any{
		"resolution": map[string]int{"width": cfg.Width, "height": cfg.Height},
		"renderer": map[string]any{
			"supersampling":        cfg.Supersample,
			"temporal_samples":     cfg.Samples,
			"shutter_seconds":      cfg.Shutter,
			"frame_center_seconds": cfg.FrameTime,
			"seed":                 cfg.Seed,
			"alpha_compositing":    "straight-alpha src-over, back to front",
			"sort_key":             []string{"depth ascending", "sequence ascending", "surface kind", "layer id"},
			"sort_stability":       "slice-stable per frame and per pixel",
			"hair_model":           "each strand is a separately depth-sorted tapered curve; no bundle-level transparent card",
			"trail_model":          "each shutter sample re-deforms every strand from root to tip",
		},
		"outputs": map[string]string{
			"color":          "color.png",
			"layer_order":    "layer_order.png",
			"motion_vectors": "motion_vectors.png",
		},
		"counts": map[string]int{
			"hair_strands":               res.HairCount,
			"glass_blades":               res.GlassCount,
			"transparent_surfaces_total": res.LayerCount,
		},
		"layer_order_mask": map[string]any{
			"encoding":         "depth-rank palette; opaque head and clothing are neutral gray",
			"transparent_ramp": "violet -> blue -> cyan -> green -> yellow -> orange -> pink",
			"depth_range":      []float64{0.68, 2.92},
			"alpha":            "palette is blended in the same stable back-to-front order as color",
		},
		"motion_vectors": map[string]any{
			"units":              "pixels/second",
			"direction_encoding": "HSV hue, red=+X, yellow-green=+Y (screen down), cyan=-X, magenta=-Y",
			"magnitude_encoding": "HSV value and arrow length",
			"max_speed":          round2(res.MaxSpeed),
		},
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
