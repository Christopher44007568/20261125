package main

import (
	"image"
	"image/color"
	"math"
)

type Canvas struct {
	*image.RGBA
}

func NewCanvas(width, height int, bg color.RGBA) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	c := &Canvas{RGBA: img}
	c.FillRect(FloatRect{MinX: 0, MinY: 0, MaxX: float64(width), MaxY: float64(height)}, bg)
	return c
}

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// FillRect blends an axis-aligned rectangle using exact per-pixel area coverage.
// The projected UI is only mildly perspective-skewed, so texel quads reduce to
// axis-aligned fractional rectangles at each glyph.
func (c *Canvas) FillRect(r FloatRect, src color.RGBA) {
	bounds := c.Bounds()
	x0 := int(math.Floor(r.MinX))
	y0 := int(math.Floor(r.MinY))
	x1 := int(math.Ceil(r.MaxX))
	y1 := int(math.Ceil(r.MaxY))
	x0 = maxInt(x0, bounds.Min.X)
	y0 = maxInt(y0, bounds.Min.Y)
	x1 = minInt(x1, bounds.Max.X)
	y1 = minInt(y1, bounds.Max.Y)
	if x0 >= x1 || y0 >= y1 {
		return
	}

	srcA := float64(src.A) / 255.0
	for y := y0; y < y1; y++ {
		covY := math.Min(r.MaxY, float64(y+1)) - math.Max(r.MinY, float64(y))
		covY = clampUnit(covY)
		for x := x0; x < x1; x++ {
			covX := math.Min(r.MaxX, float64(x+1)) - math.Max(r.MinX, float64(x))
			covX = clampUnit(covX)
			a := srcA * covX * covY
			if a <= 0 {
				continue
			}
			i := c.PixOffset(x, y)
			dstA := float64(c.Pix[i+3]) / 255.0
			outA := a + dstA*(1-a)
			if outA <= 0 {
				continue
			}
			blend := func(s, d uint8) uint8 {
				sf := float64(s) / 255.0
				df := float64(d) / 255.0
				v := (sf*a + df*dstA*(1-a)) / outA
				return uint8(math.Round(clampUnit(v) * 255))
			}
			c.Pix[i+0] = blend(src.R, c.Pix[i+0])
			c.Pix[i+1] = blend(src.G, c.Pix[i+1])
			c.Pix[i+2] = blend(src.B, c.Pix[i+2])
			c.Pix[i+3] = uint8(math.Round(outA * 255))
		}
	}
}

func (c *Canvas) StrokeRect(r FloatRect, thickness float64, col color.RGBA) {
	if r.Width() <= 0 || r.Height() <= 0 {
		return
	}
	c.FillRect(FloatRect{MinX: r.MinX, MinY: r.MinY, MaxX: r.MaxX, MaxY: r.MinY + thickness}, col)
	c.FillRect(FloatRect{MinX: r.MinX, MinY: r.MaxY - thickness, MaxX: r.MaxX, MaxY: r.MaxY}, col)
	c.FillRect(FloatRect{MinX: r.MinX, MinY: r.MinY, MaxX: r.MinX + thickness, MaxY: r.MaxY}, col)
	c.FillRect(FloatRect{MinX: r.MaxX - thickness, MinY: r.MinY, MaxX: r.MaxX, MaxY: r.MaxY}, col)
}

// DrawScreenText draws debug/HUD text in eye-texture pixels.
func (c *Canvas) DrawScreenText(text string, x, y, texel float64, col color.RGBA) {
	for i, r := range text {
		glyph := GlyphFor(r)
		ox := x + float64(i)*6*texel
		for row, line := range glyph {
			for column, ink := range line {
				if ink != '#' {
					continue
				}
				c.FillRect(FloatRect{
					MinX: ox + float64(column)*texel,
					MinY: y + float64(row)*texel,
					MaxX: ox + float64(column+1)*texel,
					MaxY: y + float64(row+1)*texel,
				}, col)
			}
		}
	}
}

type PlacedGlyph struct {
	Rune   rune
	Origin ScreenPoint
	TexelW float64
	TexelH float64
}

func (c *Canvas) DrawPlacedGlyph(g PlacedGlyph, col color.RGBA) {
	glyph := GlyphFor(g.Rune)
	for row, line := range glyph {
		for column, ink := range line {
			if ink != '#' {
				continue
			}
			x0 := g.Origin.X + float64(column)*g.TexelW
			y0 := g.Origin.Y + float64(row)*g.TexelH
			c.FillRect(FloatRect{
				MinX: x0,
				MinY: y0,
				MaxX: x0 + g.TexelW,
				MaxY: y0 + g.TexelH,
			}, col)
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
