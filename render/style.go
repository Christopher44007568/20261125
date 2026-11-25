package render

import (
	"image/color"

	"gamutmap/gamut"
)

var (
	bgColor      = color.NRGBA{9, 11, 16, 255}
	panelColor   = color.NRGBA{20, 24, 32, 255}
	lineColor    = color.NRGBA{53, 61, 75, 255}
	textColor    = color.NRGBA{238, 242, 248, 255}
	mutedColor   = color.NRGBA{142, 153, 170, 255}
	accentColor  = color.NRGBA{255, 76, 135, 255}
	maskDark     = color.NRGBA{4, 5, 8, 150}
	maskLight    = color.NRGBA{255, 255, 255, 60}
	neutralColor = color.NRGBA{215, 220, 228, 255}
)

func MapSource(source gamut.RGB, method gamut.Method) gamut.Result {
	return gamut.MapRec2020ToSRGB(source, method)
}

func BlendMask(c *Canvas, x, y int, outOfGamut bool) {
	if !outOfGamut {
		return
	}
	phase := positiveMod(x+y, 14)
	switch {
	case phase < 4:
		c.Blend(x, y, maskDark)
	case phase < 6:
		c.Blend(x, y, maskLight)
	}
}

func DrawMaskLegend(c *Canvas, x, y int) {
	c.FillRect(x, y, 34, 22, panelColor)
	for py := 0; py < 22; py++ {
		for px := 0; px < 34; px++ {
			BlendMask(c, x+px, y+py, true)
		}
	}
	DrawText(c, x+48, y+4, 2, "OUTSIDE SRGB / MASKED", mutedColor)
}

func DrawPanel(c *Canvas, x, y, w, h int, label string) {
	c.FillRect(x, y, w, h, panelColor)
	c.StrokeRect(x, y, w, h, lineColor)
	c.FillRect(x+1, y+1, w-2, 38, color.NRGBA{10, 13, 19, 255})
	c.FillRect(x+16, y+15, 4, 10, accentColor)
	DrawText(c, x+28, y+12, 2, label, textColor)
}

func positiveMod(v, mod int) int {
	r := v % mod
	if r < 0 {
		return r + mod
	}
	return r
}
