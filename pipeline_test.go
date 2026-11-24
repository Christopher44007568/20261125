package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesCompleteArtifacts(t *testing.T) {
	dir := t.TempDir()
	opts := defaultOptions()
	opts.OutDir = dir
	opts.SweepTo = 1
	opts.SweepStep = 0.05
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"outer.ppm", "inner.ppm", "section.ppm", "report.json",
		"volume_curve.csv",
	} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
	checkPPMHeader(t, filepath.Join(dir, "outer.ppm"), 1000, 760)
	checkPPMHeader(t, filepath.Join(dir, "inner.ppm"), 1000, 760)
	checkPPMHeader(t, filepath.Join(dir, "section.ppm"), 1180, 820)

	data, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if !report.Topology.Watertight {
		t.Fatalf("mesh is not watertight: %+v", report.Topology)
	}
	if report.InvertedSectionTriangles != 48 || report.SelfIntersectionLoops != 1 {
		t.Fatalf("missing diagnostics: %+v", report)
	}
	if report.RedPixelsInSection < 1000 {
		t.Fatalf("section has too few red diagnostic pixels: %d",
			report.RedPixelsInSection)
	}
	if report.Continuity.Samples != 21 {
		t.Fatalf("sweep samples: got %d", report.Continuity.Samples)
	}
	if !report.Continuity.TopologyConstant ||
		!report.Continuity.NoRegionDeletion {
		t.Fatalf("continuity policy failed: %+v", report.Continuity)
	}
	for _, sample := range report.Sweep {
		if sample.Vertices != 204 || sample.Triangles != 404 {
			t.Fatalf("topology changed in sweep: %+v", sample)
		}
	}
}

func checkPPMHeader(t *testing.T, path string, wantW, wantH int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() || sc.Text() != "P6" {
		t.Fatalf("%s: invalid PPM magic", path)
	}
	if !sc.Scan() {
		t.Fatalf("%s: missing dimensions", path)
	}
	var w, h int
	if _, err := fmt.Sscanf(sc.Text(), "%d %d", &w, &h); err != nil {
		t.Fatalf("%s: bad dimensions: %v", path, err)
	}
	if w != wantW || h != wantH {
		t.Fatalf("%s: got %dx%d want %dx%d", path, w, h, wantW, wantH)
	}
}
