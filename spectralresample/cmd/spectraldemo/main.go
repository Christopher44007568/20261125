// Command spectraldemo runs the full multi-band reflectance pipeline:
// resample measured spectra, integrate CIE tristimulus values under D65 and
// a narrow-band LED, build metameric pairs, and emit PPM charts, error
// heatmaps, spectral SVG curves and a CSV/Markdown report.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"spectralresample"
)

func main() {
	dataDir := flag.String("data", "data", "directory containing the extracted CIE/ColorChecker CSV tables")
	outDir := flag.String("out", "output", "directory for generated PPM, SVG and report files")
	flag.Parse()

	cfg := spectralresample.PipelineConfig{
		DataDir: *dataDir,
		LEDName: "NARROWBAND LED 450/530/630 NM",
		LED: []spectralresample.LEDEmitter{
			{PeakNM: 450, FWHMNM: 22, PeakPower: 0.50},
			{PeakNM: 530, FWHMNM: 22, PeakPower: 0.70},
			{PeakNM: 630, FWHMNM: 20, PeakPower: 0.85},
		},
		Metamer: spectralresample.DefaultMetamerConfig(),
	}
	pipeline, err := spectralresample.NewPipeline(cfg)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	steps := []struct {
		name string
		fn   func(*spectralresample.Pipeline, string) error
	}{
		{"illuminant spectra", writeIlluminantSVG},
		{"patch PPM charts", writePatchCharts},
		{"metamerism error maps", writeErrorMaps},
		{"family spectral curves", writeFamilySpectra},
		{"CSV report", writeCSVReport},
		{"Markdown summary", writeMarkdownReport},
	}
	for _, step := range steps {
		if err := step.fn(pipeline, *outDir); err != nil {
			fatal(fmt.Errorf("%s: %w", step.name, err))
		}
		fmt.Println("ok:", step.name)
	}
	fmt.Println("all artefacts written to", filepath.Clean(*outDir))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "spectraldemo:", err)
	os.Exit(1)
}
