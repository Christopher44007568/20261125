package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"final-sorter/sorter"
)

func TestRunSortsJSONAndReportsRejections(t *testing.T) {
	const request = `{
	  "config": {
	    "reorder_window": 2,
	    "timeout_ms": 50,
	    "latest_frame": 12
	  },
	  "submissions": [
	    {
	      "id": "world-left",
	      "frame": 12,
	      "queue": "world",
	      "eye": "left",
	      "submitted_at": "2026-11-25T10:00:00Z",
	      "completed_at": "2026-11-25T10:00:00.010Z"
	    },
	    {
	      "id": "world-left-again",
	      "frame": 12,
	      "queue": "world",
	      "eye": "left",
	      "submitted_at": "2026-11-25T10:00:00.001Z",
	      "completed_at": "2026-11-25T10:00:00.020Z"
	    },
	    {
	      "id": "old-frame",
	      "frame": 8,
	      "queue": "hand",
	      "eye": "right",
	      "submitted_at": "2026-11-25T10:00:00Z",
	      "completed_at": "2026-11-25T10:00:00.005Z"
	    }
	  ]
	}`

	var stdout, stderr bytes.Buffer
	if err := run(nil, strings.NewReader(request), &stdout, &stderr); err != nil {
		t.Fatalf("run() error = %v; stderr = %s", err, stderr.String())
	}

	var report sorter.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if report.LatestFrame != 12 {
		t.Fatalf("LatestFrame = %d, want 12", report.LatestFrame)
	}
	if len(report.Adopted) != 1 || report.Adopted[0].ID != "world-left" {
		t.Fatalf("unexpected adopted entries: %#v", report.Adopted)
	}
	if report.Counts.Duplicates != 1 {
		t.Errorf("Duplicates = %d, want 1", report.Counts.Duplicates)
	}
	if report.Counts.HistoricalBackfills != 1 {
		t.Errorf("HistoricalBackfills = %d, want 1", report.Counts.HistoricalBackfills)
	}
	for _, rejected := range report.Rejected {
		if rejected.Eye == sorter.EyeUnknown {
			t.Errorf("rejected entry %q lost eye information", rejected.ID)
		}
	}
}

func TestRunRejectsTrailingJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(nil, strings.NewReader(`{"submissions":[]} {}`), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("run() error = %v, want multiple JSON values", err)
	}
}

func TestRunRejectsNegativeTimeout(t *testing.T) {
	const request = `{"config":{"timeout_ms":-1},"submissions":[]}`
	var stdout, stderr bytes.Buffer
	err := run(nil, strings.NewReader(request), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "timeout_ms") {
		t.Fatalf("run() error = %v, want timeout_ms error", err)
	}
}
