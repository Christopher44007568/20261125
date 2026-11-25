package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesContinuousFramesAndStableMetrics(t *testing.T) {
	out := t.TempDir()
	cfg := Config{
		OutDir:      out,
		Frames:      72,
		Width:       160,
		Height:      90,
		ContactCols: 6,
	}
	if err := run(cfg); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(out, "frames"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != cfg.Frames {
		t.Fatalf("expected %d frames, got %d", cfg.Frames, len(entries))
	}
	for _, name := range []string{
		"contact_sheet.png", "metrics.csv", "triangle_counts.csv",
		"brightness_curve.svg", "triangle_counts.svg", "metrics_chart.png",
		"stability_report.json",
	} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(out, "stability_report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report StabilityReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.RandomStrandDisappear || report.IndependentStrandFade {
		t.Fatal("transition must not use independent random strand disappearance")
	}
	if report.MaxLumaFrameStep > 3.2 {
		t.Fatalf("brightness step too large: %.3f", report.MaxLumaFrameStep)
	}
	if report.MaxCoverageFrameStep > 0.025 {
		t.Fatalf("coverage step too large: %.5f", report.MaxCoverageFrameStep)
	}
	if report.MaxLumaInLODTransition > 2.0 {
		t.Fatalf("LOD transition brightness step too large: %.3f",
			report.MaxLumaInLODTransition)
	}
	if report.MaxCoverageInLODTransition > 0.015 {
		t.Fatalf("LOD transition coverage step too large: %.5f",
			report.MaxCoverageInLODTransition)
	}
}
