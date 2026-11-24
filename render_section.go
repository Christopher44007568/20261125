package main

import (
	"fmt"
	"math"
)

type Plot struct {
	X0, Y0, X1, Y1 float64
	MinX, MaxX     float64
	MinY, MaxY     float64
}

func (p Plot) Map(v Vec2) Vec2 {
	w := p.X1 - p.X0
	h := p.Y1 - p.Y0
	scale := math.Min(w/(p.MaxX-p.MinX), h/(p.MaxY-p.MinY))
	cx := (p.X0 + p.X1) / 2
	cy := (p.Y0 + p.Y1) / 2
	mx := (p.MinX + p.MaxX) / 2
	my := (p.MinY + p.MaxY) / 2
	return Vec2{
		X: cx + (v.X-mx)*scale,
		Y: cy - (v.Y-my)*scale,
	}
}

func (p Plot) Scale() float64 {
	w := p.X1 - p.X0
	h := p.Y1 - p.Y0
	return math.Min(w/(p.MaxX-p.MinX), h/(p.MaxY-p.MinY))
}

func renderSection(profile Profile, diag SectionDiagnostics) *Canvas {
	c := NewCanvas(1180, 820, 2)
	c.Fill(bgColor)
	c.Text(26, 18,
		fmt.Sprintf("SECTION Z=0   D=%.2f MM", profile.Params.Offset),
		2, textColor)

	mainPlot := Plot{
		X0: 24, Y0: 46, X1: 790, Y1: 765,
		MinX: -19.8, MaxX: 0.8, MinY: -0.8, MaxY: 19.8,
	}
	zoomPlot := Plot{
		X0: 815, Y0: 62, X1: 1150, Y1: 397,
		MinX: -3.3, MaxX: 0.8, MinY: -0.8, MaxY: 3.3,
	}
	drawSectionPlot(c, mainPlot, profile, diag, false)
	a := mainPlot.Map(Vec2{zoomPlot.MinX, zoomPlot.MinY})
	b := mainPlot.Map(Vec2{zoomPlot.MaxX, zoomPlot.MaxY})
	c.StrokeRect(a.X, b.Y, b.X, a.Y, 1.1, redColor)
	c.Text(815, 42, "CONCAVE CORNER ZOOM", 1, darkRed())
	drawSectionPlot(c, zoomPlot, profile, diag, true)
	drawLegend(c)
	return c.Downsample()
}

func darkRed() Color { return Color{152, 22, 29} }

func drawSectionPlot(c *Canvas, plot Plot, profile Profile,
	diag SectionDiagnostics, zoom bool) {
	c.SetClipRect(plot.X0, plot.Y0, plot.X1, plot.Y1)
	defer c.ClearClip()
	c.FillRect(plot.X0, plot.Y0, plot.X1, plot.Y1, panelColor)

	for _, tri := range diag.Triangles {
		fill := sectionFill
		if tri.Inverted {
			fill = redFill
		}
		c.FillTriangle2D([3]Vec2{
			plot.Map(tri.P[0]), plot.Map(tri.P[1]), plot.Map(tri.P[2]),
		}, fill)
	}
	drawGrid(c, plot)

	for _, tri := range diag.Triangles {
		if !tri.Inverted {
			continue
		}
		w := 0.8
		if zoom {
			w = 1.2
		}
		drawPolyline(c, []Vec2{tri.P[0], tri.P[1], tri.P[2], tri.P[0]},
			plot, w, redColor)
	}
	drawPolyline(c, profile.Outer, plot, 2.1, outerColor)
	drawPolyline(c, profile.Inner, plot, 2.1, innerColor)
	drawSelfLoops(c, plot, profile, diag, zoom)
	c.StrokeRect(plot.X0, plot.Y0, plot.X1, plot.Y1, 1.2, borderColor)
}

func drawGrid(c *Canvas, plot Plot) {
	for x := math.Ceil(plot.MinX); x <= math.Floor(plot.MaxX); x++ {
		a := plot.Map(Vec2{x, plot.MinY})
		b := plot.Map(Vec2{x, plot.MaxY})
		c.Line(a.X, a.Y, b.X, b.Y, 0.7, gridColor)
	}
	for y := math.Ceil(plot.MinY); y <= math.Floor(plot.MaxY); y++ {
		a := plot.Map(Vec2{plot.MinX, y})
		b := plot.Map(Vec2{plot.MaxX, y})
		c.Line(a.X, a.Y, b.X, b.Y, 0.7, gridColor)
	}
}

func drawPolyline(c *Canvas, pts []Vec2, plot Plot, width float64, col Color) {
	if len(pts) < 2 {
		return
	}
	last := plot.Map(pts[0])
	for i := 1; i < len(pts); i++ {
		next := plot.Map(pts[i])
		c.Line(last.X, last.Y, next.X, next.Y, width, col)
		last = next
	}
}

func drawSelfLoops(c *Canvas, plot Plot, profile Profile,
	diag SectionDiagnostics, zoom bool) {
	if len(diag.Intersections) == 0 {
		return
	}
	path := profile.Inner
	for _, hit := range diag.Intersections {
		p := plot.Map(hit.P)
		radius := 8.0
		if zoom {
			radius = 15
		}
		diamond := []Vec2{
			{p.X, p.Y - radius}, {p.X + radius, p.Y},
			{p.X, p.Y + radius}, {p.X - radius, p.Y},
		}
		c.FillTriangle2D([3]Vec2{diamond[0], diamond[1], diamond[2]}, redFill)
		c.FillTriangle2D([3]Vec2{diamond[0], diamond[2], diamond[3]}, redFill)
		for k := range diamond {
			a := diamond[k]
			b := diamond[(k+1)%len(diamond)]
			c.Line(a.X, a.Y, b.X, b.Y, 1.8, redColor)
		}
		c.Circle(p.X, p.Y, radius, 1.8, redColor)

		w := 2.8
		if zoom {
			w = 4.0
		}
		for _, seg := range []int{hit.I, hit.J} {
			if seg < 0 || seg+1 >= len(path) {
				continue
			}
			a, b := path[seg], path[seg+1]
			d := b.Sub(a)
			if d.Len() < 1e-12 {
				continue
			}
			u := d.Scale(0.35 / d.Len())
			left := hit.P.Sub(u)
			right := hit.P.Add(u)
			drawPolyline(c, []Vec2{left, right}, plot, w, redColor)
		}
		c.Disc(p.X, p.Y, 2.4, redColor)
	}
}

func drawLegend(c *Canvas) {
	x := 815.0
	y := 430.0
	c.Text(x, y, "LEGEND", 2, textColor)
	rows := []struct {
		label string
		kind  int
		col   Color
	}{
		{"OUTER SURFACE", 0, outerColor},
		{"INNER OFFSET", 0, innerColor},
		{"SECTION TRIANGLES", 1, sectionFill},
		{"INVERTED TRIS", 2, redColor},
		{"SELF-INTERSECTING LOOP", 3, redColor},
	}
	for i, row := range rows {
		yy := y + 42 + float64(i)*42
		switch row.kind {
		case 0:
			c.Line(x, yy, x+65, yy, 3, row.col)
		case 1:
			c.FillRect(x, yy-12, x+65, yy+12, row.col)
			c.StrokeRect(x, yy-12, x+65, yy+12, 1, borderColor)
		case 2:
			c.FillTriangle2D([3]Vec2{
				{x + 32, yy - 14}, {x + 63, yy + 13}, {x + 1, yy + 13},
			}, redFill)
			c.Line(x+32, yy-14, x+63, yy+13, 1.5, redColor)
			c.Line(x+63, yy+13, x+1, yy+13, 1.5, redColor)
			c.Line(x+1, yy+13, x+32, yy-14, 1.5, redColor)
		case 3:
			c.FillTriangle2D([3]Vec2{
				{x + 32, yy - 10}, {x + 47, yy}, {x + 32, yy + 10},
			}, redFill)
			c.FillTriangle2D([3]Vec2{
				{x + 32, yy - 10}, {x + 17, yy}, {x + 32, yy + 10},
			}, redFill)
			c.Circle(x+32, yy, 10, 1.8, redColor)
		}
		c.Text(x+82, yy-4, row.label, 1, textColor)
	}
	c.Text(x, y+270, "ALL FACES RETAINED", 1, mutedColor)
	c.Text(x, y+290, "VOLUME CONTINUOUS", 1, mutedColor)
}
