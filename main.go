package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	imageWidth    = 1280
	imageHeight   = 800
	highlightFile = "artifacts/lod_boundary_highlight.ppm"
	metricsFile   = "artifacts/metrics.json"
	reportFile    = "artifacts/lod_report.md"
)

func countTriangles(mesh *Mesh, tier uint8) int {
	count := 0
	for _, tri := range mesh.Tris {
		if tri.Tier == tier {
			count++
		}
	}
	return count
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeMarkdown(path string, report Report) error {
	content := "# CAD LOD 弦高预算与跨级缝合报告\n\n"
	content += "| 档位 | 弦高预算 mm | 网格 U×V | 三角形数 | 最大弦高 mm | 最大轮廓偏差 mm | 边界轮廓偏差 mm | 屏幕偏差 px |\n"
	content += "|---|---:|---:|---:|---:|---:|---:|---:|\n"
	for _, level := range report.Levels {
		content += markdownRow(level)
	}
	content += "\n## 跨级裂缝\n\n"
	content += bulletf("LOD 边界总长度：`%.6f mm`", report.CrackMetrics.LODBoundaryLengthMM)
	content += bulletf("未缝合时开口边总长度（两侧边累计）：`%.6f mm`", report.CrackMetrics.NaiveOpenEdgeLengthMM)
	content += bulletf("未缝合时裂缝中心线长度：`%.6f mm`", report.CrackMetrics.NaiveCrackCenterlineLengthMM)
	content += bulletf("未缝合跨级最大开口量：`%.6f mm`", report.CrackMetrics.NaiveMaxTransitionOpeningMM)
	content += bulletf("缝合后残留裂缝长度：`%.9f mm`", report.CrackMetrics.CrossLODCrackLengthMM)
	content += bulletf("缝合后非二邻接边界边数：`%d`", report.CrackMetrics.UnmatchedStitchedEdgeCount)
	content += "\n## 输出\n\n"
	for _, output := range report.Outputs {
		content += "- `" + output + "`\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func markdownRow(level LevelResult) string {
	return fmt.Sprintf("| %s | %.6f | %d×%d | %d | %.6f | %.6f | %.6f | %.4f |\n",
		level.Label,
		level.ChordHeightBudgetMM,
		level.GridU,
		level.GridV,
		level.Metrics.TriangleCount,
		level.Metrics.MaxChordHeightMM,
		level.Metrics.MaxProfileDeviationMM,
		level.Metrics.MaxBoundaryContourDeviationMM,
		level.Metrics.MaxScreenProfileDeviationPixel,
	)
}

func bulletf(format string, values ...any) string {
	return "- " + fmt.Sprintf(format, values...) + "\n"
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	if err := os.MkdirAll("artifacts", 0o755); err != nil {
		return err
	}

	surface := newSurface()
	levels := defaultLevels()
	results := make([]LevelResult, 0, len(levels))
	distances := []float64{17, 25, 36}

	for index, cfg := range levels {
		mesh := buildLevelMesh(surface, cfg.Nu, cfg.Nv, cfg.Tier)
		camera := newCamera(distances[index], imageWidth, imageHeight)
		results = append(results, measureLevel(surface, mesh, cfg, camera))
		if results[index].Metrics.MaxChordHeightMM > cfg.Budget {
			return fmt.Errorf("%s LOD exceeds chord-height budget: %.9f > %.9f",
				cfg.ID,
				results[index].Metrics.MaxChordHeightMM,
				cfg.Budget,
			)
		}
		image := renderMesh(surface, mesh, camera, false)
		if err := writePPM(cfg.File, image); err != nil {
			return err
		}
	}

	fullMesh := buildFullMesh(surface, defaultGrids())
	highlightCamera := newCamera(24, imageWidth, imageHeight)
	highlight := renderMesh(surface, fullMesh, highlightCamera, true)
	if err := writePPM(highlightFile, highlight); err != nil {
		return err
	}

	cracks := measureCracks(surface, fullMesh, defaultGrids())
	zipperCount := countTriangles(fullMesh, tierTransition)
	tieredCount := len(fullMesh.Tris) - zipperCount
	report := Report{
		Unit:                  "millimetre",
		ImageWidth:            imageWidth,
		ImageHeight:           imageHeight,
		Levels:                results,
		TieredTriangleCount:   tieredCount,
		ZipperTriangleCount:   zipperCount,
		BoundaryHighlightFile: highlightFile,
		CrackMetrics:          cracks,
		Outputs: []string{
			"artifacts/near.ppm",
			"artifacts/mid.ppm",
			"artifacts/far.ppm",
			highlightFile,
			metricsFile,
			reportFile,
		},
	}

	if err := writeJSON(metricsFile, report); err != nil {
		return err
	}
	if err := writeMarkdown(reportFile, report); err != nil {
		return err
	}

	return filepath.WalkDir("artifacts", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			println(path)
		}
		return nil
	})
}
