package render

import (
	"bufio"
	"fmt"
	"math"
	"os"
)

type RGB struct {
	R, G, B uint8
}

type Canvas struct {
	W, H int
	Pix  []uint8
}

func NewCanvas(w, h int, bg RGB) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]uint8, w*h*3)}
	for i := 0; i < w*h; i++ {
		c.Pix[i*3] = bg.R
		c.Pix[i*3+1] = bg.G
		c.Pix[i*3+2] = bg.B
	}
	return c
}

func (c *Canvas) Set(x, y int, col RGB) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := (y*c.W + x) * 3
	c.Pix[i] = col.R
	c.Pix[i+1] = col.G
	c.Pix[i+2] = col.B
}

func (c *Canvas) Blend(x, y int, col RGB, alpha float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	if alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	i := (y*c.W + x) * 3
	c.Pix[i] = uint8(float64(c.Pix[i])*(1-alpha) + float64(col.R)*alpha)
	c.Pix[i+1] = uint8(float64(c.Pix[i+1])*(1-alpha) + float64(col.G)*alpha)
	c.Pix[i+2] = uint8(float64(c.Pix[i+2])*(1-alpha) + float64(col.B)*alpha)
}

func (c *Canvas) FillRect(x0, y0, x1, y1 int, col RGB) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.Set(x, y, col)
		}
	}
}

func edge(ax, ay, bx, by, px, py float64) float64 {
	return (px-ax)*(by-ay) - (py-ay)*(bx-ax)
}

func (c *Canvas) FillTriangle(a, b, d Vec2, ca, cb, cd RGB) {
	area := edge(a.X, a.Y, b.X, b.Y, d.X, d.Y)
	if math.Abs(area) < 1e-12 {
		return
	}
	minX := int(math.Floor(math.Min(a.X, math.Min(b.X, d.X))))
	maxX := int(math.Ceil(math.Max(a.X, math.Max(b.X, d.X))))
	minY := int(math.Floor(math.Min(a.Y, math.Min(b.Y, d.Y))))
	maxY := int(math.Ceil(math.Max(a.Y, math.Max(b.Y, d.Y))))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= c.W {
		maxX = c.W - 1
	}
	if maxY >= c.H {
		maxY = c.H - 1
	}
	sign := 1.0
	if area < 0 {
		sign = -1
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			w0 := edge(b.X, b.Y, d.X, d.Y, px, py) * sign
			w1 := edge(d.X, d.Y, a.X, a.Y, px, py) * sign
			w2 := edge(a.X, a.Y, b.X, b.Y, px, py) * sign
			if w0 < -0.001 || w1 < -0.001 || w2 < -0.001 {
				continue
			}
			den := w0 + w1 + w2
			if math.Abs(den) < 1e-15 {
				continue
			}
			col := lerpRGB(ca, cb, cd, w0/den, w1/den, w2/den)
			c.Set(x, y, col)
		}
	}
}

func lerpRGB(a, b, c RGB, wa, wb, wc float64) RGB {
	return RGB{
		R: uint8(math.Min(255, float64(a.R)*wa+float64(b.R)*wb+float64(c.R)*wc)),
		G: uint8(math.Min(255, float64(a.G)*wa+float64(b.G)*wb+float64(c.G)*wc)),
		B: uint8(math.Min(255, float64(a.B)*wa+float64(b.B)*wb+float64(c.B)*wc)),
	}
}

func (c *Canvas) Line(x0, y0, x1, y1 int, col RGB, width int, alpha float64) {
	dx := intAbs(x1 - x0)
	dy := -intAbs(y1 - y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		c.Brush(x0, y0, width, col, alpha)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (c *Canvas) Brush(x, y, width int, col RGB, alpha float64) {
	if width < 1 {
		width = 1
	}
	r := width / 2
	for yy := y - r; yy <= y+r; yy++ {
		for xx := x - r; xx <= x+r; xx++ {
			c.Blend(xx, yy, col, alpha)
		}
	}
}

func intAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (c *Canvas) WritePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(w, "P6\n%d %d\n255\n", c.W, c.H); err != nil {
		return err
	}
	if _, err := w.Write(c.Pix); err != nil {
		return err
	}
	return w.Flush()
}

func (c *Canvas) Blit(src *Canvas, dx, dy int) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			sx, sy := x, y
			if dx+x < 0 || dy+y < 0 || dx+x >= c.W || dy+y >= c.H {
				continue
			}
			si := (sy*src.W + sx) * 3
			di := ((dy+y)*c.W + dx + x) * 3
			c.Pix[di], c.Pix[di+1], c.Pix[di+2] = src.Pix[si], src.Pix[si+1], src.Pix[si+2]
		}
	}
}

type Vec2 struct {
	X, Y float64
}
