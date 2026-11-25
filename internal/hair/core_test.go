package hair

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateDeterministic(t *testing.T) {
	a := Generate(300, 77, 16)
	b := Generate(300, 77, 16)
	if len(a.Strands) != 300 || len(a.Levels) != 1 {
		t.Fatalf("unexpected strand count: %d levels=%d", len(a.Strands), len(a.Levels))
	}
	for i := range a.Strands {
		if len(a.Strands[i].Points) != 16 {
			t.Fatalf("strand %d has %d points", i, len(a.Strands[i].Points))
		}
		for j := range a.Strands[i].Points {
			if a.Strands[i].Points[j] != b.Strands[i].Points[j] {
				t.Fatalf("generation is not deterministic at strand %d point %d", i, j)
			}
		}
	}
	if math.IsInf(a.Bounds.Min.X, 0) || math.IsNaN(a.Bounds.Max.Y) {
		t.Fatalf("invalid bounds: %+v", a.Bounds)
	}
}

func TestClusterMassAndCoverage(t *testing.T) {
	const sourceCount = 1200
	asset := Generate(sourceCount, 9, 20)
	BuildGuideLevels(asset, []int{64, 16}, 9)
	if len(asset.Levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(asset.Levels))
	}

	for _, level := range asset.Levels[1:] {
		if got := sumWeights(level.Entries); math.Abs(got-float64(sourceCount)) > 1e-9 {
			t.Fatalf("level %d mass %.9f, want %d", level.Index, got, sourceCount)
		}
		for _, e := range level.Entries {
			if e.Weight <= 0 {
				t.Fatalf("level %d has empty cluster %d", level.Index, e.ID)
			}
			if e.RepresentativeID < 0 || e.RepresentativeID >= sourceCount {
				t.Fatalf("bad representative id %d", e.RepresentativeID)
			}
			if len(e.SourceIDs) == 0 || len(e.Points) != 20 {
				t.Fatalf("incomplete guide %d", e.ID)
			}
		}
	}

	seen := make([]int, sourceCount)
	for _, e := range asset.Levels[1].Entries {
		for _, id := range e.SourceIDs {
			seen[id]++
		}
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("strand %d appears %d times in level 1", id, count)
		}
	}

	validParents := make(map[int]bool)
	for _, e := range asset.Levels[1].Entries {
		validParents[e.ID] = true
	}
	for _, e := range asset.Levels[2].Entries {
		for _, id := range e.SourceIDs {
			if !validParents[id] {
				t.Fatalf("level 2 references invalid guide %d", id)
			}
		}
	}
}

func TestLODSelectionBlendsWithoutPop(t *testing.T) {
	asset := Generate(1500, 13, 16)
	BuildGuideLevels(asset, []int{64, 32, 16}, 13)
	camera := NewCamera(asset.Camera, 128, 128, 10)
	before, _ := asset.RenderLOD(camera, 10-1e-4)
	after, _ := asset.RenderLOD(camera, 10+1e-4)

	var sum, max float64
	for i := range before.Data {
		d := math.Abs(before.Data[i] - after.Data[i])
		sum += d
		if d > max {
			max = d
		}
	}
	mean := sum / float64(len(before.Data))
	if mean > 1e-3 || max > 0.25 {
		t.Fatalf("level boundary pops: mean=%.6f max=%.6f", mean, max)
	}
}

func TestDollyCoverageIsContinuous(t *testing.T) {
	asset := Generate(10000, 21, 24)
	BuildGuideLevels(asset, GuideLevelCounts(256), 21)
	report := AnalyzeDolly(asset, 160, 160, 25)
	if !report.Continuous {
		t.Fatalf("coverage is not continuous: %s", report.Summary())
	}
	if report.MaxCoverageStep > 0.08 {
		t.Fatalf("coverage step too large: %.4f", report.MaxCoverageStep)
	}
	if report.MinCoverageRatio < 0.55 {
		t.Fatalf("silhouette collapsed: min ratio %.4f", report.MinCoverageRatio)
	}
	if report.FarIoU < 0.45 || report.FarSilhouetteW < 4 || report.FarSilhouetteH < 4 {
		t.Fatalf("far silhouette collapsed: IoU=%.4f size=%dx%d",
			report.FarIoU, report.FarSilhouetteW, report.FarSilhouetteH)
	}
}

func TestPipelineArtifacts(t *testing.T) {
	dir := t.TempDir()
	result, err := RunPipeline(PipelineOptions{
		OutDir:        dir,
		StrandCount:   2000,
		GuideCount:    64,
		Seed:          5,
		Points:        16,
		Width:         192,
		Height:        192,
		AnalysisSize:  128,
		DollySamples:  13,
		ErrorDistance: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, path := range result.Files {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s artifact missing at %s: %v", name, path, err)
		}
	}
	for _, name := range []string{"strands.json", "guides.json", "lod.json", "error_heatmap.png"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}

	var guides GuideDocument
	data, err := os.ReadFile(filepath.Join(dir, "guides.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &guides); err != nil {
		t.Fatal(err)
	}
	if guides.GuideCount != 64 || len(guides.Guides) != 64 {
		t.Fatalf("guide document count mismatch: %+v", guides)
	}
	var weightSum float64
	var sourceSum int
	for _, g := range guides.Guides {
		weightSum += g.DensityWeight
		sourceSum += g.SourceCount
	}
	if math.Abs(weightSum-1) > 1e-9 || sourceSum != 2000 {
		t.Fatalf("guide density weights invalid: sum=%.12f sources=%d", weightSum, sourceSum)
	}
}
