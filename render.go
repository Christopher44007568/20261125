package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
)

// Canvas is a minimal RGBA pixel buffer with alpha blending.
type Canvas struct {
	W, H int
	Pix  []color.RGBA
}

func NewCanvas(w, h int, bg color.RGBA) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]color.RGBA, w*h)}
	for i := range c.Pix {
		c.Pix[i] = bg
	}
	return c
}

func (c *Canvas) Set(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Pix[y*c.W+x] = col
}

// Blend alpha-composites col over the existing pixel.
func (c *Canvas) Blend(x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	d := &c.Pix[y*c.W+x]
	a := float32(col.A) / 255
	d.R = uint8(float32(col.R)*a + float32(d.R)*(1-a))
	d.G = uint8(float32(col.G)*a + float32(d.G)*(1-a))
	d.B = uint8(float32(col.B)*a + float32(d.B)*(1-a))
	d.A = 255
}

// Camera is an orthographic front view: we look along +Z, so the screen
// shows world X (right) and world Y (up); depth is Z.
type Camera struct {
	CX, CY float32 // screen center
	Scale  float32
}

func (cam Camera) Project(p Vec3) (float32, float32) {
	return cam.CX + p.X*cam.Scale, cam.CY - p.Y*cam.Scale
}

// DrawGroom renders strands back-to-front with a root->tip color ramp.
// fallback strands are drawn in magenta so they are easy to spot.
func DrawGroom(img *Canvas, cam Camera, g *Groom, isFallback []bool) {
	idx := make([]int, len(g.Strands))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool {
		return g.Strands[idx[a]].Root().Z > g.Strands[idx[b]].Root().Z
	})
	for _, i := range idx {
		s := g.Strands[i]
		fb := isFallback != nil && i < len(isFallback) && isFallback[i]
		n := len(s.Pts)
		for j := 0; j < n-1; j++ {
			t := float32(j) / float32(n-1)
			var col color.RGBA
			if fb {
				col = color.RGBA{255, 47, 179, 200}
			} else {
				col = color.RGBA{
					R: uint8(43 + t*155),
					G: uint8(26 + t*129),
					B: uint8(16 + t*90),
					A: 190,
				}
			}
			x0, y0 := cam.Project(s.Pts[j])
			x1, y1 := cam.Project(s.Pts[j+1])
			drawLine(img, x0, y0, x1, y1, 1.1, col)
		}
	}
}

// drawLine draws a thick anti-aliased-ish segment by splatting discs.
func drawLine(img *Canvas, x0, y0, x1, y1, radius float32, col color.RGBA) {
	dx, dy := x1-x0, y1-y0
	steps := int(math.Sqrt(float64(dx*dx+dy*dy)))*2 + 1
	for i := 0; i <= steps; i++ {
		t := float32(i) / float32(steps)
		splat(img, x0+dx*t, y0+dy*t, radius, col)
	}
}

func splat(img *Canvas, cx, cy, r float32, col color.RGBA) {
	ir := int(r + 0.5)
	for dy := -ir; dy <= ir; dy++ {
		for dx := -ir; dx <= ir; dx++ {
			d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			if d > r+0.5 {
				continue
			}
			fall := 1 - d/(r+0.5)
			c := col
			c.A = uint8(float32(col.A) * fall)
			img.Blend(int(cx)+dx, int(cy)+dy, c)
		}
	}
}

// drawRectOutline marks a missing chunk's projected region.
func drawRectOutline(img *Canvas, x0, y0, x1, y1 int, col color.RGBA) {
	for x := x0; x <= x1; x++ {
		img.Set(x, y0, col)
		img.Set(x, y0+1, col)
		img.Set(x, y1, col)
		img.Set(x, y1-1, col)
	}
	for y := y0; y <= y1; y++ {
		img.Set(x0, y, col)
		img.Set(x0+1, y, col)
		img.Set(x1, y, col)
		img.Set(x1-1, y, col)
	}
	// Cross in the middle.
	cx, cy := (x0+x1)/2, (y0+y1)/2
	for d := -6; d <= 6; d++ {
		img.Set(cx+d, cy+d, col)
		img.Set(cx+d, cy-d, col)
	}
}

// heatColor maps t in [0,1] to a blue->cyan->green->yellow->red ramp.
func heatColor(t float32) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	x := t * 4
	switch {
	case x < 1:
		return color.RGBA{0, uint8(x * 255), 255, 255}
	case x < 2:
		return color.RGBA{0, 255, uint8((2 - x) * 255), 255}
	case x < 3:
		return color.RGBA{uint8((x - 2) * 255), 255, 0, 255}
	default:
		return color.RGBA{255, uint8((4 - x) * 255), 0, 255}
	}
}

// Downsample averages sxS blocks (supersample factor 2).
func Downsample(src *Canvas) *Canvas {
	w, h := src.W/2, src.H/2
	out := NewCanvas(w, h, color.RGBA{0, 0, 0, 255})
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b int
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					p := src.Pix[(2*y+dy)*src.W+2*x+dx]
					r += int(p.R)
					g += int(p.G)
					b += int(p.B)
				}
			}
			out.Pix[y*w+x] = color.RGBA{uint8(r / 4), uint8(g / 4), uint8(b / 4), 255}
		}
	}
	return out
}

func (c *Canvas) SavePNG(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			img.SetRGBA(x, y, c.Pix[y*c.W+x])
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// drawMemBars draws a VRAM comparison strip: three bars for the tiers.
func drawMemBars(img *Canvas, x, y, maxW, h int, labels []string, bytes []int, colors []color.RGBA) {
	maxB := 1
	for _, b := range bytes {
		if b > maxB {
			maxB = b
		}
	}
	for i, b := range bytes {
		rowY := y + i*(h+14)
		w := maxW * b / maxB
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				img.Set(x+dx, rowY+dy, colors[i])
			}
		}
		drawText(img, x, rowY+h+2, 2, labels[i], color.RGBA{230, 230, 230, 255})
	}
}
