package main

import "testing"

func allConfigs() []config {
	var cs []config
	for _, ipd := range []float64{0.058, 0.064, 0.072} {
		for _, conv := range []float64{0.5, 1.5, 8.0} {
			for _, asp := range [][2]int{{16, 9}, {4, 3}, {1, 1}} {
				cs = append(cs, makeConfig(ipd, conv, asp[0], asp[1]))
			}
		}
	}
	return cs
}

func TestMatrixMatchesGeometry(t *testing.T) {
	for _, c := range allConfigs() {
		for _, x := range []float64{-c.ipd / 2, c.ipd / 2} {
			e, err := NewEye("T", x, c.in, c.conv)
			if err != nil {
				t.Fatal(err)
			}
			r := e.CheckEye(c.conv)
			if r.MaxAngleDeg > angTolDeg {
				t.Errorf("cfg %+v eye x=%+.4f: angle %.3e deg", c, x, r.MaxAngleDeg)
			}
			if r.MaxNearErr > posTolM || r.MaxFarErr > posTolM || r.MaxEdgeErr > posTolM {
				t.Errorf("cfg %+v eye x=%+.4f: pos err %.3e/%.3e/%.3e m",
					c, x, r.MaxNearErr, r.MaxFarErr, r.MaxEdgeErr)
			}
		}
	}
}

func TestDisparitySignNeverFlips(t *testing.T) {
	for _, c := range allConfigs() {
		el, _ := NewEye("L", -c.ipd/2, c.in, c.conv)
		er, _ := NewEye("R", +c.ipd/2, c.in, c.conv)
		disp := func(z float64) float64 {
			p := Vec3{0, 0, -z}
			return el.project(p).X - er.project(p).X
		}
		if d := disp(0.5 * c.conv); d <= 0 {
			t.Errorf("cfg %+v: near disparity %e, want > 0", c, d)
		}
		if d := disp(2.0 * c.conv); d >= 0 {
			t.Errorf("cfg %+v: far disparity %e, want < 0", c, d)
		}
		if d := disp(c.conv); d > 1e-9 || d < -1e-9 {
			t.Errorf("cfg %+v: convergence disparity %e, want ~0", c, d)
		}
	}
}

// TestCheckerCatchesSabotage proves the comparison is not vacuous:
// a projection matrix built with the wrong convergence shift must be
// flagged by the geometric reference check and breaks zero-parallax.
func TestCheckerCatchesSabotage(t *testing.T) {
	c := makeConfig(0.064, 1.5, 16, 9)
	good, _ := NewEye("L", -c.ipd/2, c.in, c.conv)
	bad := good
	wrong := EyeFrustum(c.in, +c.ipd/2, c.conv) // shift the wrong way
	bad.Proj = PerspectiveOffCenter(wrong)      // matrix disagrees with reference
	inv, ok := Invert4(Mul4(bad.Proj, bad.View))
	if !ok {
		t.Fatal("singular")
	}
	bad.InvPV = inv

	r := bad.CheckEye(c.conv)
	if r.MaxAngleDeg < 1.0 {
		t.Errorf("sabotaged frustum not detected: max angle %.3e deg", r.MaxAngleDeg)
	}
	p := Vec3{0, 0, -c.conv}
	if d := good.project(p).X - bad.project(p).X; d < 1e-3 && d > -1e-3 {
		t.Error("sabotaged eye must break zero-parallax at convergence")
	}
}
