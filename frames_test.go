package main

import (
	"math"
	"testing"
)

func TestFramesAreOrthonormalAndContinuous(t *testing.T) {
	stations, _ := sampleTrefoil(sampleCount)
	sets := map[string][]Station{
		"Frenet":   buildFrenet(stations),
		"parallel": mustParallel(stations),
		"fixed":    mustFixed(stations),
	}

	for name, framed := range sets {
		t.Run(name, func(t *testing.T) {
			for i, station := range framed {
				if station.Curvature <= 0.18 {
					t.Fatalf("station %d has unsafe curvature %.5f", i, station.Curvature)
				}
				assertOrthogonal(t, station.Tangent, station.E1, 1e-10)
				assertOrthogonal(t, station.Tangent, station.E2, 1e-10)
				assertOrthogonal(t, station.E1, station.E2, 1e-10)
				assertUnit(t, station.E1, 1e-10)
				assertUnit(t, station.E2, 1e-10)
			}

			n := len(framed)
			angles := unwrapFrameAngles(framed)
			for i := 0; i < n; i++ {
				angle := angles[i]
				limit := 8 * math.Pi / 180
				if math.Abs(angle) > limit {
					t.Fatalf("station %d rotates %.2f degrees; possible frame flip", i, radToDeg(angle))
				}
			}
		})
	}
}

func TestParallelTransportClosureIsCorrected(t *testing.T) {
	stations, _ := sampleTrefoil(sampleCount)
	_, holonomy, residual := buildParallelTransport(stations)
	if math.Abs(holonomy) < 0.5 {
		t.Fatalf("test curve should expose non-trivial holonomy, got %.4f", holonomy)
	}
	if residual > 1e-12 {
		t.Fatalf("parallel transport closure residual too large: %.2e", residual)
	}
}

func TestFixedReferenceNeverBecomesParallel(t *testing.T) {
	stations, _ := sampleTrefoil(sampleCount)
	_, _, margin := buildFixedFrame(stations)
	if margin < 0.01 {
		t.Fatalf("fixed reference margin too small: %.5f", margin)
	}
}

func mustParallel(stations []Station) []Station {
	framed, _, _ := buildParallelTransport(stations)
	return framed
}

func mustFixed(stations []Station) []Station {
	framed, _, _ := buildFixedFrame(stations)
	return framed
}

func assertOrthogonal(t *testing.T, a, b Vec3, tolerance float64) {
	t.Helper()
	if math.Abs(a.Dot(b)) > tolerance {
		t.Fatalf("vectors %v and %v are not orthogonal: %.2e", a, b, a.Dot(b))
	}
}

func assertUnit(t *testing.T, v Vec3, tolerance float64) {
	t.Helper()
	if math.Abs(v.Norm()-1) > tolerance {
		t.Fatalf("vector %v has norm %.12f", v, v.Norm())
	}
}
