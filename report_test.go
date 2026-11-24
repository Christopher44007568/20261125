package notify

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDefaultReportPassesAndSerializes(t *testing.T) {
	report := DefaultReport()
	if !report.Pass {
		for _, scenario := range report.Cases {
			for _, check := range scenario.Checks {
				if !check.Pass {
					t.Errorf("%s/%s: %s", scenario.ID, check.Name, check.Detail)
				}
			}
		}
		t.Fatal("default report contains failed checks")
	}
	if len(report.Cases) != 4 {
		t.Fatalf("scenario count = %d, want 4", len(report.Cases))
	}

	var buffer bytes.Buffer
	if err := report.WriteJSON(&buffer); err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatalf("report JSON is not readable: %v", err)
	}
	if !decoded.Pass || len(decoded.Cases) != len(report.Cases) {
		t.Fatalf("decoded report pass=%v cases=%d", decoded.Pass, len(decoded.Cases))
	}

	var markdown bytes.Buffer
	if err := report.WriteMarkdown(&markdown); err != nil {
		t.Fatal(err)
	}
	if markdown.Len() == 0 || !bytes.Contains(markdown.Bytes(), []byte("Overall: **PASS**")) {
		t.Fatal("markdown report missing passing summary")
	}
}
