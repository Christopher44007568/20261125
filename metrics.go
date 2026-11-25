package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type FrameMetric struct {
	Frame          int
	TimeSeconds    float64
	PathT          float64
	CameraX        float64
	CameraY        float64
	CameraZ        float64
	Distance       float64
	LODState       string
	LODFrom        int
	LODTo          int
	Blend          float64
	HairTriangles  int
	HeadTriangles  int
	TotalTriangles int
	LOD0Triangles  int
	LOD1Triangles  int
	LOD2Triangles  int
	MeanLuma       float64
	HairLuma       float64
	HairCoverage   float64
	LumaDelta      float64
	CoverageDelta  float64
}

func AnalyzeFrameBuffer(fb *FrameBuffer) (meanLuma, hairLuma, hairCoverage float64) {
	var totalLuma float64
	var hairLumaSum float64
	var hairPixels int
	pixelCount := fb.W * fb.H

	for i := 0; i < pixelCount; i++ {
		o := i * 4
		luma := 0.2126*float64(fb.Image.Pix[o+0]) +
			0.7152*float64(fb.Image.Pix[o+1]) +
			0.0722*float64(fb.Image.Pix[o+2])
		totalLuma += luma
		if fb.Mat[i] == MatHair {
			hairLumaSum += luma
			hairPixels++
		}
	}

	meanLuma = totalLuma / float64(pixelCount)
	hairCoverage = float64(hairPixels) / float64(pixelCount)
	if hairPixels > 0 {
		hairLuma = hairLumaSum / float64(hairPixels)
	}
	return meanLuma, hairLuma, hairCoverage
}

func FillTemporalDeltas(metrics []FrameMetric) {
	for i := range metrics {
		if i == 0 {
			continue
		}
		metrics[i].LumaDelta = math.Abs(metrics[i].MeanLuma - metrics[i-1].MeanLuma)
		metrics[i].CoverageDelta = math.Abs(metrics[i].HairCoverage - metrics[i-1].HairCoverage)
	}
}

func WriteMetricsCSV(path string, metrics []FrameMetric) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	header := []string{
		"frame", "time_seconds", "path_t", "camera_x", "camera_y", "camera_z",
		"distance", "lod_state", "lod_from", "lod_to", "blend",
		"hair_triangles", "head_scene_triangles", "total_triangles",
		"lod0_triangles", "lod1_triangles", "lod2_triangles",
		"mean_luma_0_255", "hair_luma_0_255", "hair_coverage",
		"luma_abs_delta", "coverage_abs_delta",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, m := range metrics {
		row := []string{
			fmt.Sprint(m.Frame),
			fmt.Sprintf("%.4f", m.TimeSeconds),
			fmt.Sprintf("%.5f", m.PathT),
			fmt.Sprintf("%.5f", m.CameraX),
			fmt.Sprintf("%.5f", m.CameraY),
			fmt.Sprintf("%.5f", m.CameraZ),
			fmt.Sprintf("%.5f", m.Distance),
			m.LODState,
			fmt.Sprint(m.LODFrom),
			fmt.Sprint(m.LODTo),
			fmt.Sprintf("%.5f", m.Blend),
			fmt.Sprint(m.HairTriangles),
			fmt.Sprint(m.HeadTriangles),
			fmt.Sprint(m.TotalTriangles),
			fmt.Sprint(m.LOD0Triangles),
			fmt.Sprint(m.LOD1Triangles),
			fmt.Sprint(m.LOD2Triangles),
			fmt.Sprintf("%.4f", m.MeanLuma),
			fmt.Sprintf("%.4f", m.HairLuma),
			fmt.Sprintf("%.6f", m.HairCoverage),
			fmt.Sprintf("%.4f", m.LumaDelta),
			fmt.Sprintf("%.6f", m.CoverageDelta),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func WriteTriangleCountsCSV(path string, metrics []FrameMetric) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write([]string{"frame", "hair_triangles", "total_scene_triangles", "lod_state"}); err != nil {
		return err
	}
	for _, m := range metrics {
		if err := w.Write([]string{
			fmt.Sprint(m.Frame),
			fmt.Sprint(m.HairTriangles),
			fmt.Sprint(m.TotalTriangles),
			m.LODState,
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

type StabilityReport struct {
	Frames                     int     `json:"frames"`
	FramesDirectory            string  `json:"frames_directory"`
	ContactSheet               string  `json:"contact_sheet"`
	MetricsCSV                 string  `json:"metrics_csv"`
	TriangleCountsCSV          string  `json:"triangle_counts_csv"`
	BrightnessCurveSVG         string  `json:"brightness_curve_svg"`
	TriangleCurveSVG           string  `json:"triangle_curve_svg"`
	MetricsChartPNG            string  `json:"metrics_chart_png"`
	LOD0Triangles              int     `json:"lod0_triangles"`
	LOD1Triangles              int     `json:"lod1_triangles"`
	LOD2Triangles              int     `json:"lod2_triangles"`
	MaxLumaFrameStep           float64 `json:"max_luma_frame_step"`
	MeanLumaFrameStep          float64 `json:"mean_luma_frame_step"`
	MaxCoverageFrameStep       float64 `json:"max_coverage_frame_step"`
	MaxLumaInLODTransition     float64 `json:"max_luma_step_in_lod_transition"`
	MaxCoverageInLODTransition float64 `json:"max_coverage_step_in_lod_transition"`
	TransitionMethod           string  `json:"transition_method"`
	IndependentStrandFade      bool    `json:"independent_strand_fade"`
	RandomStrandDisappear      bool    `json:"random_strand_disappear"`
}

func BuildStabilityReport(cfg Config, metrics []FrameMetric, model *HairModel) StabilityReport {
	var maxLuma, sumLuma, maxCoverage float64
	var maxTransitionLuma, maxTransitionCoverage float64
	steps := 0
	for i, m := range metrics {
		if i == 0 {
			continue
		}
		maxLuma = math.Max(maxLuma, m.LumaDelta)
		sumLuma += m.LumaDelta
		maxCoverage = math.Max(maxCoverage, m.CoverageDelta)
		inTransition := m.Blend > 0 && m.LODFrom != m.LODTo
		previousInTransition := i > 0 && metrics[i-1].Blend > 0 &&
			metrics[i-1].LODFrom != metrics[i-1].LODTo
		if i > 0 && (inTransition || previousInTransition) {
			maxTransitionLuma = math.Max(maxTransitionLuma, m.LumaDelta)
			maxTransitionCoverage = math.Max(maxTransitionCoverage, m.CoverageDelta)
		}
		steps++
	}
	mean := 0.0
	if steps > 0 {
		mean = sumLuma / float64(steps)
	}
	return StabilityReport{
		Frames:                     len(metrics),
		FramesDirectory:            filepath.Join(cfg.OutDir, "frames"),
		ContactSheet:               filepath.Join(cfg.OutDir, "contact_sheet.png"),
		MetricsCSV:                 filepath.Join(cfg.OutDir, "metrics.csv"),
		TriangleCountsCSV:          filepath.Join(cfg.OutDir, "triangle_counts.csv"),
		BrightnessCurveSVG:         filepath.Join(cfg.OutDir, "brightness_curve.svg"),
		TriangleCurveSVG:           filepath.Join(cfg.OutDir, "triangle_counts.svg"),
		MetricsChartPNG:            filepath.Join(cfg.OutDir, "metrics_chart.png"),
		LOD0Triangles:              model.Tris0 + HairCapTriangles,
		LOD1Triangles:              model.Tris1 + HairCapTriangles,
		LOD2Triangles:              model.Tris2 + HairCapTriangles,
		MaxLumaFrameStep:           maxLuma,
		MeanLumaFrameStep:          mean,
		MaxCoverageFrameStep:       maxCoverage,
		MaxLumaInLODTransition:     maxTransitionLuma,
		MaxCoverageInLODTransition: maxTransitionCoverage,
		TransitionMethod:           "deterministic hierarchical lock/clump geomorph with smoothstep distance windows",
		IndependentStrandFade:      false,
		RandomStrandDisappear:      false,
	}
}

func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
