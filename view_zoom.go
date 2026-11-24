package main

import (
	"fmt"
	"image"
	"math"
)

func RenderZoomView(cfg Config, asm PatchAssembly, cracks CrackReport, st StitchResult) *image.RGBA {
	W, H := 1600, 860
	canvas := newCanvas(W, H, uiBg)
	drawWindowTitle(canvas, 36, 32,
		"T-JUNCTION ZOOM: 10X AROUND MAX CRACK",
		"left: opposite q directions and mismatched boundary samples - right: unified samples and fan completion")

	pw, ph := 760, 690
	lx, rx, py := 28, 812, 112
	center := cracks.HotPointA.Lerp(cracks.HotPointB, 0.5)
	referenceScale := FitOverviewCamera(st.Mesh.Pos, 850, 660, 48).Scale
	cam := NewCamera(center, referenceScale*10, pw, ph)

	beforeMesh := combineMeshes("before", asm.MeshA, asm.MeshB)
	beforeNormals := ComputeCornerNormals(beforeMesh, cfg.HardAngleDeg)
	beforePanel := RenderMeshPanel(beforeMesh, beforeNormals, cam, 2, zoomLook(false))
	drawCrackOverlay(beforePanel, cam, cracks, 4)
	hotA := cam.Project(cracks.HotPointA)
	hotB := cam.Project(cracks.HotPointB)
	drawLine2D(beforePanel, hotA.X, hotA.Y, hotB.X, hotB.Y, 5, uiRed)
	drawDisc(beforePanel, int(hotA.X), int(hotA.Y), 6, uiRed)
	drawDisc(beforePanel, int(hotB.X), int(hotB.Y), 6, uiRed)
	labelWithBackground(beforePanel, int(clampFloat(hotA.X+18, 12, float64(pw-260))),
		int(clampFloat(hotA.Y-44, 12, float64(ph-40))),
		fmt.Sprintf("GAP %.3F MM", cracks.MaxGapMM), 2, RGB(255, 255, 255), uiRed)

	afterPanel := RenderMeshPanel(st.Mesh, st.CornerNormals, cam, 2, zoomLook(true))
	for i := 0; i+1 < len(st.Samples); i++ {
		a := cam.Project(st.Samples[i].P)
		b := cam.Project(st.Samples[i+1].P)
		drawLine2D(afterPanel, a.X, a.Y, b.X, b.Y, 3.2, Color{R: 16, G: 74, B: 54, A: 220})
	}
	for _, s := range st.Samples {
		p := cam.Project(s.P)
		drawDisc(afterPanel, int(p.X), int(p.Y), 5.5, uiGreen)
		drawDisc(afterPanel, int(p.X), int(p.Y), 2.2, RGB(235, 255, 246))
	}
	mid := cam.Project(center)
	labelWithBackground(afterPanel, int(clampFloat(mid.X+18, 12, float64(pw-280))),
		int(clampFloat(mid.Y-44, 12, float64(ph-40))),
		"MATCHING SAMPLES", 2, RGB(255, 255, 255), uiGreen)

	pasteRGBA(canvas, beforePanel, lx, py)
	pasteRGBA(canvas, afterPanel, rx, py)
	strokeRect(canvas, lx, py, pw, ph, 2, uiLine)
	strokeRect(canvas, rx, py, pw, ph, 2, uiLine)
	fillRect(canvas, lx, py, pw, 42, Color{R: 255, G: 255, B: 255, A: 246})
	fillRect(canvas, rx, py, pw, 42, Color{R: 255, G: 255, B: 255, A: 246})
	drawText(canvas, lx+18, py+12, "BEFORE: WELDING ALONE KEEPS A T-JUNCTION", 2, uiRed)
	drawText(canvas, rx+18, py+12, "AFTER: EDGES SPLIT, TRIANGLES COMPLETED", 2, uiGreen)

	drawText(canvas, 34, H-34, "10X ORTHOGRAPHIC SCALE  -  RED CONNECTORS SHOW NEAREST BOUNDARY GAP  -  GREEN NODES ARE ONE SHARED SEAM VERTEX SET", 2, uiMuted)
	return canvas
}

func drawCrackOverlay(panel *image.RGBA, cam Camera, cracks CrackReport, step int) {
	if step < 1 {
		step = 1
	}
	norm := math.Max(cracks.MaxGapMM, 1e-9)
	for i := 0; i < len(cracks.Samples); i += step {
		s := cracks.Samples[i]
		a := cam.Project(s.PA)
		b := cam.Project(s.PB)
		c := heatColor(s.HeatMM / norm)
		drawLine2D(panel, a.X, a.Y, b.X, b.Y, 2.2, c)
	}
	for i := 0; i+step < len(cracks.Samples); i += step {
		s0 := cracks.Samples[i]
		s1 := cracks.Samples[i+step]
		c0 := heatColor(s0.HeatMM / norm)
		c1 := heatColor(s1.HeatMM / norm)
		a0 := cam.Project(s0.PA)
		a1 := cam.Project(s1.PA)
		b0 := cam.Project(s0.PB)
		b1 := cam.Project(s1.PB)
		drawLine2D(panel, a0.X, a0.Y, a1.X, a1.Y, 3.5, blendColor(c0, c1, 0.5))
		drawLine2D(panel, b0.X, b0.Y, b1.X, b1.Y, 3.5, blendColor(c0, c1, 0.5))
	}
}
