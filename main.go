package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

type Config struct {
	OutDir      string
	Frames      int
	Width       int
	Height      int
	ContactCols int
}

func DefaultConfig() Config {
	return Config{
		OutDir:      filepath.Join("artifacts", "hair-lod"),
		Frames:      72,
		Width:       320,
		Height:      180,
		ContactCols: 6,
	}
}

func main() {
	cfg := DefaultConfig()
	flag.StringVar(&cfg.OutDir, "out", cfg.OutDir, "directory for frames, CSV, SVG and report")
	flag.IntVar(&cfg.Frames, "frames", cfg.Frames, "number of continuous frames")
	flag.IntVar(&cfg.Width, "width", cfg.Width, "thumbnail width in pixels")
	flag.IntVar(&cfg.Height, "height", cfg.Height, "thumbnail height in pixels")
	flag.IntVar(&cfg.ContactCols, "contact-cols", cfg.ContactCols, "columns in contact sheet")
	flag.Parse()

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "hair-lod:", err)
		os.Exit(1)
	}
}

func run(cfg Config) error {
	if cfg.Frames < 2 {
		return fmt.Errorf("frames must be at least 2")
	}
	if cfg.Width < 16 || cfg.Height < 9 {
		return fmt.Errorf("thumbnail is too small")
	}
	if cfg.ContactCols < 1 {
		cfg.ContactCols = 6
	}

	model := NewHairModel()
	renderW := cfg.Width * 2
	renderH := cfg.Height * 2
	cameraPath := DefaultCameraPath(renderW, renderH)

	frameDir := filepath.Join(cfg.OutDir, "frames")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		return err
	}

	thumbs := make([]*image.RGBA, 0, cfg.Frames)
	metrics := make([]FrameMetric, 0, cfg.Frames)

	for frame := 0; frame < cfg.Frames; frame++ {
		t := 0.0
		if cfg.Frames > 1 {
			t = float64(frame) / float64(cfg.Frames-1)
		}
		cam := cameraPath.At(t)
		distance := vlen(vsub(cameraPath.Target, cam.Pos))
		lod := BuildLODFrame(model, distance)

		fb := NewFrameBuffer(renderW, renderH)
		fb.Clear()
		tris := BuildScene(cam, model, lod)
		RenderTriangles(fb, cam, renderW, renderH, tris)

		thumb := Downsample2x(fb)
		framePath := filepath.Join(frameDir, fmt.Sprintf("frame_%04d.png", frame))
		if err := SavePNG(framePath, thumb); err != nil {
			return fmt.Errorf("write frame %d: %w", frame, err)
		}
		thumbs = append(thumbs, thumb)

		meanLuma, hairLuma, coverage := AnalyzeFrameBuffer(fb)
		totalTris := len(tris)
		hairTriangles := lod.HairTris + HairCapTriangles
		metrics = append(metrics, FrameMetric{
			Frame:          frame,
			TimeSeconds:    float64(frame) / 24.0,
			PathT:          t,
			CameraX:        cam.Pos.X,
			CameraY:        cam.Pos.Y,
			CameraZ:        cam.Pos.Z,
			Distance:       distance,
			LODState:       lod.State,
			LODFrom:        lod.FromLevel,
			LODTo:          lod.ToLevel,
			Blend:          lod.Blend,
			HairTriangles:  hairTriangles,
			HeadTriangles:  totalTris - hairTriangles,
			TotalTriangles: totalTris,
			LOD0Triangles:  model.Tris0 + HairCapTriangles,
			LOD1Triangles:  model.Tris1 + HairCapTriangles,
			LOD2Triangles:  model.Tris2 + HairCapTriangles,
			MeanLuma:       meanLuma,
			HairLuma:       hairLuma,
			HairCoverage:   coverage,
		})
	}

	FillTemporalDeltas(metrics)

	contactPath := filepath.Join(cfg.OutDir, "contact_sheet.png")
	if err := WriteContactSheet(contactPath, thumbs, cfg.ContactCols); err != nil {
		return fmt.Errorf("write contact sheet: %w", err)
	}

	metricsPath := filepath.Join(cfg.OutDir, "metrics.csv")
	if err := WriteMetricsCSV(metricsPath, metrics); err != nil {
		return fmt.Errorf("write metrics csv: %w", err)
	}
	trianglePath := filepath.Join(cfg.OutDir, "triangle_counts.csv")
	if err := WriteTriangleCountsCSV(trianglePath, metrics); err != nil {
		return fmt.Errorf("write triangle csv: %w", err)
	}
	brightnessPath := filepath.Join(cfg.OutDir, "brightness_curve.svg")
	if err := WriteBrightnessChart(brightnessPath, metrics); err != nil {
		return fmt.Errorf("write brightness chart: %w", err)
	}
	triangleChartPath := filepath.Join(cfg.OutDir, "triangle_counts.svg")
	if err := WriteTriangleChart(triangleChartPath, metrics); err != nil {
		return fmt.Errorf("write triangle chart: %w", err)
	}
	metricsPNGPath := filepath.Join(cfg.OutDir, "metrics_chart.png")
	if err := WriteMetricsPNG(metricsPNGPath, metrics); err != nil {
		return fmt.Errorf("write metrics png: %w", err)
	}

	report := BuildStabilityReport(cfg, metrics, model)
	reportPath := filepath.Join(cfg.OutDir, "stability_report.json")
	if err := WriteJSON(reportPath, report); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	if err := writeSummary(filepath.Join(cfg.OutDir, "summary.txt"), report); err != nil {
		return err
	}

	fmt.Printf("wrote %d frames to %s\n", cfg.Frames, frameDir)
	fmt.Printf("hair triangles incl. cap LOD0/1/2: %d/%d/%d\n",
		model.Tris0+HairCapTriangles, model.Tris1+HairCapTriangles, model.Tris2+HairCapTriangles)
	fmt.Printf("max luma step: %.3f; max coverage step: %.6f\n", report.MaxLumaFrameStep, report.MaxCoverageFrameStep)
	fmt.Printf("outputs: %s\n", cfg.OutDir)
	return nil
}

func writeSummary(path string, report StabilityReport) error {
	content := fmt.Sprintf(`Hair LOD fixed-camera demo

frames: %d
LOD hair triangles: %d / %d / %d
transition: %s
independent strand fade: %t
random strand disappearance: %t
max whole-frame luma step: %.3f / 255
mean whole-frame luma step: %.3f / 255
max hair-coverage step: %.6f
max luma step in LOD transition: %.3f / 255
max coverage step in LOD transition: %.6f

frame thumbnails: %s
contact sheet: %s
metrics: %s
brightness curve: %s
triangle chart: %s
PNG dashboard: %s
`, report.Frames, report.LOD0Triangles, report.LOD1Triangles, report.LOD2Triangles,
		report.TransitionMethod, report.IndependentStrandFade, report.RandomStrandDisappear,
		report.MaxLumaFrameStep, report.MeanLumaFrameStep, report.MaxCoverageFrameStep,
		report.MaxLumaInLODTransition, report.MaxCoverageInLODTransition,
		report.FramesDirectory, report.ContactSheet, report.MetricsCSV,
		report.BrightnessCurveSVG, report.TriangleCurveSVG, report.MetricsChartPNG)
	return os.WriteFile(path, []byte(content), 0o644)
}
