package main

import (
	"image"
	"math"
	"sort"
)

type SharpnessResult struct {
	Eye                 EyeID    `json:"eye"`
	ElementID           string   `json:"elementId"`
	Status              string   `json:"status"`
	Pass                bool     `json:"pass"`
	EdgeSamples         int      `json:"edgeSamples"`
	MedianTransitionPx  float64  `json:"medianTransition10To90Px"`
	P90TransitionPx     float64  `json:"p90TransitionPx"`
	MedianLuminanceStep float64  `json:"medianLuminanceStep"`
	Failures            []string `json:"failures,omitempty"`
}

func crossingNear(line []float64, threshold, center float64) (float64, bool) {
	best := 0.0
	bestDist := math.Inf(1)
	found := false
	for i := 0; i+1 < len(line); i++ {
		a, b := line[i], line[i+1]
		if (a <= threshold && b >= threshold) || (a >= threshold && b <= threshold) {
			x := float64(i)
			if math.Abs(b-a) > 1e-12 {
				x = float64(i) + (threshold-a)/(b-a)
			}
			if d := math.Abs(x - center); d < bestDist {
				bestDist = d
				best = x
				found = true
			}
		}
	}
	return best, found
}

func transitionAt(line []float64, center int) (float64, float64, bool) {
	start := center - 2
	end := center + 3
	window := line[start:end]
	low, high := window[0], window[0]
	for _, v := range window[1:] {
		low = math.Min(low, v)
		high = math.Max(high, v)
	}
	step := high - low
	if step < 0.30 || math.Abs(line[center+1]-line[center-1]) < 0.18 {
		return 0, 0, false
	}

	t10 := low + 0.10*step
	t90 := low + 0.90*step
	x10, ok10 := crossingNear(window, t10, 2)
	x90, ok90 := crossingNear(window, t90, 2)
	if !ok10 || !ok90 {
		return 0, 0, false
	}
	width := math.Abs(x90 - x10)
	if width < 0.15 || width > 3.25 {
		return 0, 0, false
	}
	return width, step, true
}

func collectTransitions(img *image.RGBA, rect image.Rectangle) ([]float64, []float64) {
	widths := make([]float64, 0, 256)
	steps := make([]float64, 0, 256)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		line := make([]float64, rect.Dx())
		for x := rect.Min.X; x < rect.Max.X; x++ {
			line[x-rect.Min.X] = luminanceAt(img, x, y)
		}
		for x := 2; x < len(line)-2; {
			width, step, ok := transitionAt(line, x)
			if !ok {
				x++
				continue
			}
			widths = append(widths, width)
			steps = append(steps, step)
			x += 2
		}
	}

	for x := rect.Min.X; x < rect.Max.X; x++ {
		line := make([]float64, rect.Dy())
		for y := rect.Min.Y; y < rect.Max.Y; y++ {
			line[y-rect.Min.Y] = luminanceAt(img, x, y)
		}
		for y := 2; y < len(line)-2; {
			width, step, ok := transitionAt(line, y)
			if !ok {
				y++
				continue
			}
			widths = append(widths, width)
			steps = append(steps, step)
			y += 2
		}
	}
	return widths, steps
}

func MeasureSharpness(img *image.RGBA, elementID string, bounds FloatRect, eye Eye) SharpnessResult {
	result := SharpnessResult{Eye: eye.ID, ElementID: elementID, Status: "pass", Pass: true}
	rect := floatRectToImageRect(bounds, img.Bounds(), 1)
	if rect.Dx() < 8 || rect.Dy() < 8 {
		result.Status = "fail"
		result.Pass = false
		result.Failures = append(result.Failures, "sharpness region too small")
		return result
	}

	widths, steps := collectTransitions(img, rect)
	result.EdgeSamples = len(widths)
	if len(widths) < 20 {
		result.Status = "fail"
		result.Pass = false
		result.Failures = append(result.Failures, "fewer than 20 measurable text edges")
		return result
	}

	sort.Float64s(widths)
	sort.Float64s(steps)
	result.MedianTransitionPx = percentileSorted(widths, 0.50)
	result.P90TransitionPx = percentileSorted(widths, 0.90)
	result.MedianLuminanceStep = percentileSorted(steps, 0.50)

	if result.MedianTransitionPx > 2.0 {
		result.Failures = append(result.Failures, "median edge transition wider than 2 device pixels")
	}
	if result.P90TransitionPx > 3.0 {
		result.Failures = append(result.Failures, "p90 edge transition wider than 3 device pixels")
	}
	if result.MedianLuminanceStep < 0.25 {
		result.Failures = append(result.Failures, "edge luminance step too weak")
	}
	if len(result.Failures) > 0 {
		result.Status = "fail"
		result.Pass = false
	}
	return result
}
