package main

import (
	"math"
	"testing"
)

func roundTripDE(fwdCAT, bwdCAT Mat3, fwdT, bwdT float64, bwdFrom, bwdTo Vec3, rgb Vec3) float64 {
	s := Scenario{
		FwdCAT: fwdCAT, FwdT: fwdT,
		BwdCAT: bwdCAT, BwdT: bwdT,
		BwdFrom: bwdFrom, BwdTo: bwdTo,
	}
	return DeltaEFromSRGB(rgb, s.RoundTrip(rgb))
}

// Paired white points + paired CAT must round-trip every patch to
// well below just-noticeable difference (dE2000 < 0.05).
func TestPairedRoundTripIsLossless(t *testing.T) {
	for _, p := range patches() {
		dE := roundTripDE(CATBradford, CATBradford, 1, 1, WhiteD50, WhiteD65, p.RGB)
		if dE > 0.05 {
			t.Errorf("patch %q: paired round trip dE2000 = %.4f, want < 0.05", p.Name, dE)
		}
	}
}

// Adapting D65->D50 but returning without inverse adaptation must shift
// D65 white visibly (yellowish), dE2000 well above 3 JND.
func TestMismatchedWhitePointShiftsWhite(t *testing.T) {
	dE := roundTripDE(CATBradford, CATBradford, 1, 1, WhiteD65, WhiteD65, Vec3{1, 1, 1})
	if dE < 3 {
		t.Errorf("mismatched white point: white dE2000 = %.4f, want > 3", dE)
	}
	out := Scenario{
		FwdCAT: CATBradford, FwdT: 1,
		BwdCAT: CATBradford, BwdT: 1,
		BwdFrom: WhiteD65, BwdTo: WhiteD65,
	}.RoundTrip(Vec3{1, 1, 1})
	if !(out[2] < out[0] && out[2] < out[1]) {
		t.Errorf("expected yellowish shift (blue channel lowest), got %v", out)
	}
}

// Swapping the CAT between legs must leave a measurable residual even
// though both matrices are "valid" adaptation transforms.
func TestMismatchedMatrixLeavesResidual(t *testing.T) {
	blue := Vec3{0, 0, 1}
	dE := roundTripDE(CATBradford, CATXYZScale, 1, 1, WhiteD50, WhiteD65, blue)
	if dE < 0.5 {
		t.Errorf("Bradford vs XYZ-scale on blue: dE2000 = %.4f, want > 0.5", dE)
	}
	dE = roundTripDE(CATBradford, CATCAT02, 1, 1, WhiteD50, WhiteD65, blue)
	if dE < 0.2 {
		t.Errorf("Bradford vs CAT02 on blue: dE2000 = %.4f, want > 0.2", dE)
	}
}

// Partial adaptation applied with the same strength both ways is not
// self-inverting: error must be zero at t=0 and t=1 and peak in between.
func TestPartialAdaptationPeaksMidway(t *testing.T) {
	white := Vec3{1, 1, 1}
	at := func(tau float64) float64 {
		return roundTripDE(CATBradford, CATBradford, tau, tau, WhiteD50, WhiteD65, white)
	}
	e0, e25, e50, e75, e1 := at(0), at(0.25), at(0.5), at(0.75), at(1)
	if e0 > 0.01 || e1 > 0.01 {
		t.Errorf("endpoints must be lossless: t=0 %.4f, t=1 %.4f", e0, e1)
	}
	if e50 <= e0 || e50 <= e1 {
		t.Errorf("mid-range t=0.5 dE %.4f must exceed endpoints %.4f/%.4f", e50, e0, e1)
	}
	if math.Abs(e25-e75) > 1e-9 {
		t.Errorf("error should be symmetric: t=0.25 %.6f vs t=0.75 %.6f", e25, e75)
	}
}

// Full forward + zero backward adaptation must equal the "mismatched
// white" scenario and be far worse than any partial-strength pairing.
func TestNoBackAdaptationIsWorst(t *testing.T) {
	white := Vec3{1, 1, 1}
	none := roundTripDE(CATBradford, CATBradford, 1, 0, WhiteD50, WhiteD65, white)
	half := roundTripDE(CATBradford, CATBradford, 0.5, 0.5, WhiteD50, WhiteD65, white)
	if none < 10 {
		t.Errorf("no back adaptation: white dE2000 = %.4f, want > 10", none)
	}
	if none <= half {
		t.Errorf("no back adaptation %.4f should exceed partial t=0.5 %.4f", none, half)
	}
}

// Sanity check of the CIEDE2000 implementation against the published
// Sharma test pair (expected dE2000 = 2.0425).
func TestDeltaE2000ReferencePair(t *testing.T) {
	lab1 := Vec3{50.0000, 2.6772, -79.7751}
	lab2 := Vec3{50.0000, 0.0000, -82.7485}
	if dE := DeltaE2000(lab1, lab2); math.Abs(dE-2.0425) > 0.0001 {
		t.Errorf("dE2000 reference pair = %.4f, want 2.0425", dE)
	}
}
