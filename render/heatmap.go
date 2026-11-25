package render

import (
	"image/color"
	"math"
	"strconv"

	"brepsection/geom"
	"brepsection/section"
)

// Heatmap draws one panel per report. Each junction error is rendered as a
// Gaussian splat colored on a jet scale normalized to that panel's maximum,
// so even tiny residuals stay visually comparable. A color bar and the raw
// maximum are annotated for quantitative reading.
func Heatmap(reports []*section.Report, width, panelH int) *Canvas {
	pad := 24
	titleH := 30
	footH := 22
	barW := 26
	cols := len(reports) / 2
	if cols < 1 {
		cols = 1
	}
	rows := (len(reports) + cols - 1) / cols
	panelW := (width - pad*(cols+1)) / cols
	height := pad + rows*(titleH+panelH+footH+pad)
	c := NewCanvas(width, height, palette.Bg)

	for i, rep := range reports {
		row, col := i/cols, i%cols
		x0 := pad + col*(panelW+pad)
		y0 := pad + row*(titleH+panelH+footH+pad) + titleH
		c.Rect(x0, y0, panelW, panelH, color.RGBA{20, 22, 30, 255})
		c.Frame(x0, y0, panelW, panelH, 1, palette.Edge)
		c.DrawText(x0+8, y0-titleH+8, 2, shortName(rep.PlaneName)+" "+rep.ModeName, palette.Text)

		maxErr := rep.MaxGap
		if maxErr <= 0 {
			maxErr = rep.Tolerance
		}
		drawHeatPanel(c, rep, x0+8, y0+8, panelW-barW-20, panelH-16, maxErr)
		drawColorBar(c, x0+panelW-barW-6, y0+8, barW, panelH-16, maxErr)

		c.DrawText(x0+8, y0+panelH+5, 1,
			"MAX GAP "+strconv.FormatFloat(rep.MaxGap, 'G', 3, 64),
			palette.Text)
	}
	return c
}

func drawHeatPanel(c *Canvas, rep *section.Report, x, y, w, h int, maxErr float64) {
	if len(rep.NodeErrors) == 0 {
		return
	}
	var lo, hi geom.Vec2
	lo = rep.NodeErrors[0].Pos
	hi = lo
	for _, ne := range rep.NodeErrors {
		lo = geom.V2(minF(lo.X, ne.Pos.X), minF(lo.Y, ne.Pos.Y))
		hi = geom.V2(maxF(hi.X, ne.Pos.X), maxF(hi.Y, ne.Pos.Y))
	}
	span := hi.Sub(lo)
	if span.X < 1e-9 {
		span.X = 1
	}
	if span.Y < 1e-9 {
		span.Y = 1
	}
	scale := float64(w) / span.X
	if sy := float64(h) / span.Y; sy < scale {
		scale = sy
	}
	radius := 10.0
	toPx := func(p geom.Vec2) (float64, float64) {
		return float64(x) + (p.X-lo.X)*scale, float64(y) + (p.Y-lo.Y)*scale
	}

	for _, ne := range rep.NodeErrors {
		cx, cy := toPx(ne.Pos)
		t := ne.Err / maxErr
		col := jetColor(t)
		r := radius * (0.6 + 0.8*t)
		for dy := -int(r); dy <= int(r); dy++ {
			for dx := -int(r); dx <= int(r); dx++ {
				d2 := float64(dx*dx + dy*dy)
				if d2 > r*r {
					continue
				}
				alpha := 0.25 + 0.75*math.Exp(-d2/(2*r*r*0.18))
				c.Blend(int(cx)+dx, int(cy)+dy, col, alpha)
			}
		}
	}
}

func drawColorBar(c *Canvas, x, y, w, h int, maxErr float64) {
	for i := 0; i < h; i++ {
		t := 1 - float64(i)/float64(h)
		c.Rect(x, y+i, w, 1, jetColor(t))
	}
	top := strconv.FormatFloat(maxErr, 'G', 2, 64)
	c.DrawText(x-4, y-12, 1, top, palette.Text)
	c.DrawText(x-4, y+h+2, 1, "0", palette.Text)
}
