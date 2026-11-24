package main

import (
	"fmt"
	"image"
)

func RenderNormalView(cfg Config, cracks CrackReport, st StitchResult) *image.RGBA {
	W, H := 1280, 820
	canvas := newCanvas(W, H, uiBg)
	drawWindowTitle(canvas, 40, 34,
		"OPPOSITE-PARAMETER SEAM STITCH",
		"normal view - unified sampling, completed seam triangles, sharp normals retained")

	px, py, pw, ph := 36, 112, 850, 660
	cam := FitOverviewCamera(st.Mesh.Pos, pw, ph, 48)
	look := defaultLook()
	panel := RenderMeshPanel(st.Mesh, st.CornerNormals, cam, 2, look)

	for i := 0; i+1 < len(st.Samples); i++ {
		a := cam.Project(st.Samples[i].P)
		b := cam.Project(st.Samples[i+1].P)
		drawLine2D(panel, a.X, a.Y, b.X, b.Y, 2.4, Color{R: 18, G: 31, B: 48, A: 210})
	}
	pasteRGBA(canvas, panel, px, py)
	strokeRect(canvas, px, py, pw, ph, 2, uiLine)

	cx, cy, cw, ch := 918, 112, 326, 660
	drawCard(canvas, cx, cy, cw, ch)
	drawText(canvas, cx+24, cy+26, "STITCH REPORT", 2, uiInk)
	passBadge(canvas, cx+24, cy+62, "SEAM WATERTIGHT", st.Watertight.SeamWatertight)

	y := cy + 122
	drawMetricLine(canvas, cx+24, y, "BEFORE MAX CRACK", formatMM(cracks.MaxGapMM), uiRed)
	y += 58
	drawMetricLine(canvas, cx+24, y, "AFTER SEAM GAP", "0.000 MM", uiGreen)
	y += 58
	drawMetricLine(canvas, cx+24, y, "UNIFIED SAMPLES", fmt.Sprintf("%d", len(st.Samples)), uiInk)
	y += 58
	drawMetricLine(canvas, cx+24, y, "NEW TRIANGLES",
		fmt.Sprintf("A +%d / B +%d", st.RefineA.NetAdded, st.RefineB.NetAdded), uiInk)
	y += 58
	drawMetricLine(canvas, cx+24, y, "SHARP NORMAL JUMP",
		fmt.Sprintf("%.1f DEG", st.SharpNormalAngle), uiBlue)
	y += 58
	drawMetricLine(canvas, cx+24, y, "OUTER PATCH EDGES",
		fmt.Sprintf("%d EXPECTED OPEN", st.Watertight.OuterBoundaryEdges), uiMuted)

	ly := cy + 486
	drawLegendDot(canvas, cx+24, ly, look.ColorA, "PATCH A / Q FORWARD")
	drawLegendDot(canvas, cx+24, ly+34, look.ColorB, "PATCH B / Q REVERSED")
	drawLegendDot(canvas, cx+24, ly+68, look.FillColor, "COMPLETED SEAM BAND")
	drawText(canvas, cx+24, cy+ch-58, "POSITION IDS ARE WELDED,", 1, uiMuted)
	drawText(canvas, cx+24, cy+ch-42, "BUT CORNER NORMALS STAY SPLIT.", 1, uiMuted)

	drawText(canvas, px+18, py+ph-28, "ORTHOGRAPHIC OVERVIEW", 2, uiMuted)
	return canvas
}
