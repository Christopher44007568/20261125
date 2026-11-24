package main

import (
	"fmt"
	"math"
	"os"
)

type Color struct {
	R, G, B uint8
}

var (
	bgColor     = Color{246, 249, 252}
	panelColor  = Color{255, 255, 255}
	textColor   = Color{31, 40, 52}
	mutedColor  = Color{83, 96, 112}
	borderColor = Color{139, 153, 169}
	gridColor   = Color{225, 232, 241}
	outerColor  = Color{38, 83, 145}
	innerColor  = Color{166, 102, 17}
	sectionFill = Color{218, 231, 247}
	redColor    = Color{220, 24, 35}
	redFill     = Color{250, 205, 205}
	orangeFill  = Color{249, 231, 205}
)

type Canvas struct {
	W, H                           int
	SS                             int
	Pix                            []uint8
	clip                           bool
	clipX0, clipY0, clipX1, clipY1 int
}

func NewCanvas(w, h, ss int) *Canvas {
	if ss < 1 {
		ss = 1
	}
	c := &Canvas{W: w, H: h, SS: ss, Pix: make([]uint8, w*h*ss*ss*3)}
	return c
}

func (c *Canvas) iw() int { return c.W * c.SS }
func (c *Canvas) ih() int { return c.H * c.SS }

func (c *Canvas) SetInternal(x, y int, col Color) {
	if x < 0 || y < 0 || x >= c.iw() || y >= c.ih() {
		return
	}
	if c.clip && (x < c.clipX0 || y < c.clipY0 ||
		x >= c.clipX1 || y >= c.clipY1) {
		return
	}
	off := (y*c.iw() + x) * 3
	c.Pix[off], c.Pix[off+1], c.Pix[off+2] = col.R, col.G, col.B
}

func (c *Canvas) SetClipRect(x0, y0, x1, y1 float64) {
	ss := float64(c.SS)
	c.clip = true
	c.clipX0 = int(math.Floor(x0 * ss))
	c.clipY0 = int(math.Floor(y0 * ss))
	c.clipX1 = int(math.Ceil(x1 * ss))
	c.clipY1 = int(math.Ceil(y1 * ss))
}

func (c *Canvas) ClearClip() { c.clip = false }

func (c *Canvas) Fill(col Color) {
	for i := 0; i < len(c.Pix); i += 3 {
		c.Pix[i], c.Pix[i+1], c.Pix[i+2] = col.R, col.G, col.B
	}
}

func (c *Canvas) FillRect(x0, y0, x1, y1 float64, col Color) {
	ss := float64(c.SS)
	xa, ya := int(math.Floor(x0*ss)), int(math.Floor(y0*ss))
	xb, yb := int(math.Ceil(x1*ss)), int(math.Ceil(y1*ss))
	for y := ya; y < yb; y++ {
		for x := xa; x < xb; x++ {
			c.SetInternal(x, y, col)
		}
	}
}

func (c *Canvas) StrokeRect(x0, y0, x1, y1, width float64, col Color) {
	c.Line(x0, y0, x1, y0, width, col)
	c.Line(x1, y0, x1, y1, width, col)
	c.Line(x1, y1, x0, y1, width, col)
	c.Line(x0, y1, x0, y0, width, col)
}

func (c *Canvas) Line(x0, y0, x1, y1, width float64, col Color) {
	ss := float64(c.SS)
	ax, ay := x0*ss, y0*ss
	bx, by := x1*ss, y1*ss
	dx, dy := bx-ax, by-ay
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps == 0 {
		c.Disc(ax/ss, ay/ss, width/2, col)
		return
	}
	r := width * ss / 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		c.DiscInternal(int(math.Round(ax+t*dx)), int(math.Round(ay+t*dy)),
			int(math.Ceil(r)), col)
	}
}

func (c *Canvas) DiscInternal(x, y, r int, col Color) {
	if r <= 0 {
		c.SetInternal(x, y, col)
		return
	}
	r2 := r * r
	for yy := -r; yy <= r; yy++ {
		for xx := -r; xx <= r; xx++ {
			if xx*xx+yy*yy <= r2 {
				c.SetInternal(x+xx, y+yy, col)
			}
		}
	}
}

func (c *Canvas) Disc(x, y, radius float64, col Color) {
	r := int(math.Ceil(radius * float64(c.SS)))
	c.DiscInternal(int(math.Round(x*float64(c.SS))),
		int(math.Round(y*float64(c.SS))), r, col)
}

func (c *Canvas) Circle(x, y, radius, width float64, col Color) {
	ss := float64(c.SS)
	steps := int(math.Ceil(2 * math.Pi * radius * ss))
	if steps < 12 {
		steps = 12
	}
	var px, py float64
	for i := 0; i <= steps; i++ {
		a := 2 * math.Pi * float64(i) / float64(steps)
		x1 := x + radius*math.Cos(a)
		y1 := y + radius*math.Sin(a)
		if i > 0 {
			c.Line(px, py, x1, y1, width, col)
		}
		px, py = x1, y1
	}
}

func (c *Canvas) FillTriangle2D(p [3]Vec2, col Color) {
	ss := float64(c.SS)
	q := [3]Vec2{}
	for i := range p {
		q[i] = Vec2{p[i].X * ss, p[i].Y * ss}
	}
	minX := int(math.Floor(math.Min(q[0].X, math.Min(q[1].X, q[2].X))))
	maxX := int(math.Ceil(math.Max(q[0].X, math.Max(q[1].X, q[2].X))))
	minY := int(math.Floor(math.Min(q[0].Y, math.Min(q[1].Y, q[2].Y))))
	maxY := int(math.Ceil(math.Max(q[0].Y, math.Max(q[1].Y, q[2].Y))))
	if c.clip {
		minX = maxInt(minX, c.clipX0)
		minY = maxInt(minY, c.clipY0)
		maxX = minInt(maxX, c.clipX1-1)
		maxY = minInt(maxY, c.clipY1-1)
	}
	area := cross2(q[0], q[1], q[2])
	if math.Abs(area) < 1e-9 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			pt := Vec2{float64(x) + 0.5, float64(y) + 0.5}
			w0 := cross2(q[1], q[2], pt) / area
			w1 := cross2(q[2], q[0], pt) / area
			w2 := cross2(q[0], q[1], pt) / area
			if w0 >= -1e-6 && w1 >= -1e-6 && w2 >= -1e-6 {
				c.SetInternal(x, y, col)
			}
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c *Canvas) Text(x, y float64, s string, scale int, col Color) {
	if scale < 1 {
		scale = 1
	}
	ss := float64(c.SS)
	advance := float64(6 * scale)
	cx := x
	for _, r := range s {
		g, ok := glyphFor(r)
		if !ok {
			cx += advance
			continue
		}
		for row := 0; row < 7; row++ {
			bits := g[row]
			for colIdx := 0; colIdx < 5; colIdx++ {
				if bits&(1<<uint(4-colIdx)) == 0 {
					continue
				}
				px := int(math.Floor((cx + float64(colIdx*scale)) * ss))
				py := int(math.Floor((y + float64(row*scale)) * ss))
				for yy := 0; yy < scale*c.SS; yy++ {
					for xx := 0; xx < scale*c.SS; xx++ {
						c.SetInternal(px+xx, py+yy, col)
					}
				}
			}
		}
		cx += advance
	}
}

func (c *Canvas) TextWidth(s string, scale int) float64 {
	return float64(len([]rune(s)) * 6 * scale)
}

func (c *Canvas) Downsample() *Canvas {
	if c.SS == 1 {
		out := NewCanvas(c.W, c.H, 1)
		copy(out.Pix, c.Pix)
		return out
	}
	out := NewCanvas(c.W, c.H, 1)
	ss := c.SS
	area := ss * ss
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			var r, g, b int
			for yy := 0; yy < ss; yy++ {
				for xx := 0; xx < ss; xx++ {
					off := ((y*ss+yy)*c.iw() + x*ss + xx) * 3
					r += int(c.Pix[off])
					g += int(c.Pix[off+1])
					b += int(c.Pix[off+2])
				}
			}
			off := (y*c.W + x) * 3
			out.Pix[off] = uint8(r / area)
			out.Pix[off+1] = uint8(g / area)
			out.Pix[off+2] = uint8(b / area)
		}
	}
	return out
}

func (c *Canvas) SavePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "P6\n%d %d\n255\n", c.W, c.H); err != nil {
		return err
	}
	_, err = f.Write(c.Pix)
	return err
}

func (c *Canvas) CountRedPixels() int {
	n := 0
	for i := 0; i < len(c.Pix); i += 3 {
		r, g, b := c.Pix[i], c.Pix[i+1], c.Pix[i+2]
		if r > 190 && int(g)*3 < int(r)*2 && int(b)*3 < int(r)*2 {
			n++
		}
	}
	return n
}
