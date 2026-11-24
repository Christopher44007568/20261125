package eyebw

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReportFilesAndJSONShape(t *testing.T) {
	cfg := testConfig()
	scenario := Scenario{Name: "write", Config: cfg, Frames: 2, DynamicGaze: true}
	report, err := RunScenarios([]Scenario{scenario})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := WriteReport(dir, report, true); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"report.json", "frames.csv", "stages.csv", "classes.csv", "summary.md", "events.csv"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.Size() == 0 {
			t.Fatalf("expected non-empty %s: %v", name, err)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var shape struct {
		Scenarios []struct {
			FrameReports []struct {
				Pipelines []struct {
					Totals json.RawMessage `json:"totals"`
				} `json:"pipelines"`
			} `json:"frame_reports"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		t.Fatal(err)
	}
	if len(shape.Scenarios) != 1 || len(shape.Scenarios[0].FrameReports) != 2 {
		t.Fatalf("unexpected JSON shape: %+v", shape)
	}
	if len(shape.Scenarios[0].FrameReports[0].Pipelines) != 3 {
		t.Fatal("expected three pipeline reports in JSON")
	}
	if len(shape.Scenarios[0].FrameReports[0].Pipelines[0].Totals) == 0 {
		t.Fatal("totals should be nested under each pipeline")
	}
}

func TestDefaultScenariosRecordBudgetAndQualityExpectations(t *testing.T) {
	report, err := DefaultReport()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Success {
		t.Fatal("default scenario expectations should pass, including the expected quality trap")
	}

	var ultra, trap *ScenarioReport
	for _, scenario := range report.Scenarios {
		switch scenario.Name {
		case "ultra_4096_msaa8_dynamic_fovea_budget":
			ultra = scenario
		case "quality_trap_aggressive_center_reuse":
			trap = scenario
		}
	}
	if ultra == nil || trap == nil {
		t.Fatal("missing ultra or quality-trap scenario")
	}
	for _, pipeline := range ultra.Pipelines {
		if pipeline.BudgetExceededFrames != ultra.Frames {
			t.Fatalf("ultra %s exceeded %d/%d frames, want all frames",
				pipeline.ID, pipeline.BudgetExceededFrames, ultra.Frames)
		}
	}
	var trapReuse PipelineSequenceSummary
	for _, pipeline := range trap.Pipelines {
		if pipeline.ID == PipelineSingleReuse {
			trapReuse = pipeline
		}
	}
	if trapReuse.QualityFailedFrames != trap.Frames {
		t.Fatalf("quality trap should fail every frame, got %d/%d",
			trapReuse.QualityFailedFrames, trap.Frames)
	}
}
