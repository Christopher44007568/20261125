package eyestage

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestAnalyzeEqualSpansStillReportsSingleEyeWait(t *testing.T) {
	input := `{"unit":"ms","frames":[{"id":"same-span",
		"left":{
			"capture_scene":{"start":0,"end":10},
			"reprojection":{"start":10,"end":20},
			"distortion":{"start":20,"end":30},
			"composition":{"start":30,"end":40}
		},
		"right":{
			"capture_scene":{"start":0,"end":5},
			"reprojection":{"start":5,"end":10},
			"distortion":{"start":30,"end":35},
			"composition":{"start":35,"end":40}
		}
	}]}`
	dataset, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	report := Analyze(dataset, DefaultConfig())
	if got := report.Frames[0].Eyes[0].ActiveSpanNS; got != 40_000_000 {
		t.Fatalf("left span = %d", got)
	}
	if got := report.Frames[0].Eyes[1].ActiveSpanNS; got != 40_000_000 {
		t.Fatalf("right span = %d", got)
	}
	if got := report.IdleWaits[0].TotalWaitNS; got != 0 {
		t.Fatalf("left wait = %d, want 0", got)
	}
	if got := report.IdleWaits[1].TotalWaitNS; got != 20_000_000 {
		t.Fatalf("right wait = %d, want 20000000", got)
	}
	rightWaitDiagnostics := 0
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == "single_eye_idle_wait" && diagnostic.Eye == EyeRight {
			rightWaitDiagnostics++
		}
		if diagnostic.Code == "single_eye_idle_wait" && diagnostic.Eye == EyeLeft {
			t.Fatal("unexpected left idle wait")
		}
	}
	if rightWaitDiagnostics != 1 {
		t.Fatalf("right wait diagnostics = %d", rightWaitDiagnostics)
	}
}

func TestAnalyzeMissingRegressionAndFallbackWithoutFrameTotalSubstitution(t *testing.T) {
	input := `{"unit":"ms","frames":[{
		"frame_id":"bad",
		"frame_duration_ms":1000,
		"left":{
			"capture_scene":{"start":10,"end":5}
		},
		"right":{
			"reprojection":{"start":10,"end":"bad-clock","duration_ms":5}
		}
	}]}`
	dataset, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	report := Analyze(dataset, DefaultConfig())
	if report.Summary.MissingStages == 0 {
		t.Fatal("missing stages were not reported")
	}
	if report.Summary.TimestampRegressions != 1 {
		t.Fatalf("regressions = %d, want 1", report.Summary.TimestampRegressions)
	}
	if report.Summary.UnsupportedTimes != 1 {
		t.Fatalf("unsupported = %d, want 1", report.Summary.UnsupportedTimes)
	}
	if report.Summary.FallbackTimes != 1 {
		t.Fatalf("fallbacks = %d, want 1", report.Summary.FallbackTimes)
	}
	var reprojection *StageStat
	for index := range report.StageStats {
		stat := &report.StageStats[index]
		if stat.Eye == EyeRight && stat.Stage == StageReprojection {
			reprojection = stat
		}
		if stat.MaxNS >= int64(time.Second) {
			t.Fatalf("frame total leaked into stage attribution: %#v", stat)
		}
	}
	if reprojection == nil || reprojection.Count != 1 || reprojection.MaxNS != 5_000_000 {
		t.Fatalf("reprojection stat = %#v", reprojection)
	}
	if !hasDiagnostic(report.Diagnostics, "missing_stage") ||
		!hasDiagnostic(report.Diagnostics, "timestamp_regression") ||
		!hasFallback(report.Diagnostics, "fallback:duration") {
		t.Fatalf("diagnostics = %#v", report.Diagnostics)
	}
	var detail *StageDetail
	for index := range report.Frames[0].Eyes[1].Stages {
		if report.Frames[0].Eyes[1].Stages[index].Stage == StageReprojection {
			detail = &report.Frames[0].Eyes[1].Stages[index]
		}
	}
	if detail == nil || detail.End == nil || detail.End.Source != "fallback:duration" {
		t.Fatalf("stage detail = %#v", detail)
	}
}

func TestLongTailUsesPerStageDistribution(t *testing.T) {
	dataset := &Dataset{Unit: "ns"}
	for index := 0; index < 3; index++ {
		frameID := "f" + string(rune('1'+index))
		left := &EyeTiming{Eye: EyeLeft, Stages: map[string]*StageTiming{}}
		right := &EyeTiming{Eye: EyeRight, Stages: map[string]*StageTiming{}}
		for _, stage := range Stages {
			left.Stages[stage] = makeInterval(0, 10_000_000)
			duration := int64(10_000_000)
			if stage == StageReprojection && index == 1 {
				duration = 60_000_000
			}
			right.Stages[stage] = makeInterval(0, duration)
		}
		dataset.Frames = append(dataset.Frames, &Frame{
			ID: frameID, Eyes: map[string]*EyeTiming{EyeLeft: left, EyeRight: right},
		})
	}
	report := Analyze(dataset, DefaultConfig())
	if len(report.LongTail) != 1 {
		t.Fatalf("long tail = %#v", report.LongTail)
	}
	entry := report.LongTail[0]
	if entry.FrameID != "f2" || entry.Eye != EyeRight || entry.Stage != StageReprojection || entry.DurationNS != 60_000_000 {
		t.Fatalf("long tail entry = %#v", entry)
	}
	if len(report.LongTailFrames) != 1 || report.LongTailFrames[0].FrameID != "f2" {
		t.Fatalf("long tail frames = %#v", report.LongTailFrames)
	}
}

func TestStageOverlapAndIdleWindows(t *testing.T) {
	dataset := &Dataset{Unit: "ns"}
	left := &EyeTiming{Eye: EyeLeft, Stages: map[string]*StageTiming{
		StageCaptureScene: makeInterval(0, 10),
		StageReprojection: makeInterval(10, 20),
		StageDistortion:   makeInterval(20, 30),
		StageComposition:  makeInterval(30, 40),
	}}
	right := &EyeTiming{Eye: EyeRight, Stages: map[string]*StageTiming{
		StageCaptureScene: makeInterval(4, 14),
		StageReprojection: makeInterval(20, 30),
		StageDistortion:   makeInterval(30, 40),
		StageComposition:  makeInterval(40, 50),
	}}
	dataset.Frames = []*Frame{{ID: "overlap", Eyes: map[string]*EyeTiming{EyeLeft: left, EyeRight: right}}}
	report := Analyze(dataset, DefaultConfig())

	capture := findOverlap(report.StageOverlaps, StageCaptureScene)
	if capture == nil || capture.OverlapNS != 6 {
		t.Fatalf("capture overlap = %#v", capture)
	}
	if len(report.CrossStageOverlaps) == 0 {
		t.Fatal("cross-stage overlap missing")
	}
	if report.Frames[0].Eyes[0].IdleWaitNS == nil || *report.Frames[0].Eyes[0].IdleWaitNS != 10 {
		t.Fatalf("left idle = %#v", report.Frames[0].Eyes[0].IdleWaitNS)
	}
}

func TestPersistentSingleEyeWait(t *testing.T) {
	dataset := &Dataset{Unit: "ns"}
	for index := 0; index < 2; index++ {
		left := &EyeTiming{Eye: EyeLeft, Stages: map[string]*StageTiming{
			StageCaptureScene: makeInterval(0, 10),
			StageReprojection: makeInterval(10, 20),
			StageDistortion:   makeInterval(20, 30),
			StageComposition:  makeInterval(30, 40),
		}}
		right := &EyeTiming{Eye: EyeRight, Stages: map[string]*StageTiming{
			StageCaptureScene: makeInterval(0, 5),
			StageReprojection: makeInterval(5, 10),
			StageDistortion:   makeInterval(30, 35),
			StageComposition:  makeInterval(35, 40),
		}}
		dataset.Frames = append(dataset.Frames, &Frame{
			ID:   "wait-" + string(rune('1'+index)),
			Eyes: map[string]*EyeTiming{EyeLeft: left, EyeRight: right},
		})
	}
	report := Analyze(dataset, DefaultConfig())
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Code == "persistent_single_eye_wait" && diagnostic.Eye == EyeRight {
			return
		}
	}
	t.Fatalf("persistent wait diagnostic missing: %#v", report.Diagnostics)
}

func TestWriteTextAndJSON(t *testing.T) {
	dataset, err := Parse(strings.NewReader(`{"unit":"ns","frames":[{
		"id":"r",
		"left":{"capture_scene":{"start_ns":0,"end_ns":10}},
		"right":{"capture_scene":{"start_ns":0,"end_ns":10}}
	}]}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	report := Analyze(dataset, DefaultConfig())
	var text bytes.Buffer
	if err := WriteText(&text, report); err != nil {
		t.Fatalf("WriteText: %v", err)
	}
	for _, marker := range []string{"阶段耗时直方图", "长尾阶段", "跨眼对应阶段重叠", "单眼空等", "诊断"} {
		if !strings.Contains(text.String(), marker) {
			t.Fatalf("text output missing %q", marker)
		}
	}
	var encoded bytes.Buffer
	if err := WriteJSON(&encoded, report); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !strings.Contains(encoded.String(), `"tail_percentile"`) ||
		!strings.Contains(encoded.String(), `"stage_stats"`) {
		t.Fatalf("json output = %s", encoded.String())
	}
	var html bytes.Buffer
	if err := WriteHTML(&html, report); err != nil {
		t.Fatalf("WriteHTML: %v", err)
	}
	if !strings.Contains(html.String(), "<!doctype html>") ||
		!strings.Contains(html.String(), "阶段耗时直方图") {
		t.Fatal("html output missing report content")
	}
}

func makeInterval(start, end int64) *StageTiming {
	duration := end - start
	return &StageTiming{
		Start:          &Timestamp{NS: start, Source: "explicit:ns"},
		End:            &Timestamp{NS: end, Source: "explicit:ns"},
		DurationNS:     &duration,
		DurationSource: "interval",
	}
}

func findOverlap(values []StageOverlap, stage string) *StageOverlap {
	for index := range values {
		if values[index].Stage == stage {
			return &values[index]
		}
	}
	return nil
}

func hasDiagnostic(values []Diagnostic, code string) bool {
	for _, diagnostic := range values {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func hasFallback(values []Diagnostic, source string) bool {
	for _, diagnostic := range values {
		if diagnostic.Code == "timestamp_fallback" && diagnostic.FallbackSource == source {
			return true
		}
	}
	return false
}
