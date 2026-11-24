package main

import (
	"fmt"
	"image"
	"image/draw"
	"math"
)

type panelSpec struct {
	mode     PanelMode
	title    string
	subtitle string
	accent   RGB
}

func maxSeamMetrics(report Report) (pos, tangent, normal, jump float64) {
	for _, seam := range report.Seams {
		pos = math.Max(pos, seam.MaxPositionErrorMM)
		tangent = math.Max(tangent, seam.MaxTangentAngleDeg)
		normal = math.Max(normal, seam.MaxNormalAngleDeg)
		jump = math.Max(jump, seam.MaxCurvatureJumpPerMM)
	}
	return
}

func renderPanel(mesh Mesh, cfg Config, report Report, spec panelSpec) *image.RGBA {
	panel := drawPanelBackground(cfg.PanelWidth, cfg.PanelHeight)
	content := Rect{X: 22, Y: 62, W: cfg.PanelWidth - 44, H: cfg.PanelHeight - 128}
	cam := NewCamera(Vec3{X: 7.5, Y: 0, Z: 0}, Vec3{X: -15, Y: -34, Z: 7})
	cam.Fit(mesh, content, 22, 18)

	surface, coverage := renderSurface(
		mesh, cam, spec.mode, cfg.PanelWidth, cfg.PanelHeight,
		cfg.SuperSample, report.Curvature.DisplayMax,
	)
	draw.Draw(panel, panel.Bounds(), surface, image.Point{}, draw.Over)
	outlineCoverage(panel, coverage, RGB{0.005, 0.008, 0.014})

	drawFrame(panel, 1, spec.accent)
	drawTextShadow(panel, 22, 13, spec.title, 2, RGB{0.92, 0.95, 0.99})
	drawText(panel, 22, 34, spec.subtitle, 1, RGB{0.50, 0.58, 0.68})
	for x := 22; x < cfg.PanelWidth-22; x++ {
		blendPixel(panel, x, 54, spec.accent, 0.35)
	}

	switch spec.mode {
	case ModeSilhouette:
		drawPassPill(panel, 22, cfg.PanelHeight-48, report.Acceptance.FilletParentSeamsPass)
		drawText(panel, 154, cfg.PanelHeight-44, "UPPER + LOWER SEAMS", 1,
			RGB{0.52, 0.61, 0.70})
	case ModeNormalSign:
		drawNormalLegend(panel, cfg.PanelHeight)
	case ModeZebra:
		drawText(panel, 22, cfg.PanelHeight-44, "REFLECTED ENVIRONMENT / STRIPES FOLLOW NORMALS", 1,
			RGB{0.58, 0.65, 0.73})
	case ModeCurvature:
		drawCurvatureLegend(panel, cfg, report.Curvature.DisplayMax)
	}
	return panel
}

func drawNormalLegend(img *image.RGBA, h int) {
	y := h - 44
	items := []struct {
		label string
		c     RGB
	}{
		{"+X", RGB{0.93, 0.22, 0.20}},
		{"+Y", RGB{0.25, 0.90, 0.36}},
		{"+Z", RGB{0.28, 0.50, 1.00}},
	}
	x := 22
	for _, item := range items {
		for iy := 0; iy < 10; iy++ {
			for ix := 0; ix < 10; ix++ {
				blendPixel(img, x+ix, y+iy, item.c, 1)
			}
		}
		drawText(img, x+15, y-1, item.label, 2, item.c)
		x += 70
	}
	drawText(img, x+4, y+1, "DARK = NEGATIVE COMPONENT", 1, RGB{0.50, 0.57, 0.65})
}

func drawCurvatureLegend(img *image.RGBA, cfg Config, maxK float64) {
	const bands = 10
	x, y := 22, cfg.PanelHeight-49
	barW, barH := cfg.PanelWidth-266, 13
	for i := 0; i < bands; i++ {
		t := (float64(i) + 0.5) / bands
		c := paletteColor(curvaturePalette, t)
		x0 := x + i*barW/bands
		x1 := x + (i+1)*barW/bands
		for iy := 0; iy < barH; iy++ {
			for ix := x0; ix < x1; ix++ {
				blendPixel(img, ix, y+iy, c, 1)
			}
		}
	}
	for ix := x; ix < x+barW; ix++ {
		blendPixel(img, ix, y, RGB{0.02, 0.025, 0.035}, 0.75)
		blendPixel(img, ix, y+barH-1, RGB{0.02, 0.025, 0.035}, 0.75)
	}
	for iy := y; iy < y+barH; iy++ {
		blendPixel(img, x, iy, RGB{0.02, 0.025, 0.035}, 0.75)
		blendPixel(img, x+barW-1, iy, RGB{0.02, 0.025, 0.035}, 0.75)
	}
	labelY := y + barH + 3
	drawText(img, x, labelY, "0", 1, RGB{0.60, 0.66, 0.73})
	mid := fmt.Sprintf("%.2f", maxK*0.5)
	drawText(img, x+barW/2-textWidth(mid, 1)/2, labelY, mid, 1, RGB{0.60, 0.66, 0.73})
	right := fmt.Sprintf("%.2f", maxK)
	drawText(img, x+barW-textWidth(right, 1), labelY, right, 1, RGB{0.68, 0.73, 0.80})
	drawText(img, x+barW+16, y-1, "|K|MAX", 2, RGB{0.88, 0.91, 0.95})
	drawText(img, x+barW+16, y+17, "1/MM", 1, RGB{0.52, 0.59, 0.67})
}

func renderFigure(mesh Mesh, cfg Config, report Report) *image.RGBA {
	const (
		outer   = 30
		gap     = 22
		headerH = 96
		footerH = 92
	)
	figW := 2*cfg.PanelWidth + 2*outer + gap
	panelTop := outer + headerH
	figH := panelTop + 2*cfg.PanelHeight + gap + footerH + outer
	fig := image.NewRGBA(image.Rect(0, 0, figW, figH))
	fillBackground(fig, RGB{0.020, 0.025, 0.035}, RGB{0.010, 0.012, 0.018})
	addRadialGlow(fig, float64(figW)*0.50, float64(figH)*0.44, float64(figW)*0.62,
		RGB{0.055, 0.077, 0.105}, 0.42)

	drawTextShadow(fig, outer+2, 19, "VARIABLE-RADIUS FILLET DIAGNOSTICS", 3,
		RGB{0.93, 0.95, 0.98})
	subtitle := fmt.Sprintf(
		"%.0f DEG WEDGE / LENGTH %.1f MM / FACE %.1f MM / R %.1f -> %.1f MM / ANALYTIC SURFACE",
		2*cfg.HalfAngle, cfg.Length, cfg.FaceWidth, cfg.RadiusStart, cfg.RadiusEnd,
	)
	drawText(fig, outer+3, 53, subtitle, 2, RGB{0.46, 0.55, 0.65})
	drawText(fig, outer+3, 73, "GO STANDARD LIBRARY / SMOOTH NORMALS / ORTHOGRAPHIC DEPTH BUFFER",
		1, RGB{0.32, 0.39, 0.47})

	pass := report.Acceptance.FilletParentSeamsPass
	statusColor := RGB{0.31, 0.82, 0.49}
	if !pass {
		statusColor = RGB{0.96, 0.34, 0.31}
	}
	drawText(fig, figW-310, 25, "G0 + G1 "+passLabel(pass), 2, statusColor)
	drawText(fig, figW-310, 47, "UPPER + LOWER SEAMS", 1, RGB{0.56, 0.64, 0.71})
	for x := outer; x < figW-outer; x++ {
		blendPixel(fig, x, panelTop-8, RGB{0.20, 0.25, 0.31}, 0.75)
	}

	specs := []panelSpec{
		{mode: ModeSilhouette, title: "01  SILHOUETTE", subtitle: "ANALYTIC SURFACE / ORTHOGRAPHIC VIEW", accent: RGB{0.30, 0.72, 0.95}},
		{mode: ModeNormalSign, title: "02  NORMAL SIGN", subtitle: "RGB = +X,+Y,+Z / DARK = NEGATIVE", accent: RGB{0.42, 0.88, 0.58}},
		{mode: ModeZebra, title: "03  ZEBRA", subtitle: "REFLECTION STRIPES / SMOOTH NORMAL FIELD", accent: RGB{0.92, 0.71, 0.28}},
		{mode: ModeCurvature, title: "04  CURVATURE", subtitle: "MAX PRINCIPAL |K| / 10 BANDS", accent: RGB{0.91, 0.46, 0.33}},
	}
	for i, spec := range specs {
		col, row := i%2, i/2
		x := outer + col*(cfg.PanelWidth+gap)
		y := panelTop + row*(cfg.PanelHeight+gap)
		panel := renderPanel(mesh, cfg, report, spec)
		draw.Draw(fig, image.Rect(x, y, x+cfg.PanelWidth, y+cfg.PanelHeight),
			panel, image.Point{}, draw.Src)
	}

	footerTop := panelTop + 2*cfg.PanelHeight + gap
	for x := outer; x < figW-outer; x++ {
		blendPixel(fig, x, footerTop-1, RGB{0.21, 0.26, 0.33}, 0.8)
	}
	pos, tangent, normal, jump := maxSeamMetrics(report)
	line1 := fmt.Sprintf(
		"G0 MAX %.2E MM  |  TANGENT MAX %.2E DEG  |  NORMAL MAX %.2E DEG  |  SEAMS + ENDPOINTS",
		pos, tangent, normal,
	)
	line2 := fmt.Sprintf(
		"G1 ACCEPTANCE: %s  |  CURVATURE JUMP %.3f 1/MM AT PLANAR SEAMS (EXPECTED)",
		passLabel(pass), jump,
	)
	drawText(fig, outer+2, footerTop+17, line1, 2, RGB{0.64, 0.71, 0.78})
	drawText(fig, outer+2, footerTop+47, line2, 2, RGB{0.54, 0.61, 0.69})
	return fig
}

func passLabel(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}
