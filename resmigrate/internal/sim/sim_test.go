package sim

import "testing"

func TestAllScenariosPassInvariants(t *testing.T) {
	for _, spec := range Scenarios() {
		r := Run(spec)
		if !r.OK {
			for _, c := range r.Checks {
				if !c.OK {
					t.Errorf("%s: check %s failed: %s", spec.ID, c.ID, c.Detail)
				}
			}
		}
	}
}

func TestNoPresentedFrameHasStereoMismatch(t *testing.T) {
	for _, spec := range Scenarios() {
		r := Run(spec)
		for _, f := range r.Frames {
			if !f.Left.Size.Equals(f.Right.Size) {
				t.Fatalf("%s frame %d presents mismatched eye sizes %s vs %s",
					spec.ID, f.Index, f.Left.Size, f.Right.Size)
			}
		}
	}
}

func TestRollbackRestoresOldSizeForBothEyes(t *testing.T) {
	for _, spec := range Scenarios() {
		if spec.Fault == nil || spec.RetryAllowed {
			continue
		}
		r := Run(spec)
		if r.Outcome != OutcomeRolledBack {
			t.Fatalf("%s: expected rolled_back, got %s", spec.ID, r.Outcome)
		}
		if !r.FinalSize.Equals(spec.Initial) {
			t.Fatalf("%s: final %s != initial %s", spec.ID, r.FinalSize, spec.Initial)
		}
		if !r.Eyes[0].FinalSize.Equals(r.Eyes[1].FinalSize) {
			t.Fatalf("%s: permanent stereo mismatch %s vs %s",
				spec.ID, r.Eyes[0].FinalSize, r.Eyes[1].FinalSize)
		}
	}
}

func TestBudgetModes(t *testing.T) {
	specs := map[string]func(r *ScenarioReport) (int, int){
		"budget_fixed_spp":   func(r *ScenarioReport) (int, int) { return r.Frames[0].Left.SPP, r.Frames[len(r.Frames)-1].Left.SPP },
		"budget_fixed_total": func(r *ScenarioReport) (int, int) { return r.Frames[0].Left.SPP, r.Frames[len(r.Frames)-1].Left.SPP },
	}
	for _, spec := range Scenarios() {
		fn, ok := specs[spec.ID]
		if !ok {
			continue
		}
		r := Run(spec)
		firstSPP, lastSPP := fn(r)
		switch spec.ID {
		case "budget_fixed_spp":
			if firstSPP != lastSPP {
				t.Fatalf("fixed_spp: SPP changed %d -> %d", firstSPP, lastSPP)
			}
		case "budget_fixed_total":
			if lastSPP >= firstSPP {
				t.Fatalf("fixed_total: SPP did not shrink %d -> %d", firstSPP, lastSPP)
			}
		}
	}
}

func TestDiscardFullyInvalidatesHistory(t *testing.T) {
	for _, spec := range Scenarios() {
		if !spec.Discard {
			continue
		}
		r := Run(spec)
		if r.Invalidation.WorstRatio != 1 {
			t.Fatalf("%s: discard invalidation %f, want 1", spec.ID, r.Invalidation.WorstRatio)
		}
	}
}
