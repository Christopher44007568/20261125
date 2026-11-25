package main

import (
	"fmt"
	"math"
)

type metricRow struct {
	Label string
	Value string
	Unit  string
	Note  string
	Tone  string
}

func pageShell(s *SVG, index, kicker, title, subtitle string) {
	s.Raw("<title>" + esc(title) + "</title>\n")
	s.Def(`<linearGradient id="pageBG" x1="0" y1="0" x2="1" y2="1">
<stop offset="0" stop-color="#071017"/><stop offset="0.52" stop-color="#09151f"/><stop offset="1" stop-color="#061018"/>
</linearGradient>
<linearGradient id="accentLine" x1="0" y1="0" x2="1" y2="0">
<stop offset="0" stop-color="#58e1d2" stop-opacity="0.95"/><stop offset="1" stop-color="#58e1d2" stop-opacity="0"/>
</linearGradient>
<pattern id="fineGrid" width="28" height="28" patternUnits="userSpaceOnUse">
<path d="M 28 0 L 0 0 0 28" fill="none" stroke="#7e9aa6" stroke-width="0.6" opacity="0.055"/>
</pattern>
`)
	s.Rect(0, 0, float64(s.Width), float64(s.Height), "url(#pageBG)")
	s.Rect(0, 0, float64(s.Width), float64(s.Height), "url(#fineGrid)")
	s.Rect(0, 0, float64(s.Width), 4, "url(#accentLine)")
	s.Text(64, 48, 12, colCyan, "700", "start", "SURFACE / FIT LAB")
	s.Mono(64, 68, 10, colMuted, "400", "start", "BRep PREVIEW  ·  ROBUST SCAN RECONSTRUCTION")
	s.Mono(float64(s.Width)-64, 56, 11, colMuted, "400", "end", index)
	s.Text(64, 118, 34, colText, "600", "start", title)
	s.Text(64, 145, 14, colMuted, "400", "start", subtitle)
	s.Line(64, 168, float64(s.Width)-64, 168, colGrid, 1, 0.9)
	s.Line(64, float64(s.Height)-48, float64(s.Width)-64, float64(s.Height)-48, colGrid, 1, 0.9)
	s.Mono(64, float64(s.Height)-24, 10, colFaint, "400", "start", "GO STANDARD LIBRARY  /  DETERMINISTIC RUN  /  20261125")
	s.Mono(float64(s.Width)-64, float64(s.Height)-24, 10, colFaint, "400", "end", "UNITS: mm  ·  ORTHO-PERSPECTIVE PROJECTION")
}

func panel(s *SVG, x, y, w, h float64, title, subtitle string) {
	s.RectEx(x, y, w, h, colPanel, colGrid, 1, 0.96, 16)
	s.RectEx(x+1, y+1, w-2, 34, "#0f202a", "", 0, 0.65, 15)
	s.Circle(x+20, y+18, 3.4, colCyan, "", 0, 1)
	s.Text(x+32, y+23, 11, colText, "600", "start", title)
	if subtitle != "" {
		s.Mono(x+w-18, y+23, 9, colMuted, "400", "end", subtitle)
	}
}

func cornerTicks(s *SVG, x, y, w, h float64) {
	clr := colGrid2
	for _, p := range [][2]float64{{x, y}, {x + w, y}, {x, y + h}, {x + w, y + h}} {
		dx, dy := 1.0, 1.0
		if p[0] > x+w*0.5 {
			dx = -1
		}
		if p[1] > y+h*0.5 {
			dy = -1
		}
		s.Line(p[0], p[1], p[0]+dx*13, p[1], clr, 1.5, 0.8)
		s.Line(p[0], p[1], p[0], p[1]+dy*13, clr, 1.5, 0.8)
	}
}

func legendChip(s *SVG, x, y float64, kind, label string) float64 {
	switch kind {
	case "clean":
		s.Circle(x+7, y-3, 3.2, colCyan, "#071017", 1, 1)
	case "noise":
		s.Circle(x+7, y-3, 5.2, colOrange, "", 0, 0.14)
		s.Circle(x+7, y-3, 2.9, colOrange, "#071017", 1, 1)
	case "hole":
		s.RectEx(x, y-10, 14, 14, "#5d1f7d", "", 0, 1, 3)
		s.Line(x+2, y-8, x+12, y+2, "#e8b8ff", 0.8, 0.8)
		s.Line(x+2, y-3, x+12, y+7, "#e8b8ff", 0.8, 0.6)
	default:
		s.Line(x, y-3, x+14, y-3, colMuted, 1.5, 1)
	}
	s.Text(x+21, y, 11, colMuted, "500", "start", label)
	return x + 40 + float64(len(label))*7.15
}

func statRail(s *SVG, x, y, w float64, audit Audit, fit *FitResult, clean, noise, holes int) {
	s.RectEx(x, y, w, 774, colPanel, colGrid, 1, 0.96, 16)
	s.Text(x+22, y+31, 11, colMuted, "700", "start", "FIT ACCEPTANCE")
	s.Text(x+w-22, y+31, 10, colCyan, "700", "end", "CONVERGED")
	s.Line(x+22, y+46, x+w-22, y+46, colGrid, 1, 1)
	rows := []metricRow{
		{"CONTROL RESIDUAL", fmt.Sprintf("%.3f", fit.InlierRMS), "mm RMS", "inlier scan points", colCyan},
		{"DENSE HEIGHT", fmt.Sprintf("%.3f", audit.DenseRMSE), "mm RMS", fmt.Sprintf("%d audit samples", audit.DenseSamples), colCyan},
		{"NORMAL ERROR", fmt.Sprintf("%.2f", audit.NormalP95), "deg p95", fmt.Sprintf("%.2f deg RMS", audit.NormalRMSE), colGold},
		{"CURVATURE", fmt.Sprintf("%.1f", audit.CurvatureRelative), "% RMS", "mean curvature vs ref", colGold},
		{"RIPPLE ENERGY", fmt.Sprintf("%.1f", audit.RippleRatio), "%", "4 mm high-pass band", colGold},
		{"SILHOUETTE IoU", fmt.Sprintf("%.5f", audit.SilhouetteIoU), "", "projected area overlap", colCyan},
		{"BOUNDARY CHAMFER", fmt.Sprintf("%.3f", audit.SilhouetteChamfer), "mm", "trim loop deviation", colCyan},
	}
	yy := y + 76
	for i, r := range rows {
		if i > 0 {
			s.Line(x+22, yy-28, x+w-22, yy-28, colGrid, 1, 0.65)
		}
		s.Text(x+22, yy, 9, colMuted, "600", "start", r.Label)
		s.Mono(x+22, yy+28, 23, r.Tone, "500", "start", r.Value)
		if r.Unit != "" {
			s.Mono(x+22+float64(len(r.Value))*13.6, yy+27, 9, colFaint, "400", "start", r.Unit)
		}
		s.Mono(x+w-22, yy+8, 9, colFaint, "400", "end", r.Note)
		yy += 60
	}
	s.Line(x+22, yy-25, x+w-22, yy-25, colGrid, 1, 0.8)
	s.Text(x+22, yy+4, 9, colMuted, "600", "start", "INPUT COVERAGE")
	s.Mono(x+22, yy+29, 12, colText, "500", "start", fmt.Sprintf("%d CLEAN", clean))
	s.Mono(x+116, yy+29, 12, colOrange, "500", "start", fmt.Sprintf("%d NOISE", noise))
	s.Mono(x+210, yy+29, 12, colMagenta, "500", "start", fmt.Sprintf("%d HOLES", holes))
	s.Mono(x+22, yy+52, 9, colFaint, "400", "start",
		fmt.Sprintf("CONTROL NET %d×%d  ·  CUBIC B-SPLINE", fit.Options.NU, fit.Options.NV))
	s.Mono(x+22, yy+70, 9, colFaint, "400", "start",
		fmt.Sprintf("ROBUST IRLS %d PASSES  ·  λ %.2f", fit.Iterations, fit.Options.Lambda))
}

func gauge(s *SVG, x, y, w float64, value, max, target float64, label, tone string) {
	s.Text(x, y, 9, colMuted, "600", "start", label)
	s.RectEx(x, y+10, w, 5, "#122732", "", 0, 1, 2.5)
	fw := clamp(value/max, 0, 1) * w
	s.RectEx(x, y+10, fw, 5, tone, "", 0, 1, 2.5)
	tx := x + clamp(target/max, 0, 1)*w
	s.Line(tx, y+6, tx, y+19, colText, 1.2, 0.8)
}

func drawAxisGizmo(s *SVG, cam Camera, ox, oy, size float64) {
	right, up, view := cam.Basis()
	_ = view
	axes := []struct {
		V Vec3
		C string
		L string
	}{
		{Vec3{1, 0, 0}, colOrange, "X"},
		{Vec3{0, 1, 0}, colCyan, "Y"},
		{Vec3{0, 0, 1}, colGold, "Z"},
	}
	s.Circle(ox, oy, 26, "#0a151d", colGrid, 1, 0.9)
	for _, a := range axes {
		dx := a.V.Dot(right) * size
		dy := -a.V.Dot(up) * size
		s.Line(ox, oy, ox+dx, oy+dy, a.C, 1.8, 0.95)
		s.Circle(ox+dx, oy+dy, 2.5, a.C, "", 0, 1)
		s.Mono(ox+dx*1.22, oy+dy*1.22+3, 9, a.C, "600", "middle", a.L)
	}
}

func drawFloorGrid(s *SVG, cam Camera, z float64) {
	for i := -5; i <= 5; i++ {
		x := float64(i) * 20
		p1 := Vec3{x, -70, z}
		p2 := Vec3{x, 70, z}
		x1, y1, _ := cam.Project(p1)
		x2, y2, _ := cam.Project(p2)
		s.Line(x1, y1, x2, y2, colGrid, 0.8, 0.32)
		y := float64(i) * 18
		p1 = Vec3{-100, y, z}
		p2 = Vec3{100, y, z}
		x1, y1, _ = cam.Project(p1)
		x2, y2, _ = cam.Project(p2)
		s.Line(x1, y1, x2, y2, colGrid, 0.8, 0.32)
	}
}

func callout(s *SVG, x1, y1, x2, y2 float64, label, tone string) {
	s.Line(x1, y1, x2, y2, tone, 1, 0.55)
	s.Circle(x1, y1, 2.1, tone, "", 0, 1)
	s.Line(x2, y2, x2+42, y2, tone, 1, 0.75)
	w := math.Max(30, float64(len(label))*6.4+14)
	s.RectEx(x2+42, y2-10, w, 20, "#07121a", tone, 1, 0.85, 3)
	s.Mono(x2+49, y2+3, 8.5, tone, "600", "start", label)
}

func drawHoleOverlay(s *SVG, cam Camera, surface *BSplineSurface, label bool) {
	labelOffsets := []Vec2{{-34, -18}, {0, 0}, {36, -17}}
	for i, h := range scanHoles {
		var poly []Vec2
		var cx, cy float64
		for k := 0; k < 72; k++ {
			t := 2 * math.Pi * float64(k) / 72
			ca, sa := math.Cos(h.Angle), math.Sin(h.Angle)
			u, v := h.RX*math.Cos(t), h.RY*math.Sin(t)
			nx := h.CX + ca*u - sa*v
			ny := h.CY + sa*u + ca*v
			x, y := nx*PatchA, ny*PatchB
			p := Vec3{x, y, surface.Z(x, y) + 0.45}
			sx, sy, _ := cam.Project(p)
			poly = append(poly, Vec2{sx, sy})
			cx += sx
			cy += sy
		}
		cx /= 72
		cy /= 72
		s.Poly(poly, "#7b2ba0", "", 0, 0.56, true)
		hatchPolygon(s, poly, "#e8b8ff", 0.7, 0.50, 8)
		s.Segmented(append(append([]Vec2{}, poly...), poly[0]), colMagenta, 1.25, 1)
		if label {
			lx, ly := cx, cy
			if i < len(labelOffsets) {
				lx += labelOffsets[i].X
				ly += labelOffsets[i].Y
			}
			if lx != cx || ly != cy {
				s.Line(cx, cy, lx, ly, colMagenta, 0.8, 0.65)
			}
			s.Circle(lx, ly, 10, "#160b20", colMagenta, 1, 0.94)
			s.Mono(lx, ly+3, 8, colMagenta, "700", "middle", fmt.Sprintf("H%d", i+1))
		}
	}
}

func hatchPolygon(s *SVG, poly []Vec2, tone string, sw, opacity, spacing float64) {
	if len(poly) < 3 {
		return
	}
	minY, maxY := poly[0].Y, poly[0].Y
	for _, p := range poly {
		minY = math.Min(minY, p.Y)
		maxY = math.Max(maxY, p.Y)
	}
	for y := minY + spacing*0.55; y < maxY; y += spacing {
		var xs []float64
		for i := range poly {
			a, b := poly[i], poly[(i+1)%len(poly)]
			if (a.Y <= y && b.Y > y) || (b.Y <= y && a.Y > y) {
				t := (y - a.Y) / (b.Y - a.Y)
				xs = append(xs, a.X+(b.X-a.X)*t)
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			s.Line(xs[i], y, xs[i+1], y, tone, sw, opacity)
		}
	}
}

func drawNoisePoints(s *SVG, cam Camera, surface *BSplineSurface, points []ScanPoint, stems bool) {
	for _, p := range points {
		if p.Kind != PointNoise {
			continue
		}
		x, y, _ := cam.Project(Vec3{p.X, p.Y, p.Z})
		if stems {
			zx := pointZ(surface, p.X, p.Y) + 0.15
			x2, y2, _ := cam.Project(Vec3{p.X, p.Y, zx})
			s.DashedLine(x, y, x2, y2, colOrange, 0.8, 0.38, "2 4")
		}
		s.Circle(x, y, 7.5, colOrange, "", 0, 0.10)
		s.Circle(x, y, 3.1, colOrange, "#071017", 1.1, 1)
	}
}

func drawSurfaceFaces(s *SVG, cam Camera, faces []SurfaceFace, colorFn func(SurfaceFace) string, edge string, edgeWidth, edgeOpacity float64) {
	proj := projectFaces(cam, faces)
	_, _, view := cam.Basis()
	light := Vec3{-0.35, -0.48, 0.80}.Unit()
	half := light.Add(view).Unit()
	for _, face := range proj {
		n := face.Face.Normal
		diff := math.Max(0, n.Dot(light))
		rim := math.Pow(1-math.Abs(n.Dot(view)), 2.2) * 0.18
		spec := math.Pow(math.Max(0, n.Dot(half)), 26) * 0.34
		base := colorFn(face.Face)
		shade := 0.50 + 0.43*diff + rim + spec
		fill := scaleColor(base, shade)
		s.Poly(face.Poly, fill, "", 0, 1, true)
		if edge != "" && edgeWidth > 0 && edgeOpacity > 0 {
			for i := range face.Poly {
				j := (i + 1) % len(face.Poly)
				s.Line(face.Poly[i].X, face.Poly[i].Y, face.Poly[j].X, face.Poly[j].Y, edge, edgeWidth, edgeOpacity)
			}
		}
	}
}

func drawSurfaceFacesStyled(s *SVG, cam Camera, faces []SurfaceFace, colorFn func(SurfaceFace) (fill, stroke string, sw, opacity float64)) {
	proj := projectFaces(cam, faces)
	_, _, view := cam.Basis()
	light := Vec3{-0.35, -0.48, 0.80}.Unit()
	for _, face := range proj {
		n := face.Face.Normal
		diff := math.Max(0, n.Dot(light))
		rim := math.Pow(1-math.Abs(n.Dot(view)), 2.2) * 0.18
		base, stroke, sw, op := colorFn(face.Face)
		shade := 0.73 + 0.27*diff + rim
		fill := scaleColor(base, shade)
		s.Poly(face.Poly, fill, "", 0, 1, true)
		if stroke != "" && sw > 0 && op > 0 {
			for i := range face.Poly {
				j := (i + 1) % len(face.Poly)
				s.Line(face.Poly[i].X, face.Poly[i].Y, face.Poly[j].X, face.Poly[j].Y, stroke, sw, op)
			}
		}
	}
}

func drawIsoparms(s *SVG, cam Camera, surface *BSplineSurface, a, b float64, count int, tone string, opacity float64) {
	for k, line := range paramCurves(surface, a, b, count, 0.16) {
		sw := 0.65
		op := opacity
		if k%2 == 0 {
			sw = 0.85
		}
		s.Segmented(projectWorld(cam, line), tone, sw, op)
	}
}

func drawTrim(s *SVG, cam Camera, surface *BSplineSurface, a, b float64, tone string, sw, opacity float64) {
	line := trimBoundary(surface, a, b, 300, 0.2)
	s.Segmented(projectWorld(cam, line), tone, sw, opacity)
}

func drawSurfaceLabel(s *SVG, x, y float64, lines []string, tone string) {
	w := 250.0
	h := 22 + float64(len(lines))*17
	s.RectEx(x, y, w, h, "#07121a", colGrid2, 1, 0.88, 8)
	s.Rect(x, y, 3, h, tone)
	for i, line := range lines {
		s.Mono(x+14, y+22+float64(i)*17, 9.5, colMuted, "400", "start", line)
	}
}
