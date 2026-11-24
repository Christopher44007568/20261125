package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunSuiteContainsRequiredCasesAndMetrics(t *testing.T) {
	rows := RunSuite(DefaultSuite())
	if len(rows) != 5*3 {
		t.Fatalf("expected 15 scenario/mode rows, got %d", len(rows))
	}

	seen := map[string]bool{}
	for _, row := range rows {
		key := row.Scenario + "/" + string(row.Mode)
		seen[key] = true
		if row.Iterations <= 0 {
			t.Fatalf("%s has no iteration count", key)
		}
		if row.DepthError < 0 {
			t.Fatalf("%s has negative depth error %g", key, row.DepthError)
		}
		if row.Status == StatusHit && row.HitLayer < 0 {
			t.Fatalf("%s hit without layer", key)
		}
		if row.Fallback && row.HitLayer != -1 {
			t.Fatalf("%s fallback layer should be -1", key)
		}
	}

	for _, scenario := range []string{"plane_forward", "steep_slope", "reverse_view", "height_out_of_bounds"} {
		for _, mode := range []SearchMode{ModeFixed, ModeBinary, ModeMaxIteration} {
			key := scenario + "/" + string(mode)
			if !seen[key] {
				t.Fatalf("missing report row %s", key)
			}
		}
	}
}

func TestJSONAndCSVOutput(t *testing.T) {
	rows := RunSuite(DefaultSuite())

	var jsonBuffer bytes.Buffer
	if err := WriteJSON(&jsonBuffer, rows[:3]); err != nil {
		t.Fatal(err)
	}
	var decoded []Row
	if err := json.Unmarshal(jsonBuffer.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(decoded) != 3 || decoded[0].HitLayer != 5 {
		t.Fatalf("unexpected JSON rows: %+v", decoded)
	}

	var csvBuffer bytes.Buffer
	if err := WriteCSV(&csvBuffer, rows); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(csvBuffer.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != len(rows)+1 {
		t.Fatalf("expected %d CSV records, got %d", len(rows)+1, len(records))
	}
	if records[0][3] != "hit_layer" || records[0][4] != "iterations" || records[0][8] != "depth_error" {
		t.Fatalf("unexpected CSV header: %v", records[0])
	}
}
