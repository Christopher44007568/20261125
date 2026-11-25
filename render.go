package main

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"
)

type ViewKind int

const (
	ViewFront ViewKind = iota
	ViewSide
	ViewTop
	ViewThreeQuarter
)

type Point2 struct {
	X float64
	Y float64
}

type Rect struct {
	X float64
	Y float64
	W float64
	H float64
}

type SceneOptions struct {
	Title            string
	Subtitle         string
	HairColor        string
	HeatHair         bool
	Ghost            bool
	MigrationScaleMM float64
	ShowPoints       bool
	ShowTrajectories bool
	ShowHeader       bool
}

func viewName(v ViewKind) string {
	switch v {
	case ViewSide:
		return "侧面 Side"
	case ViewTop:
		return "顶面 Top"
	case ViewThreeQuarter:
		return "三分之四视角 3/4"
	default:
		return "正面 Front"
	}
}

func projectPoint(local Vec3, pose Pose, view ViewKind) (Point2, float64) {
	p := pose.Apply(local)
	switch view {
	case ViewSide:
		return Point2{X: p.Z, Y: -p.Y}, p.X
	case ViewTop:
		return Point2{X: p.X, Y: -p.Z}, p.Y
	case ViewThreeQuarter:
		const angle = 0.62
		c, s := math.Cos(angle), math.Sin(angle)
		x := c*p.X - s*p.Z
		z := s*p.X + c*p.Z
		return Point2{X: x, Y: -p.Y}, z
	default:
		return Point2{X: p.X, Y: -p.Y}, p.Z
	}
}

func cameraNormal(view ViewKind) Vec3 {
	switch view {
	case ViewSide:
		return V(1, 0, 0)
	case ViewTop:
		return V(0, 1, 0)
	case ViewThreeQuarter:
		const angle = 0.62
		return V(math.Sin(angle), 0, math.Cos(angle)).Norm()
	default:
		return V(0, 0, 1)
	}
}

type sceneTransform struct {
	rect   Rect
	scale  float64
	center Point2
}

func newSceneTransform(rect Rect) sceneTransform {
	scale := math.Min((rect.W-46)/1.62, (rect.H-78)/1.86)
	return sceneTransform{
		rect:   rect,
		scale:  scale,
		center: Point2{X: rect.X + rect.W*0.5, Y: rect.Y + rect.H*0.43},
	}
}

func (st sceneTransform) mapPoint(p Point2) Point2 {
	return Point2{
		X: st.center.X + p.X*st.scale,
		Y: st.center.Y + p.Y*st.scale,
	}
}

func (st sceneTransform) project(local Vec3, pose Pose, view ViewKind) (Point2, float64) {
	p, depth := projectPoint(local, pose, view)
	return st.mapPoint(p), depth
}

func svgHeader(b *strings.Builder, width, height int, title string) {
	fmt.Fprintf(b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s">`,
		width, height, width, height, html.EscapeString(title))
	fmt.Fprintf(b, "<style>text{font-family:'Noto Sans CJK SC','Source Han Sans SC','Microsoft YaHei',Arial,sans-serif;fill:#29313a}.mono{font-family:'JetBrains Mono','SFMono-Regular',Consolas,monospace}.title{font-weight:700}.muted{fill:#6b7280}</style>")
	fmt.Fprintf(b, `<rect width="%d" height="%d" fill="#f7f4ed"/>`, width, height)
}

func svgFooter(b *strings.Builder) {
	b.WriteString("</svg>\n")
}

func drawText(b *strings.Builder, x, y float64, size int, weight int, anchor, class, text string) {
	fmt.Fprintf(b, `<text x="%.2f" y="%.2f" font-size="%d" font-weight="%d" text-anchor="%s" class="%s">%s</text>`,
		x, y, size, weight, anchor, class, html.EscapeString(text))
}

func drawPanel(b *strings.Builder, r Rect, fill, stroke string) {
	fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="12" fill="%s" stroke="%s" stroke-width="1"/>`,
		r.X, r.Y, r.W, r.H, fill, stroke)
}

func cross2(a, b, c Point2) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func convexHull(points []Point2) []Point2 {
	if len(points) < 3 {
		return points
	}
	pts := append([]Point2(nil), points...)
	sort.Slice(pts, func(i, j int) bool {
		if pts[i].X != pts[j].X {
			return pts[i].X < pts[j].X
		}
		return pts[i].Y < pts[j].Y
	})
	lower := make([]Point2, 0, len(pts))
	for _, p := range pts {
		for len(lower) >= 2 && cross2(lower[len(lower)-2], lower[len(lower)-1], p) <= 0 {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, p)
	}
	upper := make([]Point2, 0, len(pts))
	for i := len(pts) - 1; i >= 0; i-- {
		p := pts[i]
		for len(upper) >= 2 && cross2(upper[len(upper)-2], upper[len(upper)-1], p) <= 0 {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, p)
	}
	return append(lower[:len(lower)-1], upper[:len(upper)-1]...)
}

func ellipsoidPoint(theta, phi float64) Vec3 {
	return V(
		scalpRadii.X*math.Cos(phi)*math.Cos(theta),
		scalpRadii.Y*math.Sin(phi),
		scalpRadii.Z*math.Cos(phi)*math.Sin(theta),
	)
}

func ellipsoidNormal(p Vec3) Vec3 {
	return V(
		p.X/(scalpRadii.X*scalpRadii.X),
		p.Y/(scalpRadii.Y*scalpRadii.Y),
		p.Z/(scalpRadii.Z*scalpRadii.Z),
	).Norm()
}

func drawHead(b *strings.Builder, pose Pose, view ViewKind, st sceneTransform) {
	projected := make([]Point2, 0, 15*29)
	for lat := -12; lat <= 12; lat++ {
		phi := float64(lat) * math.Pi / 26
		for lon := 0; lon < 29; lon++ {
			theta := float64(lon) * 2 * math.Pi / 29
			p, _ := st.project(ellipsoidPoint(theta, phi), pose, view)
			projected = append(projected, p)
		}
	}
	hull := convexHull(projected)
	if len(hull) > 2 {
		b.WriteString(`<path d="`)
		for i, p := range hull {
			if i == 0 {
				fmt.Fprintf(b, "M%.2f %.2f", p.X, p.Y)
			} else {
				fmt.Fprintf(b, "L%.2f %.2f", p.X, p.Y)
			}
		}
		b.WriteString(`Z" fill="#e6ded0" stroke="#b9ae9c" stroke-width="1.1"/>`)
	}
	cam := cameraNormal(view)
	drawWire := func(local []Vec3) {
		visible := make([]bool, len(local))
		points := make([]Point2, len(local))
		for i, p := range local {
			points[i], _ = st.project(p, pose, view)
			normal := pose.Rotate(ellipsoidNormal(p))
			visible[i] = normal.Dot(cam) > 0.02
		}
		started := false
		for i := range points {
			if !visible[i] {
				started = false
				continue
			}
			if !started {
				fmt.Fprintf(b, `<path d="M%.2f %.2f`, points[i].X, points[i].Y)
				started = true
			} else {
				fmt.Fprintf(b, "L%.2f %.2f", points[i].X, points[i].Y)
			}
			if i+1 == len(points) || !visible[i+1] {
				b.WriteString(`" fill="none" stroke="#c8bdab" stroke-width="0.55" opacity="0.65"/>`)
				started = false
			}
		}
	}
	for lon := 0; lon < 18; lon++ {
		theta := float64(lon) * math.Pi / 9
		line := make([]Vec3, 0, 25)
		for lat := -12; lat <= 12; lat++ {
			phi := float64(lat) * math.Pi / 28
			line = append(line, ellipsoidPoint(theta, phi))
		}
		drawWire(line)
	}
	for lat := -4; lat <= 4; lat++ {
		phi := float64(lat) * math.Pi / 12
		line := make([]Vec3, 0, 37)
		for lon := 0; lon <= 36; lon++ {
			theta := float64(lon) * math.Pi / 18
			line = append(line, ellipsoidPoint(theta, phi))
		}
		drawWire(line)
	}
}

func catmullControls(points []Vec3, i int) (Vec3, Vec3, Vec3, Vec3) {
	last := len(points) - 1
	p0 := points[max(i-1, 0)]
	p1 := points[i]
	p2 := points[i+1]
	p3 := points[min(i+2, last)]
	c1 := p1.Add(p2.Sub(p0).Div(6))
	c2 := p2.Sub(p3.Sub(p1).Div(6))
	return p1, c1, c2, p2
}

type rgb struct {
	R int
	G int
	B int
}

var heatStops = []struct {
	T float64
	C rgb
}{
	{0.00, rgb{37, 99, 235}},
	{0.27, rgb{6, 182, 212}},
	{0.52, rgb{234, 179, 8}},
	{0.76, rgb{249, 115, 22}},
	{1.00, rgb{220, 38, 38}},
}

func heatRGB(t float64) rgb {
	t = clamp(t, 0, 1)
	for i := 1; i < len(heatStops); i++ {
		if t <= heatStops[i].T {
			a, b := heatStops[i-1], heatStops[i]
			u := (t - a.T) / (b.T - a.T)
			return rgb{
				R: int(math.Round(lerp(float64(a.C.R), float64(b.C.R), u))),
				G: int(math.Round(lerp(float64(a.C.G), float64(b.C.G), u))),
				B: int(math.Round(lerp(float64(a.C.B), float64(b.C.B), u))),
			}
		}
	}
	return heatStops[len(heatStops)-1].C
}

func heatColor(t float64) string {
	c := heatRGB(t)
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

func drawHeatLegend(b *strings.Builder, x, y, w float64, maxMM float64) {
	drawText(b, x, y-10, 12, 700, "start", "", "控制点迁移 A → B · migration (mm)")
	const steps = 48
	for i := 0; i < steps; i++ {
		cell := w / steps
		fmt.Fprintf(b, `<rect x="%.3f" y="%.2f" width="%.3f" height="14" fill="%s"/>`,
			x+float64(i)*cell, y, cell+0.25, heatColor(float64(i)/float64(steps-1)))
	}
	drawText(b, x, y+31, 10, 500, "start", "mono", "0")
	drawText(b, x+w/2, y+31, 10, 500, "middle", "mono", fmt.Sprintf("%.0f mm", maxMM*0.5))
	drawText(b, x+w, y+31, 10, 500, "end", "mono", fmt.Sprintf("%.0f mm", maxMM))
}

func buildCatmullPath(points []Vec3, pose Pose, view ViewKind, st sceneTransform) (string, []Point2) {
	projected := make([]Point2, len(points))
	for i, p := range points {
		projected[i], _ = st.project(p, pose, view)
	}
	var path strings.Builder
	fmt.Fprintf(&path, "M%.2f %.2f", projected[0].X, projected[0].Y)
	for i := 0; i+1 < len(points); i++ {
		_, c1, c2, p2 := catmullControls(points, i)
		q1, _ := st.project(c1, pose, view)
		q2, _ := st.project(c2, pose, view)
		q3, _ := st.project(p2, pose, view)
		fmt.Fprintf(&path, "C%.2f %.2f %.2f %.2f %.2f %.2f",
			q1.X, q1.Y, q2.X, q2.Y, q3.X, q3.Y)
	}
	return path.String(), projected
}

func drawStrand(b *strings.Builder, strand Strand, link Link, pose Pose, view ViewKind, st sceneTransform, opts SceneOptions) {
	path, projected := buildCatmullPath(strand.Controls, pose, view, st)
	underWidth := math.Max(1.25, strand.Width*0.62)
	if !opts.Ghost {
		fmt.Fprintf(b, `<path d="%s" fill="none" stroke="#1e2933" stroke-width="%.2f" stroke-linecap="round" stroke-linejoin="round" opacity="0.40"/>`,
			path, underWidth)
	}
	opacity := 0.94
	if opts.Ghost {
		opacity = 0.22
	}
	if !opts.HeatHair {
		fmt.Fprintf(b, `<path d="%s" fill="none" stroke="%s" stroke-width="%.2f" stroke-linecap="round" stroke-linejoin="round" opacity="%.2f"/>`,
			path, opts.HairColor, math.Max(0.82, strand.Width*0.40), opacity)
	} else {
		for level := 0; level+1 < len(strand.Controls); level++ {
			_, c1, c2, p2 := catmullControls(strand.Controls, level)
			q0 := projected[level]
			q1, _ := st.project(c1, pose, view)
			q2, _ := st.project(c2, pose, view)
			q3, _ := st.project(p2, pose, view)
			mig := (link.MigrationMM[level] + link.MigrationMM[level+1]) * 0.5
			norm := 0.0
			if opts.MigrationScaleMM > 1e-12 {
				norm = mig / opts.MigrationScaleMM
			}
			width := math.Max(0.82, strand.Width*0.43)
			fmt.Fprintf(b, `<path d="M%.2f %.2fC%.2f %.2f %.2f %.2f %.2f %.2f" fill="none" stroke="%s" stroke-width="%.2f" stroke-linecap="round" opacity="%.2f"/>`,
				q0.X, q0.Y, q1.X, q1.Y, q2.X, q2.Y, q3.X, q3.Y,
				heatColor(norm), width, opacity)
		}
	}
	if opts.ShowPoints && !opts.Ghost {
		for level, p := range projected {
			norm := 0.0
			if opts.MigrationScaleMM > 1e-12 {
				norm = link.MigrationMM[level] / opts.MigrationScaleMM
			}
			r := 1.05
			if level == 0 {
				r = 1.45
			}
			if level > 0 {
				r = 0.82
			}
			fmt.Fprintf(b, `<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s" stroke="#fff" stroke-width="0.35" opacity="0.92"/>`,
				p.X, p.Y, r, heatColor(norm))
		}
	}
}

func drawArrow(b *strings.Builder, a, c Point2, color string, width, opacity float64) {
	dx, dy := c.X-a.X, c.Y-a.Y
	l := math.Hypot(dx, dy)
	if l < 3 {
		return
	}
	fmt.Fprintf(b, `<path d="M%.2f %.2fL%.2f %.2f" fill="none" stroke="%s" stroke-width="%.2f" stroke-dasharray="2.6 3.0" opacity="%.2f"/>`,
		a.X, a.Y, c.X, c.Y, color, width, opacity)
	ux, uy := dx/l, dy/l
	tip := c
	base := Point2{X: c.X - ux*5, Y: c.Y - uy*5}
	left := Point2{X: base.X - uy*2.2, Y: base.Y + ux*2.2}
	right := Point2{X: base.X + uy*2.2, Y: base.Y - ux*2.2}
	fmt.Fprintf(b, `<path d="M%.2f %.2fL%.2f %.2fL%.2f %.2fZ" fill="%s" opacity="%.2f"/>`,
		tip.X, tip.Y, left.X, left.Y, right.X, right.Y, color, opacity)
}

func drawTrajectories(b *strings.Builder, frame Frame, view ViewKind, st sceneTransform, scaleMM float64) {
	step := max(1, len(frame.Links)/13)
	for j := 0; j < len(frame.Links); j += step {
		link := frame.Links[j]
		if !nearSide(frame.Strands[j].Root(), frame.Pose, view) {
			continue
		}
		for level := 1; level < len(link.MigrationMM); level += 2 {
			a, _ := st.project(link.aControls[level], frame.Pose, view)
			c, _ := st.project(link.bControls[level], frame.Pose, view)
			if math.Hypot(c.X-a.X, c.Y-a.Y) < 3 {
				continue
			}
			norm := 0.0
			if scaleMM > 1e-12 {
				norm = link.MigrationMM[level] / scaleMM
			}
			drawArrow(b, a, c, heatColor(norm), 0.8, 0.58)
		}
	}
}

type depthStrand struct {
	index int
	depth float64
}

func nearSide(root Vec3, pose Pose, view ViewKind) bool {
	normal := pose.Rotate(scalpNormal(root))
	return normal.Dot(cameraNormal(view)) >= 0
}

func drawHairLayer(b *strings.Builder, frame Frame, view ViewKind, st sceneTransform, opts SceneOptions, near bool) {
	depths := make([]depthStrand, len(frame.Strands))
	count := 0
	for i, strand := range frame.Strands {
		if nearSide(strand.Root(), frame.Pose, view) != near {
			continue
		}
		total := 0.0
		for _, p := range strand.Controls {
			_, depth := st.project(p, frame.Pose, view)
			total += depth
		}
		depths[count] = depthStrand{index: i, depth: total / float64(len(strand.Controls))}
		count++
	}
	depths = depths[:count]
	sort.SliceStable(depths, func(i, j int) bool { return depths[i].depth < depths[j].depth })
	for _, entry := range depths {
		drawStrand(b, frame.Strands[entry.index], frame.Links[entry.index], frame.Pose, view, st, opts)
	}
}

func drawHairCap(b *strings.Builder, frame Frame, view ViewKind, st sceneTransform, opts SceneOptions) {
	points := make([]Point2, 0, len(frame.Strands))
	for _, strand := range frame.Strands {
		if !nearSide(strand.Root(), frame.Pose, view) {
			continue
		}
		p, _ := st.project(strand.Root(), frame.Pose, view)
		points = append(points, p)
	}
	hull := convexHull(points)
	if len(hull) < 3 {
		return
	}
	cx, cy := 0.0, 0.0
	for _, p := range hull {
		cx += p.X
		cy += p.Y
	}
	cx /= float64(len(hull))
	cy /= float64(len(hull))
	fill := opts.HairColor
	if opts.HeatHair || fill == "" {
		fill = "#25313b"
	}
	b.WriteString(`<path d="`)
	for i, p := range hull {
		q := Point2{X: cx + (p.X-cx)*1.08, Y: cy + (p.Y-cy)*1.08}
		if i == 0 {
			fmt.Fprintf(b, "M%.2f %.2f", q.X, q.Y)
		} else {
			fmt.Fprintf(b, "L%.2f %.2f", q.X, q.Y)
		}
	}
	fmt.Fprintf(b, `Z" fill="%s" opacity="0.17"/>`, fill)
}

func drawScene(b *strings.Builder, frame Frame, view ViewKind, r Rect, opts SceneOptions) {
	drawPanel(b, r, "#fffdfa", "#d9d2c5")
	if opts.ShowHeader {
		drawText(b, r.X+14, r.Y+23, 13, 700, "start", "title", opts.Title)
		if opts.Subtitle != "" {
			drawText(b, r.X+14, r.Y+41, 10, 500, "start", "mono muted", opts.Subtitle)
		}
	}
	st := newSceneTransform(r)
	farOpts := opts
	farOpts.Ghost = true
	farOpts.ShowPoints = false
	farOpts.ShowTrajectories = false
	drawHairLayer(b, frame, view, st, farOpts, false)
	drawHead(b, frame.Pose, view, st)
	drawHairCap(b, frame, view, st, opts)
	drawHairLayer(b, frame, view, st, opts, true)
	if opts.ShowTrajectories {
		drawTrajectories(b, frame, view, st, opts.MigrationScaleMM)
	}
	drawText(b, r.X+r.W-13, r.Y+r.H-12, 10, 600, "end", "muted", viewName(view))
}

func renderThreeView(run ScenarioRun) string {
	const (
		width  = 1260
		height = 1460
		left   = 150.0
		top    = 110.0
		right  = 28.0
		bottom = 28.0
		gap    = 13.0
	)
	b := &strings.Builder{}
	svgHeader(b, width, height, run.Scenario.Title+" 三视图")
	drawText(b, 26, 35, 22, 800, "start", "title", "Groom interpolation · 中间姿态三视图")
	drawText(b, 26, 58, 12, 500, "start", "muted", "A / B / blend · unequal strand counts · deterministic scalp-atlas pairing")
	drawText(b, 26, 78, 11, 500, "start", "mono muted", fmt.Sprintf("A=%d  B=%d  blend=%d  scale=1 unit / %.0f mm", run.Report.StrandsA, run.Report.StrandsB, run.Report.BlendStrands, worldScaleMM))
	drawHeatLegend(b, float64(width)-245, 38, 210, run.Report.MigrationScaleMM)
	cellW := (width - left - right - 2*gap) / 3
	cellH := (height - top - bottom - 2*gap) / 3
	views := []ViewKind{ViewFront, ViewSide, ViewTop}
	for col, view := range views {
		x := left + float64(col)*(cellW+gap)
		drawText(b, x+cellW/2, top-14, 13, 700, "middle", "title", viewName(view))
	}
	frames := []Frame{run.Frames[0], run.Frames[len(run.Frames)-1], run.Frames[len(run.Frames)/2]}
	rows := []struct {
		label  string
		detail string
		color  string
		heat   bool
		points bool
		frame  Frame
	}{
		{"GROOM A", run.Scenario.A.Name, "#0f766e", true, false, frames[0]},
		{"GROOM B", run.Scenario.B.Name, "#b45309", true, false, frames[1]},
		{"BLEND · t=0.50", "Blend 混合结果", "", true, false, frames[2]},
	}
	for row, data := range rows {
		y := top + float64(row)*(cellH+gap)
		drawText(b, 24, y+34, 13, 800, "start", "title", data.label)
		drawText(b, 24, y+55, 11, 600, "start", "", data.detail)
		drawText(b, 24, y+74, 10, 500, "start", "mono muted", fmt.Sprintf("%d strands", len(data.frame.Strands)))
		if data.color != "" {
			fmt.Fprintf(b, `<rect x="24" y="%.2f" width="18" height="4" rx="2" fill="%s"/>`, y+86, data.color)
		} else {
			for i := 0; i < 9; i++ {
				fmt.Fprintf(b, `<rect x="%.2f" y="%.2f" width="2.2" height="4" fill="%s"/>`,
					24+float64(i)*2, y+86, heatColor(float64(i)/8))
			}
		}
		for col, view := range views {
			x := left + float64(col)*(cellW+gap)
			r := Rect{X: x, Y: y, W: cellW, H: cellH}
			opts := SceneOptions{
				Title:            data.label,
				Subtitle:         data.detail,
				HairColor:        data.color,
				HeatHair:         data.heat,
				MigrationScaleMM: run.Report.MigrationScaleMM,
				ShowPoints:       data.points,
				ShowTrajectories: false,
				ShowHeader:       false,
			}
			drawScene(b, data.frame, view, r, opts)
		}
	}
	drawText(b, 26, height-9, 10, 500, "start", "muted",
		"色标：发丝控制点从 Groom A 到 Groom B 的位移；混合发丝本身按对应控制点位移着色。")
	svgFooter(b)
	return b.String()
}

func renderMotionSheet(run ScenarioRun) string {
	const (
		width  = 1740
		height = 520
		left   = 24.0
		top    = 92.0
		gap    = 12.0
	)
	b := &strings.Builder{}
	svgHeader(b, width, height, run.Scenario.Title+" 中间帧")
	drawText(b, 24, 34, 21, 800, "start", "title", "Interpolation frames · 中间帧尖端包络")
	drawText(b, 24, 58, 11, 500, "start", "muted", "控制点凸包约束：每帧尖端半径不超过两端 groom 的线性包络。 tip envelope ≤ endpoint envelope")
	drawHeatLegend(b, float64(width)-245, 37, 210, run.Report.MigrationScaleMM)
	panelW := (width - 2*left - 4*gap) / 5
	panelH := 360.0
	for i := 0; i < 5; i++ {
		frameIndex := i * (len(run.Frames) - 1) / 4
		frame := run.Frames[frameIndex]
		x := left + float64(i)*(panelW+gap)
		r := Rect{X: x, Y: top, W: panelW, H: panelH}
		opts := SceneOptions{
			Title:            fmt.Sprintf("t = %.2f", frame.T),
			Subtitle:         fmt.Sprintf("%d strands", len(frame.Strands)),
			HeatHair:         true,
			MigrationScaleMM: run.Report.MigrationScaleMM,
			ShowPoints:       false,
			ShowHeader:       true,
		}
		drawScene(b, frame, ViewThreeQuarter, r, opts)
		metric := run.Report.Frames[frameIndex*2]
		drawText(b, x+14, top+panelH+23, 10, 500, "start", "mono muted",
			fmt.Sprintf("tip envelope %.3f · void %d", metric.MaxTipRadius, metric.TipBoundViolations))
	}
	drawText(b, 24, height-18, 10, 500, "start", "muted",
		"发丝数量随 t 连续取整：t=0 精确为 A，t=1 精确为 B；配对始终使用 scalp atlas 顺序，不使用随机索引。")
	svgFooter(b)
	return b.String()
}

func renderControlMigration(run ScenarioRun) string {
	const (
		width  = 1120
		height = 820
	)
	b := &strings.Builder{}
	svgHeader(b, width, height, run.Scenario.Title+" 控制点迁移")
	drawText(b, 30, 38, 22, 800, "start", "title", "Control-point migration · 控制点迁移")
	drawText(b, 30, 61, 12, 500, "start", "muted", "细虚线箭头：抽样发丝的控制点 A→B 轨迹；颜色与发丝热图使用同一 mm 标尺。 dashed arrows: sampled A→B control-point paths")
	mainRect := Rect{X: 28, Y: 92, W: 710, H: 690}
	frame := run.Frames[len(run.Frames)/2]
	drawScene(b, frame, ViewThreeQuarter, mainRect, SceneOptions{
		Title:            "混合结果",
		Subtitle:         fmt.Sprintf("t=0.50 · %d strands", len(frame.Strands)),
		HeatHair:         true,
		MigrationScaleMM: run.Report.MigrationScaleMM,
		ShowPoints:       true,
		ShowTrajectories: true,
		ShowHeader:       true,
	})
	side := Rect{X: 758, Y: 92, W: 334, H: 690}
	drawPanel(b, side, "#fffdfa", "#d9d2c5")
	drawHeatLegend(b, side.X+28, side.Y+54, side.W-56, run.Report.MigrationScaleMM)
	drawText(b, side.X+24, side.Y+128, 13, 800, "start", "title", "确定性配对")
	lines := []string{
		fmt.Sprintf("Groom A  %d 根", run.Report.StrandsA),
		fmt.Sprintf("Groom B  %d 根", run.Report.StrandsB),
		fmt.Sprintf("t=0.50    %d 根", run.Report.BlendStrands),
		"",
		"根部按 Hilbert scalp atlas 排序，",
		"再按分位重采样到常见中间根数。",
		"数量不足时只混合 atlas 中相邻",
		"发丝；没有随机索引或乱序配对。",
	}
	for i, line := range lines {
		drawText(b, side.X+24, side.Y+158+float64(i)*22, 11, 500, "start",
			"mono muted", line)
	}
	drawText(b, side.X+24, side.Y+365, 13, 800, "start", "title", "验收")
	stats := []string{
		fmt.Sprintf("最大迁移      %.1f mm", run.Report.MigrationScaleMM),
		fmt.Sprintf("尖端越界      %d", run.Report.Frames[len(run.Report.Frames)/2].TipBoundViolations),
		fmt.Sprintf("尖端包络比    %.4f", run.Report.TipEnvelopeRatio),
		fmt.Sprintf("控制点越界    %d", run.Report.Frames[len(run.Report.Frames)/2].ControlViolations),
		fmt.Sprintf("分段越界      %d", run.Report.Frames[len(run.Report.Frames)/2].SpanViolations),
		"",
		"hash " + run.Report.DeterminismHash[:16],
	}
	for i, line := range stats {
		drawText(b, side.X+24, side.Y+395+float64(i)*22, 11, 500, "start",
			"mono", line)
	}
	drawText(b, side.X+24, side.Y+610, 12, 700, "start", "", "读图")
	readLines := []string{
		"蓝色：位移接近最小",
		"黄/橙：中等位移",
		"红色：位移最大，多位于根部",
		"圆点：每个发丝控制点",
	}
	for i, line := range readLines {
		drawText(b, side.X+24, side.Y+635+float64(i)*20, 10, 500, "start", "muted", line)
	}
	svgFooter(b)
	return b.String()
}
