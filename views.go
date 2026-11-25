package main

import (
	"fmt"
	"image/color"
)

var (
	colHead    = color.RGBA{58, 49, 42, 255}
	colBG      = color.RGBA{16, 15, 18, 255}
	colMissing = color.RGBA{255, 60, 50, 255}
	colText    = color.RGBA{235, 235, 235, 255}
)

// renderGroomImage draws one groom view with markers, title and VRAM bars.
func renderGroomImage(path string, cam Camera, w, h int, g *Groom,
	isFallback []bool, rects []intRect, title string,
	memLabels []string, memBytes []int, memColors []color.RGBA) {
	img := NewCanvas(w, h, colBG)
	// Faint head disc for silhouette context.
	cx, cy := cam.Project(Vec3{})
	splat(img, cx, cy, headRadius*cam.Scale, color.RGBA{colHead.R, colHead.G, colHead.B, 255})
	DrawGroom(img, cam, g, isFallback)
	for _, r := range rects {
		drawRectOutline(img, r.x0, r.y0, r.x1, r.y1, colMissing)
	}
	final := Downsample(img)
	drawText(final, 12, 10, 3, title, colText)
	for _, r := range rects {
		drawText(final, r.x0/2, r.y0/2-16, 2, "MISSING", colMissing)
	}
	if len(rects) > 0 {
		legendY := 34
		splat(final, 20, float32(legendY+5), 4, color.RGBA{255, 47, 179, 255})
		drawText(final, 32, legendY, 2, "FALLBACK STRAND", colText)
	}
	drawMemBars(final, 12, final.H-104, 220, 10, memLabels, memBytes, memColors)
	drawText(final, 12, final.H-118, 2, "VRAM", colText)
	if err := final.SavePNG(path); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", path)
}

// renderHeatmap splats per-point 3D error into a colored heatmap.
func renderHeatmap(path string, cam Camera, w, h int, orig *Groom,
	res *LoadResult, rects []intRect, maxErr float32) {
	heat := make([]float32, w*h)
	splatErr := func(x, y float32, e float32) {
		ix, iy := int(x), int(y)
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				px, py := ix+dx, iy+dy
				if px < 0 || py < 0 || px >= w || py >= h {
					continue
				}
				if e > heat[py*w+px] {
					heat[py*w+px] = e
				}
			}
		}
	}
	for k, rs := range res.Groom.Strands {
		oi := res.OrigIdx[k]
		if oi < 0 {
			continue
		}
		os := orig.Strands[oi]
		ref := os
		if len(rs.Pts) != len(os.Pts) {
			ref = resample(os, len(rs.Pts))
		}
		for j := 1; j < len(rs.Pts)-1; j++ {
			e := ref.Pts[j].Dist(rs.Pts[j])
			x, y := cam.Project(rs.Pts[j])
			splatErr(x, y, e)
		}
	}
	img := NewCanvas(w, h, color.RGBA{8, 8, 12, 255})
	if maxErr <= 0 {
		maxErr = 1
	}
	for i, e := range heat {
		if e > 0 {
			img.Pix[i] = heatColor(e / maxErr)
		}
	}
	for _, r := range rects {
		drawRectOutline(img, r.x0, r.y0, r.x1, r.y1, colMissing)
	}
	final := Downsample(img)
	drawText(final, 12, 10, 3, "ERROR HEATMAP", colText)
	for _, r := range rects {
		drawText(final, r.x0/2, r.y0/2-16, 2, "MISSING", colMissing)
	}
	// Scale bar: 0 .. maxErr.
	bx, by, bw := 12, final.H-40, 220
	for dx := 0; dx < bw; dx++ {
		c := heatColor(float32(dx) / float32(bw))
		for dy := 0; dy < 10; dy++ {
			final.Set(bx+dx, by+dy, c)
		}
	}
	drawText(final, bx, by+14, 2, "0", colText)
	maxLabel := fmt.Sprintf("MAX %.3f", maxErr)
	drawText(final, bx+bw-textWidth(2, maxLabel), by+14, 2, maxLabel, colText)
	if err := final.SavePNG(path); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", path)
}
