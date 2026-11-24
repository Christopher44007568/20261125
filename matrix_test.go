package eyebw

import (
	"fmt"
	"sync"
	"testing"
)

func TestFullResolutionMSAAFoveationMatrixPassesQuality(t *testing.T) {
	scenarios := MatrixScenarios(2)
	if len(scenarios) != 3*4*2 {
		t.Fatalf("matrix size = %d, want 24", len(scenarios))
	}
	for i, scenario := range scenarios {
		t.Run(fmt.Sprintf("matrix_%02d_%dx%d_msaa%d_fovea%t",
			i, scenario.Config.Width, scenario.Config.Height,
			scenario.Config.MSAASamples, scenario.Config.Foveation.Enabled),
			func(t *testing.T) {
				report, err := RunScenario(scenario)
				if err != nil {
					t.Fatal(err)
				}
				if !report.Success {
					t.Fatalf("matrix case failed expectations: %s", report.FailureReason)
				}
				for _, pipeline := range report.Pipelines {
					if pipeline.FrameCount != scenario.Frames {
						t.Fatalf("frame count = %d, want %d", pipeline.FrameCount, scenario.Frames)
					}
					if pipeline.Average.TransferTotal() == 0 {
						t.Fatal("expected nonzero bandwidth")
					}
				}
			})
	}
}

func TestMeterConcurrentSnapshot(t *testing.T) {
	meter := NewMeter(0, PipelineAtlas, NewCache(1024*MebiByte))
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				key := fmt.Sprintf("tile:%d:%d", worker, i%8)
				meter.Write(StageOpaquePass, EyeLeft, ClassColor, key, 64)
				meter.Read(StageResolve, EyeLeft, ClassColor, key, 64)
			}
		}(worker)
	}
	wg.Wait()
	snapshot := meter.Snapshot()
	if len(snapshot) != 16*100*2 {
		t.Fatalf("access count = %d, want %d", len(snapshot), 16*100*2)
	}
}
