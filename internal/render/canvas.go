// Package render draws the scalp, hair roots, density heat map and flow
// glyphs using only the Go standard library (image + image/png).
package render

import (
	"image"
	"image/color"
	"math"
)

// Canvas is an RGBA image paired with a depth buffer. Smaller depth wins.
type Canvas struct {
	W, H  int
	img   *image.RGBA
	depth []float64
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{
		W:     w,
		H:     h,
		img:   image.NewRGBA(image.Rect(0, 0, w, h)),
		depth: make([]float64, w*h),
	}
}

func (c *Canvas) resetDepth() {
	for i := range c.depth {
		c.depth[i] = math.Inf(1)
	}
}

// Fill paints the whole canvas with one opaque color.
func (c *Canvas) Fill(col color.Color) {
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			c.img.Set(x, y, col)
		}
	}
}

// Set writes a pixel when its depth is nearer than what is stored.
func (c *Canvas) Set(x, y int, z float64, col color.Color) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	i := y*c.W + x
	if z >= c.depth[i] {
		return
	}
	c.depth[i] = z
	c.img.Set(x, y, col)
}

// Set2D writes a pixel without depth testing (used for 2D maps and glyphs).
func (c *Canvas) Set2D(x, y int, col color.Color) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.img.Set(x, y, col)
}

// Image exposes the rendered image for PNG encoding.
func (c *Canvas) Image() image.Image { return c.img }

// DepthAt returns the stored depth at a pixel and whether it was painted.
func (c *Canvas) DepthAt(x, y int) (float64, bool) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return 0, false
	}
	z := c.depth[y*c.W+x]
	return z, !math.IsInf(z, 1)
}

// Splat stamps a filled disc of radius px centered at (x, y).
func (c *Canvas) Splat(x, y int, radius float64, z float64, col color.Color) {
	r := int(math.Ceil(radius))
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= int(radius*radius) {
				c.Set(x+dx, y+dy, z, col)
			}
		}
	}
}

// Splat2D is the depth-free counterpart of Splat.
func (c *Canvas) Splat2D(x, y int, radius float64, col color.Color) {
	r := int(math.Ceil(radius))
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= int(radius*radius) {
				c.Set2D(x+dx, y+dy, col)
			}
		}
	}
}

// Line draws a 2D line in (x0,y0)->(x1,y1) using Bresenham's algorithm.
func (c *Canvas) Line(x0, y0, x1, y1 int, col color.Color) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		c.Set2D(x0, y0, col)
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

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
