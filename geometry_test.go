package main

import (
	"math"
	"testing"
)

func TestDefaultProfileMatchesReferenceTopology(t *testing.T) {
	p := DefaultParams()
	profile := buildProfile(p)
	if got := len(profile.Outer); got != 51 {
		t.Fatalf("outer profile points: got %d want 51", got)
	}
	mesh := buildMesh(profile)
	if err := mesh.Validate(); err != nil {
		t.Fatal(err)
	}
	topo := mesh.Topology()
	if topo.Vertices != 204 || topo.Triangles != 404 || topo.Edges != 606 {
		t.Fatalf("unexpected topology: %+v", topo)
	}
}

func TestDefaultVolumeAndDiagnostics(t *testing.T) {
	p := DefaultParams()
	profile := buildProfile(p)
	mesh := buildMesh(profile)
	volume := mesh.Volume()
	if math.Abs(volume-873.4230958526659) > 1e-8 {
		t.Fatalf("volume: got %.12f", volume)
	}
	diag := analyzeSection(profile)
	if diag.Inverted != 48 {
		t.Fatalf("inverted section triangles: got %d want 48", diag.Inverted)
	}
	if len(diag.Intersections) != 1 || diag.LoopCount != 1 {
		t.Fatalf("intersections=%d loops=%d", len(diag.Intersections), diag.LoopCount)
	}
	invertedWalls := 0
	for _, inv := range innerWallInverted(profile) {
		if inv {
			invertedWalls++
		}
	}
	if invertedWalls != 48 {
		t.Fatalf("inverted inner wall segments: got %d want 48", invertedWalls)
	}
	if got, want := profile.AnalyticVolume(p.Offset), 873.4247779607693; math.Abs(got-want) > 1e-9 {
		t.Fatalf("analytic volume: got %.12f want %.12f", got, want)
	}
}

func TestThresholdIsAboveCornerRadius(t *testing.T) {
	p := DefaultParams()
	p.Offset = 1.25
	if diag := analyzeSection(buildProfile(p)); len(diag.Intersections) != 0 || diag.Inverted != 0 {
		t.Fatalf("offset at radius should be degenerate, not inverted: %+v", diag)
	}
	p.Offset = 1.275
	if diag := analyzeSection(buildProfile(p)); len(diag.Intersections) != 1 || diag.Inverted != 48 {
		t.Fatalf("offset above radius should expose defects: inv=%d hits=%d",
			diag.Inverted, len(diag.Intersections))
	}
}
