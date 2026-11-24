package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	out := flag.String("out", "artifacts/stereoui", "directory for PNG, JSON, CSV and Markdown results")
	width := flag.Int("width", 1280, "single-eye frame width in pixels")
	height := flag.Int("height", 960, "single-eye frame height in pixels")
	ipd := flag.Float64("ipd", 0.064, "interpupillary distance in meters")
	strict := flag.Bool("strict", false, "exit non-zero when any validation fails")
	flag.Parse()

	report, err := Run(RunOptions{
		OutDir: *out,
		Width:  *width,
		Height: *height,
		IPD:    *ipd,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "stereoui: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("stereoui: wrote %d eye frames to %s/frames\n", report.Summary.EyeFrameCount, *out)
	fmt.Printf("stereoui: %d glyph pairs, pass=%d warn=%d fail=%d sharpness=%t\n",
		report.Summary.GlyphPairCount,
		report.Summary.PassCount,
		report.Summary.WarningCount,
		report.Summary.FailCount,
		report.Summary.SystemSharpnessPass,
	)
	fmt.Printf("stereoui: report %s/report.md\n", *out)
	if *strict && !report.Summary.OverallPass {
		os.Exit(2)
	}
}
