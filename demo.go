package xrpath

import (
	"fmt"
	"path/filepath"
)

type DemoResult struct {
	OutputDir             string        `json:"output_dir"`
	Report                Report        `json:"report"`
	StoredImageMismatches []Mismatch    `json:"stored_image_mismatches"`
	FaultArtifact         FaultArtifact `json:"fault_artifact"`
	OK                    bool          `json:"ok"`
}

func RunDemo(out string) (*DemoResult, error) {
	baseline := Simulate()
	baselineDir := filepath.Join(out, "baseline")
	replayDir := filepath.Join(out, "replay")
	if err := SaveRun(baselineDir, baseline); err != nil {
		return nil, fmt.Errorf("save baseline: %w", err)
	}

	rerun := Simulate()
	if err := SaveRun(replayDir, rerun); err != nil {
		return nil, fmt.Errorf("save replay: %w", err)
	}

	loadedBaseline, err := LoadRecords(baselineDir)
	if err != nil {
		return nil, err
	}
	loadedReplay, err := LoadRecords(replayDir)
	if err != nil {
		return nil, err
	}
	report := CompareRecords(loadedBaseline, loadedReplay)
	rawReport := CompareRuns(baseline, rerun)
	if !rawReport.OK {
		report.OK = false
		report.Mismatches = append(report.Mismatches, rawReport.Mismatches...)
		sortMismatches(report.Mismatches)
		report.MismatchCount = len(report.Mismatches)
		report.FirstDivergence = &report.Mismatches[0]
	}

	storedMismatches := VerifyStoredImages(baselineDir, loadedBaseline)
	storedMismatches = append(storedMismatches, VerifyStoredImages(replayDir, loadedReplay)...)

	suite := RunFaultDrills(baseline)
	hashDrills, err := RunHashBlindDrills(baseline.Records)
	if err != nil {
		return nil, err
	}
	faultArtifact := FaultArtifact{
		OK: suite.OK && hashDrillsOK(hashDrills), FaultSuite: suite, HashBlindDrills: hashDrills,
	}

	if err := WriteJSON(filepath.Join(out, CompareJSON), report); err != nil {
		return nil, err
	}
	if err := WriteJSON(filepath.Join(out, FaultJSON), faultArtifact); err != nil {
		return nil, err
	}
	markdown := RenderMarkdown(report, storedMismatches, suite, hashDrills)
	if err := WriteMarkdown(filepath.Join(out, CompareMD), []byte(markdown)); err != nil {
		return nil, err
	}

	result := &DemoResult{
		OutputDir: out, Report: report, StoredImageMismatches: storedMismatches,
		FaultArtifact: faultArtifact,
		OK:            report.OK && len(storedMismatches) == 0 && faultArtifact.OK,
	}
	return result, nil
}

func CompareDirectories(expectedDir, actualDir string) (Report, []Mismatch, error) {
	expected, err := LoadRecords(expectedDir)
	if err != nil {
		return NewReport(), nil, fmt.Errorf("load expected: %w", err)
	}
	actual, err := LoadRecords(actualDir)
	if err != nil {
		return NewReport(), nil, fmt.Errorf("load actual: %w", err)
	}
	report := CompareRecords(expected, actual)
	stored := VerifyStoredImages(expectedDir, expected)
	stored = append(stored, VerifyStoredImages(actualDir, actual)...)
	return report, stored, nil
}
