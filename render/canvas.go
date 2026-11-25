// Package render draws dependency-free comparison PNGs for the gamut mapper.
package render

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"gamutmap/gamut"
)

// Canvas is a small sRGB raster with source-over blending.
type Canvas struct {
	Img *image.NRGBA
}

// NewCanvas allocates an opaque black canvas.
func NewCanvas(width, height int) *Canvas {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+3] = 255
	}
	return &Canvas{Img: img}
}

// Save writes the canvas as a PNG, creating parent directories as needed.
func (c *Canvas) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, c.Img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Set writes an opaque pixel.
func (c *Canvas) Set(x, y int, col color.NRGBA) {
	if x < 0 || y < 0 || x >= c.Img.Bounds().Dx() || y >= c.Img.Bounds().Dy() {
		return
	}
	col.A = 255
	i := c.Img.PixOffset(x, y)
	c.Img.Pix[i] = col.R
	c.Img.Pix[i+1] = col.G
	c.Img.Pix[i+2] = col.B
	c.Img.Pix[i+3] = 255
}

// Blend composites a color with source-over alpha.
func (c *Canvas) Blend(x, y int, col color.NRGBA) {
	if col.A == 0 || x < 0 || y < 0 || x >= c.Img.Bounds().Dx() || y >= c.Img.Bounds().Dy() {
		return
	}
	i := c.Img.PixOffset(x, y)
	a := uint32(col.A)
	ia := 255 - a
	c.Img.Pix[i] = uint8((uint32(col.R)*a + uint32(c.Img.Pix[i])*ia + 127) / 255)
	c.Img.Pix[i+1] = uint8((uint32(col.G)*a + uint32(c.Img.Pix[i+1])*ia + 127) / 255)
	c.Img.Pix[i+2] = uint8((uint32(col.B)*a + uint32(c.Img.Pix[i+2])*ia + 127) / 255)
}

// FillRect fills an axis-aligned rectangle.
func (c *Canvas) FillRect(x, y, w, h int, col color.NRGBA) {
	for py := max(0, y); py < min(c.Img.Bounds().Dy(), y+h); py++ {
		for px := max(0, x); px < min(c.Img.Bounds().Dx(), x+w); px++ {
			c.Set(px, py, col)
		}
	}
}

// StrokeRect draws a one-pixel outline.
func (c *Canvas) StrokeRect(x, y, w, h int, col color.NRGBA) {
	c.FillRect(x, y, w, 1, col)
	c.FillRect(x, y+h-1, w, 1, col)
	c.FillRect(x, y, 1, h, col)
	c.FillRect(x+w-1, y, 1, h, col)
}

func ColorFromRGB(c gamut.RGB) color.NRGBA {
	return color.NRGBA{
		R: channel8(c.R),
		G: channel8(c.G),
		B: channel8(c.B),
		A: 255,
	}
}

func channel8(v float64) uint8 {
	v = math.Max(0, math.Min(1, v))
	return uint8(math.Round(v * 255))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
