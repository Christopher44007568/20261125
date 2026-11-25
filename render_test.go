package main

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSVGArtifactsAreWellFormed(t *testing.T) {
	scenarios, err := buildScenarios()
	if err != nil {
		t.Fatal(err)
	}
	for _, sc := range scenarios {
		run, err := analyzeScenario(sc)
		if err != nil {
			t.Fatal(err)
		}
		documents := map[string]string{
			"three_view.svg":        renderThreeView(run),
			"control_migration.svg": renderControlMigration(run),
			"motion_sheet.svg":      renderMotionSheet(run),
		}
		for name, body := range documents {
			if strings.Contains(body, "NaN") || strings.Contains(body, "Inf") {
				t.Fatalf("%s/%s contains non-finite output", sc.ID, name)
			}
			switch name {
			case "three_view.svg":
				if !strings.Contains(body, "GROOM A") || !strings.Contains(body, "GROOM B") || !strings.Contains(body, "BLEND") {
					t.Fatalf("%s/%s is missing comparison rows", sc.ID, name)
				}
			case "control_migration.svg":
				if !strings.Contains(body, "Groom A") || !strings.Contains(body, "Groom B") {
					t.Fatalf("%s/%s is missing source labels", sc.ID, name)
				}
			case "motion_sheet.svg":
				if !strings.Contains(body, "t = 0.00") || !strings.Contains(body, "t = 1.00") {
					t.Fatalf("%s/%s is missing interpolation endpoints", sc.ID, name)
				}
			}
			decoder := xml.NewDecoder(strings.NewReader(body))
			for {
				_, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("%s/%s is not well-formed XML: %v", sc.ID, name, err)
				}
			}
		}
	}
}

func TestFullRunWritesReportAndMetrics(t *testing.T) {
	dir := t.TempDir()
	if err := run(dir); err != nil {
		t.Fatal(err)
	}
	required := []string{
		"report.html",
		"metrics.json",
		"curly_to_straight/three_view.svg",
		"curly_to_straight/control_migration.svg",
		"curly_to_straight/motion_sheet.svg",
		"long_to_short/three_view.svg",
		"long_to_short/control_migration.svg",
		"long_to_short/motion_sheet.svg",
	}
	for _, name := range required {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() < 100 {
			t.Fatalf("%s is unexpectedly small", name)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "metrics.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metrics MetricsFile
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatal(err)
	}
	if len(metrics.Scenarios) != 2 {
		t.Fatalf("got %d scenarios", len(metrics.Scenarios))
	}
	for _, report := range metrics.Scenarios {
		if report.StrandsA == report.StrandsB {
			t.Fatalf("%s does not use unequal counts", report.ID)
		}
		if report.MaxTipOverBound > 1e-8 || report.MaxControlOverBound > 1e-8 || report.MaxSpanOverBound > 1e-8 {
			t.Fatalf("%s has a bound violation", report.ID)
		}
		if len(report.Frames) != 33 {
			t.Fatalf("%s has %d verification frames", report.ID, len(report.Frames))
		}
		for _, frame := range report.Frames {
			if !frame.Finite || frame.TipBoundViolations != 0 || frame.ControlViolations != 0 || frame.SpanViolations != 0 {
				t.Fatalf("%s has a failed verification frame: %+v", report.ID, frame)
			}
		}
	}
	reportHTML, err := os.ReadFile(filepath.Join(dir, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(reportHTML), "three_view.svg") ||
		!strings.Contains(string(reportHTML), "控制点迁移") {
		t.Fatal("report.html is missing expected content")
	}
}

func TestCurlyStraightAndLongShortDiffer(t *testing.T) {
	scenarios, err := buildScenarios()
	if err != nil {
		t.Fatal(err)
	}
	curly, straight := scenarios[0].A, scenarios[0].B
	long, short := scenarios[1].A, scenarios[1].B
	if averagePathLength(curly) <= averagePathLength(straight)*1.3 {
		t.Fatal("curly groom should have substantially longer curved paths")
	}
	if maxTipRadius(long) <= maxTipRadius(short)*1.6 {
		t.Fatal("long groom should have substantially longer tips")
	}
}

func averagePathLength(g Groom) float64 {
	total := 0.0
	for _, strand := range g.Strands {
		for i := 1; i < len(strand.Controls); i++ {
			total += strand.Controls[i].Dist(strand.Controls[i-1])
		}
	}
	return total / float64(len(g.Strands))
}
