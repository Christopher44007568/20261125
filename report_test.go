package stereorollback

import (
	"strings"
	"testing"
)

func TestReportJSONContainsRequiredRecords(t *testing.T) {
	report, err := RunAll()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := report.MarshalJSONBytes()
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(raw)
	for _, key := range []string{
		`"failure_stage"`,
		`"retained_resources"`,
		`"fallback_frame"`,
		`"successful_frames"`,
		`"left_eye_failure"`,
		`"right_eye_failure"`,
		`"shared_texture_corruption"`,
		`"attachment_create_failed"`,
		`"material_parameter_error"`,
		`"reprojection_timeout"`,
	} {
		if !strings.Contains(jsonText, key) {
			t.Errorf("report JSON missing %s", key)
		}
	}
}

func TestRenderTextIncludesSummary(t *testing.T) {
	report, err := RunAll()
	if err != nil {
		t.Fatal(err)
	}
	text := RenderText(report)
	if !strings.Contains(text, "all_invariants_hold=true") {
		t.Fatal("text report must state that all invariants hold")
	}
}
