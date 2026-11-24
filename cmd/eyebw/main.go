package main

import (
	"flag"
	"fmt"
	"os"

	"eyebw"
)

func main() {
	out := flag.String("out", "artifacts/eyebw", "directory for JSON/CSV/Markdown reports")
	matrix := flag.Bool("matrix", false, "run the full resolution/MSAA/foveation matrix")
	events := flag.Bool("events", false, "write every access event to events.csv")
	strictBudget := flag.Bool("strict-budget", false, "exit with code 2 when any frame exceeds its bandwidth budget")
	flag.Parse()

	var scenarios []eyebw.Scenario
	if *matrix {
		scenarios = eyebw.MatrixScenarios(4)
	} else {
		scenarios = eyebw.DefaultScenarios()
	}
	if *events {
		for i := range scenarios {
			scenarios[i].Config.RetainAccesses = true
		}
	}
	report, err := eyebw.RunScenarios(scenarios)
	if err != nil {
		fmt.Fprintf(os.Stderr, "eyebw: %v\n", err)
		os.Exit(1)
	}
	if err := eyebw.WriteReport(*out, report, *events); err != nil {
		fmt.Fprintf(os.Stderr, "eyebw: write report: %v\n", err)
		os.Exit(1)
	}

	budgetFailure := printSummary(report, *out)
	if !report.Success {
		os.Exit(1)
	}
	if *strictBudget && budgetFailure {
		os.Exit(2)
	}
}

func printSummary(report *eyebw.Report, out string) bool {
	budgetExceeded := false
	fmt.Printf("带宽报告已写入 %s/\n", out)
	for _, scenario := range report.Scenarios {
		fmt.Printf("\n%s\n", scenario.Name)
		for _, p := range scenario.Pipelines {
			if p.BudgetExceededFrames > 0 {
				budgetExceeded = true
			}
			top := "-"
			if len(p.Hotspots) > 0 {
				top = string(p.Hotspots[0].Stage)
			}
			quality := "pass"
			if !p.AllQualityPassed {
				quality = fmt.Sprintf("FAIL(%d frames)", p.QualityFailedFrames)
			}
			fmt.Printf("  %-22s avg=%-12s read=%-12s write=%-12s hit=%5.1f%% budget_over=%2d frames hotspot=%-18s quality=%s\n",
				p.ID,
				eyebw.FormatBytes(p.Average.TransferTotal()),
				eyebw.FormatBytes(p.Average.TransferReadBytes),
				eyebw.FormatBytes(p.Average.TransferWriteBytes),
				p.Average.CacheHitRatioBytes*100,
				p.BudgetExceededFrames,
				top,
				quality,
			)
		}
		if !scenario.Success {
			fmt.Printf("  expectation: FAIL: %s\n", scenario.FailureReason)
		} else if scenario.ExpectQualityFailure {
			fmt.Printf("  expectation: pass (expected %s quality failure was recorded)\n", scenario.ExpectedFailedPipeline)
		}
	}
	return budgetExceeded
}
