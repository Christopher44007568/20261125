package geom

import "testing"

func TestRationalPatchCorners(t *testing.T) {
	p := DemoPatch()
	cases := []struct {
		u, v float64
		want Vec3
	}{
		{0, 0, p.Points[0][0]},
		{1, 0, p.Points[0][3]},
		{0, 1, p.Points[3][0]},
		{1, 1, p.Points[3][3]},
	}
	for _, tc := range cases {
		got := p.Eval(tc.u, tc.v)
		if got.Sub(tc.want).Len() > 1e-10 {
			t.Fatalf("P(%.0f,%.0f)=%v want %v", tc.u, tc.v, got, tc.want)
		}
	}
}

func TestDemoTrimIsConvexAndInsideDomain(t *testing.T) {
	trim := DemoTrim()
	poly := trim.Polyline(1e-4)
	if len(poly) < 8 {
		t.Fatalf("unexpectedly short trim polyline: %d", len(poly))
	}
	if !PolylineIsConvex(poly, 1e-9) {
		t.Fatal("trim polyline is not convex")
	}
	for _, p := range poly {
		if p.X < -1e-9 || p.X > 1+1e-9 || p.Y < -1e-9 || p.Y > 1+1e-9 {
			t.Fatalf("trim point outside parameter domain: %v", p)
		}
	}
}
