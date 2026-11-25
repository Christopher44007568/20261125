package geom

import (
	"math"
	"testing"
)

func TestPlaneRoundTrip(t *testing.T) {
	pl := NewPlane(V3(1, 2, 3), V3(0.3, 0.7, 0.65), "p")
	// Build a point guaranteed to lie on the plane.
	q := pl.Origin.Add(pl.U.Scale(2.7)).Add(pl.V.Scale(-1.9))
	on := pl.Origin.Add(pl.U.Scale(pl.To2D(q).X)).Add(pl.V.Scale(pl.To2D(q).Y))
	if !q.Approx(on, 1e-12) {
		t.Fatalf("2D round trip lost precision: %v != %v", q, on)
	}
	if math.Abs(pl.U.Dot(pl.V)) > 1e-12 || math.Abs(pl.Normal.Dot(pl.U)) > 1e-12 {
		t.Fatal("plane basis not orthonormal")
	}
}

func TestSegmentCross(t *testing.T) {
	pl := NewPlane(V3(0, 0, 0), V3(0, 0, 1), "xy")
	a, b := V3(1, 1, -2), V3(1, 1, 4)
	tt, q, ok := pl.SegmentCross(a, b)
	if !ok || math.Abs(tt-1.0/3.0) > 1e-12 {
		t.Fatalf("bad crossing parameter %v ok=%v", tt, ok)
	}
	if math.Abs(q.Z) > 1e-12 {
		t.Fatalf("crossing not on plane: %v", q)
	}
}

func TestPointInLoop(t *testing.T) {
	square := []Vec2{V2(0, 0), V2(4, 0), V2(4, 4), V2(0, 4)}
	if PointInLoop(V2(2, 2), square) == 0 {
		t.Fatal("center should be inside square")
	}
	if PointInLoop(V2(5, 2), square) != 0 {
		t.Fatal("external point misclassified")
	}
}

func TestPolygonArea(t *testing.T) {
	ccw := []Vec2{V2(0, 0), V2(2, 0), V2(2, 2), V2(0, 2)}
	if PolygonArea(ccw) <= 0 {
		t.Fatal("CCW square should have positive area")
	}
	cw := []Vec2{V2(0, 0), V2(0, 2), V2(2, 2), V2(2, 0)}
	if PolygonArea(cw) >= 0 {
		t.Fatal("CW square should have negative area")
	}
}

func TestVecOps(t *testing.T) {
	a, b := V3(1, 2, 3), V3(4, 5, 6)
	if a.Dot(b) != 32 {
		t.Fatal("dot product mismatch")
	}
	cr := a.Cross(b)
	if cr.Approx(V3(-3, 6, -3), 1e-12) == false {
		t.Fatalf("cross product mismatch: %v", cr)
	}
	if V3(0, 3, 4).Len() != 5 {
		t.Fatal("length mismatch")
	}
}
