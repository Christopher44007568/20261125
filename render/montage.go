package render

import (
	"image"
	"image/color"

	"brepsection/geom"
	"brepsection/section"
)

// Montage draws the section reports in a two-row grid (the float64 run on
// top, the float32 stress run below). Each panel fits the section loops into
// its frame, fills hole loops, strokes every contour and annotates nesting
// level and closure status.
func Montage(reports []*section.Report, width, panelH int) *Canvas {
	pad := 24
	titleH := 30
	footH := 22
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
		c.Rect(x0, y0, panelW, panelH, palette.Panel)
		c.Frame(x0, y0, panelW, panelH, 1, palette.Edge)
		c.DrawText(x0+8, y0-titleH+8, 2, shortName(rep.PlaneName)+" "+rep.ModeName, palette.Text)
		drawPanel(c, rep, x0+12, y0+12, panelW-24, panelH-24)

		status := "CLOSED"
		if !rep.Closed {
			status = "OPEN"
		}
		c.DrawText(x0+8, y0+panelH+5, 1,
			status+" LOOPS "+itoa(len(rep.Loops))+" ORDER "+boolWord(rep.OrderConsistent),
			palette.Text)
	}
	return c
}

// shortName reduces long plane names so two-row titles never overlap.
func shortName(name string) string {
	switch name {
	case "z-extrusion-cut":
		return "Z AXIS CUT"
	case "x-tangent-via":
		return "X TANGENT VIA"
	case "oblique-center-cut":
		return "OBLIQUE CUT"
	default:
		return name
	}
}

func drawPanel(c *Canvas, rep *section.Report, x, y, w, h int) {
	if len(rep.Loops) == 0 {
		return
	}
	min, max := loopBounds(rep)
	span := max.Sub(min)
	scale := float64(w-20) / span.X
	if sy := float64(h-20) / span.Y; sy < scale {
		scale = sy
	}
	toPx := func(p geom.Vec2) image.Point {
		px := x + 10 + int((p.X-min.X)*scale)
		py := y + 10 + int((p.Y-min.Y)*scale)
		return image.Point{X: px, Y: py}
	}

	// Fill holes first so outer contours remain visible on top.
	for _, l := range rep.Loops {
		if l.Level%2 == 1 {
			pts := make([]image.Point, len(l.Nodes))
			for k, p := range l.Nodes {
				pts[k] = toPx(p)
			}
			c.Polygon(pts, palette.HoleFill)
		}
	}

	for _, l := range rep.Loops {
		col := palette.Section
		if l.Level%2 == 1 {
			col = color.RGBA{40, 110, 200, 255}
		}
		n := len(l.Nodes)
		for k := 0; k < n; k++ {
			a, b := toPx(l.Nodes[k]), toPx(l.Nodes[(k+1)%n])
			c.Line(a.X, a.Y, b.X, b.Y, 2, col)
		}
		for _, p := range l.Nodes {
			px := toPx(p)
			c.disc(px.X, px.Y, 2, palette.Edge)
		}
		// Nesting level label at the loop representative point.
		rp := section.RepresentativePoint(l.Nodes)
		pp := toPx(rp)
		c.DrawText(pp.X-3, pp.Y-3, 1, itoa(l.Level), palette.Text)
	}
}

func loopBounds(rep *section.Report) (geom.Vec2, geom.Vec2) {
	first := rep.Loops[0].Nodes[0]
	lo, hi := first, first
	for _, l := range rep.Loops {
		for _, p := range l.Nodes {
			lo = geom.V2(minF(lo.X, p.X), minF(lo.Y, p.Y))
			hi = geom.V2(maxF(hi.X, p.X), maxF(hi.Y, p.Y))
		}
	}
	if hi.X-lo.X < 1e-9 {
		hi.X = lo.X + 1
	}
	if hi.Y-lo.Y < 1e-9 {
		hi.Y = lo.Y + 1
	}
	return lo, hi
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [12]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func boolWord(v bool) string {
	if v {
		return "OK"
	}
	return "BAD"
}
