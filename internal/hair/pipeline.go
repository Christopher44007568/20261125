package hair

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"
)

type PipelineOptions struct {
	OutDir        string
	StrandCount   int
	GuideCount    int
	Seed          int64
	Points        int
	Width         int
	Height        int
	AnalysisSize  int
	DollySamples  int
	ErrorDistance float64
}

type PipelineResult struct {
	Asset *Asset
	Dolly DollyReport
	Error ErrorReport
	Files map[string]string
}

func GuideLevelCounts(guideCount int) []int {
	if guideCount < 2 {
		return nil
	}
	counts := []int{guideCount}
	for counts[len(counts)-1] > 24 {
		next := counts[len(counts)-1] / 2
		if next < 24 || next == counts[len(counts)-1] {
			break
		}
		counts = append(counts, next)
	}
	return counts
}

func RunPipeline(opts PipelineOptions) (*PipelineResult, error) {
	if opts.StrandCount < 100 {
		return nil, fmt.Errorf("strand count must be at least 100")
	}
	if opts.GuideCount < 2 || opts.GuideCount >= opts.StrandCount {
		return nil, fmt.Errorf("guide count must be in [2, strandCount)")
	}
	if opts.Width < 32 || opts.Height < 32 {
		return nil, fmt.Errorf("render size must be at least 32x32")
	}
	if opts.AnalysisSize < 96 {
		opts.AnalysisSize = 96
	}
	if opts.DollySamples < 2 {
		opts.DollySamples = 2
	}

	asset := Generate(opts.StrandCount, opts.Seed, opts.Points)
	BuildGuideLevels(asset, GuideLevelCounts(opts.GuideCount), opts.Seed)
	if len(asset.Levels) < 2 {
		return nil, fmt.Errorf("could not build guide level")
	}

	if opts.ErrorDistance <= asset.Camera.FocusDistance {
		opts.ErrorDistance = asset.Camera.FocusDistance * 3.6
	}
	if opts.ErrorDistance > asset.Camera.FarDistance {
		opts.ErrorDistance = asset.Camera.FarDistance
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return nil, err
	}

	result := &PipelineResult{
		Asset: asset,
		Files: make(map[string]string),
	}
	nearDistance := asset.Camera.FocusDistance * 1.35
	nearCamera := NewCamera(asset.Camera, opts.Width, opts.Height, nearDistance)
	fullNear := asset.RenderLevel(nearCamera, 0)
	guideNear := asset.RenderLevel(nearCamera, 1)
	result.Files["fullStrands"] = filepath.Join(opts.OutDir, "full_strands.png")
	result.Files["guides"] = filepath.Join(opts.OutDir, "guides.png")
	if err := WritePNG(result.Files["fullStrands"], ColorizeField(fullNear, PaletteHair)); err != nil {
		return nil, err
	}
	if err := WritePNG(result.Files["guides"], ColorizeField(guideNear, PaletteGuides)); err != nil {
		return nil, err
	}

	errorCamera := NewCamera(asset.Camera, opts.Width, opts.Height, opts.ErrorDistance)
	fullRef := asset.RenderLevel(errorCamera, 0)
	lodField, _ := asset.RenderLOD(errorCamera, opts.ErrorDistance)
	heatmap, errorReport := ErrorHeatmap(fullRef, lodField)
	result.Error = errorReport
	result.Files["fullReference"] = filepath.Join(opts.OutDir, "full_reference.png")
	result.Files["lod"] = filepath.Join(opts.OutDir, "lod.png")
	result.Files["errorHeatmap"] = filepath.Join(opts.OutDir, "error_heatmap.png")
	if err := WritePNG(result.Files["fullReference"], ColorizeField(fullRef, PaletteHair)); err != nil {
		return nil, err
	}
	if err := WritePNG(result.Files["lod"], ColorizeField(lodField, PaletteLOD)); err != nil {
		return nil, err
	}
	if err := WritePNG(result.Files["errorHeatmap"], heatmap); err != nil {
		return nil, err
	}

	result.Dolly = AnalyzeDolly(asset, opts.AnalysisSize, opts.AnalysisSize, opts.DollySamples)
	if err := writeCoverageStrip(asset, opts, result.Files); err != nil {
		return nil, err
	}
	result.Files["coverageCurve"] = filepath.Join(opts.OutDir, "coverage_curve.png")
	if err := writeCoverageChart(result.Dolly, result.Files["coverageCurve"]); err != nil {
		return nil, err
	}

	result.Files["strands"] = filepath.Join(opts.OutDir, "strands.json")
	result.Files["guidesData"] = filepath.Join(opts.OutDir, "guides.json")
	result.Files["lodData"] = filepath.Join(opts.OutDir, "lod.json")
	result.Files["coverageData"] = filepath.Join(opts.OutDir, "coverage_curve.json")
	if err := WriteJSON(result.Files["strands"], BuildStrandDocument(asset)); err != nil {
		return nil, err
	}
	if err := WriteJSON(result.Files["guidesData"], BuildGuideDocument(asset)); err != nil {
		return nil, err
	}
	if err := WriteJSON(result.Files["coverageData"], result.Dolly); err != nil {
		return nil, err
	}
	outputs := relativeOutputs(result.Files)
	lodDoc := BuildLODDocument(asset, result.Dolly, result.Error, opts.ErrorDistance, outputs)
	if err := WriteJSON(result.Files["lodData"], lodDoc); err != nil {
		return nil, err
	}
	return result, nil
}

func writeCoverageStrip(asset *Asset, opts PipelineOptions, files map[string]string) error {
	distances := []float64{
		asset.Camera.FocusDistance * 1.15,
		asset.Camera.FocusDistance * 1.7,
		asset.Camera.FocusDistance * 2.5,
		asset.Camera.FocusDistance * 3.8,
		math.Min(asset.Camera.FarDistance, asset.Camera.FocusDistance*6.2),
	}
	const panelW, panelH, gap = 220, 300, 8
	canvas := image.NewRGBA(image.Rect(0, 0, panelW*len(distances)+gap*(len(distances)-1), panelH))
	for i, distance := range distances {
		camera := NewCamera(asset.Camera, panelW, panelH, distance)
		field, _ := asset.RenderLOD(camera, distance)
		panel := ColorizeField(field, PaletteLOD)
		draw.Draw(canvas, image.Rect(i*(panelW+gap), 0, i*(panelW+gap)+panelW, panelH), panel, image.Point{}, draw.Src)
	}
	path := filepath.Join(opts.OutDir, "coverage_strip.png")
	files["coverageStrip"] = path
	return WritePNG(path, canvas)
}

func relativeOutputs(files map[string]string) map[string]string {
	out := make(map[string]string, len(files))
	for name, path := range files {
		out[name] = filepath.Base(path)
	}
	return out
}
