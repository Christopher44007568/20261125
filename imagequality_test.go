package eyebw

import (
	"image"
	"image/color"
	"testing"
)

func sharpTestImage(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			v := byte(28)
			if (x+y)%2 == 0 {
				v = 228
			}
			if x%16 == 0 || y%16 == 0 {
				v = 0
			}
			img.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func nearestHalfScale(img *image.RGBA) *image.RGBA {
	b := img.Bounds()
	size := b.Dx()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		sy := b.Min.Y + ((y-b.Min.Y)/2)*2
		for x := b.Min.X; x < b.Max.X; x++ {
			sx := b.Min.X + ((x-b.Min.X)/2)*2
			out.Set(x, y, img.At(sx, sy))
		}
	}
	_ = size
	return out
}

func TestCenterImageProbeDetectsSilentBlur(t *testing.T) {
	baseline := sharpTestImage(128)
	identical, err := CompareCenterQuality(EyeLeft, "identical", baseline, baseline, Gaze{}, 28)
	if err != nil {
		t.Fatal(err)
	}
	if identical.PSNRDB < 99 || identical.SSIM < 0.999 {
		t.Fatalf("identical image probe too low: %+v", identical)
	}

	blurred := nearestHalfScale(baseline)
	probe, err := CompareCenterQuality(EyeLeft, "half_rate_nearest_upscale", baseline, blurred, Gaze{}, 28)
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	quality := CheckQuality(cfg, PipelineSingleReuse, []CenterObservation{
		{Eye: EyeLeft, TileX: 0, TileY: 0, Divisor: 1, MSAASamples: 4, IndependentlyRendered: true},
		{Eye: EyeRight, TileX: 0, TileY: 0, Divisor: 1, MSAASamples: 4, IndependentlyRendered: true},
	}, []QualityProbe{probe})
	if quality.Passed {
		t.Fatalf("blurred center should fail probe thresholds: probe=%+v", probe)
	}
	if len(quality.ProbeViolations) == 0 {
		t.Fatal("expected PSNR/SSIM probe violations")
	}
}
