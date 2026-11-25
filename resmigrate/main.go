package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"resmigrate/internal/report"
	"resmigrate/internal/sim"
)

func main() {
	outDir := flag.String("out", "../artifacts/resmigrate-go", "output directory for report.json and report.md")
	flag.Parse()

	specs := sim.Scenarios()
	scenarios := make([]sim.ScenarioReport, 0, len(specs))
	passCount := 0
	for _, spec := range specs {
		r := sim.Run(spec)
		if r.OK {
			passCount++
		}
		scenarios = append(scenarios, *r)
	}
	root := sim.BuildReport(scenarios)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail(err)
	}
	jsonBytes, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(filepath.Join(*outDir, "report.json"), append(jsonBytes, '\n'), 0o644); err != nil {
		fail(err)
	}
	md := report.RenderMarkdown(root, passCount)
	if err := os.WriteFile(filepath.Join(*outDir, "report.md"), []byte(md), 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("scenarios=%d passed=%d failed=%d out=%s\n", len(scenarios), passCount, len(scenarios)-passCount, *outDir)
	if passCount != len(scenarios) {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "resmigrate:", err)
	os.Exit(2)
}
