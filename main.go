package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"colorlab/colorlab"
)

const maxRoundTripError = 1e-9

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "colorlab:", err)
		os.Exit(1)
	}
}

func run() error {
	outDir := flag.String("out", "out", "directory for PPM images and the report")
	cell := flag.Int("cell", 64, "patch size in pixels")
	gutter := flag.Int("gutter", 4, "black gap between mosaic bands in pixels")
	flag.Parse()

	if *cell < 1 {
		return fmt.Errorf("cell size must be positive")
	}
	if *gutter < 0 {
		return fmt.Errorf("gutter must not be negative")
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}

	chart := colorlab.StandardChart()
	var report strings.Builder
	report.WriteString("colorlab conversion report\n")
	report.WriteString("==========================\n")
	report.WriteString("chart: 11 neutral gray steps, 11 primary/secondary bars, 11 skin tones\n")
	report.WriteString("band 1: chart in source space; band 2: forward result; band 3: round trip\n\n")

	referencePath := filepath.Join(*outDir, "reference_srgb.ppm")
	if err := colorlab.RenderChart(chart, colorlab.SRGB, *cell).WritePPMFile(referencePath); err != nil {
		return err
	}
	report.WriteString("images:\n")
	report.WriteString("  " + referencePath + "\n")
	for _, space := range colorlab.AllSpaces() {
		path := filepath.Join(*outDir, "encoded_"+space.Slug()+".ppm")
		if err := colorlab.RenderChart(chart, space, *cell).WritePPMFile(path); err != nil {
			return err
		}
		report.WriteString("  " + path + "\n")
	}

	metrics := colorlab.AllChainMetrics(chart)
	allOK := true
	maxPeak := 0.0
	report.WriteString("\nround-trip chains:\n")
	for _, metric := range metrics {
		path := filepath.Join(*outDir, colorlab.ChainFileName(metric.From, metric.To))
		mosaic := colorlab.RenderChainMosaic(chart, metric.From, metric.To, *cell, *gutter)
		if err := mosaic.WritePPMFile(path); err != nil {
			return err
		}
		if metric.PeakError > maxPeak {
			maxPeak = metric.PeakError
		}
		if metric.PeakError > maxRoundTripError {
			allOK = false
		}
		report.WriteString("  " + metric.SummaryLine() + "\n")
		report.WriteString("    " + path + "\n")
	}
	report.WriteString(fmt.Sprintf("\nmax peak error: %.9g, limit %.0e\n",
		maxPeak, maxRoundTripError))

	report.WriteString("\ntransfer monotonicity on [0,1], including boundaries (tol 1e-12):\n")
	for _, check := range colorlab.CheckMonotonicity() {
		ok := check.Monotonic && check.ZeroOK && check.OneOK
		if !ok {
			allOK = false
		}
		report.WriteString(fmt.Sprintf(
			"  %-16s monotonic=%-5t endpoints=(%t,%t) max_backtrack=%.3e samples=%d\n",
			check.Name, check.Monotonic, check.ZeroOK, check.OneOK,
			check.MaxBacktrack, check.Samples,
		))
	}

	if allOK {
		report.WriteString("\nresult: PASS\n")
	} else {
		report.WriteString("\nresult: FAIL\n")
	}

	reportPath := filepath.Join(*outDir, "report.txt")
	if err := os.WriteFile(reportPath, []byte(report.String()), 0o644); err != nil {
		return err
	}
	fmt.Print(report.String())
	fmt.Printf("\nreport: %s\n", reportPath)

	if !allOK {
		return fmt.Errorf("conversion validation failed; see %s", reportPath)
	}
	return nil
}
