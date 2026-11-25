package main

import (
	"fmt"
	"math"
	"path/filepath"
)

func viewCamera(az, el float64) Camera {
	c := NewCamera(az, el, 1600, 1100, 5.10)
	c.Center = Vec3{0, 0, 29}
	c.PanX = -310
	c.PanY = 42
	return c
}

func drawReferenceTrim(s *SVG, cam Camera, a, b float64, tone string) {
	ref := Reference{}
	var pts []Vec3
	for i := 0; i <= 300; i++ {
		t := 2 * math.Pi * float64(i) / 300
		x, y := a*math.Cos(t), b*math.Sin(t)
		pts = append(pts, Vec3{x, y, ref.Z(x, y) + 0.3})
	}
	proj := projectWorld(cam, pts)
	for i := 0; i < len(proj)-1; i += 2 {
		j := i + 1
		if j >= len(proj) {
			break
		}
		s.Line(proj[i].X, proj[i].Y, proj[j].X, proj[j].Y, tone, 1.25, 0.82)
	}
}

func RenderRotatedView(path, index, title, subtitle, viewName string, az, el float64, points []ScanPoint, fit *FitResult, audit Audit, trimA, trimB float64, clean, noise int) error {
	s := NewSVG(1600, 1100)
	pageShell(s, index, "ORBIT", title, subtitle)
	panel(s, 48, 190, 1074, 850, viewName, fmt.Sprintf("AZ %.0f°  ·  EL %.0f°", az, el))
	cam := viewCamera(az, el)
	faces := buildSurfaceFaces(fit.Surface, Reference{}, trimA, trimB, 62, 40, 0)
	colorFn := func(f SurfaceFace) string {
		t := clamp((f.Center.Z-17)/29, 0, 1)
		if az > 90 {
			return mixColor("#104b73", "#60e7de", 0.12+0.78*t)
		}
		return mixColor("#0a4e64", "#72f0d2", 0.10+0.80*t)
	}
	drawSurfaceFaces(s, cam, faces, colorFn, "#08353d", 0.30, 0.07)
	drawIsoparms(s, cam, fit.Surface, trimA, trimB, 9, "#a8fff5", 0.12)
	drawTrim(s, cam, fit.Surface, trimA, trimB, "#061116", 6.2, 0.62)
	drawTrim(s, cam, fit.Surface, trimA, trimB, colCyan, 2.2, 1)
	drawReferenceTrim(s, cam, PatchA, PatchB, colGold)
	drawHoleOverlay(s, cam, fit.Surface, true)
	drawNoisePoints(s, cam, fit.Surface, points, true)
	drawAxisGizmo(s, cam, 1018, 934, 33)

	s.RectEx(76, 856, 350, 92, "#07131b", colGrid2, 1, 0.82, 9)
	s.Mono(94, 881, 9, colMuted, "600", "start", "SILHOUETTE OVERLAY")
	s.Line(94, 899, 160, 899, colCyan, 2.2, 1)
	s.Mono(170, 902, 9, colText, "400", "start", "FIT  /  TRIM LOOP")
	s.Line(94, 921, 160, 921, colGold, 1.3, 0.85)
	s.Mono(170, 924, 9, colText, "400", "start", "REFERENCE  /  DASHED")
	s.Mono(94, 942, 8.5, colFaint, "400", "start", fmt.Sprintf("CHAMFER %.3f mm  ·  IoU %.5f", audit.SilhouetteChamfer, audit.SilhouetteIoU))

	lx := 76.0
	lx = legendChip(s, lx, 1005, "clean", "SURFACE")
	lx = legendChip(s, lx, 1005, "noise", "NOISE")
	_ = legendChip(s, lx, 1005, "hole", "OCCLUSION")

	railX, railY, railW := 1146.0, 190.0, 390.0
	s.RectEx(railX, railY, railW, 850, colPanel, colGrid, 1, 0.96, 16)
	s.Text(railX+22, railY+31, 11, colMuted, "700", "start", "VIEW TELEMETRY")
	s.Mono(railX+railW-22, railY+31, 9, colCyan, "600", "end", viewName)
	s.Line(railX+22, railY+46, railX+railW-22, railY+46, colGrid, 1, 1)
	s.Mono(railX+22, railY+88, 33, colText, "500", "start", fmt.Sprintf("%+.0f°", az))
	s.Text(railX+22, railY+106, 9, colMuted, "500", "start", "AZIMUTH")
	s.Mono(railX+194, railY+88, 33, colText, "500", "start", fmt.Sprintf("%.0f°", el))
	s.Text(railX+194, railY+106, 9, colMuted, "500", "start", "ELEVATION")
	s.Text(railX+22, railY+154, 9, colMuted, "700", "start", "ORBIT POSITION")
	drawOrbitMap(s, railX+22, railY+170, railW-44, az)
	s.Line(railX+22, railY+348, railX+railW-22, railY+348, colGrid, 1, 0.8)
	metrics := []struct {
		Label string
		Value string
		Note  string
		Tone  string
	}{
		{"SILHOUETTE IoU", fmt.Sprintf("%.5f", audit.SilhouetteIoU), "projected mask overlap", colCyan},
		{"BOUNDARY CHAMFER", fmt.Sprintf("%.3f mm", audit.SilhouetteChamfer), "fit vs. reference", colCyan},
		{"NORMAL P95", fmt.Sprintf("%.2f°", audit.NormalP95), "surface orientation", colGold},
		{"CURVATURE ERROR", fmt.Sprintf("%.1f%%", audit.CurvatureRelative), "mean curvature RMS", colGold},
		{"RIPPLE ENERGY", fmt.Sprintf("%.1f%%", audit.RippleRatio), "4 mm band", colGold},
	}
	yy := railY + 385
	for i, m := range metrics {
		if i > 0 {
			s.Line(railX+22, yy-22, railX+railW-22, yy-22, colGrid, 1, 0.6)
		}
		s.Text(railX+22, yy, 9, colMuted, "600", "start", m.Label)
		s.Mono(railX+railW-22, yy+2, 15, m.Tone, "500", "end", m.Value)
		s.Mono(railX+22, yy+20, 8.5, colFaint, "400", "start", m.Note)
		yy += 58
	}
	s.Line(railX+22, railY+676, railX+railW-22, railY+676, colGrid, 1, 0.8)
	gauge(s, railX+22, railY+706, railW-44, audit.NormalP95, 4, 2, "NORMAL P95 / 4° BAND", colGold)
	gauge(s, railX+22, railY+751, railW-44, (1-audit.SilhouetteIoU)*100, 1.5, 1, "SILHOUETTE MISS / %", colCyan)
	s.Line(railX+22, railY+782, railX+railW-22, railY+782, colGrid, 1, 0.8)
	s.Mono(railX+22, railY+807, 8.5, colFaint, "400", "start",
		fmt.Sprintf("%d CLEAN  ·  %d NOISE  ·  3 VOIDS", clean, noise))
	s.Mono(railX+22, railY+824, 8.5, colFaint, "400", "start",
		"REFERENCE CONTOUR USED FOR AUDIT ONLY")
	return s.Save(filepath.Clean(path))
}

func drawOrbitMap(s *SVG, x, y, w float64, az float64) {
	cx, cy := x+w*0.5, y+78
	rx, ry := w*0.32, 54.0
	s.Poly([]Vec2{
		{cx - rx, cy}, {cx, cy - ry}, {cx + rx, cy}, {cx, cy + ry},
	}, "#0a1a23", colGrid2, 1, 1, true)
	s.Poly([]Vec2{
		{cx - rx*0.58, cy + 5}, {cx, cy - ry*0.58}, {cx + rx*0.58, cy + 5}, {cx, cy + ry*0.58},
	}, "#15323b", colCyan, 1, 0.55, true)
	for i, a := range []float64{-38, 137} {
		rad := a * math.Pi / 180
		px := cx + rx*math.Cos(rad)
		py := cy - ry*math.Sin(rad)
		tone := colGold
		if i == 0 {
			tone = colCyan
		}
		s.Line(cx, cy, px, py, tone, 1, 0.6)
		s.Circle(px, py, 5.5, tone, "#071017", 1.2, 1)
		s.Mono(px, py-11, 8, tone, "700", "middle", fmt.Sprintf("%c", 'A'+rune(i)))
	}
	rad := az * math.Pi / 180
	s.Circle(cx+rx*math.Cos(rad), cy-ry*math.Sin(rad), 12, "", colText, 1, 0.82)
	s.Mono(cx, y+151, 8, colFaint, "400", "middle", "TOP VIEW  /  A FRONT · B REAR")
}
