package render

import (
	"image"
	"image/color"
	"math"
)

// Canvas wraps an RGBA image with minimal drawing primitives.
type Canvas struct {
	Img *image.RGBA
}

var (
	ColDark  = color.RGBA{R: 10, G: 11, B: 15, A: 255}
	ColWhite = color.RGBA{R: 235, G: 238, B: 245, A: 255}
)

func NewCanvas(w, h int) *Canvas {
	return &Canvas{Img: image.NewRGBA(image.Rect(0, 0, w, h))}
}

func (c *Canvas) Bounds() image.Rectangle { return c.Img.Bounds() }

func (c *Canvas) Fill(col color.Color) {
	b := c.Img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c.Img.Set(x, y, col)
		}
	}
}

func (c *Canvas) Set(x, y int, col color.Color) {
	if image.Pt(x, y).In(c.Img.Bounds()) {
		c.Img.Set(x, y, col)
	}
}

func (c *Canvas) Line(x0, y0, x1, y1 int, col color.Color) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		c.Set(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
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

func (c *Canvas) Rect(x0, y0, x1, y1 int, col color.Color) {
	c.Line(x0, y0, x1, y0, col)
	c.Line(x1, y0, x1, y1, col)
	c.Line(x1, y1, x0, y1, col)
	c.Line(x0, y1, x0, y0, col)
}

func (c *Canvas) FillRect(x0, y0, x1, y1 int, col color.Color) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			c.Set(x, y, col)
		}
	}
}

func (c *Canvas) Circle(cx, cy, r int, col color.Color, filled bool) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			d := math.Sqrt(float64(x*x + y*y))
			if filled && d <= float64(r) {
				c.Set(cx+x, cy+y, col)
			} else if !filled && d >= float64(r)-0.7 && d <= float64(r)+0.7 {
				c.Set(cx+x, cy+y, col)
			}
		}
	}
}

// Text draws a compact caption using the built-in 5x7 bitmap font.
func (c *Canvas) Text(x, y int, s string, col color.Color) {
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch >= 'a' && ch <= 'z' {
			ch -= 'a' - 'A'
		}
		glyph, ok := font5x7[ch]
		if !ok {
			glyph = font5x7['?']
		}
		for row := 0; row < 7; row++ {
			bits := glyph[row]
			for colIdx := 0; colIdx < 5; colIdx++ {
				if bits&(1<<(4-colIdx)) != 0 {
					c.Set(x+i*6+colIdx, y+row, col)
				}
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
