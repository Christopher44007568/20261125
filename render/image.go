package render

import (
	"bufio"
	"fmt"
	"io"
	"math"
)

type RGB struct {
	R, G, B uint8
}

type Image struct {
	W, H int
	Pix  []RGB
}

func NewImage(w, h int, bg RGB) *Image {
	im := &Image{W: w, H: h, Pix: make([]RGB, w*h)}
	for i := range im.Pix {
		im.Pix[i] = bg
	}
	return im
}

func Crop(src *Image, x, y, w, h int) *Image {
	dst := NewImage(w, h, Background)
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			if x+xx >= 0 && x+xx < src.W && y+yy >= 0 && y+yy < src.H {
				dst.Pix[yy*w+xx] = src.Pix[(y+yy)*src.W+x+xx]
			}
		}
	}
	return dst
}

func (im *Image) Set(x, y int, c RGB) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return
	}
	im.Pix[y*im.W+x] = c
}

func (im *Image) Blend(x, y int, c RGB, alpha float64) {
	if x < 0 || y < 0 || x >= im.W || y >= im.H {
		return
	}
	if alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	i := y*im.W + x
	bg := im.Pix[i]
	im.Pix[i] = RGB{
		R: uint8(float64(bg.R)*(1-alpha) + float64(c.R)*alpha),
		G: uint8(float64(bg.G)*(1-alpha) + float64(c.G)*alpha),
		B: uint8(float64(bg.B)*(1-alpha) + float64(c.B)*alpha),
	}
}

type PixelPoint struct {
	X, Y float64
}

func (im *Image) Rect(x0, y0, x1, y1 int, c RGB) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			im.Set(x, y, c)
		}
	}
}

func (im *Image) FillTriangle(a, b, c PixelPoint, color RGB, alpha float64) {
	minX := int(math.Floor(math.Min(a.X, math.Min(b.X, c.X))))
	maxX := int(math.Ceil(math.Max(a.X, math.Max(b.X, c.X))))
	minY := int(math.Floor(math.Min(a.Y, math.Min(b.Y, c.Y))))
	maxY := int(math.Ceil(math.Max(a.Y, math.Max(b.Y, c.Y))))
	area := edge(a, b, c)
	if math.Abs(area) < 1e-9 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := PixelPoint{X: float64(x) + 0.5, Y: float64(y) + 0.5}
			w0, w1, w2 := edge(b, c, p), edge(c, a, p), edge(a, b, p)
			if area > 0 {
				if w0 >= 0 && w1 >= 0 && w2 >= 0 {
					im.Blend(x, y, color, alpha)
				}
			} else if w0 <= 0 && w1 <= 0 && w2 <= 0 {
				im.Blend(x, y, color, alpha)
			}
		}
	}
}

func edge(a, b, p PixelPoint) float64 {
	return (p.X-a.X)*(b.Y-a.Y) - (p.Y-a.Y)*(b.X-a.X)
}

func (im *Image) Line(a, b PixelPoint, color RGB, width int) {
	if width < 1 {
		width = 1
	}
	dx, dy := b.X-a.X, b.Y-a.Y
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps == 0 {
		im.Disc(int(math.Round(a.X)), int(math.Round(a.Y)), width/2, color)
		return
	}
	radius := (width - 1) / 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(a.X + dx*t))
		y := int(math.Round(a.Y + dy*t))
		if radius == 0 {
			im.Set(x, y, color)
		} else {
			im.Disc(x, y, radius, color)
		}
	}
}

func (im *Image) Disc(cx, cy, r int, color RGB) {
	if r <= 0 {
		im.Set(cx, cy, color)
		return
	}
	r2 := r*r + r
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r2 {
				im.Set(cx+x, cy+y, color)
			}
		}
	}
}

func (im *Image) Circle(cx, cy, r int, color RGB, width int) {
	steps := int(math.Max(24, float64(r)*8))
	for i := 0; i < steps; i++ {
		t0 := 2 * math.Pi * float64(i) / float64(steps)
		t1 := 2 * math.Pi * float64(i+1) / float64(steps)
		im.Line(PixelPoint{X: float64(cx) + float64(r)*math.Cos(t0), Y: float64(cy) + float64(r)*math.Sin(t0)},
			PixelPoint{X: float64(cx) + float64(r)*math.Cos(t1), Y: float64(cy) + float64(r)*math.Sin(t1)},
			color, width)
	}
}

func WritePPM(w io.Writer, im *Image) error {
	bw := bufio.NewWriterSize(w, 64*1024)
	if _, err := fmt.Fprintf(bw, "P6\n# ruled surface intersection diagnostic\n%d %d\n255\n", im.W, im.H); err != nil {
		return err
	}
	for _, p := range im.Pix {
		if err := bw.WriteByte(p.R); err != nil {
			return err
		}
		if err := bw.WriteByte(p.G); err != nil {
			return err
		}
		if err := bw.WriteByte(p.B); err != nil {
			return err
		}
	}
	return bw.Flush()
}
