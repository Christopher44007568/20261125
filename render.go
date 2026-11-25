package main

import "math"

var (
	colBarbA  = RGB{0.62, 0.49, 0.30}
	colBarbB  = RGB{0.84, 0.72, 0.50}
	colShaft  = RGB{0.93, 0.83, 0.62}
	colCore   = RGB{0.38, 0.26, 0.14}
	colAnchor = RGB{0.95, 0.45, 0.12}
	colArrow  = RGB{0.10, 0.72, 0.82}
)

type Map struct {
	X0, Y0, Scale float64
	H             int
}

func (m Map) p(v Vec) (float64, float64) {
	return m.X0 + v.X*m.Scale, float64(m.H) - (m.Y0 + v.Y*m.Scale)
}

func heatColor(t float64) RGB {
	stops := []struct {
		p float64
		c RGB
	}{
		{0, RGB{0.05, 0.20, 0.75}},
		{0.45, RGB{0.10, 0.75, 0.85}},
		{0.72, RGB{0.98, 0.82, 0.18}},
		{1, RGB{0.90, 0.16, 0.12}},
	}
	for i := 0; i < len(stops)-1; i++ {
		a, b := stops[i], stops[i+1]
		if t <= b.p {
			u := (t - a.p) / (b.p - a.p)
			return RGB{
				lerp(a.c.R, b.c.R, u),
				lerp(a.c.G, b.c.G, u),
				lerp(a.c.B, b.c.B, u),
			}
		}
	}
	return stops[len(stops)-1].c
}

func mixColor(a, b RGB, t float64) RGB {
	return RGB{lerp(a.R, b.R, t), lerp(a.G, b.G, t), lerp(a.B, b.B, t)}
}

func drawBarb(l *Layer, m Map, b Barb, steps int, heat bool) {
	var prevX, prevY float64
	var prevR float64
	var prevC RGB
	for i := 0; i < steps; i++ {
		u0 := float64(i) / float64(steps)
		u1 := float64(i+1) / float64(steps)
		u := (u0 + u1) / 2
		pt := barbPoint(b, u)
		w := barbWeight(b, u)
		x, y := m.p(pt)
		var c RGB
		if heat {
			c = heatColor(w)
		} else {
			c = mixColor(colBarbA, colBarbB, b.T)
		}
		radius := b.Width * m.Scale * (0.32 + 0.68*math.Sqrt(w))
		if i > 0 {
			l.line(prevX, prevY, x, y, math.Max(prevR, radius)*0.92, prevC)
		}
		l.stamp(x, y, radius, w, c)
		prevX, prevY, prevR, prevC = x, y, radius, c
	}
}

func drawRachis(l *Layer, m Map, heat bool) {
	steps := 130
	for i := 0; i < steps; i++ {
		t0 := float64(i) / float64(steps)
		t1 := float64(i+1) / float64(steps)
		p0 := rachisPoint(t0)
		p1 := rachisPoint(t1)
		x0, y0 := m.p(p0)
		x1, y1 := m.p(p1)
		w := rachisWidth((t0 + t1) / 2)
		c := colShaft
		if heat {
			c = heatColor(1 - 0.25*(t0+t1)/2)
		}
		l.line(x0, y0, x1, y1, w*m.Scale*0.5, c)
	}
	steps = 110
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(steps)
		x, y := m.p(rachisPoint(t))
		l.stamp(x, y, rachisWidth(t)*m.Scale*0.16, 0.9, colCore)
	}
	x, y := m.p(rachisPoint(0))
	l.disc(x, y, rachisWidth(0)*m.Scale*0.62, colShaft)
}

func drawAnchors(l *Layer, m Map, f Feather, every int, radius float64) {
	for i, a := range f.Anchors {
		if every > 1 && i%every != 0 {
			continue
		}
		x, y := m.p(a)
		l.ring(x, y, radius, radius*0.16, colAnchor)
	}
}

func drawArrow(l *Layer, x, y, angle, size, radius float64, c RGB) {
	dir := Vec{math.Cos(angle), math.Sin(angle)}
	tip := Vec{x, y}
	start := tip.Sub(dir.Scale(size))
	l.line(start.X, start.Y, tip.X, tip.Y, radius, c)
	side := dir.Rot(2.45)
	l.line(tip.X, tip.Y, tip.X+side.X*size*0.34, tip.Y+side.Y*size*0.34, radius, c)
	side = dir.Rot(-2.45)
	l.line(tip.X, tip.Y, tip.X+side.X*size*0.34, tip.Y+side.Y*size*0.34, radius, c)
}

func drawTangentArrows(l *Layer, m Map, f Feather, indices []int) {
	for _, idx := range indices {
		for _, b := range f.Barbs {
			if b.Anchor != idx {
				continue
			}
			u := 0.52
			pt := barbPoint(b, u)
			tan := barbTangent(b, u)
			x, y := m.p(pt)
			sx, sy := m.p(pt.Add(tan))
			angle := math.Atan2(sy-y, sx-x)
			drawArrow(l, x, y, angle, 30, 3.2, colArrow)
		}
	}
}

func drawWindArrow(l *Layer, x, y float64, f Feather) {
	angle := math.Atan2(f.Wind.Y, f.Wind.X)
	drawArrow(l, x, y, angle, 150, 7, colArrow)
}

func drawWeightLabels(l *Layer, m Map, f Feather, anchor int, us []float64) {
	b := Barb{}
	for _, cand := range f.Barbs {
		if cand.Anchor == anchor && cand.Side > 0 {
			b = cand
			break
		}
	}
	for _, u := range us {
		pt := barbPoint(b, u)
		x, y := m.p(pt)
		w := barbWeight(b, u)
		labelY := y - 34 + 76*u
		l.line(x, y, x+10, labelY+4, 1.6, RGB{1, 1, 1})
		drawText(l, int(x+12), int(labelY), 2, formatWeight(w), RGB{1, 1, 1})
		l.ring(x, y, 5, 1.4, RGB{1, 1, 1})
	}
}

func drawColorbar(l *Layer, x, y, w, h int) {
	for i := 0; i < h; i++ {
		t := 1 - float64(i)/float64(h-1)
		for j := 0; j < w; j++ {
			l.stamp(float64(x+j)+0.5, float64(y+i)+0.5, 1, 1, heatColor(t))
		}
	}
	for i, label := range []string{"1.0", "0.5", "0.0"} {
		yy := y + i*h/2
		drawText(l, x+w+14, yy-8, 2, label, RGB{0.12, 0.14, 0.18})
	}
}
