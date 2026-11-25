package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	out := flag.String("out", "artifacts", "output directory")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	scenarios, err := buildScenarios()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	runs := make([]ScenarioRun, 0, len(scenarios))
	for _, sc := range scenarios {
		result, err := analyzeScenario(sc)
		if err != nil {
			return err
		}
		dir := filepath.Join(outDir, sc.ID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		files := []struct {
			name string
			body string
		}{
			{"three_view.svg", renderThreeView(result)},
			{"control_migration.svg", renderControlMigration(result)},
			{"motion_sheet.svg", renderMotionSheet(result)},
		}
		for _, file := range files {
			path := filepath.Join(dir, file.name)
			if err := os.WriteFile(path, []byte(file.body), 0o644); err != nil {
				return err
			}
		}
		runs = append(runs, result)
	}
	if err := writeMetrics(filepath.Join(outDir, "metrics.json"), runs); err != nil {
		return err
	}
	if err := writeReport(filepath.Join(outDir, "report.html"), runs); err != nil {
		return err
	}
	for _, run := range runs {
		fmt.Printf("%s: A=%d B=%d blend=%d tip-over=%.2g span-over=%.2g hash=%s\n",
			run.Scenario.ID,
			run.Report.StrandsA,
			run.Report.StrandsB,
			run.Report.BlendStrands,
			run.Report.MaxTipOverBound,
			run.Report.MaxSpanOverBound,
			run.Report.DeterminismHash[:12])
	}
	fmt.Printf("wrote %s\n", outDir)
	return nil
}
