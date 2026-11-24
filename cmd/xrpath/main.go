package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"xrpath"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "record":
			recordCommand(os.Args[2:])
			return
		case "compare":
			compareCommand(os.Args[2:])
			return
		case "fault":
			faultCommand(os.Args[2:])
			return
		case "demo":
			demoCommand(os.Args[2:])
			return
		case "-h", "--help", "help":
			printUsage()
			return
		}
	}
	demoCommand(os.Args[1:])
}

func demoCommand(args []string) {
	flags := flag.NewFlagSet("demo", flag.ExitOnError)
	out := flags.String("out", "artifacts/xrpath", "report/artifact directory")
	_ = flags.Parse(args)

	result, err := xrpath.RunDemo(*out)
	if err != nil {
		fatal(err)
	}
	printDemo(result)
	if !result.OK {
		os.Exit(2)
	}
}

func recordCommand(args []string) {
	flags := flag.NewFlagSet("record", flag.ExitOnError)
	out := flags.String("out", "artifacts/xrpath/run", "directory for this run")
	_ = flags.Parse(args)

	run := xrpath.Simulate()
	if err := xrpath.SaveRun(*out, run); err != nil {
		fatal(err)
	}
	fmt.Printf("recorded %d frames to %s\n", len(run.Records), *out)
}

func compareCommand(args []string) {
	flags := flag.NewFlagSet("compare", flag.ExitOnError)
	expected := flags.String("a", "artifacts/xrpath/baseline", "first/expected run directory")
	actual := flags.String("b", "artifacts/xrpath/replay", "second/actual run directory")
	reportPath := flags.String("report", "", "optional JSON report output path")
	_ = flags.Parse(args)

	report, stored, err := xrpath.CompareDirectories(*expected, *actual)
	if err != nil {
		fatal(err)
	}
	if *reportPath != "" {
		if err := xrpath.WriteJSON(*reportPath, map[string]any{
			"expected_dir": *expected, "actual_dir": *actual,
			"report": report, "stored_image_mismatches": stored,
		}); err != nil {
			fatal(err)
		}
	}

	fmt.Printf("compared %d/%d frames: checks=%d mismatches=%d image_file_issues=%d\n",
		report.FrameCountExpected, report.FrameCountActual, report.CheckCount, report.MismatchCount, len(stored))
	if report.FirstDivergence != nil {
		d := report.FirstDivergence
		fmt.Printf("first divergence: frame=%d subsystem=%s item=%s field=%s\n", d.Frame, d.Subsystem, d.Item, d.Field)
	}
	if !report.OK || len(stored) > 0 {
		os.Exit(2)
	}
}

func faultCommand(args []string) {
	flags := flag.NewFlagSet("fault", flag.ExitOnError)
	out := flags.String("out", "artifacts/xrpath/manual-fault-report.json", "JSON report path")
	kind := flags.String("kind", xrpath.FaultPoseDrift, "fault kind")
	frame := flags.Int("at", xrpath.FaultPoseFrame, "injection frame")
	target := flags.String("target", "", "optional target")
	_ = flags.Parse(args)

	baseline := xrpath.Simulate()
	fault := xrpath.Fault{Frame: *frame, Kind: *kind, Target: *target}
	report := xrpath.CompareRuns(baseline, xrpath.Simulate(fault))
	if err := xrpath.WriteJSON(*out, map[string]any{
		"fault": fault, "expected_subsystem": xrpath.ExpectedSubsystemForFault(*kind), "report": report,
	}); err != nil {
		fatal(err)
	}
	printFault(fault, report, *out)
	if report.OK {
		os.Exit(2)
	}
}

func printDemo(result *xrpath.DemoResult) {
	fmt.Println("XR fixed-path verification complete")
	fmt.Printf("  output: %s\n", result.OutputDir)
	fmt.Printf("  frames: %d\n", result.Report.FrameCountExpected)
	fmt.Printf("  checks: %d\n", result.Report.CheckCount)
	if result.Report.FirstDivergence != nil {
		d := result.Report.FirstDivergence
		fmt.Printf("  first divergence: frame=%d subsystem=%s item=%s field=%s\n", d.Frame, d.Subsystem, d.Item, d.Field)
	} else {
		fmt.Println("  first divergence: none")
	}
	fmt.Printf("  fault drills: ok=%v (%d injections)\n", result.FaultArtifact.FaultSuite.OK, len(result.FaultArtifact.FaultSuite.Drills))
	fmt.Printf("  stored image mismatches: %d\n", len(result.StoredImageMismatches))
}

func printFault(fault xrpath.Fault, report xrpath.Report, path string) {
	fmt.Printf("Manual fault %s at frame %d\n", fault.Kind, fault.Frame)
	if report.FirstDivergence != nil {
		d := report.FirstDivergence
		fmt.Printf("  first divergence: frame=%d subsystem=%s item=%s field=%s\n", d.Frame, d.Subsystem, d.Item, d.Field)
	} else {
		fmt.Println("  no divergence detected")
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	fmt.Printf("  report: %s\n", path)
}

func printUsage() {
	fmt.Println(`xrpath commands:
  xrpath [demo] [-out artifacts/xrpath]
  xrpath record -out <dir>
  xrpath compare -a <expected-dir> -b <actual-dir> [-report result.json]
  xrpath fault -kind <kind> -at <frame> [-target id] [-out report.json]`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "xrpath:", err)
	os.Exit(1)
}
