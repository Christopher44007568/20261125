package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"geomlife"
)

func main() {
	outPath := flag.String("out", "artifacts/report.json", "JSON report output path; empty string disables file output")
	asJSON := flag.Bool("json", false, "print the full JSON report to stdout")
	flag.Parse()

	report := geomlife.RunSimulation()

	if *asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "encode report: %v\n", err)
			os.Exit(1)
		}
	}

	if *outPath != "" {
		if err := writeReport(*outPath, report); err != nil {
			fmt.Fprintf(os.Stderr, "write report %s: %v\n", *outPath, err)
			os.Exit(1)
		}
	}

	printSummary(report, *outPath)
	if !report.OK {
		os.Exit(2)
	}
}

func writeReport(path string, report geomlife.Report) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func printSummary(report geomlife.Report, path string) {
	fmt.Printf("Geometry arena lifecycle simulation: ok=%t\n", report.OK)

	fmt.Println("\nChecks:")
	for _, check := range report.Canonical.Checks {
		mark := "PASS"
		if !check.OK {
			mark = "FAIL"
		}
		if check.Detail == "" {
			fmt.Printf("  [%s] %s\n", mark, check.Name)
		} else {
			fmt.Printf("  [%s] %s (%s)\n", mark, check.Name, check.Detail)
		}
	}
	fmt.Printf("  [%s] parallel_double_frame (%d frames)\n",
		passMark(report.Parallel.OK), len(report.Parallel.Frames))
	fmt.Printf("  [%s] geometry_canary (unsafe_corruption=%t safe_intact=%t active_release_blocked=%t)\n",
		passMark(report.Canary.OK),
		report.Canary.UnsafeDrawCorrupted,
		report.Canary.SafeDrawIntact,
		report.Canary.ActiveReleaseBlocked)

	fmt.Println("\nObjects:")
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "SEQ\tOBJECT_STATE\tLIFECYCLE\tOFFSET\tSIZE\tLAST_SUB\tFRAME\tRELEASE_REASON\tDUP\tSTALE\tHELD_BLOCKS")
	for _, object := range report.Canonical.Snapshot.Objects {
		fmt.Fprintf(writer, "%d\t%s\t%s\t%d\t%d\t%d\t%d\t%s\t%d\t%d\t%d\n",
			object.AllocationSeq,
			object.State,
			object.LifecycleState,
			object.Offset,
			object.Size,
			object.LastUsedSubmissionID,
			object.LastUsedFrameID,
			object.ReleaseReason,
			object.DuplicateReleaseAttempts,
			object.StaleReleaseAttempts,
			object.BlockedHeldReleaseAttempts,
		)
	}
	_ = writer.Flush()

	fmt.Printf("\nEvents recorded: %d\n", len(report.Canonical.Snapshot.Events))
	if path != "" {
		fmt.Printf("JSON report: %s\n", path)
	}
}

func passMark(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
