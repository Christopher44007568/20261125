package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

type Pixel struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func newRGBA(width, height int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, width, height))
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// HashImage hashes the canonical RGBA pixel bytes and dimensions. The hash is
// independent of PNG encoder details.
func HashImage(img *image.RGBA) string {
	if img == nil {
		return ""
	}
	header := fmt.Sprintf("rgba:%d:%d:", img.Bounds().Dx(), img.Bounds().Dy())
	hash := sha256.New()
	_, _ = hash.Write([]byte(header))
	_, _ = hash.Write(img.Pix)
	return hex.EncodeToString(hash.Sum(nil))
}

// FirstPollutedPixel reports the first channel mismatch in raster order:
// top-to-bottom, then left-to-right.
func FirstPollutedPixel(reference, actual *image.RGBA) *Pixel {
	if reference == nil || actual == nil {
		return nil
	}
	if reference.Bounds() != actual.Bounds() {
		return &Pixel{}
	}
	bounds := reference.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ri := reference.PixOffset(x, y)
			ai := actual.PixOffset(x, y)
			if !bytes.Equal(reference.Pix[ri:ri+4], actual.Pix[ai:ai+4]) {
				return &Pixel{X: x, Y: y}
			}
		}
	}
	return nil
}

func CountPollutedPixels(reference, actual *image.RGBA) int {
	if reference == nil || actual == nil || reference.Bounds() != actual.Bounds() {
		return 0
	}
	count := 0
	bounds := reference.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ri := reference.PixOffset(x, y)
			ai := actual.PixOffset(x, y)
			if !bytes.Equal(reference.Pix[ri:ri+4], actual.Pix[ai:ai+4]) {
				count++
			}
		}
	}
	return count
}

func PixelRGBA(img *image.RGBA, point *Pixel) *[4]uint8 {
	if img == nil || point == nil || !image.Pt(point.X, point.Y).In(img.Bounds()) {
		return nil
	}
	offset := img.PixOffset(point.X, point.Y)
	return &[4]uint8{
		img.Pix[offset],
		img.Pix[offset+1],
		img.Pix[offset+2],
		img.Pix[offset+3],
	}
}

func writePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		return err
	}
	return file.Close()
}

// WritePollutionDiff highlights differing pixels and marks the first mismatch.
func WritePollutionDiff(path string, reference, actual *image.RGBA) error {
	if reference == nil || actual == nil || reference.Bounds() != actual.Bounds() {
		return fmt.Errorf("cannot diff images with different bounds")
	}
	diff := newRGBA(reference.Bounds().Dx(), reference.Bounds().Dy())
	first := FirstPollutedPixel(reference, actual)
	bounds := reference.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			offset := diff.PixOffset(x, y)
			if pixelsDiffer(reference, actual, x, y) {
				diff.Pix[offset+0] = 255
				diff.Pix[offset+1] = 24
				diff.Pix[offset+2] = 96
				diff.Pix[offset+3] = 255
			} else {
				diff.Pix[offset+0] = 8
				diff.Pix[offset+1] = 10
				diff.Pix[offset+2] = 14
				diff.Pix[offset+3] = 255
			}
		}
	}
	if first != nil {
		drawMarker(diff, first.X, first.Y)
	}
	return writePNG(path, diff)
}

func pixelsDiffer(a, b *image.RGBA, x, y int) bool {
	ai := a.PixOffset(x, y)
	bi := b.PixOffset(x, y)
	return !bytes.Equal(a.Pix[ai:ai+4], b.Pix[bi:bi+4])
}

func drawMarker(img *image.RGBA, x, y int) {
	marker := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for offset := -4; offset <= 4; offset++ {
		setPixel(img, x+offset, y, marker)
		setPixel(img, x, y+offset, marker)
	}
}

func setPixel(img *image.RGBA, x, y int, value color.RGBA) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	offset := img.PixOffset(x, y)
	img.Pix[offset+0] = value.R
	img.Pix[offset+1] = value.G
	img.Pix[offset+2] = value.B
	img.Pix[offset+3] = value.A
}
