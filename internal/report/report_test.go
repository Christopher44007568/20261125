package report

import (
	"bytes"
	"strings"
	"testing"

	"eyediag/internal/model"
	"eyediag/internal/synth"
)

func TestBuildAggregationAndAsymmetry(t *testing.T) {
	c, _ := synth.Generate(synth.ScenOneEye, 1)
	r := Build(c, model.Defaults())

	if r.CorrectEyes != 4 || r.FaultyEyes != 4 {
		t.Fatalf("eye counts = %d/%d, want 4/4", r.CorrectEyes, r.FaultyEyes)
	}
	if !r.Asymmetric {
		t.Fatal("one-eye scenario must be flagged asymmetric")
	}
	if len(r.CauseStats) != 4 {
		t.Fatalf("cause stats = %d, want 4", len(r.CauseStats))
	}
	for _, s := range r.CauseStats {
		if s.EyeHits[model.Left] != 0 || s.EyeHits[model.Right] != 1 {
			t.Fatalf("cause %s eye hits = %v", s.Cause, s.EyeHits)
		}
	}
}

func TestTextReportCarriesPixelDiffCaveat(t *testing.T) {
	c, _ := synth.Generate(synth.ScenEqualDiffDiffCau, 1)
	r := Build(c, model.Defaults())
	var buf bytes.Buffer
	r.WriteText(&buf)
	out := buf.String()
	if !strings.Contains(out, "总像素差") && !strings.Contains(out, "像素差") {
		t.Fatal("text report missing pixel-diff context")
	}
	if !strings.Contains(out, "不参与任何因果判定") {
		t.Fatal("text report missing pixel-diff non-causality caveat")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	c, _ := synth.Generate(synth.ScenBudget, 1)
	r := Build(c, model.Defaults())
	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"degradedHits"`) {
		t.Fatal("JSON report missing degradedHits field")
	}
}
