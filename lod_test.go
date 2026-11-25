package main

import (
	"testing"
)

func TestHairLODCountsAreNestedAndDecreasing(t *testing.T) {
	model := NewHairModel()
	if !(model.Tris0 > model.Tris1 && model.Tris1 > model.Tris2) {
		t.Fatalf("expected decreasing triangles, got %d/%d/%d",
			model.Tris0, model.Tris1, model.Tris2)
	}
	if len(model.LOD0) != len(model.LOD1)*6 {
		t.Fatalf("each lock should own six strands: %d strands, %d locks",
			len(model.LOD0), len(model.LOD1))
	}
	if len(model.LOD1) != len(model.LOD2)*4 {
		t.Fatalf("each clump should own four locks: %d locks, %d clumps",
			len(model.LOD1), len(model.LOD2))
	}
	for _, s := range model.LOD0 {
		if s.Parent < 0 || s.Parent >= len(model.LOD1) {
			t.Fatalf("strand %d has invalid lock parent %d", s.ID, s.Parent)
		}
	}
}

func TestHairGenerationIsDeterministic(t *testing.T) {
	a := NewHairModel()
	b := NewHairModel()
	if len(a.LOD0) != len(b.LOD0) {
		t.Fatalf("strand count changed: %d vs %d", len(a.LOD0), len(b.LOD0))
	}
	for i := range a.LOD0 {
		if len(a.LOD0[i].Points) != len(b.LOD0[i].Points) {
			t.Fatalf("strand %d point count differs", i)
		}
		for j := range a.LOD0[i].Points {
			if a.LOD0[i].Points[j] != b.LOD0[i].Points[j] {
				t.Fatalf("strand %d point %d is not deterministic", i, j)
			}
		}
	}
}

func TestLODPathUsesBothAdjacentLevels(t *testing.T) {
	sawMorph01 := false
	sawMorph12 := false
	sawStable0 := false
	sawStable2 := false
	for d := 1.5; d <= 5.4; d += 0.03 {
		lod := BuildLODFrame(NewHairModel(), d)
		switch lod.State {
		case "LOD0-stable":
			sawStable0 = true
		case "geomorph-0-to-1":
			sawMorph01 = true
			if lod.Blend < 0 || lod.Blend > 1 {
				t.Fatalf("bad 0->1 blend %f at distance %f", lod.Blend, d)
			}
		case "geomorph-1-to-2":
			sawMorph12 = true
		case "LOD2-stable":
			sawStable2 = true
		}
	}
	if !sawStable0 || !sawMorph01 || !sawMorph12 || !sawStable2 {
		t.Fatalf("missing LOD phase: stable0=%v morph01=%v morph12=%v stable2=%v",
			sawStable0, sawMorph01, sawMorph12, sawStable2)
	}
}

func TestCameraPathIsFixedAndMonotonicEnough(t *testing.T) {
	path := DefaultCameraPath(320, 180)
	prev := 0.0
	for i := 0; i <= 20; i++ {
		cam := path.At(float64(i) / 20)
		d := vlen(vsub(path.Target, cam.Pos))
		if i > 0 && d <= prev {
			t.Fatalf("distance did not increase at sample %d: %f <= %f", i, d, prev)
		}
		prev = d
	}
	if path.At(0.37).Pos != path.At(0.37).Pos {
		t.Fatal("camera path must be reproducible")
	}
}
