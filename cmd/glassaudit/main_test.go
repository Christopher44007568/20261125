package main

import (
	"testing"

	"glassaudit/glass"
)

func TestBuildSuiteIsStableAndComplete(t *testing.T) {
	report, err := buildSuite(glass.DefaultBudget(), glass.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.StableAcrossScenes {
		t.Fatal("suite is not stable across all submissions")
	}
	if len(report.Scenarios) != 3 {
		t.Fatalf("scenario count = %d, want 3", len(report.Scenarios))
	}
	for _, scenario := range report.Scenarios {
		if !scenario.StableAcrossSubmissions {
			t.Fatalf("%s is not stable", scenario.Kind)
		}
		if scenario.SubmissionOrdersTested != 120 {
			t.Fatalf("%s tested %d submissions, want 120", scenario.Kind, scenario.SubmissionOrdersTested)
		}
		if err := scenario.Audit.ValidateEvidence(); err != nil {
			t.Fatalf("%s: %v", scenario.Kind, err)
		}
	}
}
