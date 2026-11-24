package main

import "fmt"
import "image"

var (
	uiBg        = RGB(244, 247, 251)
	uiInk       = RGB(28, 38, 54)
	uiMuted     = RGB(92, 105, 122)
	uiCard      = RGB(255, 255, 255)
	uiLine      = RGB(214, 222, 232)
	uiBlue      = RGB(69, 155, 219)
	uiOrange    = RGB(235, 157, 64)
	uiGreen     = RGB(34, 166, 110)
	uiRed       = RGB(219, 58, 52)
	uiDarkPanel = RGB(235, 240, 247)
)

func drawWindowTitle(img *image.RGBA, x, y int, title, subtitle string) {
	drawText(img, x, y, title, 3, uiInk)
	drawText(img, x, y+34, subtitle, 2, uiMuted)
}

func drawCard(img *image.RGBA, x, y, w, h int) {
	fillRect(img, x, y, w, h, uiCard)
	strokeRect(img, x, y, w, h, 2, uiLine)
}

func drawLegendDot(img *image.RGBA, x, y int, c Color, label string) {
	fillRect(img, x, y, 18, 18, c)
	strokeRect(img, x, y, 18, 18, 1, uiInk)
	drawText(img, x+28, y+3, label, 2, uiInk)
}

func drawMetricLine(img *image.RGBA, x, y int, label, value string, valueColor Color) {
	drawText(img, x, y, label, 2, uiMuted)
	drawText(img, x, y+24, value, 2, valueColor)
}

func passBadge(img *image.RGBA, x, y int, text string, ok bool) {
	c := uiGreen
	if !ok {
		c = uiRed
	}
	w := textWidth(text, 2) + 28
	fillRect(img, x, y, w, 34, c)
	drawText(img, x+14, y+9, text, 2, RGB(255, 255, 255))
}

func combineMeshes(name string, meshes ...*Mesh) *Mesh {
	out := &Mesh{Name: name}
	for _, src := range meshes {
		offset := len(out.Pos)
		out.Pos = append(out.Pos, src.Pos...)
		for _, t := range src.Tri {
			out.Tri = append(out.Tri, Triangle{
				A: t.A + offset, B: t.B + offset, C: t.C + offset,
				Group: t.Group, Filled: t.Filled,
			})
		}
	}
	return out
}

func defaultLook() MeshLook {
	return MeshLook{
		ColorA:    RGB(67, 157, 214),
		ColorB:    RGB(236, 164, 72),
		FillColor: RGB(255, 92, 64),
		FillTint:  0.16,
		Wireframe: true,
		WireColor: Color{R: 24, G: 35, B: 52, A: 58},
		WireWidth: 0.7,
	}
}

func zoomLook(after bool) MeshLook {
	look := defaultLook()
	look.WireColor = Color{R: 18, G: 28, B: 44, A: 120}
	look.WireWidth = 1.0
	if after {
		look.FillTint = 0.34
	} else {
		look.FillTint = 0
	}
	return look
}

func formatMM(v float64) string {
	if v == 0 {
		return "0 MM"
	}
	if v < 0.01 {
		return fmt.Sprintf("%.4F MM", v)
	}
	return fmt.Sprintf("%.3F MM", v)
}
