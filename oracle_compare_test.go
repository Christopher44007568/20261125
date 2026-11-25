package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type oracleDoc struct {
	Scenarios []struct {
		ID     string        `json:"id"`
		Frames []FrameReport `json:"frames"`
	} `json:"scenarios"`
}

func ufLoadOracle(t *testing.T) oracleDoc {
	raw, err := os.ReadFile("artifacts/degrade_report.json")
	if err != nil {
		t.Skip("oracle not available")
	}
	var doc oracleDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestFrameStatesMatchOracle(t *testing.T) {
	doc := ufLoadOracle(t)
	cfg := DefaultConfig()
	defs := AllScenarios()
	for i, def := range defs {
		_, frames := ExecuteScenario(def, cfg)
		want := doc.Scenarios[i]
		if want.ID != def.ID {
			t.Fatalf("scenario order mismatch %s vs %s", want.ID, def.ID)
		}
		if len(frames) != len(want.Frames) {
			t.Fatalf("%s frame count %d want %d", def.ID, len(frames), len(want.Frames))
		}
		for j, got := range frames {
			wf := want.Frames[j]
			if got.Frame != wf.Frame {
				t.Errorf("%s f%d frame no %d", def.ID, j+1, got.Frame)
			}
			if !reflect.DeepEqual(got.State, wf.State) {
				t.Errorf("%s f%d STATE got=%+v want=%+v", def.ID, j+1, got.State, wf.State)
			}
			if !ufMetricsEqual(got.Metrics, wf.Metrics) {
				t.Errorf("%s f%d METRICS got=%+v want=%+v", def.ID, j+1, got.Metrics, wf.Metrics)
			}
			if got.StatusLevel != wf.StatusLevel || got.Headline != wf.Headline {
				t.Errorf("%s f%d STATUS got=(%s,%s) want=(%s,%s)", def.ID, j+1,
					got.StatusLevel, got.Headline, wf.StatusLevel, wf.Headline)
			}
			if got.ChainID != wf.ChainID {
				t.Errorf("%s f%d FRAMECHAIN got=%q want=%q", def.ID, j+1, got.ChainID, wf.ChainID)
			}
			if !reflect.DeepEqual(got.StatusDetail.Lines, wf.StatusDetail.Lines) {
				t.Errorf("%s f%d LINES\n got=%#v\nwant=%#v", def.ID, j+1,
					got.StatusDetail.Lines, wf.StatusDetail.Lines)
			}
			if !reflect.DeepEqual(got.StatusDetail.ActiveModes, wf.StatusDetail.ActiveModes) {
				t.Errorf("%s f%d MODES got=%v want=%v", def.ID, j+1,
					got.StatusDetail.ActiveModes, wf.StatusDetail.ActiveModes)
			}
			if got.StatusDetail.ChainID != wf.StatusDetail.ChainID {
				t.Errorf("%s f%d SDCHAIN got=%q want=%q", def.ID, j+1,
					got.StatusDetail.ChainID, wf.StatusDetail.ChainID)
			}
			if got.StatusDetail.FloorWarning != wf.StatusDetail.FloorWarning {
				t.Errorf("%s f%d FLOOR got=%v want=%v", def.ID, j+1,
					got.StatusDetail.FloorWarning, wf.StatusDetail.FloorWarning)
			}
		}
	}
}

func ufMetricsEqual(a, b Metrics) bool {
	return a.Frame == b.Frame &&
		a.FrameTimeMS == b.FrameTimeMS &&
		a.BudgetMS == b.BudgetMS &&
		a.FPS == b.FPS &&
		a.BudgetHeadroomMS == b.BudgetHeadroomMS &&
		a.EyeCount == b.EyeCount && a.Stereo == b.Stereo &&
		a.SampleLevel == b.SampleLevel && a.SamplingScale == b.SamplingScale &&
		a.Reprojection == b.Reprojection && a.HistoryCapture == b.HistoryCapture &&
		a.Interaction == b.Interaction && a.RelativeLoad == b.RelativeLoad &&
		a.QualityScore == b.QualityScore
}
