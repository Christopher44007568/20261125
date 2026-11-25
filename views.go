package main

func makeMap(w, h int, x0, y0, x1, y1 float64) Map {
	pad := 60.0
	sx := float64(w) / (x1 - x0 + pad*2)
	sy := float64(h) / (y1 - y0 + pad*2)
	scale := sx
	if sy < scale {
		scale = sy
	}
	return Map{
		X0:    float64(w)/2 - (x0+x1)/2*scale,
		Y0:    float64(h)/2 - (y0+y1)/2*scale,
		Scale: scale,
		H:     h,
	}
}

func renderFeatherLayer(w, h int, m Map, f Feather, heat bool) *Layer {
	l := newLayer(w, h)
	for _, b := range f.Barbs {
		drawBarb(l, m, b, 30, heat)
	}
	drawRachis(l, m, heat)
	return l
}

func topView(f Feather) (*Layer, Map) {
	w, h := 760, 1180
	m := makeMap(w, h, -210, -580, 240, 590)
	l := renderFeatherLayer(w, h, m, f, false)
	drawAnchors(l, m, f, 4, 7)
	drawText(l, 28, 28, 3, "TOP VIEW", colCore)
	drawText(l, 28, h-44, 2, "WIND BEND - FIXED RACHIS", colCore)
	drawWindArrow(l, 110, 120, f)
	return l, m
}

func featherOnlyView(f Feather) *Layer {
	w, h := 760, 1180
	m := makeMap(w, h, -210, -580, 240, 590)
	l := renderFeatherLayer(w, h, m, f, false)
	drawAnchors(l, m, f, 4, 7)
	return l
}

func obliqueView(src *Layer, f Feather) *Layer {
	w, h := 1180, 900
	out := newLayer(w, h)
	srcPts := [4]Vec{{0, 0}, {float64(src.W), 0}, {float64(src.W), float64(src.H)}, {0, float64(src.H)}}
	dstPts := [4]Vec{{470, 70}, {950, 215}, {740, 805}, {265, 650}}
	homo := findHomography(srcPts, dstPts)
	inv, ok := matInv(homo)
	if !ok {
		panic("invalid homography")
	}
	warped := warpLayer(src, inv, w, h)
	for i := range out.P {
		out.P[i] = warped.P[i]
	}
	drawText(out, 28, 28, 3, "OBLIQUE VIEW", colCore)
	drawText(out, 28, h-44, 2, "PERSPECTIVE WARP - ALPHA PRESERVED", colCore)
	return out
}

func closeupView(f Feather) *Layer {
	w, h := 1120, 860
	plotW, plotH := 800, h
	rx0, rx1 := -225.0, 235.0
	ry0, ry1 := -390.0, -90.0
	m := makeMap(plotW, plotH, rx0, ry0, rx1, ry1)
	l := renderFeatherLayer(plotW, plotH, m, f, true)
	indices := []int{12, 16, 20, 24}
	drawTangentArrows(l, m, f, indices)
	drawAnchors(l, m, f, 4, 6.5)
	drawWeightLabels(l, m, f, 20, []float64{0.15, 0.5, 0.85})

	out := newLayer(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < plotW; x++ {
			si, di := l.idx(x, y), out.idx(x, y)
			for k := 0; k < 4; k++ {
				out.P[di+k] = l.P[si+k]
			}
		}
	}
	drawText(out, plotW+54, 110, 3, "WEIGHT", colCore)
	drawColorbar(out, plotW+70, 180, 26, 320)
	drawText(out, plotW+54, 560, 2, "TANGENT", colArrow)
	drawArrow(out, float64(plotW+110), 620, 0, 90, 4, colArrow)
	drawText(out, plotW+54, 690, 2, "ANCHOR", colAnchor)
	out.ring(float64(plotW+154), 704, 13, 2.4, colAnchor)
	drawText(out, 28, 28, 2, "CLOSEUP 6X - BARB FIELD", colCore)
	drawText(out, 28, h-44, 2, "COVERAGE WEIGHT AND DIRECTION", colCore)
	return out
}
