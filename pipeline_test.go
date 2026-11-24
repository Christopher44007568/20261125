package eyebw

import (
	"testing"
)

func testConfig() FrameConfig {
	cfg := DefaultConfig()
	cfg.Width = 480
	cfg.Height = 480
	cfg.MSAASamples = 4
	cfg.Budget.FrameBytes = 0
	cfg.CacheCapacityBytes = 8 * MebiByte
	cfg.RetainAccesses = true
	return cfg
}

func runTestFrames(t *testing.T, cfg FrameConfig, frames int) *ScenarioReport {
	t.Helper()
	scenario := Scenario{
		Name:        "test",
		Config:      cfg,
		Frames:      frames,
		DynamicGaze: true,
	}
	report, err := RunScenario(scenario)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	return report
}

func TestAccessLedgerNeverCountsHitAsTransfer(t *testing.T) {
	report := runTestFrames(t, testConfig(), 5)
	for _, frame := range report.FrameReports {
		for _, pipeline := range frame.Pipelines {
			recomputed := AggregateAccesses(pipeline.Accesses)
			if recomputed.TransferTotal() != pipeline.TransferTotal() {
				t.Fatalf("%s frame %d aggregate mismatch: events=%d report=%d",
					pipeline.ID, frame.Index, recomputed.TransferTotal(), pipeline.TransferTotal())
			}
			for _, access := range pipeline.Accesses {
				if !access.Class.Valid() {
					t.Fatalf("invalid class %q", access.Class)
				}
				if access.TransferBytes > access.LogicalBytes {
					t.Fatalf("transfer exceeds logical bytes: %+v", access)
				}
				if access.CacheHit && (access.TransferBytes != 0 || access.Direction != DirRead) {
					t.Fatalf("cache hit must be a zero-transfer read: %+v", access)
				}
				if access.TransferBytes == 0 && !access.CacheHit {
					t.Fatalf("zero-transfer access without cache hit: %+v", access)
				}
			}
		}
	}
}

func TestAllPipelinesRecordFourClassesAndHotspots(t *testing.T) {
	report := runTestFrames(t, testConfig(), 6)
	for _, summary := range report.Pipelines {
		classBytes := map[DataClass]uint64{}
		for _, class := range summary.Classes {
			classBytes[DataClass(class.Name)] = class.TransferReadBytes + class.TransferWriteBytes
		}
		for _, class := range AllClasses() {
			if classBytes[class] == 0 {
				t.Fatalf("pipeline %s has no %s transfer bytes", summary.ID, class)
			}
		}
		if len(summary.Hotspots) == 0 {
			t.Fatalf("pipeline %s has no hotspot stages", summary.ID)
		}
		if summary.Hotspots[0].Rank != 1 || summary.Hotspots[0].TransferBytes == 0 {
			t.Fatalf("invalid top hotspot for %s: %+v", summary.ID, summary.Hotspots[0])
		}
	}
}

func TestAtlasMatchesSeparatePixelTrafficAndGapAddsBytes(t *testing.T) {
	cfg := testConfig()
	report := runTestFrames(t, cfg, 3)
	for _, frame := range report.FrameReports {
		separate := frame.Get(PipelineSeparateEyes)
		atlas := frame.Get(PipelineAtlas)
		if separate.TransferTotal() != atlas.TransferTotal() {
			t.Fatalf("frame %d atlas=%d separate=%d; gapless atlas should not change pixel bytes",
				frame.Index, atlas.TransferTotal(), separate.TransferTotal())
		}
	}

	gapped := testConfig()
	gapped.AtlasGapPixels = 16
	gapReport := runTestFrames(t, gapped, 1)
	atlas := gapReport.FrameReports[0].Get(PipelineAtlas)
	separate := gapReport.FrameReports[0].Get(PipelineSeparateEyes)
	if atlas.TransferTotal() <= separate.TransferTotal() {
		t.Fatalf("atlas gap should add color clear/scanout bytes: atlas=%d separate=%d",
			atlas.TransferTotal(), separate.TransferTotal())
	}
}

func TestSafeSingleEyeReuseSavesBandwidthAndPassesQuality(t *testing.T) {
	report := runTestFrames(t, testConfig(), 6)
	byID := map[PipelineID]PipelineSequenceSummary{}
	for _, pipeline := range report.Pipelines {
		byID[pipeline.ID] = pipeline
		if !pipeline.AllQualityPassed {
			t.Fatalf("safe pipeline %s unexpectedly failed quality", pipeline.ID)
		}
	}
	separate := byID[PipelineSeparateEyes]
	reuse := byID[PipelineSingleReuse]
	if reuse.Average.TransferTotal() >= separate.Average.TransferTotal() {
		t.Fatalf("reuse avg %s should be below separate %s",
			eyebwFormat(reuse.Average.TransferTotal()), eyebwFormat(separate.Average.TransferTotal()))
	}
}

func eyebwFormat(n uint64) string { return FormatBytes(n) }
