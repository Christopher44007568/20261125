package main

import (
	"image"
	"math"
)

func blendPixel(img *image.RGBA, x, y int, c RGB, alpha float64) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	alpha = clamp(alpha, 0, 1)
	o := y*img.Stride + x*4
	srcR, srcG, srcB := c.Bytes()
	r := float64(img.Pix[o+0])*(1-alpha) + float64(srcR)*alpha
	g := float64(img.Pix[o+1])*(1-alpha) + float64(srcG)*alpha
	b := float64(img.Pix[o+2])*(1-alpha) + float64(srcB)*alpha
	img.Pix[o+0] = uint8(clamp(r+0.5, 0, 255))
	img.Pix[o+1] = uint8(clamp(g+0.5, 0, 255))
	img.Pix[o+2] = uint8(clamp(b+0.5, 0, 255))
	img.Pix[o+3] = 255
}

func fillBackground(img *image.RGBA, top, bottom RGB) {
	b := img.Bounds()
	for y := 0; y < b.Dy(); y++ {
		t := float64(y) / float64(maxInt(1, b.Dy()-1))
		c := top.Lerp(bottom, t)
		for x := 0; x < b.Dx(); x++ {
			o := y*img.Stride + x*4
			r, g, bl := c.Bytes()
			img.Pix[o+0] = r
			img.Pix[o+1] = g
			img.Pix[o+2] = bl
			img.Pix[o+3] = 255
		}
	}
}

func addRadialGlow(img *image.RGBA, cx, cy, radius float64, c RGB, strength float64) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dx := (float64(x) - cx) / radius
			dy := (float64(y) - cy) / radius
			d := math.Sqrt(dx*dx + dy*dy)
			if d >= 1 {
				continue
			}
			a := strength * math.Pow(1-d, 2)
			blendPixel(img, x, y, c, a)
		}
	}
}

func drawPanelBackground(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fillBackground(img,
		RGB{0.043, 0.052, 0.069},
		RGB{0.020, 0.025, 0.035},
	)
	addRadialGlow(img, float64(w)*0.52, float64(h)*0.46, float64(w)*0.52,
		RGB{0.075, 0.105, 0.140}, 0.55)
	return img
}

func outlineCoverage(img *image.RGBA, coverage []float32, c RGB) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			idx := y*w + x
			if coverage[idx] < 0.46 {
				continue
			}
			if coverage[idx-1] < 0.20 || coverage[idx+1] < 0.20 ||
				coverage[idx-w] < 0.20 || coverage[idx+w] < 0.20 {
				blendPixel(img, x, y, c, 0.72)
			}
		}
	}
}

func drawFrame(img *image.RGBA, inset int, accent RGB) {
	b := img.Bounds()
	line := RGB{0.12, 0.15, 0.20}
	for x := inset; x < b.Dx()-inset; x++ {
		blendPixel(img, x, inset, line, 0.9)
		blendPixel(img, x, b.Dy()-inset-1, line, 0.9)
	}
	for y := inset; y < b.Dy()-inset; y++ {
		blendPixel(img, inset, y, line, 0.9)
		blendPixel(img, b.Dx()-inset-1, y, line, 0.9)
	}
	for y := inset; y < inset+38; y++ {
		blendPixel(img, inset, y, accent, 0.85)
	}
}

func drawPassPill(img *image.RGBA, x, y int, pass bool) {
	c := RGB{0.31, 0.82, 0.49}
	label := "G0 + G1 PASS"
	if !pass {
		c = RGB{0.96, 0.34, 0.31}
		label = "CHECK FAILED"
	}
	w := textWidth(label, 1) + 18
	for iy := 0; iy < 18; iy++ {
		for ix := 0; ix < w; ix++ {
			blendPixel(img, x+ix, y+iy, c, 0.16)
		}
	}
	for ix := 0; ix < w; ix++ {
		blendPixel(img, x+ix, y, c, 0.7)
		blendPixel(img, x+ix, y+17, c, 0.7)
	}
	drawText(img, x+9, y+5, label, 1, c)
}
