package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func ftoa(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }

func writeSummaryCSV(cfg Config, results []SceneResult) string {
	path := filepath.Join(cfg.OutDir, "summary.csv")
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{
		"scene", "motion", "strategy", "eye",
		"flicker_pixels", "periph_flicker", "mean_lum_var", "periph_lum_var",
		"max_lum_var", "rmse", "sharpness", "rel_sharpness",
		"inter_eye_residual", "false_stability",
	})
	for _, r := range results {
		for _, e := range r.Eyes {
			_ = w.Write([]string{
				r.Scene, r.Motion, r.Strategy, e.Eye,
				strconv.Itoa(e.FlickerPixels), strconv.Itoa(e.PeriphFlicker),
				ftoa(e.MeanLumVar), ftoa(e.PeriphVar), ftoa(e.MaxLumVar),
				ftoa(e.RMSE), ftoa(e.Sharpness), ftoa(e.RelSharpness),
				ftoa(r.InterEyeResidual), strconv.FormatBool(r.FalseStability),
			})
		}
	}
	return path
}

func writeMarkdownReport(cfg Config, results []SceneResult) string {
	path := filepath.Join(cfg.OutDir, "report.md")
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintf(f, "# Foveated sampling flicker report\n\n")
	fmt.Fprintf(f, "Frames=%d  viewport=%dx%d  gt=%dx%d jittered samples/pixel  tau=%.2f\n\n",
		cfg.Frames, cfg.W, cfg.H, cfg.GTSpp, cfg.GTSpp, cfg.Tau)
	fmt.Fprintln(f, "| scene | motion | strategy | eye | flicker px | periph flicker | periph var | rmse | rel sharpness | inter-eye | false stable |")
	fmt.Fprintln(f, "|---|---|---|---|---:|---:|---:|---:|---:|---:|:---:|")
	for _, r := range results {
		for _, e := range r.Eyes {
			fmt.Fprintf(f, "| %s | %s | %s | %s | %d | %d | %.5f | %.4f | %.3f | %.5f | %t |\n",
				r.Scene, r.Motion, r.Strategy, e.Eye,
				e.FlickerPixels, e.PeriphFlicker, e.PeriphVar,
				e.RMSE, e.RelSharpness, r.InterEyeResidual, r.FalseStability)
		}
	}

	fmt.Fprint(f, "\n## Worst temporal series (left eye, top variance pixel per run)\n\n")
	fmt.Fprintln(f, "| scene | motion | strategy | pixel | variance | luminance series |")
	fmt.Fprintln(f, "|---|---|---|---|---:|---|")
	for _, r := range results {
		if len(r.Eyes) == 0 || len(r.Eyes[0].TopPixels) == 0 {
			continue
		}
		p := r.Eyes[0].TopPixels[0]
		series := ""
		for i, v := range p.Lum {
			if i > 0 {
				series += " "
			}
			series += ftoa(v)
		}
		fmt.Fprintf(f, "| %s | %s | %s | (%d,%d) | %.5f | %s |\n",
			r.Scene, r.Motion, r.Strategy, p.X, p.Y, p.Var, series)
	}
	return path
}
