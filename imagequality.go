package eyebw

import (
	"fmt"
	"image"
	"math"
)

// CompareCenterQuality measures PSNR and global SSIM inside a circular center
// region. It is an optional bridge for real captured eye buffers; the byte
// model alone cannot prove that a full-rate tile was not silently blurred.
func CompareCenterQuality(eye EyeID, source string, baseline, candidate image.Image, gaze Gaze, radiusPixels float64) (QualityProbe, error) {
	if baseline == nil || candidate == nil {
		return QualityProbe{}, fmt.Errorf("baseline and candidate images must be non-nil")
	}
	bb := baseline.Bounds()
	cb := candidate.Bounds()
	if bb.Dx() != cb.Dx() || bb.Dy() != cb.Dy() {
		return QualityProbe{}, fmt.Errorf("image dimensions differ: baseline=%dx%d candidate=%dx%d",
			bb.Dx(), bb.Dy(), cb.Dx(), cb.Dy())
	}
	if radiusPixels <= 0 {
		radiusPixels = float64(bb.Dy()) * 0.12
	}
	centerX := float64(bb.Min.X) + float64(bb.Dx())*(clampUnit(gaze.X)+1)/2
	centerY := float64(bb.Min.Y) + float64(bb.Dy())*(clampUnit(gaze.Y)+1)/2

	var n float64
	var sumX, sumY, sumXX, sumYY, sumXY, sumSquaredError float64
	for y := bb.Min.Y; y < bb.Max.Y; y++ {
		for x := bb.Min.X; x < bb.Max.X; x++ {
			dx := float64(x) + 0.5 - centerX
			dy := float64(y) + 0.5 - centerY
			if math.Hypot(dx, dy) > radiusPixels {
				continue
			}
			vx := luminanceAt(baseline, x, y)
			vy := luminanceAt(candidate, x-bb.Min.X+cb.Min.X, y-bb.Min.Y+cb.Min.Y)
			n++
			sumX += vx
			sumY += vy
			sumXX += vx * vx
			sumYY += vy * vy
			sumXY += vx * vy
			d := vx - vy
			sumSquaredError += d * d
		}
	}
	if n == 0 {
		return QualityProbe{}, fmt.Errorf("center quality region contains no pixels")
	}

	meanX := sumX / n
	meanY := sumY / n
	varX := math.Max(0, sumXX/n-meanX*meanX)
	varY := math.Max(0, sumYY/n-meanY*meanY)
	covXY := sumXY/n - meanX*meanY
	mse := sumSquaredError / n
	psnr := 100.0
	if mse > 1e-12 {
		psnr = math.Min(100, 10*math.Log10((255*255)/mse))
	}

	c1 := math.Pow(0.01*255, 2)
	c2 := math.Pow(0.03*255, 2)
	ssim := ((2*meanX*meanY + c1) * (2*covXY + c2)) /
		((meanX*meanX + meanY*meanY + c1) * (varX + varY + c2))
	if math.IsNaN(ssim) || math.IsInf(ssim, 0) {
		ssim = 0
	}
	return QualityProbe{Eye: eye, PSNRDB: psnr, SSIM: math.Max(0, math.Min(1, ssim)), Source: source}, nil
}

func luminanceAt(img image.Image, x, y int) float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	// RGBA returns alpha-premultiplied 16-bit values. For opaque test buffers
	// dividing by 257 maps the color channels back to the usual 0..255 range.
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 257.0
}
