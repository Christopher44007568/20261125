package render

import (
	"gamutmap/gamut"
)

type skinTone struct {
	label string
	srgb  gamut.RGB
}

var skinTones = []skinTone{
	{"LIGHT", gamut.RGB{R: 0.956, G: 0.827, B: 0.761}},
	{"LIGHT-MED", gamut.RGB{R: 0.878, G: 0.690, B: 0.561}},
	{"MEDIUM", gamut.RGB{R: 0.753, G: 0.514, B: 0.376}},
	{"TAN", gamut.RGB{R: 0.553, G: 0.353, B: 0.243}},
	{"DEEP", gamut.RGB{R: 0.369, G: 0.227, B: 0.173}},
}

// SkinTonesPNG renders realistic and deliberately chroma-boosted skin samples.
func SkinTonesPNG(path string) error {
	const (
		width  = 1500
		height = 870
	)
	c := NewCanvas(width, height)
	c.FillRect(0, 0, width, height, bgColor)
	DrawText(c, 36, 28, 4, "SKIN TONE COMPARISON", textColor)
	DrawText(c, 38, 68, 2, "BASE SKIN STAYS IN GAMUT / BOOSTED CHROMA EXPOSES MAPPING DIFFERENCES", mutedColor)

	drawSkinPanel(c, 36, 112, 700, 708, gamut.Clip)
	drawSkinPanel(c, 764, 112, 700, 708, gamut.Perceptual)
	DrawMaskLegend(c, 38, 834)
	DrawText(c, width-492, 836, 2, "NEUTRAL ROW IS IDENTICAL IN BOTH PANELS", mutedColor)
	return c.Save(path)
}

func drawSkinPanel(c *Canvas, x, y, w, h int, method gamut.Method) {
	label := "DIRECT CHANNEL CLIP"
	if method == gamut.Perceptual {
		label = "PERCEPTUAL OKLCH COMPRESSION"
	}
	DrawPanel(c, x, y, w, h, label)

	const (
		leftLabel = 64
		gutter    = 10
		pad       = 18
	)
	gridX := x + pad + leftLabel
	gridW := w - 2*pad - leftLabel
	cellW := (gridW - 2*gutter) / 3
	topY := y + 80
	cellH := 82
	rowGap := 10
	voicings := []struct {
		label  string
		factor float64
	}{
		{"BASE", 1},
		{"CHROMA X2", 2},
		{"CHROMA X3", 3},
	}
	for col, v := range voicings {
		cx := gridX + col*(cellW+gutter)
		DrawText(c, cx, y+52, 1, v.label, mutedColor)
	}

	for row, skin := range skinTones {
		ry := topY + row*(cellH+rowGap)
		DrawText(c, x+pad, ry+cellH/2-3, 1, skin.label, mutedColor)
		rec2020 := gamut.SRGBToRec2020(skin.srgb)
		for col, v := range voicings {
			cx := gridX + col*(cellW+gutter)
			source := saturationBoosted(rec2020, v.factor)
			out := MapSource(source, method)
			c.FillRect(cx, ry, cellW, cellH, ColorFromRGB(out.SRGB))
			for py := 0; py < cellH; py++ {
				for px := 0; px < cellW; px++ {
					BlendMask(c, cx+px, ry+py, out.OutOfGamut)
				}
			}
			c.StrokeRect(cx, ry, cellW, cellH, lineColor)
		}
	}

	neutralY := topY + len(skinTones)*(cellH+rowGap) + 6
	DrawText(c, x+pad, neutralY+6, 1, "NEUTRAL", mutedColor)
	neutrals := []gamut.RGB{
		{R: 0, G: 0, B: 0},
		{R: 0.18, G: 0.18, B: 0.18},
		{R: 0.5, G: 0.5, B: 0.5},
		{R: 0.82, G: 0.82, B: 0.82},
		{R: 1, G: 1, B: 1},
	}
	chipW := gridW / len(neutrals)
	for i, neutral := range neutrals {
		out := MapSource(neutral, method)
		col := ColorFromRGB(out.SRGB)
		cx := gridX + i*chipW
		c.FillRect(cx, neutralY, chipW-4, 18, col)
		c.StrokeRect(cx, neutralY, chipW-4, 18, lineColor)
	}
	DrawText(c, x+pad, y+h-38, 1, "SAME RGB ON NEUTRAL AXIS: NO SATURATION GAIN", accentColor)
}
