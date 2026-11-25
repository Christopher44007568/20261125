package main

import (
	"math"
	"testing"
)

func TestStableDepthOrdering(t *testing.T) {
	samples := []DeepSample{
		{Depth: 1.0, Sequence: 7, LayerID: 1, Color: RGB(1, 0, 0), Alpha: 1},
		{Depth: 1.0, Sequence: 7, LayerID: 2, Color: RGB(0, 1, 0), Alpha: 1},
		{Depth: 0.5, Sequence: 9, LayerID: 3, Color: RGB(0, 0, 1), Alpha: 1},
	}
	sortDeep(samples)
	if samples[0].LayerID != 3 {
		t.Fatalf("nearest-to-back sample first: got layer %d", samples[0].LayerID)
	}
	if samples[1].LayerID != 1 || samples[2].LayerID != 2 {
		t.Fatalf("equal-key samples were not kept stable: %d, %d", samples[1].LayerID, samples[2].LayerID)
	}
}

func TestCompositeUsesFrontmostTransparentSurface(t *testing.T) {
	bg := RGB(0, 0, 0)
	samples := []DeepSample{
		{Depth: 2, Sequence: 2, Kind: KindGlass, Color: RGB(1, 0, 0), Alpha: 1},
		{Depth: 1, Sequence: 1, Kind: KindHair, Color: RGB(0, 0, 1), Alpha: 1},
	}
	sortDeep(samples)
	got, _ := composite(samples, bg)
	if math.Abs(got.R-1) > 1e-9 || got.B > 1e-9 {
		t.Fatalf("front sample should win: got %+v", got)
	}
}

func TestHairTipsCarryTheirOwnMotion(t *testing.T) {
	cfg := Config{
		Width: 1280, Height: 800, Supersample: 1, Samples: 3,
		Shutter: 0.06, FrameTime: 0.665, Seed: 20261125,
	}
	scene := NewScene(cfg.Width, cfg.Height, cfg.Seed)
	start := cfg.FrameTime - cfg.Shutter/2
	end := cfg.FrameTime + cfg.Shutter/2

	bestTip, bestRoot := 0.0, 0.0
	for i := range scene.Strands {
		st := &scene.Strands[i]
		rootA := st.pointAt(0, start, scene.Pivot)
		rootB := st.pointAt(0, end, scene.Pivot)
		tipA := st.pointAt(1, start, scene.Pivot)
		tipB := st.pointAt(1, end, scene.Pivot)
		rootTravel := math.Hypot(rootB.X-rootA.X, rootB.Y-rootA.Y)
		tipTravel := math.Hypot(tipB.X-tipA.X, tipB.Y-tipA.Y)
		bestRoot = math.Max(bestRoot, rootTravel)
		bestTip = math.Max(bestTip, tipTravel)
		if tipTravel < 8 {
			t.Fatalf("strand %d has no visible shutter trail: %.2f px", st.ID, tipTravel)
		}
	}
	if bestTip <= bestRoot {
		t.Fatalf("longest tip trail %.2f px does not exceed root travel %.2f px", bestTip, bestRoot)
	}
}

func TestRendererIsDeterministic(t *testing.T) {
	cfg := Config{
		Width: 96, Height: 64, Supersample: 1, Samples: 2,
		Shutter: 0.04, FrameTime: 0.665, Seed: 42,
	}
	a, err := Render(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Render(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Color) != len(b.Color) || len(a.Motion) != len(b.Motion) {
		t.Fatal("unexpected output size")
	}
	for i := range a.Color {
		if a.Color[i] != b.Color[i] || a.Motion[i] != b.Motion[i] {
			t.Fatalf("render differs at pixel %d", i)
		}
	}
}
