package main

import (
	"fmt"
	"image/color"
)

func rgba(rgb [3]uint8, alpha uint8) color.RGBA {
	return color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: alpha}
}

func withAlpha(c color.RGBA, alpha uint8) color.RGBA {
	c.A = alpha
	return c
}

func renderEyeFrame(
	canvas *Canvas,
	scenario Scenario,
	layer Layer,
	eye Eye,
	geometries []ElementGeometry,
) FloatRect {
	panelBounds := ProjectPanelRect(layer, eye)
	accent := rgba(layer.AccentRGB, 255)

	canvas.FillRect(panelBounds, color.RGBA{R: 15, G: 23, B: 42, A: 242})
	canvas.StrokeRect(panelBounds, 3.0, accent)

	buttonFill := color.RGBA{R: 30, G: 41, B: 59, A: 245}
	for _, rectSpec := range layer.Rects {
		rect := ProjectWorldRect(layer, rectSpec, eye)
		if rectSpec.Button {
			canvas.FillRect(rect, buttonFill)
			canvas.StrokeRect(rect, 2.0, accent)
		} else {
			canvas.FillRect(rect, withAlpha(accent, 180))
		}
	}

	textColor := color.RGBA{R: 248, G: 250, B: 252, A: 255}
	for _, geo := range geometries {
		for _, pair := range geo.Glyphs {
			var glyph EyeGlyph
			if eye.ID == LeftEye {
				glyph = pair.Left
			} else {
				glyph = pair.Right
			}
			if glyph.Valid {
				canvas.DrawPlacedGlyph(glyph.View, textColor)
				if geo.Spec.Style.Bold {
					bold := glyph.View
					bold.Origin.X += 0.55
					canvas.DrawPlacedGlyph(bold, textColor)
				}
			}
		}
	}

	safe := eye.SafeRect()
	canvas.StrokeRect(safe, 1.4, withAlpha(accent, 105))

	cx, cy := eye.principalX(), eye.principalY()
	guide := withAlpha(accent, 170)
	canvas.FillRect(FloatRect{MinX: cx - 9, MinY: cy - 1, MaxX: cx + 9, MaxY: cy + 1}, guide)
	canvas.FillRect(FloatRect{MinX: cx - 1, MinY: cy - 9, MaxX: cx + 1, MaxY: cy + 9}, guide)

	hud := color.RGBA{R: 203, G: 213, B: 225, A: 255}
	canvas.DrawScreenText(
		fmt.Sprintf("%s / %s / %s", scenario.ID, eye.ID, eye.Tracking.State),
		18, 16, 2.0, hud,
	)
	canvas.DrawScreenText("SAFE RECT + PROJECTION CENTER", 18, float64(eye.Height)-34, 1.6, withAlpha(hud, 210))

	return panelBounds
}
