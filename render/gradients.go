package render

import (
	"math"

	"gamutmap/gamut"
)

type gradientStop func(t float64) gamut.RGB

type gradientRow struct {
	label string
	hint  string
	stop  gradientStop
}

// GradientsPNG renders four ramps through both mapping methods.
func GradientsPNG(path string) error {
	const (
		width  = 1500
		height = 900
	)
	c := NewCanvas(width, height)
	c.FillRect(0, 0, width, height, bgColor)
	DrawText(c, 36, 28, 4, "GRADIENT STRESS TEST", textColor)
	DrawText(c, 38, 68, 2, "HIGH-CHROMA REC.2020 RAMPS / MASK SHOWS SRGB EXCEEDANCE", mutedColor)

	rows := []gradientRow{
		{
			label: "RED CHROMA: NEUTRAL TO REC.2020 RED",
			hint:  "CLIPPING FLATTENS THE TOP SATURATION RANGE",
			stop: func(t float64) gamut.RGB {
				return hsvToRec2020(0.005, 0.02+0.96*t, 0.94)
			},
		},
		{
			label: "RED FAMILY: CONSTANT RED, SHIFTING GREEN/BLUE",
			hint:  "EVERY STOP IS OUT OF GAMUT; CLIP COLLAPSES MANY TO ONE RED",
			stop: func(t float64) gamut.RGB {
				return gamut.RGB{R: 1, G: 0.20 * (1 - t), B: 0.20 * t}
			},
		},
		{
			label: "FULL HUE SWEEP AT HIGH CHROMA",
			hint:  "HUE DIRECTION IS RETAINED BY BOTH SCHEMES",
			stop: func(t float64) gamut.RGB {
				return hsvToRec2020(t, 0.95, 0.92)
			},
		},
		{
			label: "NEUTRAL AXIS: BLACK TO WHITE",
			hint:  "NO MASK, NO TINT, NO SATURATION ADDED",
			stop: func(t float64) gamut.RGB {
				v := 0.02 + 0.96*t
				return gamut.RGB{R: v, G: v, B: v}
			},
		},
	}

	drawGradientPanel(c, 36, 112, 700, 726, gamut.Clip, rows)
	drawGradientPanel(c, 764, 112, 700, 726, gamut.Perceptual, rows)
	DrawMaskLegend(c, 38, 854)
	DrawText(c, width-356, 856, 2, "MASK = SOURCE EXCEEDS SRGB", mutedColor)
	return c.Save(path)
}

func drawGradientPanel(c *Canvas, x, y, w, h int, method gamut.Method, rows []gradientRow) {
	label := "DIRECT CHANNEL CLIP"
	if method == gamut.Perceptual {
		label = "PERCEPTUAL OKLCH COMPRESSION"
	}
	DrawPanel(c, x, y, w, h, label)
	rowHeight := (h - 58) / len(rows)
	for i, row := range rows {
		ry := y + 48 + i*rowHeight
		DrawText(c, x+22, ry, 2, row.label, textColor)
		DrawText(c, x+22, ry+18, 1, row.hint, mutedColor)
		drawGradientStrip(c, x+22, ry+36, w-44, rowHeight-52, method, row.stop)
	}
}

func drawGradientStrip(c *Canvas, x, y, w, h int, method gamut.Method, stop gradientStop) {
	for px := 0; px < w; px++ {
		t := float64(px) / float64(w-1)
		source := stop(t)
		out := MapSource(source, method)
		col := ColorFromRGB(out.SRGB)
		for py := 0; py < h; py++ {
			c.Set(x+px, y+py, col)
			BlendMask(c, x+px, y+py, out.OutOfGamut)
		}
	}
	c.StrokeRect(x, y, w, h, lineColor)
}

func saturationBoosted(c gamut.RGB, factor float64) gamut.RGB {
	h, s, v := rec2020ToHSV(c)
	return hsvToRec2020(h, math.Min(0.98, s*factor), v)
}
