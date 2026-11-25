package hair

import (
	"fmt"
	"math"
)

type DollySample struct {
	Distance       float64 `json:"distance"`
	LevelA         int     `json:"levelA"`
	LevelB         int     `json:"levelB"`
	Blend          float64 `json:"blend"`
	EffectiveCount float64 `json:"effectiveCount"`
	LODPixels      int     `json:"lodPixels"`
	FullPixels     int     `json:"fullPixels"`
	CoverageRatio  float64 `json:"coverageRatio"`
	IoU            float64 `json:"iou"`
	SilhouetteW    int     `json:"silhouetteWidth"`
	SilhouetteH    int     `json:"silhouetteHeight"`
}

type DollyReport struct {
	Threshold        float64       `json:"threshold"`
	SampleCount      int           `json:"sampleCount"`
	MaxCoverageStep  float64       `json:"maxCoverageStep"`
	MinCoverageRatio float64       `json:"minCoverageRatio"`
	FarCoverageRatio float64       `json:"farCoverageRatio"`
	FarIoU           float64       `json:"farIoU"`
	FarSilhouetteW   int           `json:"farSilhouetteWidth"`
	FarSilhouetteH   int           `json:"farSilhouetteHeight"`
	Continuous       bool          `json:"continuous"`
	Samples          []DollySample `json:"samples"`
}

func AnalyzeDolly(asset *Asset, width, height, sampleCount int) DollyReport {
	const threshold = 0.02
	start := asset.Camera.FocusDistance
	end := math.Min(asset.Camera.FarDistance, 32)
	distances := SortedDistances(start, end, sampleCount)
	report := DollyReport{
		Threshold:        threshold,
		SampleCount:      len(distances),
		MinCoverageRatio: math.Inf(1),
	}
	previousRatio := 0.0
	for i, distance := range distances {
		camera := NewCamera(asset.Camera, width, height, distance)
		full := asset.RenderLevel(camera, 0)
		lod, selection := asset.RenderLOD(camera, distance)
		fullCoverage := full.Coverage(threshold)
		lodCoverage := lod.Coverage(threshold)
		ratio := 0.0
		if fullCoverage.Pixels > 0 {
			ratio = float64(lodCoverage.Pixels) / float64(fullCoverage.Pixels)
		}
		iou := maskIoU(full.Occupancy(threshold), lod.Occupancy(threshold))
		sample := DollySample{
			Distance:       distance,
			LevelA:         selection.LevelA,
			LevelB:         selection.LevelB,
			Blend:          selection.Blend,
			EffectiveCount: selection.EffectiveCount,
			LODPixels:      lodCoverage.Pixels,
			FullPixels:     fullCoverage.Pixels,
			CoverageRatio:  ratio,
			IoU:            iou,
			SilhouetteW:    lodCoverage.Width,
			SilhouetteH:    lodCoverage.Height,
		}
		report.Samples = append(report.Samples, sample)
		if i > 0 {
			report.MaxCoverageStep = math.Max(report.MaxCoverageStep, math.Abs(ratio-previousRatio))
		}
		previousRatio = ratio
		if ratio < report.MinCoverageRatio {
			report.MinCoverageRatio = ratio
		}
		report.FarCoverageRatio = ratio
		report.FarIoU = iou
		report.FarSilhouetteW = lodCoverage.Width
		report.FarSilhouetteH = lodCoverage.Height
	}
	report.Continuous = report.MaxCoverageStep <= 0.08 &&
		report.MinCoverageRatio >= 0.55 &&
		report.FarIoU >= 0.45 &&
		report.FarSilhouetteW >= 4 &&
		report.FarSilhouetteH >= 4
	return report
}

func (r DollyReport) Summary() string {
	return fmt.Sprintf(
		"coverage ratio %.3f..%.3f, max step %.3f, far IoU %.3f, continuous=%t",
		r.MinCoverageRatio, r.FarCoverageRatio, r.MaxCoverageStep, r.FarIoU, r.Continuous,
	)
}
