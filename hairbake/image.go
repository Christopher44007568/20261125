package main

import (
	"image"
	"image/png"
	"math"
	"os"
)

// RGBA is a simple premultiplied-free RGBA8 canvas (straight alpha).
type RGBA struct {
	W, H int
	Px   []uint8
}

func newRGBA(w, h int) *RGBA {
	return &RGBA{W: w, H: h, Px: make([]uint8, w*h*4)}
}

func (im *RGBA) set(x, y int, c [4]float64) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return
	}
	idx := (y*im.W + x) * 4
	im.Px[idx+0] = uint8(clampf(c[0], 0, 1) * 255)
	im.Px[idx+1] = uint8(clampf(c[1], 0, 1) * 255)
	im.Px[idx+2] = uint8(clampf(c[2], 0, 1) * 255)
	im.Px[idx+3] = uint8(clampf(c[3], 0, 1) * 255)
}

func (im *RGBA) get(x, y int) [4]float64 {
	idx := (y*im.W + x) * 4
	return [4]float64{
		float64(im.Px[idx+0]) / 255,
		float64(im.Px[idx+1]) / 255,
		float64(im.Px[idx+2]) / 255,
		float64(im.Px[idx+3]) / 255,
	}
}

// blendOver alpha-over composites c onto the canvas.
func (im *RGBA) blend(x, y int, c [4]float64) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return
	}
	dst := im.get(x, y)
	a := clampf(c[3], 0, 1)
	out := [4]float64{}
	for k := 0; k < 3; k++ {
		out[k] = c[k]*a + dst[k]*(1-a)
	}
	out[3] = a + dst[3]*(1-a)
	im.set(x, y, out)
}

// fill paints the whole canvas with one opaque color.
func (im *RGBA) fill(c [3]float64) {
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			im.set(x, y, [4]float64{c[0], c[1], c[2], 1})
		}
	}
}

func (im *RGBA) writePNG(path string) error {
	out := image.NewRGBA(image.Rect(0, 0, im.W, im.H))
	copy(out.Pix, im.Px)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

func atlasToRGBA(a *Atlas, flow bool) *RGBA {
	im := newRGBA(a.Size, a.Size)
	src := a.RGBA
	if flow {
		src = a.Flow
	}
	copy(im.Px, src)
	if flow {
		// Make the flow map opaque for inspection.
		for i := 0; i < a.Size*a.Size; i++ {
			im.Px[i*4+3] = 255
		}
	}
	return im
}

// drawLine is a Xiaolin-Wu style antialiased line in straight-alpha color.
func (im *RGBA) drawLine(x0, y0, x1, y1 float64, col [3]float64, alpha, width float64) {
	steep := math.Abs(y1-y0) > math.Abs(x1-x0)
	if steep {
		x0, y0 = y0, x0
		x1, y1 = y1, x1
	}
	if x0 > x1 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
	}
	dx := x1 - x0
	dy := y1 - y0
	grad := 0.0
	if dx != 0 {
		grad = dy / dx
	}
	r := width * 0.5
	for x := x0; x <= x1; x++ {
		y := y0 + grad*(x-x0)
		for oy := -r; oy <= r; oy++ {
			cov := alpha * clampf(r-math.Abs(oy)+0.5, 0, 1)
			px, py := int(math.Round(x)), int(math.Round(y+oy))
			if steep {
				px, py = py, px
			}
			im.blend(px, py, [4]float64{col[0], col[1], col[2], cov})
		}
	}
}

// drawDashedLine overlays a dashed marker line (used for seam callouts).
func (im *RGBA) drawDashedLine(x0, y0, x1, y1 float64, col [3]float64, alpha float64) {
	dx, dy := x1-x0, y1-y0
	length := math.Hypot(dx, dy)
	const dash = 7.0
	for d := 0.0; d < length; d += 2 * dash {
		t0 := d / length
		t1 := math.Min((d+dash)/length, 1)
		im.drawLine(x0+dx*t0, y0+dy*t0, x0+dx*t1, y0+dy*t1, col, alpha, 1.6)
	}
}

// drawRect strokes an axis-aligned rectangle.
func (im *RGBA) drawRect(x, y, w, h int, col [3]float64, alpha float64) {
	im.drawLine(float64(x), float64(y), float64(x+w), float64(y), col, alpha, 1.5)
	im.drawLine(float64(x+w), float64(y), float64(x+w), float64(y+h), col, alpha, 1.5)
	im.drawLine(float64(x+w), float64(y+h), float64(x), float64(y+h), col, alpha, 1.5)
	im.drawLine(float64(x), float64(y+h), float64(x), float64(y), col, alpha, 1.5)
}
