package render

import (
	"image/color"
	"math"

	"gamutmap/gamut"
)

// ColorWheelPNG renders Rec.2020 hue wheels through both mapping methods.
func ColorWheelPNG(path string) error {
	const (
		width  = 1500
		height = 840
	)
	c := NewCanvas(width, height)
	c.FillRect(0, 0, width, height, bgColor)
	DrawText(c, 36, 30, 4, "REC.2020 TO SRGB: HIGH-CHROMA HUE WHEEL", textColor)
	DrawText(c, 38, 70, 2, "VIRTUAL REC.2020 WHEEL / SAME OUT-OF-GAMUT MASK IN BOTH PANELS", mutedColor)

	drawWheelPanel(c, 36, 112, 700, 646, gamut.Clip)
	drawWheelPanel(c, 764, 112, 700, 646, gamut.Perceptual)

	DrawMaskLegend(c, 38, 786)
	DrawText(c, width-462, 788, 2, "NEUTRAL CENTER STAYS NEUTRAL", mutedColor)
	return c.Save(path)
}

func drawWheelPanel(c *Canvas, x, y, w, h int, method gamut.Method) {
	label := "DIRECT CHANNEL CLIP"
	if method == gamut.Perceptual {
		label = "PERCEPTUAL OKLCH COMPRESSION"
	}
	DrawPanel(c, x, y, w, h, label)

	cx := float64(x + w/2)
	cy := float64(y + 38 + (h-38)/2)
	radius := float64(min(w, h-38)/2 - 34)
	drawWheelDisc(c, cx, cy, radius, method)

	hint := "CHANNEL CLAMPING COLLAPSES RED DETAIL"
	if method == gamut.Perceptual {
		hint = "HUE + LIGHTNESS HELD, CHROMA REDUCED"
	}
	DrawText(c, x+42, y+h-34, 2, hint, mutedColor)
}

func drawWheelDisc(c *Canvas, cx, cy, radius float64, method gamut.Method) {
	x0 := int(math.Floor(cx - radius - 2))
	x1 := int(math.Ceil(cx + radius + 2))
	y0 := int(math.Floor(cy - radius - 2))
	y1 := int(math.Ceil(cy + radius + 2))
	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			dx := float64(px) + 0.5 - cx
			dy := float64(py) + 0.5 - cy
			dist := math.Hypot(dx, dy)
			if dist > radius+1 {
				continue
			}

			var sum gamut.RGB
			var count float64
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					ox := (float64(sx) + 0.5) / 2
					oy := (float64(sy) + 0.5) / 2
					sdx := dx - 0.5 + ox
					sdy := dy - 0.5 + oy
					sdist := math.Hypot(sdx, sdy)
					if sdist > radius {
						continue
					}
					hue := math.Mod(math.Atan2(sdy, sdx)+math.Pi/2, 2*math.Pi) / (2 * math.Pi)
					saturation := math.Min(0.97, sdist/radius*1.04)
					source := hsvToRec2020(hue, saturation, 0.92)
					out := MapSource(source, method).SRGB
					sum.R += out.R
					sum.G += out.G
					sum.B += out.B
					count++
				}
			}
			if count == 0 {
				continue
			}
			avg := gamut.RGB{R: sum.R / count, G: sum.G / count, B: sum.B / count}
			// Edge coverage blends the disc into the dark panel.
			coverage := math.Max(0, math.Min(1, radius+0.5-dist))
			col := ColorFromRGB(avg)
			if coverage < 1 {
				col = blendOver(panelColor, col, coverage)
			}
			c.Set(px, py, col)

			centerSource := wheelSourceAt(dx, dy, radius)
			BlendMask(c, px, py, gamut.MapRec2020ToSRGB(centerSource, gamut.Clip).OutOfGamut)
		}
	}
}

func wheelSourceAt(dx, dy, radius float64) gamut.RGB {
	hue := math.Mod(math.Atan2(dy, dx)+math.Pi/2, 2*math.Pi) / (2 * math.Pi)
	saturation := math.Min(0.97, math.Hypot(dx, dy)/radius*1.04)
	return hsvToRec2020(hue, saturation, 0.92)
}

func hsvToRec2020(h, s, v float64) gamut.RGB {
	h = math.Mod(math.Mod(h, 1)+1, 1)
	hh := h * 6
	i := math.Floor(hh)
	f := hh - i
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	switch int(i) % 6 {
	case 0:
		return gamut.RGB{R: v, G: t, B: p}
	case 1:
		return gamut.RGB{R: q, G: v, B: p}
	case 2:
		return gamut.RGB{R: p, G: v, B: t}
	case 3:
		return gamut.RGB{R: p, G: q, B: v}
	case 4:
		return gamut.RGB{R: t, G: p, B: v}
	default:
		return gamut.RGB{R: v, G: p, B: q}
	}
}

func blendOver(bottom, top color.NRGBA, alpha float64) color.NRGBA {
	a := math.Max(0, math.Min(1, alpha))
	return color.NRGBA{
		R: uint8(math.Round(float64(top.R)*a + float64(bottom.R)*(1-a))),
		G: uint8(math.Round(float64(top.G)*a + float64(bottom.G)*(1-a))),
		B: uint8(math.Round(float64(top.B)*a + float64(bottom.B)*(1-a))),
		A: 255,
	}
}
