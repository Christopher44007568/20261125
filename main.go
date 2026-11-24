package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	quick := flag.Bool("quick", false, "smaller viewport and fewer frames")
	out := flag.String("out", "artifacts/results", "output directory")
	flag.Parse()

	cfg := defaultConfig()
	if *quick {
		cfg = testConfig()
		cfg.OutDir = *out
	}
	cfg.OutDir = *out
	ensureDir(cfg.OutDir)

	results := runAll(cfg)
	csvPath := writeSummaryCSV(cfg, results)
	mdPath := writeMarkdownReport(cfg, results)

	fmt.Println("strategy            scene                motion          eye   flicker periphVar rmse   relSharp falseStable")
	for _, r := range results {
		for _, e := range r.Eyes {
			fmt.Printf("%-19s %-20s %-15s %-5s %7d %.5f   %.4f %.3f    %t\n",
				r.Strategy, r.Scene, r.Motion, e.Eye,
				e.FlickerPixels, e.PeriphVar, e.RMSE, e.RelSharpness, r.FalseStability)
		}
	}
	fmt.Printf("\nwrote %s\nwrote %s\nartifacts under %s\n",
		csvPath, mdPath, filepath.Clean(cfg.OutDir))
	os.Stdout.Sync()
}
