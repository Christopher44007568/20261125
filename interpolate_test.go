package main

import (
	"math"
	"testing"
)

func TestScenariosHaveDifferentStrandCounts(t *testing.T) {
	scenarios, err := buildScenarios()
	if err != nil {
		t.Fatal(err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("got %d scenarios", len(scenarios))
	}
	for _, sc := range scenarios {
		if len(sc.A.Strands) == len(sc.B.Strands) {
			t.Fatalf("%s should exercise unequal strand counts", sc.ID)
		}
		if err := sc.A.Validate(); err != nil {
			t.Fatal(err)
		}
		if err := sc.B.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEndpointFramesAreExact(t *testing.T) {
	scenarios, err := buildScenarios()
	if err != nil {
		t.Fatal(err)
	}
	for _, sc := range scenarios {
		it, err := NewInterpolator(sc.A, sc.B)
		if err != nil {
			t.Fatal(err)
		}
		testEndpoint(t, sc.ID, "A", it.Frame(0), sc.A)
		testEndpoint(t, sc.ID, "B", it.Frame(1), sc.B)
	}
}

func testEndpoint(t *testing.T, id, side string, frame Frame, source Groom) {
	t.Helper()
	if len(frame.Strands) != len(source.Strands) {
		t.Fatalf("%s endpoint %s: got %d strands, want %d", id, side, len(frame.Strands), len(source.Strands))
	}
	seen := make([]bool, len(source.Strands))
	for i, strand := range frame.Strands {
		link := frame.Links[i]
		ref := link.A
		if side == "B" {
			ref = link.B
		}
		if math.Abs(ref.Weights[0]-1) > 1e-9 || math.Abs(ref.Weights[1]) > 1e-9 {
			t.Fatalf("%s endpoint %s: output %d is not an exact source strand: %+v", id, side, i, ref)
		}
		index := ref.Indices[0]
		if index < 0 || index >= len(seen) || seen[index] {
			t.Fatalf("%s endpoint %s: duplicate or invalid source index %d", id, side, index)
		}
		seen[index] = true
		want := source.Strands[index]
		if strand.Width != want.Width {
			t.Fatalf("%s endpoint %s: width %g != %g", id, side, strand.Width, want.Width)
		}
		for level := range want.Controls {
			if got := strand.Controls[level].Dist(want.Controls[level]); got > 1e-9 {
				t.Fatalf("%s endpoint %s: level %d error %g", id, side, level, got)
			}
		}
	}
	for i, ok := range seen {
		if !ok {
			t.Fatalf("%s endpoint %s: source strand %d was not represented", id, side, i)
		}
	}
	wantPose := source.Pose
	gotPose := frame.Pose
	if math.Abs(gotPose.Yaw-wantPose.Yaw) > 1e-12 ||
		math.Abs(gotPose.Pitch-wantPose.Pitch) > 1e-12 ||
		math.Abs(gotPose.Roll-wantPose.Roll) > 1e-12 ||
		gotPose.Translate.Dist(wantPose.Translate) > 1e-12 {
		t.Fatalf("%s endpoint %s pose mismatch", id, side)
	}
}

func TestNoDivergentTipsOrControlSpikes(t *testing.T) {
	scenarios, err := buildScenarios()
	if err != nil {
		t.Fatal(err)
	}
	for _, sc := range scenarios {
		it, err := NewInterpolator(sc.A, sc.B)
		if err != nil {
			t.Fatal(err)
		}
		envelope := math.Max(maxTipRadius(sc.A), maxTipRadius(sc.B))
		for i := 0; i <= 128; i++ {
			tf := float64(i) / 128
			frame := it.Frame(tf)
			m := frameMetrics(frame, envelope)
			if !m.Finite {
				t.Fatalf("%s t=%.4f: non-finite output", sc.ID, tf)
			}
			if m.MaxTipOverBound > 1e-8 || m.TipBoundViolations != 0 {
				t.Fatalf("%s t=%.4f: tip bound exceeded by %g (%d violations)",
					sc.ID, tf, m.MaxTipOverBound, m.TipBoundViolations)
			}
			if m.MaxControlOverBound > 1e-8 || m.ControlViolations != 0 {
				t.Fatalf("%s t=%.4f: control bound exceeded by %g (%d violations)",
					sc.ID, tf, m.MaxControlOverBound, m.ControlViolations)
			}
			if m.MaxSpanOverBound > 1e-8 || m.SpanViolations != 0 {
				t.Fatalf("%s t=%.4f: segment bound exceeded by %g (%d violations)",
					sc.ID, tf, m.MaxSpanOverBound, m.SpanViolations)
			}
			if m.MaxTipRadius > 1+1e-8 {
				t.Fatalf("%s t=%.4f: tip escaped endpoint envelope: %g", sc.ID, tf, m.MaxTipRadius)
			}
		}
	}
}

func TestPairingIsDeterministicAndMassPreserving(t *testing.T) {
	scenarios, err := buildScenarios()
	if err != nil {
		t.Fatal(err)
	}
	for _, sc := range scenarios {
		it, err := NewInterpolator(sc.A, sc.B)
		if err != nil {
			t.Fatal(err)
		}
		first := analyzeDeterministicHash(t, sc)
		second := analyzeDeterministicHash(t, sc)
		if first != second {
			t.Fatalf("%s: non-deterministic interpolation hash", sc.ID)
		}
		for i := 0; i <= 64; i++ {
			frame := it.Frame(float64(i) / 64)
			if len(frame.Links) != len(frame.Strands) {
				t.Fatalf("%s: link count mismatch", sc.ID)
			}
			for j, link := range frame.Links {
				if len(link.MigrationMM) != sc.A.Levels {
					t.Fatalf("%s: link %d migration levels %d", sc.ID, j, len(link.MigrationMM))
				}
				checkRef(t, sc.ID, "A", link.A, len(sc.A.Strands), frame)
				checkRef(t, sc.ID, "B", link.B, len(sc.B.Strands), frame)
				sum := link.A.Weights[0] + link.A.Weights[1]
				if math.Abs(sum-1) > 1e-12 {
					t.Fatalf("%s: A weights sum %g", sc.ID, sum)
				}
				sum = link.B.Weights[0] + link.B.Weights[1]
				if math.Abs(sum-1) > 1e-12 {
					t.Fatalf("%s: B weights sum %g", sc.ID, sum)
				}
			}
		}
	}
}

func checkRef(t *testing.T, id, side string, ref StrandRef, n int, frame Frame) {
	t.Helper()
	if ref.Indices[0] < 0 || ref.Indices[0] >= n || ref.Indices[1] < 0 || ref.Indices[1] >= n {
		t.Fatalf("%s %s ref out of range: %+v", id, side, ref)
	}
	if len(frame.Links) == 0 {
		t.Fatal("empty links")
	}
}

func analyzeDeterministicHash(t *testing.T, sc Scenario) string {
	t.Helper()
	run, err := analyzeScenario(sc)
	if err != nil {
		t.Fatal(err)
	}
	return run.Report.DeterminismHash
}

func TestScenarioReportsPassAcceptance(t *testing.T) {
	scenarios, err := buildScenarios()
	if err != nil {
		t.Fatal(err)
	}
	for _, sc := range scenarios {
		run, err := analyzeScenario(sc)
		if err != nil {
			t.Fatal(err)
		}
		if run.Report.StrandsA == run.Report.StrandsB {
			t.Fatalf("%s does not test unequal counts", sc.ID)
		}
		if run.Report.MaxTipOverBound > 1e-8 ||
			run.Report.MaxControlOverBound > 1e-8 ||
			run.Report.MaxSpanOverBound > 1e-8 ||
			run.Report.TipEnvelopeRatio > 1+1e-8 {
			t.Fatalf("%s failed acceptance: %+v", sc.ID, run.Report)
		}
		if run.Report.DeterminismHash == "" {
			t.Fatalf("%s has no determinism hash", sc.ID)
		}
	}
}
