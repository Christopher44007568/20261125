package main

const (
	sceneWidth  = 32
	sceneHeight = 20
)

var glyphs = map[rune][][]float64{
	'G': {
		{0, 1, 1, 1, 0},
		{1, 0, 0, 0, 0},
		{1, 0, 0, 0, 0},
		{1, 0, 1, 1, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
	},
	'O': {
		{0, 1, 1, 1, 0},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{1, 0, 0, 0, 1},
		{0, 1, 1, 1, 0},
	},
}

func glyphCoverage(x, y int) float64 {
	glyphX, glyphY := -1, -1

	if x >= 8 && x < 13 && y >= 7 && y < 14 {
		glyphX, glyphY = x-8, y-7
		if glyphs['G'][glyphY][glyphX] == 1 {
			return 1
		}
	}
	if x >= 15 && x < 20 && y >= 7 && y < 14 {
		glyphX, glyphY = x-15, y-7
		if glyphs['O'][glyphY][glyphX] == 1 {
			return 1
		}
	}
	return 0
}

func eyeFactor(layer string, eye int) float64 {
	if eye == 0 {
		switch layer {
		case "shadow":
			return 1
		case "text":
			return 0.96
		case "highlight":
			return 0.62
		}
	}
	switch layer {
	case "shadow":
		return 0.72
	case "text":
		return 0.82
	case "highlight":
		return 0.88
	}
	return 1
}

func textColor(x int) (r, g, b float64) {
	if x < 14 {
		return 0.05, 0.92, 1.0
	}
	return 1.0, 0.18, 0.82
}

func highlightCoverage(x, y int) float64 {
	if x < 17 || x > 28 || y < 4 || y > 9 {
		return 0
	}
	if (x == 17 || x == 28) && (y == 4 || y == 9) {
		return 0
	}
	return 1
}

func buildScene() Scene {
	return Scene{
		Width:  sceneWidth,
		Height: sceneHeight,
		Layers: []Layer{
			{
				Name:   "hdr_background",
				Space:  Linear,
				Premul: false,
				Render: func(x, y, eye int) Sample {
					fx := float64(x) / float64(sceneWidth-1)
					fy := float64(y) / float64(sceneHeight-1)
					return Sample{
						R: 0.55 + 1.35*fx,
						G: 0.35 + 1.15*fy,
						B: 0.85 + 0.95*(1-fx),
						A: 1,
					}
				},
			},
			{
				Name:   "black_shadow_srgb_straight",
				Space:  SRGB,
				Premul: false,
				Render: func(x, y, eye int) Sample {
					coverage := glyphCoverage(x-1, y-1)
					return Sample{R: 0, G: 0, B: 0, A: 0.55 * coverage * eyeFactor("shadow", eye)}
				},
			},
			{
				Name:   "colored_text_srgb_premultiplied",
				Space:  SRGB,
				Premul: true,
				Render: func(x, y, eye int) Sample {
					coverage := glyphCoverage(x, y)
					alpha := coverage * eyeFactor("text", eye)
					r, g, b := textColor(x)
					return Sample{R: r * alpha, G: g * alpha, B: b * alpha, A: alpha}
				},
			},
			{
				Name:   "highlight_srgb_straight",
				Space:  SRGB,
				Premul: false,
				Render: func(x, y, eye int) Sample {
					alpha := 0.48 * highlightCoverage(x, y) * eyeFactor("highlight", eye)
					return Sample{R: 1, G: 0.91, B: 0.28, A: alpha}
				},
			},
		},
	}
}
