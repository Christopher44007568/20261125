package main

import "testing"

func TestBuildReportSummaryAndInvariants(t *testing.T) {
	r := BuildReport(DefaultConfig())
	s := r.Summary
	if s.ScenarioCount != 9 || s.EpisodeCount != 10 || s.ChainCount != 7 {
		t.Fatalf("counts=%d/%d/%d want 9/10/7", s.ScenarioCount, s.EpisodeCount, s.ChainCount)
	}
	if len(s.FalsePositiveAlerts) != 1 || len(s.FalsePositiveEpisodes) != 1 {
		t.Fatalf("fp alert=%d ep=%d", len(s.FalsePositiveAlerts), len(s.FalsePositiveEpisodes))
	}
	if len(s.ContinuousChains) != 2 || len(s.PartialRecoveries) != 2 ||
		len(s.IsolatedRecoveredChains) != 3 {
		t.Fatalf("classification counts cont=%d partial=%d isolated=%d",
			len(s.ContinuousChains), len(s.PartialRecoveries), len(s.IsolatedRecoveredChains))
	}
	if s.SafetyEvents != 2 {
		t.Fatalf("safety events=%d want 2", s.SafetyEvents)
	}
	if len(s.InvariantChecks) != 5 {
		t.Fatalf("invariant checks=%d", len(s.InvariantChecks))
	}
	for _, ic := range s.InvariantChecks {
		if !ic.Passed {
			t.Fatalf("invariant %s failed: %s", ic.ID, ic.Evidence)
		}
	}
}

func TestEyeFloorNeverZero(t *testing.T) {
	for _, def := range AllScenarios() {
		c, frames := ExecuteScenario(def, DefaultConfig())
		for _, f := range frames {
			if f.State.EyeCount < 1 {
				t.Fatalf("%s frame %d has %d eyes", def.ID, f.Frame, f.State.EyeCount)
			}
			if !f.State.Interaction || !f.State.Overlay {
				t.Fatalf("%s frame %d lost interaction/overlay", def.ID, f.Frame)
			}
		}
		for _, ev := range c.SafetyEvents {
			if ev.Code != "eyes_zero_rejected" && ev.Code != "history_silent_disable_rejected" {
				t.Fatalf("unexpected safety code %s", ev.Code)
			}
		}
	}
}

func TestSevereFrameDegradesImmediately(t *testing.T) {
	c := NewController(DefaultConfig())
	f := c.Step(Signals{FrameTimeMS: 20})
	if f.State.SampleLevel != 1 || len(c.Alerts) != 1 || c.Alerts[0].Verdict != "confirmed" {
		t.Fatalf("severe frame should confirm+downshift immediately: %+v alert=%+v",
			f.State, c.Alerts)
	}
}
