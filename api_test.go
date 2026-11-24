package main

import "testing"

func TestCompareSearchesConvenienceAPIs(t *testing.T) {
	field := testPlaneField()
	ray := testPlaneRay(false)

	results := CompareSearches(field, ray, 8, 10, 40, 1e-9)
	if len(results) != 3 {
		t.Fatalf("expected three results, got %d", len(results))
	}
	if results[0].Mode != ModeFixed || results[1].Mode != ModeBinary || results[2].Mode != ModeMaxIteration {
		t.Fatalf("unexpected mode order: %q, %q, %q", results[0].Mode, results[1].Mode, results[2].Mode)
	}

	binary := BinarySearch(field, ray, 8, 10)
	if !binary.Hit || binary.BinaryIterations != 10 {
		t.Fatalf("binary alias returned unexpected result: %+v", binary)
	}

	maxResult := MaximumIterationSearch(field, ray, 8, 40, 1e-9)
	maxResult.ScoreAgainst(0.6)
	if !maxResult.Hit || maxResult.DepthError > 1e-8 {
		t.Fatalf("max alias returned unexpected result: %+v", maxResult)
	}
}
