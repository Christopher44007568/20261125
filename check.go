package main

import (
	"image"
	"math"
)

type FrameMetrics struct {
	DisplayID          string     `json:"display_id"`
	GrayMeanRGB        [3]float64 `json:"gray_mean_rgb_code"`
	GrayLuma           float64    `json:"gray_luma_code"`
	GrayTargetCode     float64    `json:"gray_target_code"`
	GrayMaxError       float64    `json:"gray_max_error"`
	GrayNeutralDelta   float64    `json:"gray_neutral_delta"`
	NearWhiteFraction  float64    `json:"near_white_fraction"`
	WholeMeanLuma      float64    `json:"whole_mean_luma"`
	ControlMeanLuma    float64    `json:"control_mean_luma"`
	HighlightStdLuma   float64    `json:"highlight_std_luma"`
	HighlightOccupancy int        `json:"highlight_occupied_bins"`
	Pass               bool       `json:"pass"`
	Reasons            []string   `json:"reasons,omitempty"`
}

func sceneAuditROIs(id string) (control, highlight Rect, controlMax float64) {
	switch id {
	case "window":
		return Rect{X0: 20, Y0: 96, X1: 190, Y1: 224},
			Rect{X0: 248, Y0: 46, X1: 578, Y1: 220}, 86
	case "lights":
		return Rect{X0: 12, Y0: 150, X1: 62, Y1: 242},
			Rect{X0: 42, Y0: 56, X1: 598, Y1: 116}, 104
	default:
		return Rect{X0: 232, Y0: 72, X1: 430, Y1: 190},
			Rect{X0: 76, Y0: 148, X1: 178, Y1: 198}, 78
	}
}

func AnalyzeFrame(scene *Scene, frame *DisplayFrame) FrameMetrics {
	grayROI := Rect{
		X0: scene.Gray.X0 + 10, Y0: scene.Gray.Y0 + 10,
		X1: scene.Gray.X1 - 10, Y1: scene.Gray.Y1 - 10,
	}
	gr, gg, gb, grayLuma := roiMeans(frame.Image, grayROI)
	grayMaxError := math.Max(math.Abs(gr-grayProofCode),
		math.Max(math.Abs(gg-grayProofCode), math.Abs(gb-grayProofCode)))
	neutralDelta := math.Max(gr, math.Max(gg, gb)) - math.Min(gr, math.Min(gg, gb))

	var whitePixels int
	var lumaSum float64
	for y := 0; y < frame.Image.Rect.Dy(); y++ {
		for x := 0; x < frame.Image.Rect.Dx(); x++ {
			l := pixelLuma(frame.Image, x, y)
			lumaSum += l
			if l >= 245 {
				whitePixels++
			}
		}
	}
	pixelCount := float64(sceneW * sceneH)

	controlROI, highlightROI, controlMax := sceneAuditROIs(scene.ID)
	_, _, _, controlLuma := roiMeans(frame.Image, controlROI)
	highlightStd, highlightBins := roiLumaSpread(frame.Image, highlightROI)

	metrics := FrameMetrics{
		DisplayID:          frame.Spec.ID,
		GrayMeanRGB:        [3]float64{round1(gr), round1(gg), round1(gb)},
		GrayLuma:           round1(grayLuma),
		GrayTargetCode:     grayProofCode,
		GrayMaxError:       round2(grayMaxError),
		GrayNeutralDelta:   round2(neutralDelta),
		NearWhiteFraction:  round4(float64(whitePixels) / pixelCount),
		WholeMeanLuma:      round1(lumaSum / pixelCount),
		ControlMeanLuma:    round1(controlLuma),
		HighlightStdLuma:   round1(highlightStd),
		HighlightOccupancy: highlightBins,
		Pass:               true,
	}

	if grayMaxError > 2.0 {
		metrics.Pass = false
		metrics.Reasons = append(metrics.Reasons, "gray card missed declared code 118")
	}
	if neutralDelta > 2.0 {
		metrics.Pass = false
		metrics.Reasons = append(metrics.Reasons, "gray card is not neutral")
	}
	if controlLuma > controlMax {
		metrics.Pass = false
		metrics.Reasons = append(metrics.Reasons, "shadow/midground control ROI washed upward")
	}
	if metrics.NearWhiteFraction > 0.18 {
		metrics.Pass = false
		metrics.Reasons = append(metrics.Reasons, "too many near-white pixels")
	}
	if highlightStd < 16 || highlightBins < 18 {
		metrics.Pass = false
		metrics.Reasons = append(metrics.Reasons, "highlight ROI lacks recovered tonal separation")
	}
	return metrics
}

func pixelLuma(img *image.NRGBA, x, y int) float64 {
	i := img.PixOffset(x, y)
	r := float64(img.Pix[i+0])
	g := float64(img.Pix[i+1])
	b := float64(img.Pix[i+2])
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func roiMeans(img *image.NRGBA, r Rect) (float64, float64, float64, float64) {
	values := roiPixels(img, r)
	if len(values) == 0 {
		return 0, 0, 0, 0
	}
	var sr, sg, sb, sl float64
	for _, p := range values {
		sr += p[0]
		sg += p[1]
		sb += p[2]
		sl += 0.2126*p[0] + 0.7152*p[1] + 0.0722*p[2]
	}
	n := float64(len(values))
	return sr / n, sg / n, sb / n, sl / n
}

func roiLumaSpread(img *image.NRGBA, r Rect) (float64, int) {
	pixels := roiPixels(img, r)
	if len(pixels) == 0 {
		return 0, 0
	}
	var sum float64
	seen := map[uint8]bool{}
	lumas := make([]float64, len(pixels))
	for i, p := range pixels {
		l := 0.2126*p[0] + 0.7152*p[1] + 0.0722*p[2]
		lumas[i] = l
		sum += l
		seen[uint8(math.Round(l))] = true
	}
	mean := sum / float64(len(lumas))
	var variance float64
	for _, l := range lumas {
		variance += (l - mean) * (l - mean)
	}
	return math.Sqrt(variance / float64(len(lumas))), len(seen)
}

func roiPixels(img *image.NRGBA, r Rect) [][3]float64 {
	x0 := int(math.Max(0, math.Floor(r.X0)))
	y0 := int(math.Max(0, math.Floor(r.Y0)))
	x1 := int(math.Min(sceneW, math.Ceil(r.X1)))
	y1 := int(math.Min(sceneH, math.Ceil(r.Y1)))
	out := make([][3]float64, 0, (x1-x0)*(y1-y0))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			i := img.PixOffset(x, y)
			out = append(out, [3]float64{
				float64(img.Pix[i+0]), float64(img.Pix[i+1]), float64(img.Pix[i+2]),
			})
		}
	}
	return out
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
