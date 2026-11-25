package main

import (
	"fmt"
	"math"
	"path/filepath"
)

func scatterCamera() Camera {
	c := NewCamera(-38, 20, 1600, 1100, 5.35)
	c.Center = Vec3{0, 0, 30}
	c.PanX = -255
	c.PanY = 42
	return c
}

func RenderPointCloud(path string, points []ScanPoint, holes HoleField, fit *FitResult, clean, noise int, trimA, trimB float64) error {
	s := NewSVG(1600, 1100)
	pageShell(s, "01 / 06", "INPUT", "Sparse scan before fitting",
		"Raw samples with independent noise and scanner occlusion masks; no dense reference is supplied to the solver.")
	panel(s, 48, 190, 1120, 850, "POINT CLOUD / RAW SAMPLE FIELD", "AZ −38°  ·  EL 20°")
	cam := scatterCamera()
	drawFloorGrid(s, cam, 11.0)
	drawHoleOverlay(s, cam, fit.Surface, true)
	maxZ, minZ := -math.MaxFloat64, math.MaxFloat64
	for _, p := range points {
		maxZ = math.Max(maxZ, p.Z)
		minZ = math.Min(minZ, p.Z)
	}
	for _, p := range points {
		if p.Kind != PointClean {
			continue
		}
		x, y, depth := cam.Project(Vec3{p.X, p.Y, p.Z})
		t := clamp((p.Z-minZ)/(maxZ-minZ+1e-9), 0, 1)
		r := 2.15 * clamp(525/(depth+120), 0.70, 1.18)
		clr := mixColor("#4fb7c8", "#bafff4", 0.28+0.58*t)
		s.Circle(x, y, r+2.5, clr, "", 0, 0.055)
		s.Circle(x, y, r, clr, "#082029", 0.55, 0.92)
	}
	drawNoisePoints(s, cam, fit.Surface, points, true)

	drawSurfaceLabel(s, 78, 900, []string{
		fmt.Sprintf("X RANGE   %6.1f … %6.1f mm", -PatchA, PatchA),
		fmt.Sprintf("Y RANGE   %6.1f … %6.1f mm", -PatchB, PatchB),
		fmt.Sprintf("Z RANGE   %6.1f … %6.1f mm", minZ, maxZ),
	}, colCyan)
	drawAxisGizmo(s, cam, 1068, 934, 34)
	lx := 380.0
	lx = legendChip(s, lx, 1002, "clean", "CLEAN SCAN")
	lx = legendChip(s, lx, 1002, "noise", "NOISE / OUTLIER")
	_ = legendChip(s, lx, 1002, "hole", "OCCLUSION VOID")

	railX, railY, railW := 1192.0, 190.0, 344.0
	s.RectEx(railX, railY, railW, 850, colPanel, colGrid, 1, 0.96, 16)
	s.Text(railX+22, railY+31, 11, colMuted, "700", "start", "SCAN COVERAGE")
	s.Mono(railX+railW-22, railY+31, 9, colCyan, "600", "end", "UNCURATED INPUT")
	s.Line(railX+22, railY+46, railX+railW-22, railY+46, colGrid, 1, 1)
	s.Mono(railX+22, railY+78, 36, colText, "500", "start", fmt.Sprintf("%d", clean+noise))
	s.Text(railX+22, railY+98, 10, colMuted, "500", "start", "TOTAL SAMPLES")
	s.Line(railX+22, railY+116, railX+railW-22, railY+116, colGrid, 1, 0.7)
	stats := []struct {
		Label string
		Value int
		Tone  string
	}{
		{"CLEAN / INLIER", clean, colCyan},
		{"NOISE / OUTLIER", noise, colOrange},
		{"OCCLUSION VOIDS", len(scanHoles), colMagenta},
	}
	for i, st := range stats {
		yy := railY + 151 + float64(i)*77
		s.Text(railX+22, yy, 9, colMuted, "600", "start", st.Label)
		s.Mono(railX+22, yy+30, 24, st.Tone, "500", "start", fmt.Sprintf("%d", st.Value))
		s.RectEx(railX+130, yy+19, 186, 5, "#122732", "", 0, 1, 2.5)
		frac := float64(st.Value) / float64(clean+noise)
		s.RectEx(railX+130, yy+19, 186*clamp(frac, 0.04, 1), 5, st.Tone, "", 0, 1, 2.5)
	}
	s.Line(railX+22, railY+392, railX+railW-22, railY+392, colGrid, 1, 0.8)
	s.Text(railX+22, railY+419, 9, colMuted, "700", "start", "OCCLUSION MASK / H1–H3")
	for i, h := range scanHoles {
		yy := railY + 444 + float64(i)*42
		s.RectEx(railX+22, yy-10, 14, 14, "#5d1f7d", "", 0, 1, 3)
		s.Line(railX+24, yy-8, railX+34, yy+2, "#e8b8ff", 0.8, 0.8)
		s.Line(railX+24, yy-3, railX+34, yy+7, "#e8b8ff", 0.8, 0.6)
		s.Mono(railX+46, yy, 10, colMagenta, "600", "start", fmt.Sprintf("H%d", i+1))
		s.Mono(railX+80, yy, 9, colMuted, "400", "start",
			fmt.Sprintf("%.1f × %.1f mm", h.RX*2*PatchA, h.RY*2*PatchB))
		s.Mono(railX+railW-22, yy, 9, colFaint, "400", "end",
			fmt.Sprintf("%.0f, %.0f", h.CX*PatchA, h.CY*PatchB))
	}
	s.Line(railX+22, railY+585, railX+railW-22, railY+585, colGrid, 1, 0.8)
	s.Text(railX+22, railY+612, 9, colMuted, "700", "start", "FIT CONTRACT")
	s.Text(railX+22, railY+636, 11, colText, "500", "start", "Silhouette + curvature")
	s.Text(railX+22, railY+653, 11, colText, "500", "start", "acceptance, not control")
	s.Text(railX+22, railY+670, 11, colText, "500", "start", "point residual alone.")
	s.Mono(railX+22, railY+706, 9, colFaint, "400", "start", "ROBUST TUKEY IRLS")
	s.Mono(railX+22, railY+723, 9, colFaint, "400", "start", "SECOND-DIFFERENCE SMOOTHING")
	s.Mono(railX+22, railY+740, 9, colFaint, "400", "start", "DENSE REFERENCE: AUDIT ONLY")
	s.Text(railX+22, railY+790, 9, colMuted, "700", "start", "ESTIMATED TRIM LOOP")
	s.Mono(railX+22, railY+814, 10, colText, "500", "start",
		fmt.Sprintf("%.2f × %.2f mm", trimA, trimB))
	return s.Save(filepath.Clean(path))
}
