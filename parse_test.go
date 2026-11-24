package eyestage

import (
	"strings"
	"testing"
)

func TestParseNestedFramesAndAliases(t *testing.T) {
	input := `{
		"unit": "ms",
		"frames": [{
			"frame": 7,
			"eyes": {
				"L": {
					"采集场景": {"start": 1, "end": 3},
					"reproject": {"start_ms": 3, "end_ms": 5},
					"畸变": {"start": 5, "end": 7},
					"compose": {"start": 7, "end": 9}
				},
				"R": {
					"capture_scene": [11, 13],
					"reprojection": {"start": 13, "end": 15},
					"distortion": {"start": 15, "end": 17},
					"composition": {"start": 17, "end": 19}
				}
			}
		}]
	}`
	dataset, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(dataset.Frames) != 1 || dataset.Frames[0].ID != "7" {
		t.Fatalf("unexpected frames: %#v", dataset.Frames)
	}
	left := dataset.Frames[0].Eyes[EyeLeft]
	if left == nil {
		t.Fatal("left eye missing")
	}
	if got := *left.Stages[StageCaptureScene].DurationNS; got != 2_000_000 {
		t.Fatalf("capture duration = %d, want 2000000", got)
	}
	if got := *left.Stages[StageComposition].DurationNS; got != 2_000_000 {
		t.Fatalf("composition duration = %d, want 2000000", got)
	}
	if got := *dataset.Frames[0].Eyes[EyeRight].Stages[StageCaptureScene].DurationNS; got != 2_000_000 {
		t.Fatalf("right capture duration = %d, want 2000000", got)
	}
}

func TestParseJSONLFlatRecordsAndDefaultUnitFallback(t *testing.T) {
	input := `{"frame_id":"a","eye":"left","stage":"capture_scene","start":10,"end":20}
{"frame_id":"a","eye":"right","stage":"capture_scene","start":12,"end":22}
`
	dataset, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(dataset.Frames) != 1 {
		t.Fatalf("frames = %d, want 1", len(dataset.Frames))
	}
	left := dataset.Frames[0].Eyes[EyeLeft].Stages[StageCaptureScene]
	if left.Start.Source != "fallback:default_ns" {
		t.Fatalf("start source = %q, want fallback:default_ns", left.Start.Source)
	}
	if *left.DurationNS != 10 {
		t.Fatalf("duration = %d, want 10", *left.DurationNS)
	}
	found := false
	for _, diagnostic := range dataset.Diagnostics {
		if diagnostic.Code == "timestamp_fallback" && diagnostic.FallbackSource == "fallback:default_ns" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing fallback source diagnostic")
	}
}

func TestParseUnsupportedTimestampDerivesFromDuration(t *testing.T) {
	input := `{"unit":"ms","frames":[{"id":"x","left":{"reprojection":{
		"start":10,"end":"clock-unavailable","duration_ms":5
	}}}]}`
	dataset, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	stage := dataset.Frames[0].Eyes[EyeLeft].Stages[StageReprojection]
	if stage.End == nil || stage.End.NS != 15_000_000 {
		t.Fatalf("derived end = %#v, want 15000000ns", stage.End)
	}
	if stage.End.Source != "fallback:duration" {
		t.Fatalf("end source = %q, want fallback:duration", stage.End.Source)
	}
	hasUnsupported := false
	hasFallback := false
	for _, diagnostic := range dataset.Diagnostics {
		if diagnostic.Code == "unsupported_timestamp" {
			hasUnsupported = true
		}
		if diagnostic.Code == "timestamp_fallback" && diagnostic.FallbackSource == "fallback:duration" {
			hasFallback = true
		}
	}
	if !hasUnsupported || !hasFallback {
		t.Fatalf("diagnostics = %#v", dataset.Diagnostics)
	}
}

func TestParseRFC3339(t *testing.T) {
	input := `{"frames":[{"id":"r","left":{"capture_scene":{
		"start":"2026-11-25T10:00:00Z","end":"2026-11-25T10:00:00.010Z"
	}}}]}`
	dataset, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	stage := dataset.Frames[0].Eyes[EyeLeft].Stages[StageCaptureScene]
	if stage.Start.Source != "explicit:rfc3339" || *stage.DurationNS != 10_000_000 {
		t.Fatalf("stage = %#v", stage)
	}
}

func TestParseUnsupportedUnitAndZonelessRFC3339MarkFallback(t *testing.T) {
	input := `{"unit":"ticks","frames":[{"id":"u","left":{"capture_scene":{
		"start":"2026-11-25T10:00:00","end":"2026-11-25T10:00:00.010"
	}}}]}`
	dataset, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	stage := dataset.Frames[0].Eyes[EyeLeft].Stages[StageCaptureScene]
	if stage.Start.Source != "fallback:no_timezone_utc" {
		t.Fatalf("start source = %q", stage.Start.Source)
	}
	hasUnit := false
	hasTimezone := false
	for _, diagnostic := range dataset.Diagnostics {
		if diagnostic.Code == "unsupported_unit" && diagnostic.FallbackSource == "fallback:unknown_unit_ns" {
			hasUnit = true
		}
		if diagnostic.Code == "timestamp_fallback" && diagnostic.FallbackSource == "fallback:no_timezone_utc" {
			hasTimezone = true
		}
	}
	if !hasUnit || !hasTimezone {
		t.Fatalf("diagnostics = %#v", dataset.Diagnostics)
	}
}
