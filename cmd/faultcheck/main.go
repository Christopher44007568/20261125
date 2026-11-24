package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"binocularfault/binocular"
)

func main() {
	out := flag.String("out", "artifacts/report.json", "JSON report path")
	cycles := flag.Int("cycles", 2, "reconnect cycles per disconnect scope")
	flag.Parse()

	scopes := []binocular.DisconnectScope{
		binocular.DisconnectLeftEye,
		binocular.DisconnectRightEye,
		binocular.DisconnectDevice,
	}
	scenarios := make([]binocular.ScenarioReport, 0, len(scopes))
	for _, scope := range scopes {
		scenario, err := binocular.RunScenario(scope, binocular.ScenarioOptions{Cycles: *cycles})
		if err != nil {
			fmt.Fprintf(os.Stderr, "scope %s failed: %v\n", scope, err)
			os.Exit(1)
		}
		scenarios = append(scenarios, scenario)
	}
	report := binocular.BuildReport(scenarios)
	if err := writeReport(*out, report); err != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", err)
		os.Exit(1)
	}
	printSummary(report, *out)
	if !report.Summary.AllClean {
		os.Exit(2)
	}
}

func writeReport(path string, report binocular.Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func printSummary(report binocular.Report, path string) {
	fmt.Printf("Go %s; report: %s\n", report.GoVersion, path)
	for _, scenario := range report.Scenarios {
		for _, cycle := range scenario.Cycles {
			fmt.Printf("%-10s cycle=%d clean=%t running=%d queued=%d stopped=%d queued_cancel=%d still=%d unreleased=%d frame_diff=%d reconnect=%.3fms\n",
				scenario.Scope,
				cycle.Cycle,
				cycle.Clean,
				cycle.Release.Tasks.Running,
				cycle.Release.Tasks.Queued,
				cycle.Release.Tasks.Stopped,
				cycle.Release.Tasks.QueuedCancel,
				cycle.Release.Tasks.StillRunning,
				len(cycle.Unreleased),
				cycle.FirstFrame.DiffBytes,
				cycle.Reconnect.TotalMS,
			)
			for _, category := range cycle.Release.Categories {
				fmt.Printf("  %-12s discovered=%d released=%d failed=%d ok=%t",
					category.Name, category.Discovered, category.Released, category.Failed, category.OK)
				if category.Name == "cache_keys" {
					fmt.Printf(" keys=%v", category.Keys)
				}
				fmt.Println()
			}
		}
	}
	fmt.Printf("all_clean=%t unreleased=%d max_first_frame_diff=%d latencies_ns=%v\n",
		report.Summary.AllClean,
		report.Summary.TotalUnreleased,
		report.Summary.MaxFirstFrameDiff,
		report.Summary.ReconnectLatencies,
	)
}
