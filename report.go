package main

import (
	"image"
	"math"
	"sort"
	"time"
)

type LatencyReport struct {
	Samples             int     `json:"samples"`
	Measurement         string  `json:"measurement"`
	BudgetMS            float64 `json:"budget_ms"`
	MinMS               float64 `json:"min_ms"`
	MeanMS              float64 `json:"mean_ms"`
	P95MS               float64 `json:"p95_ms"`
	MaxMS               float64 `json:"max_ms"`
	WithinBudget        bool    `json:"within_budget"`
	WithinBudgetSamples int     `json:"within_budget_samples"`
}

type DepthReport struct {
	Source                     string  `json:"source"`
	InvalidPolicy              string  `json:"invalid_policy"`
	InvalidFraction            float64 `json:"invalid_fraction"`
	InvalidInSphereFraction    float64 `json:"invalid_in_sphere_fraction"`
	VisibleSphereFraction      float64 `json:"visible_sphere_fraction"`
	OccludedSphereFraction     float64 `json:"occluded_sphere_fraction"`
	VirtualFrontPixels         int     `json:"virtual_front_pixels"`
	OccludedPixels             int     `json:"occluded_pixels"`
	InvalidDepthInSpherePixels int     `json:"invalid_depth_in_sphere_pixels"`
	SphereFootprintPixels      int     `json:"sphere_footprint_pixels"`
	MeanConfidence             float64 `json:"mean_confidence"`
}

type ExposureReport struct {
	Method              string         `json:"method"`
	ReferenceIndex      int            `json:"reference_index"`
	Frames              []ExposureStat `json:"frames"`
	FusionInputs        int            `json:"fusion_inputs"`
	MaxAbsDeviationEV   float64        `json:"max_abs_deviation_ev"`
	MeanClippedFraction float64        `json:"mean_clipped_fraction"`
}

type ColorReport struct {
	Method           string  `json:"method"`
	GainR            float64 `json:"gain_r"`
	GainG            float64 `json:"gain_g"`
	GainB            float64 `json:"gain_b"`
	BlendSpace       string  `json:"blend_space"`
	AppliedBeforeMix bool    `json:"applied_before_mix"`
}

type SystemReport struct {
	MeanAlpha  float64 `json:"mean_alpha"`
	Coverage   float64 `json:"coverage"`
	BlendSpace string  `json:"blend_space"`
	BlendOrder string  `json:"blend_order"`
}

type Report struct {
	GeneratedAt    string            `json:"generated_at"`
	Resolution     [2]int            `json:"resolution"`
	FramesRendered int               `json:"frames_rendered"`
	Latency        LatencyReport     `json:"passthrough_latency"`
	Depth          DepthReport       `json:"depth"`
	Exposure       ExposureReport    `json:"exposure"`
	Color          ColorReport       `json:"color_alignment"`
	SystemLayer    SystemReport      `json:"system_layer"`
	Outputs        map[string]string `json:"outputs"`
}

func latencyReport(samples []time.Duration, budget time.Duration) LatencyReport {
	report := LatencyReport{
		Samples:     len(samples),
		Measurement: "frame_available_to_composite",
		BudgetMS:    float64(budget) / float64(time.Millisecond),
	}
	if len(samples) == 0 {
		return report
	}
	values := make([]float64, len(samples))
	total := 0.0
	for i, d := range samples {
		ms := float64(d) / float64(time.Millisecond)
		values[i] = ms
		total += ms
		if d <= budget {
			report.WithinBudgetSamples++
		}
	}
	sort.Float64s(values)
	report.MinMS = values[0]
	report.MaxMS = values[len(values)-1]
	report.MeanMS = total / float64(len(values))
	idx := int(math.Ceil(0.95*float64(len(values)))) - 1
	if idx < 0 {
		idx = 0
	}
	report.P95MS = values[idx]
	report.WithinBudget = report.WithinBudgetSamples == len(values)
	return report
}

func depthReport(bundle *CaptureBundle, stats VisibilityStats, conf []float64) DepthReport {
	invalid := 0
	confSum := 0.0
	confCount := 0
	for p, valid := range bundle.DepthValid {
		if !valid {
			invalid++
			continue
		}
		confSum += conf[p]
		confCount++
	}
	report := DepthReport{
		Source:                     "aligned-reference-depth",
		InvalidPolicy:              "do-not-assert-virtual-in-front",
		InvalidFraction:            float64(invalid) / float64(len(bundle.DepthValid)),
		VirtualFrontPixels:         stats.VisiblePixels,
		OccludedPixels:             stats.OccludedPixels,
		InvalidDepthInSpherePixels: stats.InvalidDepthPixels,
		SphereFootprintPixels:      stats.FootprintPixels,
	}
	if stats.FootprintPixels > 0 {
		report.VisibleSphereFraction = float64(stats.VisiblePixels) / float64(stats.FootprintPixels)
		report.OccludedSphereFraction = float64(report.OccludedPixels) / float64(stats.FootprintPixels)
		report.InvalidInSphereFraction = float64(report.InvalidDepthInSpherePixels) / float64(stats.FootprintPixels)
	}
	if confCount > 0 {
		report.MeanConfidence = confSum / float64(confCount)
	}
	return report
}

func exposureReport(fused FusionResult) ExposureReport {
	report := ExposureReport{
		Method:            "linear-irradiance-weighted-fusion",
		ReferenceIndex:    fused.ReferenceIndex,
		Frames:            fused.ExposureStats,
		FusionInputs:      len(fused.ExposureStats),
		MaxAbsDeviationEV: fused.MaxAbsDeviation,
	}
	if len(fused.ExposureStats) > 0 {
		sum := 0.0
		for _, stat := range fused.ExposureStats {
			sum += stat.ClippedFraction
		}
		report.MeanClippedFraction = sum / float64(len(fused.ExposureStats))
	}
	return report
}

func systemReport(layer *image.NRGBA) SystemReport {
	if layer == nil {
		return SystemReport{BlendSpace: "display-srgb", BlendOrder: "after-virtual"}
	}
	var alphaSum float64
	covered := 0
	for y := layer.Rect.Min.Y; y < layer.Rect.Max.Y; y++ {
		for x := layer.Rect.Min.X; x < layer.Rect.Max.X; x++ {
			a := layer.NRGBAAt(x, y).A
			alphaSum += float64(a) / 255
			if a > 0 {
				covered++
			}
		}
	}
	pixels := layer.Rect.Dx() * layer.Rect.Dy()
	if pixels == 0 {
		return SystemReport{BlendSpace: "display-srgb", BlendOrder: "after-virtual"}
	}
	return SystemReport{
		MeanAlpha:  alphaSum / float64(pixels),
		Coverage:   float64(covered) / float64(pixels),
		BlendSpace: "display-srgb",
		BlendOrder: "after-virtual",
	}
}
