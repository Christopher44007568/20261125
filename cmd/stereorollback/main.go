package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	stereo "stereorollback"
)

func main() {
	out := flag.String("out", "artifacts", "directory for JSON/text report and fallback frames")
	alsoReports := flag.Bool("reports-dir", true, "also write a tracked copy under ./reports")
	check := flag.Bool("check", false, "exit non-zero if any invariant fails")
	flag.Parse()

	report, err := stereo.RunAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %v\n", err)
		os.Exit(2)
	}

	text := stereo.RenderText(report)
	fmt.Print(text)

	targets := []string{*out}
	if *alsoReports {
		targets = append(targets, "reports")
	}
	for _, dir := range targets {
		if err := writeBundle(dir, report, text); err != nil {
			fmt.Fprintf(os.Stderr, "write to %s failed: %v\n", dir, err)
			os.Exit(2)
		}
	}

	if *check && !report.Summary.AllInvariantsHold {
		os.Exit(1)
	}
}

func writeBundle(dir string, report *stereo.Report, text string) error {
	framesDir := filepath.Join(dir, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		return err
	}
	jsonBytes, err := report.MarshalJSONBytes()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "rollback-report.json"), jsonBytes, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "stereo-rollback-report.txt"), []byte(text), 0o644); err != nil {
		return err
	}
	for _, sc := range report.Scenarios {
		for _, fr := range sc.Frames {
			if fr.FallbackFrame == nil {
				continue
			}
			img := stereo.ComposeFallbackImage(*fr.FallbackFrame)
			pngBytes, err := img.PNG()
			if err != nil {
				return err
			}
			name := fmt.Sprintf("%s-f%d-fallback.png", sc.ID, fr.Index)
			if err := os.WriteFile(filepath.Join(framesDir, name), pngBytes, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
