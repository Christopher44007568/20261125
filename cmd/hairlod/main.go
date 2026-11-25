package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"hairlod/internal/hair"
)

func main() {
	out := flag.String("out", "out", "artifact output directory")
	strands := flag.Int("strands", hair.DefaultStrandCount, "number of full strands")
	guides := flag.Int("guides", hair.DefaultGuideCount, "number of level-1 guides")
	seed := flag.Int64("seed", hair.DefaultSeed, "deterministic generation/clustering seed")
	points := flag.Int("points", 32, "points per strand")
	width := flag.Int("width", 720, "preview image width")
	height := flag.Int("height", 720, "preview image height")
	analysisSize := flag.Int("analysis-size", 200, "coverage-curve raster size")
	dollySamples := flag.Int("dolly-samples", 25, "number of camera distances in continuity check")
	errorDistance := flag.Float64("error-distance", 18, "distance used for LOD and error heatmap")
	flag.Parse()

	start := time.Now()
	fmt.Printf("hairlod: generating %d strands and %d guides\n", *strands, *guides)
	result, err := hair.RunPipeline(hair.PipelineOptions{
		OutDir:        *out,
		StrandCount:   *strands,
		GuideCount:    *guides,
		Seed:          *seed,
		Points:        *points,
		Width:         *width,
		Height:        *height,
		AnalysisSize:  *analysisSize,
		DollySamples:  *dollySamples,
		ErrorDistance: *errorDistance,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hairlod:", err)
		os.Exit(1)
	}

	fmt.Printf("guides: %v\n", levelCounts(result.Asset))
	fmt.Printf("dolly:  %s\n", result.Dolly.Summary())
	fmt.Printf("error:  mean=%.5f p95=%.5f max=%.5f IoU=%.3f\n",
		result.Error.MeanAbs, result.Error.P95Abs, result.Error.MaxAbs, result.Error.IoU)
	fmt.Printf("wrote %d files to %s in %s\n", len(result.Files), *out, time.Since(start).Round(time.Millisecond))
	if !result.Dolly.Continuous {
		fmt.Fprintln(os.Stderr, "hairlod: warning: continuity acceptance thresholds were not met")
	}
}

func levelCounts(asset *hair.Asset) []int {
	out := make([]int, len(asset.Levels))
	for i, level := range asset.Levels {
		out[i] = len(level.Entries)
	}
	return out
}
