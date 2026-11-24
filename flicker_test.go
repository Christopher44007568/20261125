package main

import (
	"os"
	"path/filepath"
	"testing"
)

func collect(t *testing.T, cfg Config, sceneName, motionName, strategy string) SceneResult {
	t.Helper()
	base := buildScene(sceneName, cfg)
	var mot Motion
	for _, m := range allMotions() {
		if m.Name == motionName {
			mot = m
		}
	}
	gt := renderGroundTruthTrack(cfg, base, mot)
	_, eyes, inter, falseStab := runStrategy(cfg, base, mot, strategy, gt)
	return SceneResult{
		Scene: sceneName, Motion: motionName, Strategy: strategy,
		Eyes: eyes, InterEyeResidual: inter, FalseStability: falseStab,
	}
}

func find(results []SceneResult, strategy string) SceneResult {
	for _, r := range results {
		if r.Strategy == strategy {
			return r
		}
	}
	return SceneResult{}
}

func TestGroundTruthIsTemporallyStableWhenCameraStatic(t *testing.T) {
	cfg := testConfig()
	r := collect(t, cfg, sceneTriangles, "static", stratGT)
	for _, e := range r.Eyes {
		if e.FlickerPixels != 0 {
			t.Fatalf("GT flicker = %d pixels, want 0", e.FlickerPixels)
		}
		if e.MaxLumVar != 0 {
			t.Fatalf("GT max variance = %v, want 0", e.MaxLumVar)
		}
		if e.RMSE > 1e-12 {
			t.Fatalf("GT RMSE against itself = %v", e.RMSE)
		}
	}
}

func TestLowSamplingFlickersMoreThanGroundTruth(t *testing.T) {
	cfg := testConfig()
	var totalFlicker int
	for _, scene := range allScenes() {
		r := collect(t, cfg, scene, "slow-turn", stratIndependent)
		for _, e := range r.Eyes {
			totalFlicker += e.FlickerPixels
			if e.PeriphVar <= 0 {
				t.Fatalf("%s: expected peripheral temporal variance, got 0", scene)
			}
			if e.RelSharpness <= 0 {
				t.Fatalf("%s: sharpness not measured", scene)
			}
		}
	}
	if totalFlicker == 0 {
		t.Fatal("foveated renderer produced no flicker pixels across scenes")
	}
}

func TestBlurFlaggedAsFalseStability(t *testing.T) {
	cfg := testConfig()
	var flagged int
	for _, scene := range allScenes() {
		r := collect(t, cfg, scene, "fast-saccade", stratBlurred)
		if r.FalseStability {
			flagged++
		}
		for _, e := range r.Eyes {
			if e.RelSharpness >= 1 {
				t.Fatalf("%s: blur retained full sharpness %.3f", scene, e.RelSharpness)
			}
		}
	}
	if flagged == 0 {
		t.Fatal("heavy blur was never flagged as false stability")
	}

	if got := blurFalseStability(stratBlurred, 0.02, 1.0, 0.4, 0.001, 0.002); !got {
		t.Fatal("blurFalseStability should flag heavy blur with reduced variance")
	}
	if got := blurFalseStability(stratIndependent, 0.02, 1.0, 0.4, 0.001, 0.002); got {
		t.Fatal("blurFalseStability must only apply to the blurred strategy")
	}
	if got := blurFalseStability(stratBlurred, 0.02, 1.0, 0.9, 0.001, 0.002); got {
		t.Fatal("blurFalseStability must not flag when sharpness is retained")
	}
}

func TestSharedCoverageIsBinocularConsistent(t *testing.T) {
	cfg := testConfig()
	for _, scene := range allScenes() {
		base := buildScene(scene, cfg)
		mot := allMotions()[2]
		gt := renderGroundTruthTrack(cfg, base, mot)
		work, _, _, _ := runStrategy(cfg, base, mot, stratShared, gt)
		indWork, _, _, _ := runStrategy(cfg, base, mot, stratIndependent, gt)
		var lf, rf, ilf, irf []*Frame
		for fr := range work {
			lf = append(lf, work[fr].left)
			rf = append(rf, work[fr].right)
			ilf = append(ilf, indWork[fr].left)
			irf = append(irf, indWork[fr].right)
		}
		sharedDiv := interEyeDivergence(cfg, lf, rf)
		indDiv := interEyeDivergence(cfg, ilf, irf)
		for fr := range work {
			for i := range work[fr].left.Pix {
				if work[fr].left.Pix[i] != work[fr].right.Pix[i] {
					t.Fatalf("%s frame %d pixel %d: shared eyes diverged", scene, fr, i)
				}
			}
		}
		if sharedDiv != 0 {
			t.Fatalf("%s: shared inter-eye divergence = %v, want 0", scene, sharedDiv)
		}
		if indDiv <= 0 {
			t.Fatalf("%s: independent baseline inter-eye divergence not measured", scene)
		}
	}
}

func TestDeterministicRunsAndArtifacts(t *testing.T) {
	cfg := testConfig()
	cfg.OutDir = filepath.Join(t.TempDir(), "out")
	first := runAll(cfg)
	second := runAll(cfg)
	if len(first) != len(second) || len(first) == 0 {
		t.Fatalf("result length mismatch: %d vs %d", len(first), len(second))
	}
	for i := range first {
		a, b := first[i], second[i]
		for e := range a.Eyes {
			if a.Eyes[e].MaxLumVar != b.Eyes[e].MaxLumVar ||
				a.Eyes[e].FlickerPixels != b.Eyes[e].FlickerPixels {
				t.Fatalf("non-deterministic result at %d", i)
			}
		}
	}
	writeSummaryCSV(cfg, first)
	writeMarkdownReport(cfg, first)
	paths := []string{
		filepath.Join(cfg.OutDir, "summary.csv"),
		filepath.Join(cfg.OutDir, "report.md"),
		filepath.Join(cfg.OutDir, sceneTriangles, "static", stratIndependent, "flicker-heatmap-left.png"),
		filepath.Join(cfg.OutDir, sceneHighlights, "slow-turn", stratShared, "timeseries-left.csv"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing artifact %s: %v", p, err)
		}
	}
}
