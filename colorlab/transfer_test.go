package colorlab

import (
	"math"
	"testing"
)

func TestTransferKnownValues(t *testing.T) {
	tests := []struct {
		name string
		fn   func(float64) float64
		in   float64
		want float64
	}{
		{"sRGB decode midpoint", sRGBToLinear, 0.5, 0.21404114048223255},
		{"sRGB encode midpoint", linearToSRGB, 0.21404114048223255, 0.5},
		{"PQ decode 0.5", pqToLinear, 0.5, 0.009224570899406526},
		{"PQ encode 0.5", linearToPQ, 0.5, 0.9265467040826304},
		{"HLG decode 0.5", hlgToLinear, 0.5, 1.0 / 12.0},
		{"HLG encode 0.75", linearToHLG, 0.75, 0.9470986439694851},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.in); math.Abs(got-tt.want) > 1e-12 {
				t.Fatalf("got %.15g, want %.15g", got, tt.want)
			}
		})
	}
}

func TestTransferBoundariesAndMonotonicity(t *testing.T) {
	checks := CheckMonotonicity()
	if len(checks) != 8 {
		t.Fatalf("got %d curve checks, want 8", len(checks))
	}
	for _, check := range checks {
		if !check.Monotonic {
			t.Errorf("%s is not monotonic (backtrack %.3g)", check.Name, check.MaxBacktrack)
		}
		if !check.ZeroOK || !check.OneOK {
			t.Errorf("%s endpoints invalid: zero=%t one=%t",
				check.Name, check.ZeroOK, check.OneOK)
		}
	}

	boundaries := []struct {
		name    string
		decode  func(float64) float64
		encode  func(float64) float64
		encoded float64
		linear  float64
	}{
		{"sRGB", sRGBToLinear, linearToSRGB, srgbEncodedThreshold, srgbLinearThreshold},
		{"Rec.2020", rec2020ToLinear, linearToRec2020, 4.5 * rec2020Beta, rec2020Beta},
		{"HLG", hlgToLinear, linearToHLG, 0.5, 1.0 / 12.0},
	}
	for _, boundary := range boundaries {
		if got := boundary.decode(boundary.encoded); math.Abs(got-boundary.linear) > 1e-12 {
			t.Errorf("%s decode boundary = %.15g, want %.15g",
				boundary.name, got, boundary.linear)
		}
		if got := boundary.encode(boundary.linear); math.Abs(got-boundary.encoded) > 1e-12 {
			t.Errorf("%s encode boundary = %.15g, want %.15g",
				boundary.name, got, boundary.encoded)
		}
	}
}

func TestRoundTripEndpointsAndMidpoint(t *testing.T) {
	values := []RGB{
		{0, 0, 0},
		{1, 1, 1},
		{0.5, 0.5, 0.5},
		{0.8, 0.2, 0.1},
	}
	for _, from := range AllSpaces() {
		for _, to := range AllSpaces() {
			for _, source := range values {
				reference := Convert(source, SRGB, from)
				got := Convert(Convert(reference, from, to), to, from)
				for _, delta := range channelDeltas(reference, got) {
					if delta > 1e-9 {
						t.Fatalf("%s -> %s -> %s with %+v: peak %.3g",
							from, to, from, source, delta)
					}
				}
			}
		}
	}
}

func TestRoundTripNativePrimaries(t *testing.T) {
	linearColors := []RGB{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
		{0.25, 0.5, 0.75},
	}
	for _, from := range AllSpaces() {
		for _, to := range AllSpaces() {
			for _, linear := range linearColors {
				reference := Encode(linear, from)
				got := Convert(Convert(reference, from, to), to, from)
				for _, delta := range channelDeltas(reference, got) {
					if delta > 1e-9 {
						t.Fatalf("%s -> %s -> %s with linear %+v: peak %.3g",
							from, to, from, linear, delta)
					}
				}
			}
		}
	}
}
