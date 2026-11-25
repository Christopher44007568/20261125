package section

import (
	"math"
	"strconv"
	"testing"

	"brepsection/brep"
	"brepsection/geom"
)

func planes() []geom.Plane {
	solid := brep.SampleSolid()
	c := solid.AABB().Center()
	return []geom.Plane{
		geom.NewPlane(geom.V3(0, 0, 1.25), geom.V3(0, 0, 1), "z"),
		geom.NewPlane(geom.V3(6, 0, 0), geom.V3(1, 0, 0), "x"),
		geom.NewPlane(c, geom.V3(0.3, 0.7, 0.65), "oblique"),
	}
}

// TestAllCutsClosedAndOrdered is the core contract: every direction, in both
// precision modes, yields closed contours with consistent nesting and no
// self-intersections.
func TestAllCutsClosedAndOrdered(t *testing.T) {
	solid := brep.SampleSolid()
	for _, pl := range planes() {
		for _, mode := range StandardModes {
			rep := Extract(solid, pl, mode)
			if !rep.Closed {
				t.Fatalf("%s/%s not closed: %d dangling junction(s)", pl.Name, mode.Name, rep.Junctions)
			}
			if !rep.OrderConsistent {
				t.Fatalf("%s/%s nesting order inconsistent", pl.Name, mode.Name)
			}
			if !rep.WindingConsistent {
				t.Fatalf("%s/%s same-level winding is not uniform", pl.Name, mode.Name)
			}
			if rep.SelfIntersect {
				for _, l := range rep.Loops {
					if len(l.Crossings) > 0 {
						t.Fatalf("%s/%s loop %d self-crosses: %v", pl.Name, mode.Name, l.Index, l.Crossings)
					}
				}
			}
			if len(rep.Loops) == 0 {
				t.Fatalf("%s/%s produced no loops", pl.Name, mode.Name)
			}
		}
	}
}

// TestZCutHasTwoHoles verifies the extrusion cut sees both through-holes.
func TestZCutHasTwoHoles(t *testing.T) {
	solid := brep.SampleSolid()
	pl := geom.NewPlane(geom.V3(0, 0, 1.25), geom.V3(0, 0, 1), "z")
	rep := Extract(solid, pl, StandardModes[0])
	holes := 0
	for _, l := range rep.Loops {
		if l.Level == 1 {
			holes++
		}
	}
	if holes != 2 {
		t.Fatalf("expected 2 hole loops, got %d", holes)
	}
}

// TestTangentViaIsClosed exercises the exact tangent cut through the
// circular via: the contour pair must close without crossing.
func TestTangentViaIsClosed(t *testing.T) {
	solid := brep.SampleSolid()
	pl := geom.NewPlane(geom.V3(6, 0, 0), geom.V3(1, 0, 0), "x")
	rep := Extract(solid, pl, StandardModes[0])
	if len(rep.Loops) != 2 {
		t.Fatalf("expected outer + hole loop, got %d", len(rep.Loops))
	}
	if rep.Loops[0].Level != 0 || rep.Loops[1].Level != 1 {
		t.Fatalf("unexpected levels: %d, %d", rep.Loops[0].Level, rep.Loops[1].Level)
	}
}

// TestStressModeResidualBounded ensures the float32 run stays within its
// advertised closure tolerance even though its residuals are non-zero.
func TestStressModeResidualBounded(t *testing.T) {
	solid := brep.SampleSolid()
	for _, pl := range planes() {
		rep := Extract(solid, pl, StandardModes[1])
		if rep.MaxGap > rep.Tolerance {
			t.Fatalf("%s float32 gap %s exceeds tolerance %s",
				pl.Name, fnum(rep.MaxGap), fnum(rep.Tolerance))
		}
		if !rep.Closed {
			t.Fatalf("%s float32 failed closure", pl.Name)
		}
	}
}

// TestSelfIntersectionDetector feeds an intentionally crossing bowtie loop to
// prove the detector flags what the real pipeline never produces.
func TestSelfIntersectionDetector(t *testing.T) {
	bowtie := []geom.Vec2{
		geom.V2(0, 0), geom.V2(2, 2), geom.V2(2, 0), geom.V2(0, 2),
	}
	l := &Loop{Nodes: bowtie}
	if got := selfIntersections(l); len(got) == 0 {
		t.Fatal("bowtie loop was not detected as self-intersecting")
	}
	// A simple square must pass the same check.
	square := []geom.Vec2{
		geom.V2(0, 0), geom.V2(1, 0), geom.V2(1, 1), geom.V2(0, 1),
	}
	if got := selfIntersections(&Loop{Nodes: square}); len(got) != 0 {
		t.Fatalf("simple square reported false crossings: %v", got)
	}
}

// TestDeterministicOutput confirms repeated runs yield identical topology.
func TestDeterministicOutput(t *testing.T) {
	solid := brep.SampleSolid()
	pl := geom.NewPlane(geom.V3(0, 0, 1.25), geom.V3(0, 0, 1), "z")
	first := Extract(solid, pl, StandardModes[0])
	for k := 0; k < 3; k++ {
		again := Extract(solid, pl, StandardModes[0])
		if len(again.Loops) != len(first.Loops) || again.RawSegments != first.RawSegments {
			t.Fatal("section extraction is not deterministic")
		}
	}
}

func fnum(v float64) string {
	return strconv.FormatFloat(v, 'G', 3, 64)
}

func TestFiniteGaps(t *testing.T) {
	solid := brep.SampleSolid()
	for _, pl := range planes() {
		rep := Extract(solid, pl, StandardModes[0])
		for _, l := range rep.Loops {
			if math.IsNaN(l.ClosureGap) || math.IsInf(l.ClosureGap, 0) {
				t.Fatalf("%s loop %d non-finite gap", pl.Name, l.Index)
			}
		}
	}
}
