package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"
)

func DefaultOptions() RunOptions {
	return RunOptions{
		OutDir: "artifacts/stereoui",
		Width:  1280,
		Height: 960,
		IPD:    0.064,
	}
}

func worstStatus(current, next string) string {
	rank := map[string]int{"pass": 0, "visible": 0, "warning": 1, "edge_risk": 1, "fail": 2, "clipped": 2, "culled": 2}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

func summarizeParallax(glyphs []GlyphPair) ParallaxSummary {
	summary := ParallaxSummary{
		GlyphCount:        len(glyphs),
		MinCopyErrorPx:    math.Inf(1),
		CoordinatesCopied: false,
	}
	if len(glyphs) == 0 {
		summary.MinCopyErrorPx = 0
		return summary
	}

	var hSum, aSum, copySum float64
	allIndependent := true
	path := ""
	for _, g := range glyphs {
		hSum += g.Parallax.HorizontalPx
		aSum += g.Parallax.AngularArcMin
		copySum += g.Parallax.NaiveCopyErrorPx
		summary.MinCopyErrorPx = math.Min(summary.MinCopyErrorPx, g.Parallax.NaiveCopyErrorPx)
		summary.MaxAbsVerticalPx = math.Max(summary.MaxAbsVerticalPx, math.Abs(g.Parallax.VerticalPx))
		summary.MaxAbsVerticalArcMin = math.Max(summary.MaxAbsVerticalArcMin, math.Abs(g.Parallax.VerticalAngularArcMin))
		summary.CoordinatesCopied = summary.CoordinatesCopied || g.Parallax.CoordinatesCopied
		allIndependent = allIndependent && g.Parallax.ProjectionPath != ""
		path = g.Parallax.ProjectionPath
	}
	n := float64(len(glyphs))
	summary.MeanHorizontalPx = hSum / n
	summary.MeanAngularArcMin = aSum / n
	summary.MeanCopyErrorPx = copySum / n
	summary.IndependentProjection = allIndependent && !summary.CoordinatesCopied
	summary.ProjectionPath = path
	summary.Status = "pass"
	if summary.CoordinatesCopied || !summary.IndependentProjection {
		summary.Status = "fail"
		summary.Failures = append(summary.Failures, "eye coordinate was copied instead of independently projected")
	}
	if summary.MaxAbsVerticalArcMin > 3.0 {
		summary.Status = worstStatus(summary.Status, "fail")
		summary.Failures = append(summary.Failures, "vertical angular disparity exceeds 3 arcmin")
	}
	if summary.MinCopyErrorPx < 0.5 {
		summary.Status = worstStatus(summary.Status, "warning")
		summary.Warnings = append(summary.Warnings, "left/right coordinates nearly coincide; verify geometry, do not copy by construction")
	}
	return summary
}

func savePNG(path string, canvas *Canvas) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, canvas.RGBA)
}

func Run(options RunOptions) (*Report, error) {
	if options.Width <= 0 || options.Height <= 0 {
		def := DefaultOptions()
		if options.Width <= 0 {
			options.Width = def.Width
		}
		if options.Height <= 0 {
			options.Height = def.Height
		}
	}
	if options.IPD <= 0 {
		options.IPD = DefaultOptions().IPD
	}
	if options.OutDir == "" {
		options.OutDir = DefaultOptions().OutDir
	}

	frameDir := filepath.Join(options.OutDir, "frames")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		return nil, err
	}

	scenarios := Scenarios()
	layers := DefaultLayers()
	report := &Report{
		Schema:      "stereoui-report/v1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Generator:   "stereoui Go stdlib reference pipeline",
		Options:     options,
	}
	report.Summary.Artifacts = map[string]string{
		"framesDir":      "frames",
		"reportJson":     "report.json",
		"reportMarkdown": "report.md",
		"parallaxCsv":    "parallax.csv",
		"boundaryCsv":    "boundary.csv",
		"readabilityCsv": "readability.csv",
		"sharpnessCsv":   "sharpness.csv",
	}
	report.Summary.CopyGuard = "independent_per_eye_off_axis_projection; same_screen_coordinate_copy_is_never_used"
	report.Summary.LayerCount = len(layers)
	report.Summary.ScenarioCount = len(scenarios)
	systemSharpnessPass := true

	for _, scenario := range scenarios {
		eyes := scenario.BuildEyes(options.Width, options.Height, options.IPD)
		scenarioReport := ScenarioReport{Scenario: scenario, Eyes: eyes}

		for _, layer := range layers {
			layerReport := LayerReport{
				LayerID:     layer.ID,
				NameZH:      layer.NameZH,
				DepthClass:  layer.DepthClass,
				FrameFiles:  map[EyeID]string{},
				PanelBounds: map[EyeID]FloatRect{},
				Status:      "pass",
			}

			geometries := make([]ElementGeometry, 0, len(layer.Texts))
			for _, spec := range layer.Texts {
				geo := ProjectText(spec, layer, eyes)
				geometries = append(geometries, geo)
				layerReport.Glyphs = append(layerReport.Glyphs, geo.Glyphs...)
			}

			canvases := map[EyeID]*Canvas{}
			for eyeIndex, eye := range eyes {
				canvas := NewCanvas(options.Width, options.Height, color.RGBA{R: 7, G: 11, B: 31, A: 255})
				panelBounds := renderEyeFrame(canvas, scenario, layer, eye, geometries)
				canvases[eye.ID] = canvas
				layerReport.PanelBounds[eye.ID] = panelBounds

				filename := fmt.Sprintf("%s_%s_%s.png", scenario.ID, layer.ID, eye.ID)
				absolute := filepath.Join(frameDir, filename)
				if err := savePNG(absolute, canvas); err != nil {
					return nil, err
				}
				relative := filepath.Join("frames", filename)
				layerReport.FrameFiles[eye.ID] = relative
				report.Summary.EyeFrameCount++
				_ = eyeIndex
			}

			for elementIndex, spec := range layer.Texts {
				geo := geometries[elementIndex]
				elementReport := ElementReport{
					Spec:        spec,
					Status:      "pass",
					Parallax:    summarizeParallax(geo.Glyphs),
					Boundary:    map[EyeID]BoundaryResult{},
					Readability: map[EyeID]ReadabilityResult{},
					Sharpness:   map[EyeID]SharpnessResult{},
				}
				elementReport.Status = worstStatus(elementReport.Status, elementReport.Parallax.Status)

				for _, eye := range eyes {
					var bounds FloatRect
					if eye.ID == LeftEye {
						bounds = geo.LeftBounds
					} else {
						bounds = geo.RightBounds
					}
					boundary := CheckBoundary(bounds, eye)
					elementReport.Boundary[eye.ID] = boundary
					elementReport.Status = worstStatus(elementReport.Status, boundary.Status)

					readability := AnalyzeReadability(canvases[eye.ID].RGBA, geo, eye, scenario)
					elementReport.Readability[eye.ID] = readability
					elementReport.Status = worstStatus(elementReport.Status, readability.Status)
					report.Summary.ElementEyeChecks++

					if spec.CheckSharpness {
						sharpness := MeasureSharpness(canvases[eye.ID].RGBA, spec.ID, bounds, eye)
						elementReport.Sharpness[eye.ID] = sharpness
						elementReport.Status = worstStatus(elementReport.Status, sharpness.Status)
						if !sharpness.Pass {
							systemSharpnessPass = false
						}
					}
				}

				layerReport.Elements = append(layerReport.Elements, elementReport)
				layerReport.Status = worstStatus(layerReport.Status, elementReport.Status)
				switch elementReport.Status {
				case "pass":
					report.Summary.PassCount++
				case "warning", "edge_risk":
					report.Summary.WarningCount++
				default:
					report.Summary.FailCount++
				}
				report.Summary.GlyphPairCount += len(geo.Glyphs)
			}

			scenarioReport.Layers = append(scenarioReport.Layers, layerReport)
		}
		report.Scenarios = append(report.Scenarios, scenarioReport)
	}

	report.Summary.SystemSharpnessPass = systemSharpnessPass
	report.Summary.OverallPass = report.Summary.FailCount == 0 && systemSharpnessPass

	if err := writeJSONReport(options.OutDir, report); err != nil {
		return nil, err
	}
	if err := writeParallaxCSV(options.OutDir, report); err != nil {
		return nil, err
	}
	if err := writeBoundaryCSV(options.OutDir, report); err != nil {
		return nil, err
	}
	if err := writeReadabilityCSV(options.OutDir, report); err != nil {
		return nil, err
	}
	if err := writeSharpnessCSV(options.OutDir, report); err != nil {
		return nil, err
	}
	if err := writeMarkdownReport(options.OutDir, report); err != nil {
		return nil, err
	}
	return report, nil
}

func writeJSONReport(outDir string, report *Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "report.json"), data, 0o644)
}
