package main

import (
	"math"
	"testing"
)

func TestHomographyRoundTrip(t *testing.T) {
	src := [4]Vec{{0, 0}, {100, 0}, {100, 200}, {0, 200}}
	dst := [4]Vec{{40, 60}, {240, 20}, {180, 360}, {10, 300}}
	h := findHomography(src, dst)
	inv, ok := matInv(h)
	if !ok {
		t.Fatal("singular homography")
	}
	for i, s := range src {
		x, y := matApply(h, s.X, s.Y)
		if math.Abs(x-dst[i].X) > 1e-6 || math.Abs(y-dst[i].Y) > 1e-6 {
			t.Fatalf("point %d mapped to %.3f %.3f want %.3f %.3f", i, x, y, dst[i].X, dst[i].Y)
		}
		rx, ry := matApply(inv, x, y)
		if math.Abs(rx-s.X) > 1e-6 || math.Abs(ry-s.Y) > 1e-6 {
			t.Fatalf("inverse mismatch %.3f %.3f", rx, ry)
		}
	}
}

func TestBarbAnchorFixed(t *testing.T) {
	f := buildFeather()
	for _, b := range f.Barbs {
		p0 := barbPoint(b, 0)
		if math.Abs(p0.X-b.Base.X) > 1e-9 || math.Abs(p0.Y-b.Base.Y) > 1e-9 {
			t.Fatalf("barb base moved: %v vs %v", p0, b.Base)
		}
		a := f.Anchors[b.Anchor]
		if math.Abs(a.X-b.Base.X) > 1e-9 || math.Abs(a.Y-b.Base.Y) > 1e-9 {
			t.Fatalf("anchor mismatch %v vs %v", a, b.Base)
		}
	}
}
