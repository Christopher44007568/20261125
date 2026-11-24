package main

import "image"
import "math"

func newCanvas(w, h int, bg Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			setPixel(img, x, y, bg)
		}
	}
	return img
}

func fillRect(img *image.RGBA, x, y, w, h int, c Color) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			setPixel(img, xx, yy, c)
		}
	}
}

func strokeRect(img *image.RGBA, x, y, w, h, thick int, c Color) {
	fillRect(img, x, y, w, thick, c)
	fillRect(img, x, y+h-thick, w, thick, c)
	fillRect(img, x, y, thick, h, c)
	fillRect(img, x+w-thick, y, thick, h, c)
}

func drawLine2D(img *image.RGBA, x0, y0, x1, y1, width float64, c Color) {
	steps := int(math.Ceil(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))))
	if steps == 0 {
		drawDisc(img, int(math.Round(x0)), int(math.Round(y0)), width/2, c)
		return
	}
	radius := int(math.Ceil(width / 2))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(x0 + (x1-x0)*t))
		y := int(math.Round(y0 + (y1-y0)*t))
		drawDisc(img, x, y, float64(radius), c)
	}
}

func drawDisc(img *image.RGBA, cx, cy int, radius float64, c Color) {
	r := int(math.Ceil(radius))
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := float64(x) + 0.5 - (float64(cx) + 0.5)
			dy := float64(y) + 0.5 - (float64(cy) + 0.5)
			if dx*dx+dy*dy <= radius*radius+0.5 {
				setPixel(img, x, y, c)
			}
		}
	}
}

func pasteRGBA(dst, src *image.RGBA, x, y int) {
	b := src.Bounds()
	for sy := b.Min.Y; sy < b.Max.Y; sy++ {
		for sx := b.Min.X; sx < b.Max.X; sx++ {
			sc := src.RGBAAt(sx, sy)
			setPixel(dst, x+sx-b.Min.X, y+sy-b.Min.Y, Color{R: sc.R, G: sc.G, B: sc.B, A: sc.A})
		}
	}
}

func labelWithBackground(img *image.RGBA, x, y int, text string, scale int, fg, bg Color) {
	w := textWidth(text, scale) + 10*scale
	h := 9 * scale
	fillRect(img, x-4*scale, y-3*scale, w, h, bg)
	drawText(img, x+scale, y+scale, text, scale, fg)
}
