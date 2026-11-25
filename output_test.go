package main

import (
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesThreeRenderPasses(t *testing.T) {
	dir := t.TempDir()
	if err := run(dir, 180, 224, 96, 7, ""); err != nil {
		t.Fatal(err)
	}

	names := []string{"beauty.png", "shadow_only.png", "hair_transmission.png"}
	for _, name := range names {
		img := decodeTestPNG(t, filepath.Join(dir, name))
		if bounds := img.Bounds(); bounds.Dx() != 180 || bounds.Dy() != 224 {
			t.Fatalf("%s has bounds %v", name, bounds)
		}
	}

	beauty := decodeTestPNG(t, filepath.Join(dir, "beauty.png"))
	skinMean, skinDark := regionStats(beauty, 0.30, 0.70, 0.70, 0.92)
	hairMean, hairStdDev := regionMeanStd(beauty, 0.06, 0.20, 0.94, 0.66)
	t.Logf("skin mean=%.1f darkFraction=%.3f; hair mean=%.1f stddev=%.1f", skinMean, skinDark, hairMean, hairStdDev)

	if skinMean < 125 {
		t.Fatalf("forehead is too dark: mean %.1f", skinMean)
	}
	if skinDark > 0.04 {
		t.Fatalf("forehead contains too many near-black pixels: %.3f", skinDark)
	}
	if hairStdDev < 24 {
		t.Fatalf("hair region lacks strand/layer contrast: stddev %.1f", hairStdDev)
	}

	shadow := decodeTestPNG(t, filepath.Join(dir, "shadow_only.png"))
	shadowMean, shadowStd := regionMeanStd(shadow, 0.0, 0.0, 1.0, 1.0)
	t.Logf("shadow mean=%.1f stddev=%.1f", shadowMean, shadowStd)
	if shadowMean < 185 || shadowStd < 10 {
		t.Fatalf("shadow pass should be a bright field with readable gray layers: mean=%.1f std=%.1f", shadowMean, shadowStd)
	}

	transmission := decodeTestPNG(t, filepath.Join(dir, "hair_transmission.png"))
	transMean, transStd := regionMeanStd(transmission, 0.0, 0.12, 1.0, 0.82)
	t.Logf("transmission mean=%.1f stddev=%.1f", transMean, transStd)
	if transMean < 205 || transStd < 8 {
		t.Fatalf("transmission pass should show warm multilayer hair: mean=%.1f std=%.1f", transMean, transStd)
	}
}

func decodeTestPNG(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func regionStats(img image.Image, x0, y0, x1, y1 float64) (mean float64, darkFraction float64) {
	mean, std := regionMeanStd(img, x0, y0, x1, y1)
	_ = std
	b := img.Bounds()
	px0 := b.Min.X + int(float64(b.Dx())*x0)
	py0 := b.Min.Y + int(float64(b.Dy())*y0)
	px1 := b.Min.X + int(float64(b.Dx())*x1)
	py1 := b.Min.Y + int(float64(b.Dy())*y1)
	var dark, count int
	for y := py0; y < py1; y++ {
		for x := px0; x < px1; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			luma := 0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(bl>>8)
			if luma < 34 {
				dark++
			}
			count++
		}
	}
	return mean, float64(dark) / float64(maxInt(count, 1))
}

func regionMeanStd(img image.Image, x0, y0, x1, y1 float64) (float64, float64) {
	b := img.Bounds()
	px0 := b.Min.X + int(float64(b.Dx())*x0)
	py0 := b.Min.Y + int(float64(b.Dy())*y0)
	px1 := b.Min.X + int(float64(b.Dx())*x1)
	py1 := b.Min.Y + int(float64(b.Dy())*y1)

	var sum, sumSquares, count float64
	for y := py0; y < py1; y++ {
		for x := px0; x < px1; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			luma := 0.2126*float64(r>>8) + 0.7152*float64(g>>8) + 0.0722*float64(bl>>8)
			sum += luma
			sumSquares += luma * luma
			count++
		}
	}
	mean := sum / count
	variance := math.Max(0, sumSquares/count-mean*mean)
	return mean, math.Sqrt(variance)
}
