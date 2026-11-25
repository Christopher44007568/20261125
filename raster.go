package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type Layer struct {
	W, H int
	P    []float64
}

func newLayer(w, h int) *Layer {
	return &Layer{W: w, H: h, P: make([]float64, w*h*4)}
}

func (l *Layer) idx(x, y int) int { return (y*l.W + x) * 4 }

type RGB struct{ R, G, B float64 }

func (l *Layer) stamp(x, y, radius, coverage float64, c RGB) {
	r := math.Ceil(radius + 1)
	x0, x1 := int(math.Floor(x-r)), int(math.Ceil(x+r))
	y0, y1 := int(math.Floor(y-r)), int(math.Ceil(y+r))
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 >= l.W {
		x1 = l.W - 1
	}
	if y1 >= l.H {
		y1 = l.H - 1
	}
	inner := math.Max(0, radius-1.2)
	inner *= inner
	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			dx, dy := float64(px)+0.5-x, float64(py)+0.5-y
			d2 := dx*dx + dy*dy
			if d2 > (radius+1)*(radius+1) {
				continue
			}
			a := coverage
			if d2 > inner {
				a *= clamp01(radius + 0.5 - math.Sqrt(d2))
			}
			if a <= 0 {
				continue
			}
			i := l.idx(px, py)
			sa := clamp01(a)
			inv := 1 - sa
			l.P[i] = c.R*sa + l.P[i]*inv
			l.P[i+1] = c.G*sa + l.P[i+1]*inv
			l.P[i+2] = c.B*sa + l.P[i+2]*inv
			l.P[i+3] = sa + l.P[i+3]*inv
		}
	}
}

func (l *Layer) disc(x, y, radius float64, c RGB) {
	l.stamp(x, y, radius, 1, c)
}

func (l *Layer) ring(x, y, radius, thickness float64, c RGB) {
	steps := int(radius*8) + 8
	for i := 0; i < steps; i++ {
		a := float64(i) / float64(steps) * 2 * math.Pi
		l.stamp(x+math.Cos(a)*radius, y+math.Sin(a)*radius, thickness, 1, c)
	}
}

func (l *Layer) line(x0, y0, x1, y1, radius float64, c RGB) {
	d := math.Hypot(x1-x0, y1-y0)
	n := int(d/math.Max(0.5, radius*0.7)) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		l.stamp(lerp(x0, x1, t), lerp(y0, y1, t), radius, 1, c)
	}
}

func composite(bg *image.RGBA, l *Layer) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, l.W, l.H))
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			i := l.idx(x, y)
			sa := clamp01(l.P[i+3])
			bi := bg.PixOffset(x, y)
			br := float64(bg.Pix[bi]) / 255
			bgC := float64(bg.Pix[bi+1]) / 255
			bb := float64(bg.Pix[bi+2]) / 255
			var r, g, b float64
			if sa > 0 {
				r = l.P[i]/sa*(sa) + br*(1-sa)
				g = l.P[i+1]/sa*(sa) + bgC*(1-sa)
				b = l.P[i+2]/sa*(sa) + bb*(1-sa)
			} else {
				r, g, b = br, bgC, bb
			}
			o := out.PixOffset(x, y)
			out.Pix[o] = uint8(clamp01(r) * 255)
			out.Pix[o+1] = uint8(clamp01(g) * 255)
			out.Pix[o+2] = uint8(clamp01(b) * 255)
			out.Pix[o+3] = 255
		}
	}
	return out
}

func layerToRGBA(l *Layer) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, l.W, l.H))
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			i := l.idx(x, y)
			sa := clamp01(l.P[i+3])
			o := out.PixOffset(x, y)
			if sa <= 0 {
				continue
			}
			out.Pix[o] = uint8(clamp01(l.P[i]/sa) * 255)
			out.Pix[o+1] = uint8(clamp01(l.P[i+1]/sa) * 255)
			out.Pix[o+2] = uint8(clamp01(l.P[i+2]/sa) * 255)
			out.Pix[o+3] = uint8(sa * 255)
		}
	}
	return out
}

func alphaToGray(l *Layer) *image.Gray {
	out := image.NewGray(image.Rect(0, 0, l.W, l.H))
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			i := l.idx(x, y)
			out.SetGray(x, y, color.Gray{Y: uint8(clamp01(l.P[i+3]) * 255)})
		}
	}
	return out
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func checkerBackground(w, h, cell int) *image.RGBA {
	bg := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			light := ((x/cell)+(y/cell))%2 == 0
			var r, g, b uint8
			if light {
				r, g, b = 238, 240, 243
			} else {
				r, g, b = 202, 208, 216
			}
			if x > w*2/3 {
				r = uint8(float64(r)*0.55 + 34)
				g = uint8(float64(g)*0.55 + 44)
				b = uint8(float64(b)*0.55 + 78)
			}
			o := bg.PixOffset(x, y)
			bg.Pix[o], bg.Pix[o+1], bg.Pix[o+2], bg.Pix[o+3] = r, g, b, 255
		}
	}
	return bg
}

func warpLayer(src *Layer, inv Mat3, w, h int) *Layer {
	dst := newLayer(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx, sy := matApply(inv, float64(x)+0.5, float64(y)+0.5)
			fx, fy := sx-0.5, sy-0.5
			x0 := int(math.Floor(fx))
			y0 := int(math.Floor(fy))
			if x0 < 0 || y0 < 0 || x0+1 >= src.W || y0+1 >= src.H {
				continue
			}
			tx, ty := fx-float64(x0), fy-float64(y0)
			i00 := src.idx(x0, y0)
			i10 := src.idx(x0+1, y0)
			i01 := src.idx(x0, y0+1)
			i11 := src.idx(x0+1, y0+1)
			o := dst.idx(x, y)
			for k := 0; k < 4; k++ {
				v0 := lerp(src.P[i00+k], src.P[i10+k], tx)
				v1 := lerp(src.P[i01+k], src.P[i11+k], tx)
				dst.P[o+k] = lerp(v0, v1, ty)
			}
		}
	}
	return dst
}
