package main

import (
	"image"
	"testing"
)

func TestPremultipliedComposite(t *testing.T) {
	l := newLayer(3, 3)
	l.stamp(1, 1, 2.2, 0.5, RGB{1, 0.5, 0.25})
	edgeAlpha := 0.0
	for _, p := range []struct{ x, y int }{{0, 0}, {2, 0}, {0, 2}, {2, 2}} {
		a := l.P[l.idx(p.x, p.y)+3]
		if a < 0 || a > 1 {
			t.Fatalf("alpha out of range: %f", a)
		}
		if a > edgeAlpha {
			edgeAlpha = a
		}
	}
	if edgeAlpha == 0 {
		t.Fatal("expected antialiased edge coverage")
	}
	bg := image.NewRGBA(image.Rect(0, 0, 3, 3))
	for i := range bg.Pix {
		bg.Pix[i] = 40
		if i%4 == 3 {
			bg.Pix[i] = 255
		}
	}
	out := composite(bg, l)
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			i := l.idx(x, y)
			a := l.P[i+3]
			if a > 1 {
				t.Fatal("coverage exceeded one")
			}
			oi := out.PixOffset(x, y)
			if out.Pix[oi+3] != 255 {
				t.Fatal("composite background must be opaque")
			}
			wantR := l.P[i] + (40.0/255)*(1-a)
			if abs(float64(out.Pix[oi])/255-wantR) > 0.02 {
				t.Fatalf("premultiplied composite mismatch")
			}
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
