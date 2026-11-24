package scenes

import (
	"math"
	"testing"

	"ruledintersect/geom"
	"ruledintersect/intersect"
)

func TestScenesProduceContinuousSurfaceIntersections(t *testing.T) {
	for _, scene := range All() {
		t.Run(scene.Name, func(t *testing.T) {
			opts := intersect.DefaultOptions()
			result := intersect.Surfaces(scene.A, scene.B, opts)
			var curveLen float64
			for _, c := range result.Curves {
				curveLen += polylineLength(c.Points)
			}
			t.Logf("raw=%d segments=%d valid=%d duplicate=%d curves=%d length=%.3f maxgap=%g rejected=%d",
				result.RawCount, len(result.Segments), len(result.Valid),
				len(result.Duplicates), len(result.Curves), curveLen,
				result.MaxSurfaceGap, result.RejectedCount)
			if len(result.Valid) == 0 {
				t.Fatal("no valid intersection segments")
			}
			if len(result.Curves) == 0 {
				t.Fatal("no stitched intersections")
			}
			if len(result.Curves) != 1 {
				t.Fatalf("expected one continuous curve, got %d", len(result.Curves))
			}
			if curveLen < 0.45 {
				t.Fatalf("stitched curve is too short: %g", curveLen)
			}
			if result.MaxSurfaceGap > opts.OnSurfaceTolerance {
				t.Fatalf("surface gap %g exceeds tolerance", result.MaxSurfaceGap)
			}
			for _, seg := range result.Segments {
				checkOnBothSurfaces(t, scene.A, scene.B, seg.A, opts.OnSurfaceTolerance)
				checkOnBothSurfaces(t, scene.A, scene.B, seg.B, opts.OnSurfaceTolerance)
			}
			for _, curve := range result.Curves {
				for _, p := range curve.Points {
					checkOnBothSurfaces(t, scene.A, scene.B, p, opts.OnSurfaceTolerance)
				}
			}
			switch scene.Kind {
			case Tangent:
				if len(result.Duplicates) == 0 {
					t.Fatal("tangent contact should expose duplicate triangle-pair pieces")
				}
				for _, seg := range result.Valid {
					if math.Abs(seg.Midpoint().Z-0.35) > 2e-4 {
						t.Fatalf("tangent curve left contact plane: z=%g", seg.Midpoint().Z)
					}
				}
			case NearParallel:
				if curveLen < 0.8 {
					t.Fatalf("near-parallel curve should be long, got %g", curveLen)
				}
			case Crossing:
				if curveLen < 0.8 {
					t.Fatalf("crossing curve should be long, got %g", curveLen)
				}
			}
		})
	}
}

func checkOnBothSurfaces(t *testing.T, a, b *geom.RuledSurface, p geom.Vec3, tol float64) {
	t.Helper()
	if !p.IsFinite() {
		t.Fatal("intersection contains a non-finite point")
	}
	if d := a.Distance(p); d > tol {
		t.Fatalf("point is %g from surface A (tolerance %g)", d, tol)
	}
	if d := b.Distance(p); d > tol {
		t.Fatalf("point is %g from surface B (tolerance %g)", d, tol)
	}
}

func polylineLength(points []geom.Vec3) float64 {
	var total float64
	for i := 1; i < len(points); i++ {
		total += points[i].Dist(points[i-1])
	}
	return total
}
