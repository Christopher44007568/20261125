package main

import (
	"fmt"
	"math"
	"path/filepath"
)

func errorCamera() Camera {
	c := NewCamera(-31, 27, 1600, 1100, 5.15)
	c.Center = Vec3{0, 0, 29}
	c.PanX = -345
	c.PanY = 48
	return c
}

func RenderNormalError(path string, points []ScanPoint, fit *FitResult, audit Audit, trimA, trimB float64, clean, noise int) error {
	s := NewSVG(1600, 1100)
	vmax := math.Max(1.2, math.Min(3.5, audit.NormalP95*1.10))
	pageShell(s, "03 / 06", "AUDIT", "Normal error field",
		"Per-face angular deviation to the dense reference normal. The reference is withheld from fitting and used only for acceptance.")
	panel(s, 48, 190, 1048, 850, "NORMAL ERROR / FACE-AVERAGED", fmt.Sprintf("SCALE 0–%.1f°", vmax))
	cam := errorCamera()
	faces := buildSurfaceFaces(fit.Surface, Reference{}, trimA, trimB, 60, 38, 0)
	colorFn := func(f SurfaceFace) (fill, stroke string, sw, opacity float64) {
		t := f.Error / vmax
		return heatColor(t), "#071118", 0.18, 0.075
	}
	drawSurfaceFacesStyled(s, cam, faces, colorFn)
	drawTrim(s, cam, fit.Surface, trimA, trimB, "#061116", 5.0, 0.65)
	drawTrim(s, cam, fit.Surface, trimA, trimB, "#dffdf7", 1.35, 0.9)
	drawHoleOverlay(s, cam, fit.Surface, true)
	drawNoisePoints(s, cam, fit.Surface, points, true)
	drawAxisGizmo(s, cam, 1008, 935, 32)
	drawSurfaceLabel(s, 76, 890, []string{
		fmt.Sprintf("FACE COUNT       %d", len(faces)),
		fmt.Sprintf("NORMAL RMS       %.2f°", audit.NormalRMSE),
		fmt.Sprintf("NORMAL MAX       %.2f°", audit.NormalMax),
	}, colGold)
	lx := 72.0
	lx = legendChip(s, lx, 998, "noise", "NOISE / OUTLIER")
	_ = legendChip(s, lx, 998, "hole", "OCCLUSION VOID")

	railX, railY, railW := 1120.0, 190.0, 416.0
	s.RectEx(railX, railY, railW, 850, colPanel, colGrid, 1, 0.96, 16)
	s.Text(railX+22, railY+31, 11, colMuted, "700", "start", "ANGULAR DEVIATION")
	s.Mono(railX+railW-22, railY+31, 9, colCyan, "600", "end", "DENSE REFERENCE")
	s.Line(railX+22, railY+46, railX+railW-22, railY+46, colGrid, 1, 1)
	barX, barY, barW, barH := railX+22, railY+88, railW-44, 14.0
	const shadeBins = 28
	for i := 0; i < shadeBins; i++ {
		t0 := float64(i) / shadeBins
		s.Rect(barX+barW*t0, barY, barW/float64(shadeBins)+0.4, barH, heatColor(t0+0.5/shadeBins))
	}
	for i := 0; i <= 4; i++ {
		t := float64(i) / 4
		x := barX + t*barW
		s.Line(x, barY+barH, x, barY+barH+5, colMuted, 1, 0.8)
		s.Mono(x, barY+barH+17, 8.5, colMuted, "400", "middle", fmt.Sprintf("%.2f°", t*vmax))
	}
	px := barX + clamp(audit.NormalP95/vmax, 0, 1)*barW
	s.Line(px, barY-6, px, barY+barH+6, colText, 1.4, 0.95)
	s.Mono(px+5, barY-7, 8.5, colText, "600", "start", fmt.Sprintf("p95 %.2f°", audit.NormalP95))

	s.Text(railX+22, railY+154, 9, colMuted, "700", "start", "ERROR DISTRIBUTION")
	hist := normalHistogram(fit, trimA, trimB, 14, vmax)
	histMax := 0.0
	for _, v := range hist {
		histMax = math.Max(histMax, v)
	}
	hx, hy, hw, hh := railX+22, railY+174, railW-44, 104.0
	s.Line(hx, hy+hh, hx+hw, hy+hh, colGrid2, 1, 0.9)
	for i, v := range hist {
		x := hx + float64(i)*hw/float64(len(hist))
		hgt := v / math.Max(histMax, 1e-9) * (hh - 10)
		s.Rect(x+0.8, hy+hh-hgt, hw/float64(len(hist))-1.6, hgt,
			heatColor((float64(i)+0.5)/float64(len(hist))))
	}
	s.Mono(hx, hy+hh+16, 8, colFaint, "400", "start", "0°")
	s.Mono(hx+hw, hy+hh+16, 8, colFaint, "400", "end", fmt.Sprintf("%.1f°", vmax))

	yy := railY + 346
	rows := []metricRow{
		{"NORMAL RMSE", fmt.Sprintf("%.2f", audit.NormalRMSE), "deg", "all mesh vertices", colGold},
		{"NORMAL P95", fmt.Sprintf("%.2f", audit.NormalP95), "deg", "acceptance band", colText},
		{"NORMAL MAX", fmt.Sprintf("%.2f", audit.NormalMax), "deg", "local maximum", colOrange},
		{"CURVATURE ERROR", fmt.Sprintf("%.1f", audit.CurvatureRelative), "% RMS", "mean curvature", colCyan},
		{"RIPPLE BAND", fmt.Sprintf("%.1f", audit.RippleRatio), "%", "4 mm high-pass", colCyan},
		{"SILHOUETTE IoU", fmt.Sprintf("%.5f", audit.SilhouetteIoU), "", "projected overlap", colCyan},
		{"BOUNDARY CHAMFER", fmt.Sprintf("%.3f", audit.SilhouetteChamfer), "mm", "trim-loop distance", colCyan},
	}
	for i, r := range rows {
		if i > 0 {
			s.Line(railX+22, yy-25, railX+railW-22, yy-25, colGrid, 1, 0.62)
		}
		s.Text(railX+22, yy, 9, colMuted, "600", "start", r.Label)
		s.Mono(railX+22, yy+25, 21, r.Tone, "500", "start", r.Value)
		if r.Unit != "" {
			s.Mono(railX+22+float64(len(r.Value))*12.5, yy+24, 9, colFaint, "400", "start", r.Unit)
		}
		s.Mono(railX+railW-22, yy+7, 8.5, colFaint, "400", "end", r.Note)
		yy += 60
	}
	s.Line(railX+22, railY+774, railX+railW-22, railY+774, colGrid, 1, 0.8)
	s.Mono(railX+22, railY+798, 8.5, colFaint, "400", "start",
		fmt.Sprintf("%d CLEAN  ·  %d NOISE  ·  3 OCCLUSION VOIDS", clean, noise))
	s.Mono(railX+22, railY+815, 8.5, colFaint, "400", "start",
		"HEAT IS INDEPENDENT OF CONTROL-POINT RESIDUAL")
	return s.Save(filepath.Clean(path))
}

func normalHistogram(fit *FitResult, trimA, trimB float64, bins int, vmax float64) []float64 {
	out := make([]float64, bins)
	ref := Reference{}
	for iy := 0; iy < 40; iy++ {
		y := -trimB + (float64(iy)+0.5)/40*2*trimB
		for ix := 0; ix < 60; ix++ {
			x := -trimA + (float64(ix)+0.5)/60*2*trimA
			if trimRadius(x, y) > 0.95 {
				continue
			}
			e := angleDeg(fit.Surface.Normal(x, y), ref.Normal(x, y))
			idx := int(clamp(e/vmax, 0, 0.9999) * float64(bins))
			out[idx]++
		}
	}
	return out
}
