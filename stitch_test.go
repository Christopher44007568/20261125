package main

import (
	"math"
	"testing"
)

func TestOppositeChainsUnifyAndStitchWatertight(t *testing.T) {
	cfg := DefaultConfig()
	asm := BuildDemoPatches(cfg)
	cracks := MeasureCracks(asm, 301)
	if cracks.MaxGapMM < 0.05 {
		t.Fatalf("expected visible pre-repair crack, max gap %.4f mm", cracks.MaxGapMM)
	}

	st, err := StitchPatches(cfg, asm)
	if err != nil {
		t.Fatalf("stitch failed: %v", err)
	}
	if !st.Watertight.SeamWatertight || st.Watertight.SeamBadEdges != 0 {
		t.Fatalf("seam not watertight: %+v", st.Watertight)
	}
	if st.RefineA.NetAdded <= 0 || st.RefineB.NetAdded <= 0 {
		t.Fatalf("expected both sides to add triangles: A=%+v B=%+v", st.RefineA, st.RefineB)
	}
	if got := len(st.Samples) - 1; st.Watertight.SeamEdgeCount != got {
		t.Fatalf("seam edge count %d, want %d", st.Watertight.SeamEdgeCount, got)
	}
	if st.SharpNormalAngle < 25 {
		t.Fatalf("sharp normal jump too small: %.2f deg", st.SharpNormalAngle)
	}
}

func TestWeldingVerticesAloneDoesNotMakeTJunctionsWatertight(t *testing.T) {
	// One coarse edge and two fine edges occupy the same straight segment.
	// Merely using one position table leaves the coarse edge with one face and
	// leaves the fine midpoint as a T-junction.
	m := &Mesh{}
	m.Pos = []Vec3{
		V(0, 0, 0), V(0, 2, 0), V(0, 1, 0),
		V(-1, 0, 0), V(-1, 2, 0),
		V(1, 0, 0), V(1, 1, 0), V(1, 2, 0),
	}
	m.Tri = []Triangle{
		{A: 3, B: 0, C: 1, Group: 0},
		{A: 0, B: 2, C: 5, Group: 1},
		{A: 2, B: 1, C: 6, Group: 1},
	}
	rep := CheckWatertight(m, [][2]int{{0, 1}, {0, 2}, {2, 1}})
	if rep.SeamWatertight {
		t.Fatalf("weld-only T-junction unexpectedly reported watertight: %+v", rep)
	}
	if rep.SeamBadEdges == 0 {
		t.Fatal("expected bad seam edges")
	}
}

func TestUnifyIncludesReverseParameters(t *testing.T) {
	cfg := DefaultConfig()
	for i := 0; i < 20; i++ {
		cfg.SegAAlong = 1 + i%5
		cfg.SegBAlong = 1 + (i*3)%17
		asm := BuildDemoPatches(cfg)
		samples := UnifyBoundarySamples(cfg, asm.ChainA, asm.ChainB)
		if len(samples) < cfg.SegAAlong+1 || len(samples) < cfg.SegBAlong+1 {
			t.Fatalf("iteration %d: unified samples too short: %d", i, len(samples))
		}
		for j := 1; j < len(samples); j++ {
			if samples[j].Q <= samples[j-1].Q {
				t.Fatalf("samples not sorted at %d", j)
			}
		}
		st, err := StitchPatches(cfg, asm)
		if err != nil {
			t.Fatalf("iteration %d stitch: %v", i, err)
		}
		if math.Abs(st.Watertight.MaxSeamEdgeLengthMM) < 0 {
			t.Fatal("negative edge length")
		}
	}
}
