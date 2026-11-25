// Package render draws the cutaway solid, the 2D section montage and the
// endpoint error heat map using only the Go standard image packages.
package render

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

// Canvas is a simple RGBA pixel buffer with primitive 2D drawing helpers.
type Canvas struct {
	img *image.RGBA
	W   int
	H   int
}

// NewCanvas allocates a width x height canvas filled with bg.
func NewCanvas(w, h int, bg color.Color) *Canvas {
	c := &Canvas{img: image.NewRGBA(image.Rect(0, 0, w, h)), W: w, H: h}
	c.Fill(bg)
	return c
}

// Fill paints the whole canvas.
func (c *Canvas) Fill(col color.Color) {
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			c.img.Set(x, y, col)
		}
	}
}

// Set writes one pixel with bounds clipping.
func (c *Canvas) Set(x, y int, col color.Color) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.img.Set(x, y, col)
}

// Line draws a Bresenham line with the given stroke thickness.
func (c *Canvas) Line(x0, y0, x1, y1 int, thick int, col color.Color) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	r := thick / 2
	for {
		c.disc(x0, y0, r, col)
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

func (c *Canvas) disc(x, y, r int, col color.Color) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r+r {
				c.Set(x+dx, y+dy, col)
			}
		}
	}
}

// Rect fills an axis-aligned rectangle.
func (c *Canvas) Rect(x, y, w, h int, col color.Color) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c.Set(x+i, y+j, col)
		}
	}
}

// Frame strokes a rectangle outline.
func (c *Canvas) Frame(x, y, w, h, thick int, col color.Color) {
	c.Line(x, y, x+w, y, thick, col)
	c.Line(x+w, y, x+w, y+h, thick, col)
	c.Line(x+w, y+h, x, y+h, thick, col)
	c.Line(x, y+h, x, y, thick, col)
}

// Polygon fills a simple polygon via an integer scanline rasterizer.
func (c *Canvas) Polygon(pts []image.Point, col color.Color) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts[1:] {
		minY = minInt(minY, p.Y)
		maxY = maxInt(maxY, p.Y)
	}
	for y := minY; y <= maxY; y++ {
		var xs []int
		for i := 0; i < len(pts); i++ {
			a, b := pts[i], pts[(i+1)%len(pts)]
			if (a.Y > y) != (b.Y > y) {
				x := a.X + (y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
				xs = append(xs, x)
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			lo, hi := xs[i], xs[i+1]
			if lo > hi {
				lo, hi = hi, lo
			}
			for x := lo; x <= hi; x++ {
				c.Set(x, y, col)
			}
		}
	}
}

// Blend composites src over the canvas at a point with the given alpha.
func (c *Canvas) Blend(x, y int, src color.Color, alpha float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	sr, sg, sb, _ := src.RGBA()
	dr, dg, db, da := c.img.At(x, y).RGBA()
	a := clamp01(alpha)
	mix := func(s, d uint32) uint8 {
		v := float64(s)/65535*a + float64(d)/65535*(1-a)
		return uint8(math.Round(v * 255))
	}
	c.img.SetRGBA(x, y, color.RGBA{
		R: mix(sr, dr), G: mix(sg, dg), B: mix(sb, db),
		A: uint8(da >> 8),
	})
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// SavePNG encodes the canvas to a PNG stream.
func (c *Canvas) SavePNG(w io.Writer) error {
	return png.Encode(w, c.img)
}

// Image exposes the underlying buffer for layered compositing.
func (c *Canvas) Image() *image.RGBA { return c.img }

// BgColor returns the shared background color used by the multi-panel outputs.
func BgColor() color.Color { return palette.Bg }

// Blit copies src onto c with its top-left at (ox, oy).
func (c *Canvas) Blit(src *Canvas, ox, oy int) {
	sb := src.Image().Bounds()
	for y := sb.Min.Y; y < sb.Max.Y; y++ {
		for x := sb.Min.X; x < sb.Max.X; x++ {
			c.Set(ox+x, oy+y, src.Image().At(x, y))
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
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
