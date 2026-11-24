package classifier

import (
	"testing"

	"eyediag/internal/model"
	"eyediag/internal/synth"
)

func causesFor(t *testing.T, v MarkerVerdict, eye model.Eye) map[model.Cause]bool {
	t.Helper()
	ev, ok := v.Eyes[eye]
	if !ok {
		t.Fatalf("missing eye %s verdict", eye)
	}
	out := map[model.Cause]bool{}
	for _, l := range ev.Labels {
		out[l.Cause] = true
	}
	return out
}

func verdictFor(t *testing.T, c model.Case, id string) MarkerVerdict {
	t.Helper()
	for _, m := range c.Markers {
		if m.ID == id {
			return Classify(m, model.Defaults())
		}
	}
	t.Fatalf("marker %s not found", id)
	return MarkerVerdict{}
}

// Each single-cause scenario must fire exactly its intended cause on both
// eyes, with no collateral labels.
func TestSingleCauseIsolation(t *testing.T) {
	cases := map[string]model.Cause{
		synth.ScenSorting:  model.Sorting,
		synth.ScenParallax: model.Parallax,
		synth.ScenAlpha:    model.Alpha,
		synth.ScenAA:       model.AA,
	}
	for scen, want := range cases {
		t.Run(scen, func(t *testing.T) {
			c, err := synth.Generate(scen, 1)
			if err != nil {
				t.Fatal(err)
			}
			for _, m := range c.Markers {
				v := Classify(m, model.Defaults())
				for _, eye := range []model.Eye{model.Left, model.Right} {
					got := causesFor(t, v, eye)
					if len(got) != 1 || !got[want] {
						t.Fatalf("%s/%s = %v, want only %s", m.ID, eye, got, want)
					}
				}
			}
		})
	}
}

func TestCleanHasNoLabels(t *testing.T) {
	c, _ := synth.Generate(synth.ScenClean, 1)
	for _, m := range c.Markers {
		v := Classify(m, model.Defaults())
		for _, eye := range []model.Eye{model.Left, model.Right} {
			ev := v.Eyes[eye]
			if !ev.Correct || len(ev.Labels) != 0 {
				t.Fatalf("clean marker %s/%s fired %v", m.ID, eye, ev.Labels)
			}
		}
	}
}

func TestMultiCauseFiresMultipleLabels(t *testing.T) {
	c, _ := synth.Generate(synth.ScenMulti, 1)

	sortAlpha := verdictFor(t, c, "multi-sort-alpha")
	for _, eye := range []model.Eye{model.Left, model.Right} {
		got := causesFor(t, sortAlpha, eye)
		if !got[model.Sorting] || !got[model.Alpha] {
			t.Fatalf("sort+alpha marker/%s = %v", eye, got)
		}
	}

	parAA := verdictFor(t, c, "multi-parallax-aa")
	for _, eye := range []model.Eye{model.Left, model.Right} {
		got := causesFor(t, parAA, eye)
		if !got[model.Parallax] || !got[model.AA] {
			t.Fatalf("parallax+aa marker/%s = %v", eye, got)
		}
	}

	wantStages := map[model.Stage]bool{
		model.VertexStage: true,
		model.LayerStage:  true,
		model.SampleStage: true,
	}
	for _, v := range []MarkerVerdict{sortAlpha, parAA} {
		for _, s := range v.Stages {
			if !wantStages[s] {
				t.Fatalf("unexpected stage %s", s)
			}
		}
	}
}

func TestOneEyeCorrectOtherFaulty(t *testing.T) {
	c, _ := synth.Generate(synth.ScenOneEye, 1)
	for _, m := range c.Markers {
		v := Classify(m, model.Defaults())
		if !v.Eyes[model.Left].Correct || v.Eyes[model.Right].Correct {
			t.Fatalf("%s: want left correct / right faulty, got %+v", m.ID, v.Eyes)
		}
	}
}

func TestBudgetDegradationAnnotatesButDoesNotInvent(t *testing.T) {
	c, _ := synth.Generate(synth.ScenBudget, 1)
	for _, m := range c.Markers {
		v := Classify(m, model.Defaults())
		for _, eye := range []model.Eye{model.Left, model.Right} {
			ev := v.Eyes[eye]
			if len(ev.Labels) == 0 {
				t.Fatalf("%s/%s expected evidence-based labels", m.ID, eye)
			}
			if len(ev.DegradedBy) == 0 {
				t.Fatalf("%s/%s missing degradation factors", m.ID, eye)
			}
		}
	}

	// A degraded but actually-correct frame must stay label-free:
	// degradation context never fabricates a causal label.
	clean, _ := synth.Generate(synth.ScenClean, 1)
	m := clean.Markers[0]
	m.Budget = synth.DegradedBudgetForTest()
	v := Classify(m, model.Defaults())
	for _, eye := range []model.Eye{model.Left, model.Right} {
		if !v.Eyes[eye].Correct || len(v.Eyes[eye].DegradedBy) != 0 {
			t.Fatalf("degraded clean marker/%s = %+v", eye, v.Eyes[eye])
		}
	}
}

// The core anti-pattern: identical total pixel difference must still yield
// different causal labels, proving pixel diff cannot replace classification.
func TestEqualPixelDiffDifferentCauses(t *testing.T) {
	c, _ := synth.Generate(synth.ScenEqualDiffDiffCau, 1)
	sortV := verdictFor(t, c, "eqdiff-sorting")
	parV := verdictFor(t, c, "eqdiff-parallax")

	diffL := sortV.Eyes[model.Left].TotalPixelDiff
	diffR := parV.Eyes[model.Left].TotalPixelDiff
	if abs(diffL-diffR) > 1e-9 {
		t.Fatalf("pixel diffs differ: %.6f vs %.6f", diffL, diffR)
	}
	if !causesFor(t, sortV, model.Left)[model.Sorting] {
		t.Fatalf("eqdiff-sorting misclassified: %v", sortV.Eyes[model.Left].Labels)
	}
	if !causesFor(t, parV, model.Left)[model.Parallax] {
		t.Fatalf("eqdiff-parallax misclassified: %v", parV.Eyes[model.Left].Labels)
	}
}

func TestStageMapping(t *testing.T) {
	want := map[model.Cause]model.Stage{
		model.Sorting:  model.LayerStage,
		model.Parallax: model.VertexStage,
		model.Alpha:    model.LayerStage,
		model.AA:       model.SampleStage,
	}
	for cause, stage := range want {
		if got := stageFor(cause); got != stage {
			t.Fatalf("stageFor(%s) = %s, want %s", cause, got, stage)
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
