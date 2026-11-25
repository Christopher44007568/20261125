package colorlab

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
)

// Image is a packed 8-bit RGB raster.
type Image struct {
	Width  int
	Height int
	Pix    []byte
}

// NewImage allocates an opaque black RGB image.
func NewImage(width, height int) *Image {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return &Image{Width: width, Height: height, Pix: make([]byte, width*height*3)}
}

// Set stores one color, clamping and rounding to 8 bits.
func (img *Image) Set(x, y int, c RGB) {
	if x < 0 || y < 0 || x >= img.Width || y >= img.Height {
		return
	}
	i := (y*img.Width + x) * 3
	img.Pix[i+0] = quantize8(c.R)
	img.Pix[i+1] = quantize8(c.G)
	img.Pix[i+2] = quantize8(c.B)
}

// FillRect fills an axis-aligned rectangle.
func (img *Image) FillRect(x, y, width, height int, c RGB) {
	for yy := y; yy < y+height; yy++ {
		for xx := x; xx < x+width; xx++ {
			img.Set(xx, yy, c)
		}
	}
}

func quantize8(v float64) byte {
	if math.IsNaN(v) || v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return byte(math.Round(v * 255.0))
}

// WritePPM writes a binary P6 PPM stream.
func (img *Image) WritePPM(w io.Writer) error {
	bw := bufio.NewWriter(w)
	if _, err := fmt.Fprintf(bw, "P6\n%d %d\n255\n", img.Width, img.Height); err != nil {
		return err
	}
	if _, err := bw.Write(img.Pix); err != nil {
		return err
	}
	return bw.Flush()
}

// WritePPMFile creates or replaces a PPM file.
func (img *Image) WritePPMFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := img.WritePPM(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
