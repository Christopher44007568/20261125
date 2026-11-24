package main

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"highlightlab/sim"
)

func TestTimelineEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/timeline?seconds=2", nil)
	response := httptest.NewRecorder()
	handleTimeline(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if got := response.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content type = %q", got)
	}

	var timeline sim.Timeline
	if err := json.NewDecoder(response.Body).Decode(&timeline); err != nil {
		t.Fatalf("decode timeline: %v", err)
	}
	if len(timeline.Eyes) != 2 {
		t.Fatalf("eyes = %d, want 2", len(timeline.Eyes))
	}
	if timeline.Audit.CrossEyeSamples != 0 || timeline.Audit.EnergyCapViolations != 0 {
		t.Fatalf("audit failed: %+v", timeline.Audit)
	}
}

func TestTimelineEndpointRejectsBadDuration(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/timeline?seconds=99", nil)
	response := httptest.NewRecorder()
	handleTimeline(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestLedgerEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/ledger.csv", nil)
	response := httptest.NewRecorder()
	handleLedger(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	records, err := csv.NewReader(response.Body).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(records) != 1+361*2*3 {
		t.Fatalf("csv rows = %d, want %d", len(records), 1+361*2*3)
	}
	expectedHeader := []string{
		"frame", "time_s", "eye", "object", "center_x", "center_y",
		"stretch_px", "peak", "energy", "history_x", "history_y",
		"history_energy", "composite_energy", "history_eye", "capped",
	}
	if strings.Join(records[0], ",") != strings.Join(expectedHeader, ",") {
		t.Fatalf("header = %v", records[0])
	}
	if records[1][14] != "false" {
		t.Fatalf("first history sample should not be capped: %v", records[1])
	}
}
