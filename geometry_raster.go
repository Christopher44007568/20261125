package main

import (
	"math"
	"sort"
)

func (r *renderer) buildGlassRows() frameRows {
	rows := make(frameRows, r.ih)
	for _, blade := range r.scene.Glass {
		minX, minY, maxX, maxY := polyBounds(blade.Points)
		x0 := clampInt(int(math.Floor(minX*float64(r.ss)))-1, 0, r.iw-1)
		x1 := clampInt(int(math.Ceil(maxX*float64(r.ss)))+1, 0, r.iw-1)
		y0 := clampInt(int(math.Floor(minY*float64(r.ss)))-1, 0, r.ih-1)
		y1 := clampInt(int(math.Ceil(maxY*float64(r.ss)))+1, 0, r.ih-1)
		for iy := y0; iy <= y1; iy++ {
			py := (float64(iy) + 0.5) / float64(r.ss)
			for ix := x0; ix <= x1; ix++ {
				px := (float64(ix) + 0.5) / float64(r.ss)
				p := Vec2{X: px, Y: py}
				if !pointInPoly(p, blade.Points) {
					continue
				}
				edgeDist := distanceToPoly(p, blade.Points)
				edge := 1 - smoothstep(0, blade.EdgeW*2.4, edgeDist)
				interior := 0.82 + 0.18*math.Sin((px*0.018+py*0.011)+blade.Tilt)
				col := blade.Fill.Mul(interior).Lerp(blade.Edge, edge*0.48)
				a := blade.Alpha * (0.78 + 0.46*edge)
				rows[iy] = append(rows[iy], FrameSample{
					X: int32(ix), Depth: blade.Depth, LayerID: int32(blade.ID),
					Kind: KindGlass, Color: col, Alpha: clamp01(a),
					Sequence: uint32(blade.ID) * 64,
				})
			}
		}
	}
	for y := range rows {
		sort.Slice(rows[y], func(i, j int) bool { return rows[y][i].X < rows[y][j].X })
	}
	return rows
}

func (r *renderer) buildFrameRows(t float64) frameRows {
	rows := make(frameRows, r.ih)
	for i := range r.scene.Strands {
		st := &r.scene.Strands[i]
		r.rasterHair(rows, st, t)
	}
	for i := range r.scene.Opaque {
		r.rasterOpaque(rows, &r.scene.Opaque[i], t)
	}
	for i := range r.scene.Ink {
		r.rasterFeature(rows, &r.scene.Ink[i], t)
	}
	for y := range rows {
		if len(rows[y]) > 1 {
			sort.Slice(rows[y], func(i, j int) bool { return rows[y][i].X < rows[y][j].X })
		}
	}
	return rows
}

func (r *renderer) rasterHair(rows frameRows, st *Strand, t float64) {
	length := 0.0
	last := st.pointAt(0, t, r.scene.Pivot)
	for i := 1; i <= 20; i++ {
		u := float64(i) / 20
		p := st.pointAt(u, t, r.scene.Pivot)
		length += math.Hypot(p.X-last.X, p.Y-last.Y)
		last = p
	}
	steps := int(length / (0.72 / float64(r.ss)))
	if steps < 16 {
		steps = 16
	}
	if steps > 1800 {
		steps = 1800
	}
	r.touched = r.touched[:0]
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		p := st.pointAt(u, t, r.scene.Pivot)
		width := st.widthAt(u)
		rad := math.Max(0.34, width*0.5)
		feather := 0.55
		motion := st.motionAt(u, t, r.scene.Pivot)
		x0 := clampInt(int(math.Floor((p.X-rad)*float64(r.ss))), 0, r.iw-1)
		x1 := clampInt(int(math.Ceil((p.X+rad)*float64(r.ss))), 0, r.iw-1)
		y0 := clampInt(int(math.Floor((p.Y-rad)*float64(r.ss))), 0, r.ih-1)
		y1 := clampInt(int(math.Ceil((p.Y+rad)*float64(r.ss))), 0, r.ih-1)
		for iy := y0; iy <= y1; iy++ {
			py := (float64(iy) + 0.5) / float64(r.ss)
			dy := py - p.Y
			for ix := x0; ix <= x1; ix++ {
				px := (float64(ix) + 0.5) / float64(r.ss)
				dx := px - p.X
				dist := math.Hypot(dx, dy)
				a := 1 - smoothstep(rad-feather, rad, dist)
				if a <= 0 {
					continue
				}
				idx := iy*r.iw + ix
				b := &r.best[idx]
				if a <= b.alpha {
					continue
				}
				if b.alpha == 0 {
					r.touched = append(r.touched, idx)
				}
				*b = bestStamp{alpha: a, depth: p.Z, u: u, vx: motion.X, vy: motion.Y}
			}
		}
	}
	for _, idx := range r.touched {
		b := r.best[idx]
		iy, ix := idx/r.iw, idx%r.iw
		rows[iy] = append(rows[iy], FrameSample{
			X: int32(ix), Depth: b.depth, LayerID: int32(st.ID), Kind: KindHair,
			Color: st.colorAt(b.u), Alpha: clamp01(b.alpha * st.Alpha),
			Motion: Vec2{X: b.vx, Y: b.vy}, Sequence: uint32(st.ID) * 64,
		})
		r.best[idx] = bestStamp{}
	}
}

func (r *renderer) rasterOpaque(rows frameRows, sh *OpaqueShape, t float64) {
	pts := make([]Vec2, len(sh.Points))
	for i, p := range sh.Points {
		if sh.HeadSpace {
			pts[i] = projectHeadAt(p, t, r.scene.Pivot)
		} else {
			pts[i] = p
		}
	}
	minX, minY, maxX, maxY := polyBounds(pts)
	x0 := clampInt(int(math.Floor(minX*float64(r.ss))), 0, r.iw-1)
	x1 := clampInt(int(math.Ceil(maxX*float64(r.ss))), 0, r.iw-1)
	y0 := clampInt(int(math.Floor(minY*float64(r.ss))), 0, r.ih-1)
	y1 := clampInt(int(math.Ceil(maxY*float64(r.ss))), 0, r.ih-1)
	for iy := y0; iy <= y1; iy++ {
		py := (float64(iy) + 0.5) / float64(r.ss)
		for ix := x0; ix <= x1; ix++ {
			px := (float64(ix) + 0.5) / float64(r.ss)
			p := Vec2{X: px, Y: py}
			if !pointInPoly(p, pts) {
				continue
			}
			col := opaqueShade(sh, p, r.scene.Width, r.scene.Height)
			motion := Vec2{}
			if sh.HeadSpace {
				motion = headMotionAt(p, t, r.scene.Pivot)
			}
			alpha := sh.Alpha
			if alpha <= 0 {
				alpha = 1
			}
			rows[iy] = append(rows[iy], FrameSample{
				X: int32(ix), Depth: sh.Depth, LayerID: int32(sh.ID), Kind: sh.Kind,
				Color: col, Alpha: alpha, Motion: motion, Sequence: uint32(sh.ID) * 64,
			})
		}
	}
}

func (r *renderer) rasterFeature(rows frameRows, ft *Feature, t float64) {
	const subdiv = 20
	curve := make([]Vec2, 0, subdiv*(len(ft.Points)-1)+1)
	for i := 0; i < len(ft.Points)-1; i++ {
		p0 := ft.Points[maxInt(0, i-1)]
		p1 := ft.Points[i]
		p2 := ft.Points[i+1]
		p3 := ft.Points[minInt(len(ft.Points)-1, i+2)]
		for k := 0; k < subdiv; k++ {
			curve = append(curve, catmullRom(p0, p1, p2, p3, float64(k)/subdiv))
		}
	}
	curve = append(curve, ft.Points[len(ft.Points)-1])

	r.touched = r.touched[:0]
	for i, base := range curve {
		p := projectHeadAt(base, t, r.scene.Pivot)
		motion := headMotionAt(base, t, r.scene.Pivot)
		rad := math.Max(0.42, ft.Width*0.5)
		x0 := clampInt(int(math.Floor((p.X-rad)*float64(r.ss))), 0, r.iw-1)
		x1 := clampInt(int(math.Ceil((p.X+rad)*float64(r.ss))), 0, r.iw-1)
		y0 := clampInt(int(math.Floor((p.Y-rad)*float64(r.ss))), 0, r.ih-1)
		y1 := clampInt(int(math.Ceil((p.Y+rad)*float64(r.ss))), 0, r.ih-1)
		for iy := y0; iy <= y1; iy++ {
			py := (float64(iy) + 0.5) / float64(r.ss)
			for ix := x0; ix <= x1; ix++ {
				px := (float64(ix) + 0.5) / float64(r.ss)
				dist := math.Hypot(px-p.X, py-p.Y)
				a := 1 - smoothstep(rad-0.55, rad, dist)
				if a <= 0 {
					continue
				}
				idx := iy*r.iw + ix
				b := &r.best[idx]
				if a <= b.alpha {
					continue
				}
				if b.alpha == 0 {
					r.touched = append(r.touched, idx)
				}
				u := 0.0
				if len(curve) > 1 {
					u = float64(i) / float64(len(curve)-1)
				}
				*b = bestStamp{alpha: a, depth: ft.Depth, u: u, vx: motion.X, vy: motion.Y}
			}
		}
	}
	for _, idx := range r.touched {
		b := r.best[idx]
		iy, ix := idx/r.iw, idx%r.iw
		rows[iy] = append(rows[iy], FrameSample{
			X: int32(ix), Depth: ft.Depth, LayerID: int32(ft.ID), Kind: ft.Kind,
			Color: ft.Color, Alpha: clamp01(b.alpha * ft.Alpha),
			Motion: Vec2{X: b.vx, Y: b.vy}, Sequence: uint32(ft.ID) * 64,
		})
		r.best[idx] = bestStamp{}
	}
}

func opaqueShade(sh *OpaqueShape, p Vec2, w, h float64) Color {
	nx, ny := p.X/w, p.Y/h
	light := 0.76 + 0.28*(1-nx*0.45) + 0.10*(1-ny)
	c := sh.Color.Mul(light)
	if sh.Kind == KindSkin && sh.ID == 4 {
		cheek := math.Exp(-((nx-0.36)*(nx-0.36)*65 + (ny-0.36)*(ny-0.36)*95))
		c = c.Lerp(RGB(0.66, 0.33, 0.31), cheek*0.22)
		rim := math.Exp(-math.Pow((nx-0.322)*70, 2)) * smoothstep(0.22, 0.62, ny)
		c = c.Add(RGB(0.36, 0.19, 0.20).Mul(rim * 0.24))
	}
	return clampColor(c)
}

func headMotionAt(p Vec2, t float64, pivot Vec2) Vec2 {
	const dt = 0.0008
	a := projectHeadAt(p, t-dt, pivot)
	b := projectHeadAt(p, t+dt, pivot)
	return Vec2{X: (b.X - a.X) / (2 * dt), Y: (b.Y - a.Y) / (2 * dt)}
}

func polyBounds(pts []Vec2) (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
		maxX = math.Max(maxX, p.X)
		maxY = math.Max(maxY, p.Y)
	}
	return
}

func pointInPoly(p Vec2, pts []Vec2) bool {
	inside := false
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		a, b := pts[i], pts[j]
		if (a.Y > p.Y) != (b.Y > p.Y) &&
			p.X < (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y)+a.X {
			inside = !inside
		}
	}
	return inside
}

func distanceToPoly(p Vec2, pts []Vec2) float64 {
	best := math.Inf(1)
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		best = math.Min(best, pointSegmentDistance(p, pts[j], pts[i]))
	}
	return best
}

func pointSegmentDistance(p, a, b Vec2) float64 {
	ab := b.Sub(a)
	den := ab.Dot(ab)
	if den == 0 {
		return p.Sub(a).Len()
	}
	t := clamp01(p.Sub(a).Dot(ab) / den)
	return p.Sub(a.Add(ab.Mul(t))).Len()
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
