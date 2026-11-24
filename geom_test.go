package main

import (
	"image"
	"math"
	"testing"
)

func TestRadiusProfileEndpointsAndDerivatives(t *testing.T) {
	cfg := DefaultConfig()
	if got := cfg.Radius(0); math.Abs(got-cfg.RadiusStart) > 1e-14 {
		t.Fatalf("radius at v=0: got %.16f, want %.16f", got, cfg.RadiusStart)
	}
	if got := cfg.Radius(1); math.Abs(got-cfg.RadiusEnd) > 1e-14 {
		t.Fatalf("radius at v=1: got %.16f, want %.16f", got, cfg.RadiusEnd)
	}
	for _, v := range []float64{0, 1} {
		r1, _ := cfg.RadiusDerivatives(v)
		if math.Abs(r1) > 1e-12 {
			t.Fatalf("end derivative at v=%.0f is not zero: r'=%.3g", v, r1)
		}
	}
	previous := cfg.Radius(0)
	for i := 1; i <= 200; i++ {
		r := cfg.Radius(float64(i) / 200)
		if r < previous-1e-13 {
			t.Fatalf("radius profile is not monotone at sample %d", i)
		}
		previous = r
	}
}

func TestFilletParentSeamsAreG1AtBothContacts(t *testing.T) {
	cfg := DefaultConfig()
	report := buildReport(cfg)
	if !report.Acceptance.FilletParentSeamsPass {
		t.Fatal("default fillet failed its seam acceptance")
	}
	for _, seam := range report.Seams {
		if seam.MaxPositionErrorMM > 1e-11 {
			t.Errorf("%s: G0 error %.3e mm is too large", seam.Name, seam.MaxPositionErrorMM)
		}
		if seam.MaxTangentAngleDeg > 1e-10 {
			t.Errorf("%s: tangent mismatch %.3e deg is too large", seam.Name, seam.MaxTangentAngleDeg)
		}
		if seam.MaxNormalAngleDeg > 1e-10 {
			t.Errorf("%s: normal mismatch %.3e deg is too large", seam.Name, seam.MaxNormalAngleDeg)
		}
		if !seam.PositionContinuous || !seam.TangentContinuous || !seam.NormalOrientationValid {
			t.Errorf("%s: a continuity flag failed: %+v", seam.Name, seam)
		}
		if !seam.EndpointContinuityPassed {
			t.Errorf("%s: endpoint continuity failed: %+v", seam.Name, seam)
		}
	}
}

func TestFilletCurvatureTracksVariableRadius(t *testing.T) {
	cfg := DefaultConfig()
	fillet := NewFilletSurface(cfg)
	k0 := fillet.Eval(0.5, 0).Curvatures().MaxAbs()
	k1 := fillet.Eval(0.5, 1).Curvatures().MaxAbs()
	if math.Abs(k0-1/cfg.RadiusStart) > 2e-3 {
		t.Errorf("curvature near r=%.1f mm: got %.6f, want about %.6f", cfg.RadiusStart, k0, 1/cfg.RadiusStart)
	}
	if math.Abs(k1-1/cfg.RadiusEnd) > 2e-3 {
		t.Errorf("curvature near r=%.1f mm: got %.6f, want about %.6f", cfg.RadiusEnd, k1, 1/cfg.RadiusEnd)
	}
}

func TestVariableRadiusMeshRendersAllFourPanels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SamplesU, cfg.SamplesV = 28, 52
	cfg.PanelWidth, cfg.PanelHeight = 360, 250
	cfg.SuperSample = 1
	mesh := BuildMesh(cfg)
	report := buildReport(cfg)
	if len(mesh.Indices) == 0 {
		t.Fatal("mesh has no triangles")
	}
	fig := renderFigure(mesh, cfg, report)
	if fig.Bounds() != image.Rect(0, 0, 2*cfg.PanelWidth+2*30+22,
		30+96+2*cfg.PanelHeight+22+92+30) {
		t.Fatalf("unexpected figure bounds: %v", fig.Bounds())
	}
	nonBackground := 0
	for y := 0; y < fig.Bounds().Dy(); y += 5 {
		for x := 0; x < fig.Bounds().Dx(); x += 5 {
			o := y*fig.Stride + x*4
			if fig.Pix[o] > 80 || fig.Pix[o+1] > 80 || fig.Pix[o+2] > 80 {
				nonBackground++
			}
		}
	}
	if nonBackground < 100 {
		t.Fatalf("figure appears empty: only %d bright samples", nonBackground)
	}
}
