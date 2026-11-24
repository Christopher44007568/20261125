package eyebw

import "testing"

func TestDynamicFoveationNeverDowngradesCenter(t *testing.T) {
	cfg := testConfig()
	cfg.Width = 720
	cfg.Height = 720
	cfg.MSAASamples = 4
	report := runTestFrames(t, cfg, 10)

	for _, frame := range report.FrameReports {
		for _, id := range AllPipelines() {
			p := frame.Get(id)
			if !p.Quality.Passed {
				t.Fatalf("frame %d pipeline %s quality failed: %v", frame.Index, id, p.Quality.Violations)
			}
			if p.Quality.MinDivisor != 1 {
				t.Fatalf("%s frame %d min center divisor=%d, want 1", id, frame.Index, p.Quality.MinDivisor)
			}
			if p.Quality.MinMSAASamples != cfg.MSAASamples {
				t.Fatalf("%s frame %d MSAA=%d, want %d", id, frame.Index, p.Quality.MinMSAASamples, cfg.MSAASamples)
			}
			for _, eye := range p.FoveationByEye {
				if eye.CenterEffective != eye.CenterFullPixels {
					t.Fatalf("%s %s center effective=%d full=%d", id, eye.Eye, eye.CenterEffective, eye.CenterFullPixels)
				}
			}
			for _, obs := range p.Quality.Observations {
				if !obs.IndependentlyRendered || obs.Divisor != 1 || obs.MSAASamples != cfg.MSAASamples {
					t.Fatalf("unsafe center observation for %s frame %d: %+v", id, frame.Index, obs)
				}
			}
		}
	}

	separate := report.Pipelines[0]
	if totalFoveationRepaint(separate) == 0 {
		t.Fatal("dynamic gaze should promote tiles and record foveation repaint bytes")
	}
}

func totalFoveationRepaint(s PipelineSequenceSummary) uint64 {
	for _, class := range s.Classes {
		if DataClass(class.Name) == ClassRepaint {
			return class.TransferTotal()
		}
	}
	return 0
}

func TestAggressiveCenterReuseIsQualityFailureNotSilentSaving(t *testing.T) {
	cfg := testConfig()
	cfg.SingleEye.AggressiveCenterReuse = true
	report := runTestFrames(t, cfg, 3)

	var baseline, aggressive *PipelineReport
	for _, p := range report.FrameReports[2].Pipelines {
		switch p.ID {
		case PipelineSeparateEyes:
			baseline = p
		case PipelineSingleReuse:
			aggressive = p
		}
	}
	if aggressive.Quality.Passed {
		t.Fatal("aggressive center reuse must fail the center quality gate")
	}
	if aggressive.TransferTotal() >= baseline.TransferTotal() {
		t.Fatalf("test setup should produce lower bytes: aggressive=%d baseline=%d",
			aggressive.TransferTotal(), baseline.TransferTotal())
	}
	err := RejectSilentBandwidthSaving(PipelineQualityBandwidth{
		Pipeline:      PipelineSeparateEyes,
		TransferBytes: baseline.TransferTotal(),
		QualityPassed: true,
	}, PipelineQualityBandwidth{
		Pipeline:      PipelineSingleReuse,
		TransferBytes: aggressive.TransferTotal(),
		QualityPassed: aggressive.Quality.Passed,
		Violations:    aggressive.Quality.Violations,
	})
	if err == nil {
		t.Fatal("expected silent quality regression to be rejected")
	}
}

func oneFrameTransfer(t *testing.T, width, height, msaa int, foveated bool) uint64 {
	t.Helper()
	cfg := testConfig()
	cfg.Width = width
	cfg.Height = height
	cfg.MSAASamples = msaa
	cfg.Foveation.Enabled = foveated
	sim, err := NewSimulator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := sim.Step(FrameInput{Index: 0, Gazes: SweepGaze(0, 4)})
	if err != nil {
		t.Fatal(err)
	}
	return frame.Get(PipelineSeparateEyes).TransferTotal()
}

func TestHighResolutionAndMSAAIncreaseBandwidth(t *testing.T) {
	lowMSAA := oneFrameTransfer(t, 512, 512, 1, false)
	highMSAA := oneFrameTransfer(t, 512, 512, 8, false)
	if highMSAA <= lowMSAA {
		t.Fatalf("MSAA8 bytes %d should exceed MSAA1 bytes %d", highMSAA, lowMSAA)
	}

	lowRes := oneFrameTransfer(t, 256, 256, 1, false)
	highRes := oneFrameTransfer(t, 1024, 1024, 1, false)
	if highRes <= lowRes*4 {
		t.Fatalf("1024² bytes %d should materially exceed 256² bytes %d", highRes, lowRes)
	}

	fullRate := oneFrameTransfer(t, 768, 768, 4, false)
	foveated := oneFrameTransfer(t, 768, 768, 4, true)
	if foveated >= fullRate {
		t.Fatalf("foveated bytes %d should be below full-rate bytes %d", foveated, fullRate)
	}
}

func TestBudgetOverageIsRecorded(t *testing.T) {
	cfg := testConfig()
	cfg.Width = 768
	cfg.Height = 768
	cfg.MSAASamples = 8
	cfg.Foveation.Enabled = false
	cfg.Budget.FrameBytes = 4 * KibiByte
	sim, err := NewSimulator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := sim.Step(FrameInput{Index: 0})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range frame.Pipelines {
		if !p.Budget.Exceeded || p.Budget.OverageBytes == 0 {
			t.Fatalf("pipeline %s should exceed tiny budget: %+v", p.ID, p.Budget)
		}
		if p.Budget.ActualBytes != p.TransferTotal() {
			t.Fatalf("budget actual %d != transfer total %d", p.Budget.ActualBytes, p.TransferTotal())
		}
	}
}
