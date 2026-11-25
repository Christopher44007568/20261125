package main

import "testing"

func TestTransferRoundTrips(t *testing.T) {
	for _, nits := range []float64{0.1, 1, 18, 100, 203, 1000, 4000} {
		got := pqDecodeNits(pqEncodeNits(nits))
		if absFloat(got-nits) > 0.01*maxFloat(1, nits) {
			t.Fatalf("PQ round trip for %v nits: got %v", nits, got)
		}
	}

	for _, e := range []float64{0, 0.018, 0.083333, 0.265, 0.75, 1} {
		got := hlgDecodeScene(hlgEncodeScene(e))
		if absFloat(got-e) > 1e-9 {
			t.Fatalf("HLG round trip for %v: got %v", e, got)
		}
	}
}

func TestDisplayGrayAnchor(t *testing.T) {
	scene := &Scene{
		Width: 1, Height: 1,
		Pixels: []RGB{{grayNits, grayNits, grayNits}},
	}
	for _, spec := range DisplaySpecs() {
		frame := RenderDisplay(scene, spec)
		r, g, b := frame.Image.Pix[0], frame.Image.Pix[1], frame.Image.Pix[2]
		if r != 118 || g != 118 || b != 118 {
			t.Fatalf("%s gray = (%d,%d,%d), want 118", spec.ID, r, g, b)
		}
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
