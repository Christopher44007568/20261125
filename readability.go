package main

import (
	"image"
	"math"
	"sort"
)

type ReadabilityResult struct {
	Eye               EyeID    `json:"eye"`
	Status            string   `json:"status"`
	Pass              bool     `json:"pass"`
	CapHeightPx       float64  `json:"capHeightPx"`
	StrokeWidthPx     float64  `json:"strokeWidthPx"`
	CapHeightArcMin   float64  `json:"capHeightArcMin"`
	PixelsPerDegree   float64  `json:"pixelsPerDegree"`
	ContrastRatio     float64  `json:"contrastRatio"`
	MotionResidualPx  float64  `json:"motionResidualPx"`
	PersistenceBlurPx float64  `json:"persistenceBlurPx"`
	TrackingErrorPx   float64  `json:"trackingErrorPx"`
	Warnings          []string `json:"warnings,omitempty"`
	Failures          []string `json:"failures,omitempty"`
}

func floatRectToImageRect(r FloatRect, b image.Rectangle, pad int) image.Rectangle {
	x0 := int(math.Floor(r.MinX)) - pad
	y0 := int(math.Floor(r.MinY)) - pad
	x1 := int(math.Ceil(r.MaxX)) + pad
	y1 := int(math.Ceil(r.MaxY)) + pad
	return image.Rect(
		maxInt(x0, b.Min.X),
		maxInt(y0, b.Min.Y),
		minInt(x1, b.Max.X),
		minInt(y1, b.Max.Y),
	)
}

func relativeLuminance(r, g, b uint8) float64 {
	linear := func(v uint8) float64 {
		s := float64(v) / 255.0
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}

func luminanceAt(img *image.RGBA, x, y int) float64 {
	i := img.PixOffset(x, y)
	return relativeLuminance(img.Pix[i+0], img.Pix[i+1], img.Pix[i+2])
}

func percentileSorted(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	idx := int(math.Round(p * float64(len(values)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}

func contrastInRect(img *image.RGBA, rect image.Rectangle) float64 {
	values := make([]float64, 0, rect.Dx()*rect.Dy())
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			values = append(values, luminanceAt(img, x, y))
		}
	}
	if len(values) == 0 {
		return 1
	}
	sort.Float64s(values)
	dark := percentileSorted(values, 0.08)
	light := percentileSorted(values, 0.97)
	return (light + 0.05) / (dark + 0.05)
}

func AnalyzeReadability(img *image.RGBA, geo ElementGeometry, eye Eye, scenario Scenario) ReadabilityResult {
	result := ReadabilityResult{Eye: eye.ID, Status: "pass", Pass: true}
	glyphs := geo.Glyphs
	var capSum, strokeSum float64
	count := 0
	for _, pair := range glyphs {
		var g EyeGlyph
		if eye.ID == LeftEye {
			g = pair.Left
		} else {
			g = pair.Right
		}
		if !g.Valid || g.Bounds.Height() <= 0 {
			continue
		}
		capSum += g.Bounds.Height()
		strokeSum += g.View.TexelW
		count++
	}
	if count == 0 {
		result.Status = "fail"
		result.Pass = false
		result.Failures = append(result.Failures, "no projected glyphs")
		return result
	}

	result.CapHeightPx = capSum / float64(count)
	result.StrokeWidthPx = strokeSum / float64(count)
	result.PixelsPerDegree = eye.focalY() * math.Pi / 180.0
	result.CapHeightArcMin = arcMinutes(result.CapHeightPx / eye.focalY())

	bounds := geo.LeftBounds
	if eye.ID == RightEye {
		bounds = geo.RightBounds
	}
	pixelRect := floatRectToImageRect(bounds, img.Bounds(), 2)
	result.ContrastRatio = contrastInRect(img, pixelRect)

	result.PersistenceBlurPx = math.Abs(scenario.YawVelocityRadPerSec) *
		scenario.PersistenceSeconds * eye.focalX()
	trackingRadians := scenario.TrackingErrorArcMin * math.Pi / (180.0 * 60.0)
	result.TrackingErrorPx = trackingRadians * eye.focalX()
	result.MotionResidualPx = math.Hypot(result.PersistenceBlurPx, result.TrackingErrorPx)

	if result.CapHeightArcMin < 24 {
		result.Failures = append(result.Failures, "cap height below 24 arcmin")
	}
	if result.StrokeWidthPx < 1.20 {
		result.Failures = append(result.Failures, "stroke narrower than 1.2 device pixels")
	}
	if result.ContrastRatio < 4.5 {
		result.Failures = append(result.Failures, "contrast below WCAG 4.5:1")
	}
	if result.MotionResidualPx > result.StrokeWidthPx*1.8 {
		result.Warnings = append(result.Warnings, "motion residual approaches stroke width")
	}
	if len(result.Failures) > 0 {
		result.Status = "fail"
		result.Pass = false
	} else if len(result.Warnings) > 0 {
		result.Status = "warning"
	}
	return result
}
