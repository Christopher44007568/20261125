package main

import (
	"math"
	"testing"
)

func TestHybridBlendDoesNotAddOpacity(t *testing.T) {
	shell := HDR{R: 0.8, G: 0, B: 0, A: 0.8}
	strand := HDR{R: 0, G: 0.8, B: 0, A: 0.8}
	got, ok := blendHybridPixel(shell, strand, 0.5)
	if !ok {
		t.Fatal("expected a covered pixel")
	}
	if math.Abs(got.A-0.8) > 1e-9 {
		t.Fatalf("alpha grew to %v; representation layers must not add", got.A)
	}
	if math.Abs(got.R-0.4) > 1e-9 || math.Abs(got.G-0.4) > 1e-9 {
		t.Fatalf("unexpected premultiplied color: %#v", got)
	}
}

func TestHybridBlendPreservesSingleSidedTip(t *testing.T) {
	shell := HDR{R: 1, G: 0.8, B: 0.4, A: 1}
	empty := HDR{}
	got, ok := blendHybridPixel(shell, empty, 0.5)
	if !ok {
		t.Fatal("expected a covered pixel")
	}
	if math.Abs(got.A-1) > 1e-9 {
		t.Fatalf("single-sided coverage faded to %v", got.A)
	}
	if math.Abs(got.R-1) > 1e-9 || math.Abs(got.G-0.8) > 1e-9 {
		t.Fatalf("single-sided color was diluted: %#v", got)
	}
}

func TestMaskMetrics(t *testing.T) {
	a := Mask{W: 4, H: 1, A: []bool{true, true, false, false}, X0: 0, X1: 1, Y0: 0, Y1: 0}
	b := Mask{W: 4, H: 1, A: []bool{true, false, true, false}, X0: 0, X1: 2, Y0: 0, Y1: 0}
	s := compareMasks(a, b)
	if math.Abs(s.IoU-1.0/3.0) > 1e-9 {
		t.Fatalf("IoU = %v", s.IoU)
	}
	if math.Abs(s.Xor-2.0/3.0) > 1e-9 {
		t.Fatalf("XOR = %v", s.Xor)
	}
}
