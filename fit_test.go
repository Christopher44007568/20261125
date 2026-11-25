package main

import (
	"math"
	"testing"
)

func TestBasisPartition(t *testing.T) {
	for _, n := range []int{4, 9, 12} {
		b := NewClampedBasis(n, 3)
		buf := make([]float64, 4)
		for k := 0; k <= 100; k++ {
			u := float64(k) / 100
			start := b.Nonzero(u, buf)
			sum := 0.0
			for _, v := range buf {
				sum += v
			}
			if math.Abs(sum-1) > 1e-9 || start < 0 || start+3 >= n {
				t.Fatalf("n=%d u=%.2f start=%d vals=%v sum=%.12f", n, u, start, buf, sum)
			}
		}
	}
}

func TestSolveSPD(t *testing.T) {
	for n := 4; n <= 108; n += 13 {
		a := make([]float64, n*n)
		want := make([]float64, n)
		for i := 0; i < n; i++ {
			want[i] = math.Sin(float64(i+1) * 0.7)
			for j := 0; j < n; j++ {
				a[i*n+j] = 1.0 / (1 + math.Abs(float64(i-j)))
			}
			a[i*n+i] += 2
		}
		b := make([]float64, n)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				b[i] += a[i*n+j] * want[j]
			}
		}
		got, err := solveSPD(a, b, n)
		if err != nil {
			t.Fatal(err)
		}
		for i := range want {
			if math.Abs(got[i]-want[i]) > 1e-8 {
				t.Fatalf("n=%d component %d: got %g want %g", n, i, got[i], want[i])
			}
		}
	}
}

func TestFitPlane(t *testing.T) {
	points := GenerateScan(Reference{})
	for i := range points {
		points[i].Z = 0.01*points[i].X + 0.02*points[i].Y + 3
		points[i].Kind = PointClean
	}
	f, err := FitSurface(points, FitOptions{NU: 6, NV: 5, Lambda: 0.1, RobustIters: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, xy := range [][2]float64{{0, 0}, {20, 10}, {-50, -20}, {60, 30}} {
		want := 0.01*xy[0] + 0.02*xy[1] + 3
		if got := f.Surface.Z(xy[0], xy[1]); math.Abs(got-want) > 0.03 {
			t.Fatalf("plane at %v: got %.5f want %.5f", xy, got, want)
		}
	}
}

func TestFitKnownBSpline(t *testing.T) {
	opt := FitOptions{NU: 6, NV: 5, Lambda: 0.01, RobustIters: 1}
	u, v := NewClampedBasis(opt.NU, 3), NewClampedBasis(opt.NV, 3)
	c := make([][]float64, opt.NU)
	for i := range c {
		c[i] = make([]float64, opt.NV)
	}
	c[2][2] = 12
	c[3][1] = -5
	src := &BSplineSurface{U: u, V: v, C: c}
	points := GenerateScan(Reference{})
	for i := range points {
		points[i].Z = src.Z(points[i].X, points[i].Y)
		points[i].Kind = PointClean
	}
	f, err := FitSurface(points, opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, xy := range [][2]float64{{0, 0}, {20, 10}, {-50, -20}, {60, 30}} {
		got, want := f.Surface.Z(xy[0], xy[1]), src.Z(xy[0], xy[1])
		if math.Abs(got-want) > 0.25 {
			t.Fatalf("reconstruction at %v: got %.5f want %.5f", xy, got, want)
		}
	}
}

func TestAcceptanceAndResolutionPlateau(t *testing.T) {
	points := GenerateScan(Reference{})
	holes := OcclusionMask(points)
	if len(holes.Components) != 3 {
		t.Fatalf("expected 3 occlusion masks, got %d", len(holes.Components))
	}
	trimA, trimB := EstimateTrim(points)
	grids := [][2]int{{6, 5}, {8, 6}, {10, 7}, {12, 9}}
	results := make([]*FitResult, 0, len(grids))
	audits := make([]Audit, 0, len(grids))
	for _, g := range grids {
		f, err := FitSurface(points, FitOptions{NU: g[0], NV: g[1], Lambda: 0.02, RobustIters: 8})
		if err != nil {
			t.Fatal(err)
		}
		results = append(results, f)
		audits = append(audits, EvaluateFit(f.Surface, trimA, trimB))
	}
	for i := 1; i < len(audits); i++ {
		prev, cur := audits[i-1], audits[i]
		if cur.DenseRMSE >= prev.DenseRMSE ||
			cur.NormalP95 >= prev.NormalP95 ||
			cur.CurvatureRelative >= prev.CurvatureRelative ||
			cur.RippleRatio >= prev.RippleRatio ||
			cur.SilhouetteChamfer >= prev.SilhouetteChamfer ||
			cur.SilhouetteIoU <= prev.SilhouetteIoU {
			t.Errorf("acceptance did not improve from %v to %v: %+v -> %+v", grids[i-1], grids[i], prev, cur)
		}
	}
	final := results[len(results)-1]
	audit := audits[len(audits)-1]
	if final.InlierRMS > 0.25 {
		t.Errorf("control residual %.4f mm exceeds 0.25", final.InlierRMS)
	}
	if audit.DenseRMSE > 0.10 {
		t.Errorf("dense height RMSE %.4f mm exceeds 0.10", audit.DenseRMSE)
	}
	if audit.NormalP95 > 1.3 {
		t.Errorf("normal p95 %.3f deg exceeds 1.3", audit.NormalP95)
	}
	if audit.CurvatureRelative > 15 {
		t.Errorf("curvature error %.2f%% exceeds 15%%", audit.CurvatureRelative)
	}
	if audit.RippleRatio > 18 {
		t.Errorf("ripple band error %.2f%% exceeds 18%%", audit.RippleRatio)
	}
	if audit.SilhouetteIoU < 0.998 {
		t.Errorf("silhouette IoU %.5f below 0.998", audit.SilhouetteIoU)
	}
	if audit.SilhouetteChamfer > 0.06 {
		t.Errorf("silhouette chamfer %.4f mm exceeds 0.06", audit.SilhouetteChamfer)
	}

	next, err := FitSurface(points, FitOptions{NU: 14, NV: 10, Lambda: 0.02, RobustIters: 8})
	if err != nil {
		t.Fatal(err)
	}
	nextAudit := EvaluateFit(next.Surface, trimA, trimB)
	if next.InlierRMS >= final.InlierRMS*0.995 {
		t.Errorf("expected finer net to reduce control residual: %.4f -> %.4f", final.InlierRMS, next.InlierRMS)
	}
	if nextAudit.CurvatureRelative <= audit.CurvatureRelative {
		t.Errorf("expected curvature/ripple rejection to be exercised; curvature %.2f%% -> %.2f%%",
			audit.CurvatureRelative, nextAudit.CurvatureRelative)
	}
	if nextAudit.RippleRatio <= audit.RippleRatio {
		t.Errorf("expected ripple to regress: %.2f%% -> %.2f%%", audit.RippleRatio, nextAudit.RippleRatio)
	}
	if nextAudit.SilhouetteIoU > audit.SilhouetteIoU+0.0001 {
		t.Errorf("expected no material silhouette gain: %.5f -> %.5f", audit.SilhouetteIoU, nextAudit.SilhouetteIoU)
	}
}
