package main

import (
	"fmt"
)

func sideCamera() Camera {
	return NewCamera(V(-5.35, 0.30, 0), V(0, 0.18, 0), 37, 1)
}

func renderSideDiff(path string) error {
	scene := buildScene()
	cam := sideCamera()
	const rw, rh = 760, 760
	fmt.Println("rendering side shell mask...")
	shellFur := renderShell(&scene, cam, rw, rh)
	fmt.Println("rendering side strand mask...")
	strandFur := renderStrand(&scene, cam, rw, rh, 1)
	shellMask := maskFromAlpha(shellFur, 0.90)
	strandMask := maskFromAlpha(strandFur, 0.90)
	b := unionBounds(shellMask, strandMask)
	stats := compareMasks(shellMask, strandMask)

	out := NewCanvas(1800, 900)
	for y := 0; y < out.H; y++ {
		ny := 1 - 2*(float64(y)+0.5)/float64(out.H)
		for x := 0; x < out.W; x++ {
			nx := 2*(float64(x)+0.5)/float64(out.W) - 1
			col := bgColor(nx, ny).Mul(0.68)
			out.Pix[y*out.W+x] = HDR{col.R, col.G, col.B, 1}
		}
	}
	panelY, panelH := 112, 640
	panels := []Panel{
		{X: 64, Y: panelY, W: 500, H: panelH},
		{X: 650, Y: panelY, W: 500, H: panelH},
		{X: 1236, Y: panelY, W: 500, H: panelH},
	}
	panelBG := C(0.018, 0.035, 0.041)
	border := C(0.24, 0.32, 0.31)
	for _, p := range panels {
		fillRect(out, p, panelBG, 1)
		strokeRect(out, p, border, 0.55, 1)
	}

	shellTop, shellBottom := C(0.98, 0.54, 0.16), C(0.34, 0.11, 0.035)
	strandTop, strandBottom := C(0.22, 0.83, 0.74), C(0.025, 0.20, 0.23)
	inner := func(p Panel) Panel {
		return Panel{X: p.X + 8, Y: p.Y + 38, W: p.W - 16, H: p.H - 48}
	}
	drawMaskPanel(out, shellMask, b, inner(panels[0]), shellTop, shellBottom, C(1.25, 0.78, 0.34))
	drawMaskPanel(out, strandMask, b, inner(panels[1]), strandTop, strandBottom, C(0.45, 1.15, 1.02))
	drawDiffPanel(out, shellMask, strandMask, b, inner(panels[2]),
		C(0.98, 0.38, 0.075), C(0.06, 0.78, 0.73), C(0.48, 0.50, 0.43))

	labelCol := C(0.80, 0.75, 0.62)
	subCol := C(0.42, 0.55, 0.53)
	drawText(out, 64, 30, "SIDE SILHOUETTE DIFFERENCE", 4, labelCol, 0.94, 4)
	drawText(out, 66, 73, "SAME GROOM / ALPHA THRESHOLD 0.90 / ORTHOGRAPHIC-LIKE SIDE VIEW", 2, subCol, 0.90, 2)
	drawText(out, 66, 96, "MUZZLE +Z  /  CAMERA -X", 2, C(0.64, 0.40, 0.20), 0.88, 2)

	titles := []string{"SHELL PROFILE", "STRAND PROFILE", "MASK DIFFERENCE"}
	notes := []string{"FAR LOD / CONTINUOUS ENVELOPE", "NEAR LOD / EXPLICIT FIBERS", "AMBER SHELL-ONLY / CYAN STRAND-ONLY"}
	for i, p := range panels {
		drawText(out, p.X+16, p.Y+15, titles[i], 3, labelCol, 0.92, 2)
		drawText(out, p.X+16, p.Y+p.H+18, notes[i], 2, subCol, 0.90, 2)
	}

	fillRect(out, Panel{X: panels[2].X + 8, Y: panels[2].Y + panels[2].H - 78, W: 484, H: 64},
		C(0.006, 0.014, 0.018), 0.90)
	drawLegendSwatch(out, panels[2].X+18, panels[2].Y+panels[2].H-68, 22, 14, C(0.98, 0.38, 0.075), "SHELL ONLY", 0.92)
	drawLegendSwatch(out, panels[2].X+18, panels[2].Y+panels[2].H-40, 22, 14, C(0.06, 0.78, 0.73), "STRAND ONLY", 0.92)
	drawLegendSwatch(out, panels[2].X+236, panels[2].Y+panels[2].H-68, 22, 14, C(0.48, 0.50, 0.43), "OVERLAP", 0.92)

	metric := fmt.Sprintf("IOU %.3f   XOR %.1f%%   AREA %+.1f%%", stats.IoU, stats.Xor*100, stats.AreaDelta*100)
	drawText(out, 64, 824, metric, 3, labelCol, 0.95, 3)
	drawText(out, 64, 858, "DIFFERENCE IS MEASURED FROM THE SAME CAMERA AND CROP BOX", 2, subCol, 0.90, 2)

	applyBloom(out)
	applyFinish(out)
	return savePNG(path, out)
}
