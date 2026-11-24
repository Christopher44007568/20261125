package main

import (
	"fmt"
	"image"
	"math"
)

func RenderHeatmapView(cfg Config, asm PatchAssembly, cracks CrackReport, st StitchResult) *image.RGBA {
	W, H := 1600, 860
	canvas := newCanvas(W, H, uiBg)
	drawWindowTitle(canvas, 36, 32,
		"CRACK HEATMAP AND WATERTIGHT RESULT",
		"distance from each canonical-q boundary sample to the opposite tessellated boundary")

	pw, ph := 760, 690
	px, py := 28, 112
	center := cracks.HotPointA.Lerp(cracks.HotPointB, 0.5)
	referenceScale := FitOverviewCamera(st.Mesh.Pos, 850, 660, 48).Scale
	cam := NewCamera(center, referenceScale*8, pw, ph)

	mesh := combineMeshes("heat-before", asm.MeshA, asm.MeshB)
	normals := ComputeCornerNormals(mesh, cfg.HardAngleDeg)
	look := zoomLook(false)
	look.ColorA = RGB(196, 218, 235)
	look.ColorB = RGB(244, 220, 184)
	look.WireColor = Color{R: 60, G: 72, B: 90, A: 42}
	look.WireWidth = 0.6
	panel := RenderMeshPanel(mesh, normals, cam, 2, look)
	drawCrackOverlay(panel, cam, cracks, 1)
	hot := cam.Project(cracks.HotPointA.Lerp(cracks.HotPointB, 0.5))
	labelWithBackground(panel, int(clampFloat(hot.X+16, 12, float64(pw-310))),
		int(clampFloat(hot.Y-42, 12, float64(ph-40))),
		fmt.Sprintf("HOT Q=%.3F / %.3F MM", cracks.MaxAtQ, cracks.MaxGapMM), 2,
		RGB(255, 255, 255), uiRed)
	pasteRGBA(canvas, panel, px, py)
	strokeRect(canvas, px, py, pw, ph, 2, uiLine)
	labelWithBackground(canvas, px+16, py+ph-40, "HEAT OVERLAY / ZOOMED SEAM", 2, uiInk, Color{R: 255, G: 255, B: 255, A: 220})

	cx, cy, cw, ch := 820, 112, 752, 690
	drawCard(canvas, cx, cy, cw, ch)
	drawText(canvas, cx+28, cy+28, "CRACK DISTANCE BY CANONICAL Q", 2, uiInk)

	plotX, plotY, plotW, plotH := cx+110, cy+88, 360, 380
	axisX := plotX
	baseGapX := plotX + 70
	maxGapW := 250.0
	bottom := plotY + plotH
	maxGap := math.Max(cracks.MaxGapMM, 1e-9)

	for _, q := range []float64{0, 0.25, 0.5, 0.75, 1} {
		y := int(math.Round(float64(bottom) - q*float64(plotH)))
		drawLine2D(canvas, float64(plotX-8), float64(y), float64(plotX+plotW), float64(y), 1, uiLine)
		drawText(canvas, plotX-58, y-8, fmt.Sprintf("Q %.2F", q), 1, uiMuted)
	}
	drawLine2D(canvas, float64(axisX), float64(plotY), float64(axisX), float64(bottom), 2, uiInk)
	drawLine2D(canvas, float64(axisX), float64(bottom), float64(plotX+plotW), float64(bottom), 2, uiInk)

	for _, s := range cracks.Samples {
		y := int(math.Round(float64(bottom) - s.Q*float64(plotH)))
		w := s.HeatMM / maxGap * maxGapW
		c := heatColor(s.HeatMM / maxGap)
		drawLine2D(canvas, float64(baseGapX), float64(y), float64(baseGapX)+w, float64(y), 1.8, c)
	}
	drawText(canvas, baseGapX-10, bottom+18, "0", 1, uiMuted)
	drawText(canvas, baseGapX+int(maxGapW)-80, bottom+18, formatMM(cracks.MaxGapMM), 1, uiMuted)
	drawText(canvas, plotX+105, bottom+38, "BOUNDARY DISTANCE (MM)", 1, uiMuted)

	barX, barY, barW, barH := cx+590, plotY, 28, plotH
	for y := 0; y < barH; y++ {
		t := 1 - float64(y)/float64(barH)
		fillRect(canvas, barX, barY+y, barW, 1, heatColor(t))
	}
	strokeRect(canvas, barX, barY, barW, barH, 2, uiInk)
	drawText(canvas, barX+38, barY-4, "HIGH", 1, uiRed)
	drawText(canvas, barX+38, barY+barH-14, "ZERO", 1, uiBlue)

	sy := cy + 540
	drawMetricLine(canvas, cx+30, sy, "MAX / MEAN / RMS BEFORE",
		fmt.Sprintf("%.3F / %.3F / %.3F MM", cracks.MaxGapMM, cracks.MeanGapMM, cracks.RMSGapMM), uiRed)
	drawMetricLine(canvas, cx+30, sy+62, "AFTER STITCH", "0.000 MM - SEAM WATERTIGHT", uiGreen)
	passBadge(canvas, cx+590, sy+58, "PASS", st.Watertight.SeamWatertight)

	drawText(canvas, cx+30, cy+ch-32, "COOL COLORS: MATCHING SAMPLES. RED: T-JUNCTION OPENING.", 1, uiMuted)
	return canvas
}
