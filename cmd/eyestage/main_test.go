package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const validInput = `{"unit":"ms","frames":[{
	"id":"a",
	"left":{
		"capture_scene":{"start":0,"end":10},
		"reprojection":{"start":10,"end":20},
		"distortion":{"start":20,"end":30},
		"composition":{"start":30,"end":40}
	},
	"right":{
		"capture_scene":{"start":5,"end":15},
		"reprojection":{"start":15,"end":25},
		"distortion":{"start":25,"end":35},
		"composition":{"start":35,"end":45}
	}
}]}`

func TestRunJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-format", "json"}, strings.NewReader(validInput), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout.String())
	}
	if _, ok := decoded["stage_stats"]; !ok {
		t.Fatalf("missing stage_stats: %s", stdout.String())
	}
}

func TestRunFailOnDiagnostics(t *testing.T) {
	input := `{"frames":[{"id":"bad","left":{}}]}`
	var stdout, stderr bytes.Buffer
	code := run([]string{"-fail-on-diagnostics"}, strings.NewReader(input), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2; stderr = %s", code, stderr.String())
	}
}

func TestRunRejectsUnknownFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-format", "xml"}, strings.NewReader(validInput), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestRunHTML(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-format", "html"}, strings.NewReader(validInput), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "<!doctype html>") {
		t.Fatalf("unexpected html: %s", stdout.String())
	}
}
