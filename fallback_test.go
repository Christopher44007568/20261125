package main

import (
	"math"
	"strings"
	"testing"
)

func TestHeightOutOfBoundsReturnsGeometrySurface(t *testing.T) {
	scenario := Scenario{
		Name: "invalid",
		Field: HeightFunc(func(uv Vec2) (float64, error) {
			if uv.X > 0.45 {
				return 1.25, nil
			}
			return 0.4, nil
		}),
		Ray: Ray{
			GeometryUV: Vec2{X: 0.3, Y: 0.5},
			UVVelocity: Vec2{X: 0.8},
			PreviousUV: Vec2{X: 0.91, Y: 0.92},
		},
	}

	for _, mode := range []SearchMode{ModeFixed, ModeBinary, ModeMaxIteration} {
		t.Run(string(mode), func(t *testing.T) {
			result := Trace(suiteConfig(mode), scenario.Field, scenario.Ray)
			assertGeometryFallback(t, result, scenario.Ray, 3)
			if !strings.Contains(result.Reason, ErrHeightOutOfRange.Error()) {
				t.Fatalf("expected height-range reason, got %q", result.Reason)
			}
			if result.DepthError != 0 {
				t.Fatalf("geometry fallback should report zero accepted depth error, got %g", result.DepthError)
			}
		})
	}
}

func TestNegativeHeightSampleFallsBackImmediately(t *testing.T) {
	ray := Ray{
		GeometryUV: Vec2{X: 0.25, Y: 0.75},
		PreviousUV: Vec2{X: 0.8, Y: 0.1},
	}
	result := FixedStepSearch(HeightFunc(func(Vec2) (float64, error) {
		return -0.01, nil
	}), ray, 8)

	assertGeometryFallback(t, result, ray, 1)
}

func TestUVOutOfBoundsFallsBack(t *testing.T) {
	ray := Ray{
		GeometryUV: Vec2{X: 0.5, Y: 0.5},
		UVVelocity: Vec2{X: 4},
		PreviousUV: Vec2{X: 0.123, Y: 0.456},
	}
	result := FixedStepSearch(testPlaneField(), ray, 8)
	assertGeometryFallback(t, result, ray, 3)
	if !strings.Contains(result.Reason, ErrUVOutOfBounds.Error()) {
		t.Fatalf("expected UV bounds reason, got %q", result.Reason)
	}
}

func TestDepthLimitWithoutHitFallsBack(t *testing.T) {
	ray := testPlaneRay(false)
	ray.MaxDepth = 0.5
	result := FixedStepSearch(testPlaneField(), ray, 8)

	assertGeometryFallback(t, result, ray, 9)
	if !strings.Contains(result.Reason, "no intersection") {
		t.Fatalf("expected no-intersection reason, got %q", result.Reason)
	}
}

func TestPreviousFrameCoordinateIsNeverReused(t *testing.T) {
	field := HeightFunc(func(uv Vec2) (float64, error) {
		if uv.X > 0.45 {
			return 1.25, nil
		}
		return 0.4, nil
	})
	ray := Ray{
		GeometryUV: Vec2{X: 0.31, Y: 0.42},
		UVVelocity: Vec2{X: 0.8},
		PreviousUV: Vec2{X: 0.777, Y: 0.888},
	}

	result := MaxIterationSearch(field, ray, 8, 40, 1e-9)
	assertGeometryFallback(t, result, ray, 3)
	if result.UV == ray.PreviousUV {
		t.Fatal("fallback reused previous-frame UV")
	}

	first := Trace(suiteConfig(ModeMaxIteration), testPlaneField(), testPlaneRay(false))
	_ = Trace(suiteConfig(ModeFixed), field, ray)
	again := Trace(suiteConfig(ModeMaxIteration), testPlaneField(), testPlaneRay(false))
	if first != again {
		t.Fatalf("trace is not frame-independent:\nfirst=%+v\nagain=%+v", first, again)
	}
}

func TestMaxIterationCapIsHardLimit(t *testing.T) {
	cfg := DefaultConfig(ModeBinary)
	cfg.Layers = 8
	cfg.BinaryIterations = 10
	cfg.MaxIterations = 8

	result := Trace(cfg, testPlaneField(), testPlaneRay(false))
	if result.Iterations > cfg.MaxIterations {
		t.Fatalf("used %d iterations above cap %d", result.Iterations, cfg.MaxIterations)
	}
	assertGeometryFallback(t, result, testPlaneRay(false), cfg.MaxIterations)
	if !strings.Contains(result.Reason, "maximum iteration") {
		t.Fatalf("expected iteration cap reason, got %q", result.Reason)
	}
}

func TestMaxIterationModeCapReturnsGeometryFallback(t *testing.T) {
	cfg := DefaultConfig(ModeMaxIteration)
	cfg.Layers = 8
	cfg.MaxIterations = 10
	cfg.Tolerance = 1e-12

	ray := testPlaneRay(false)
	result := Trace(cfg, testPlaneField(), ray)
	if result.Iterations > cfg.MaxIterations {
		t.Fatalf("used %d iterations above cap %d", result.Iterations, cfg.MaxIterations)
	}
	assertGeometryFallback(t, result, ray, cfg.MaxIterations)
}

func TestIterationBudgetNeverExceeded(t *testing.T) {
	ray := testPlaneRay(false)
	for _, mode := range []SearchMode{ModeFixed, ModeBinary, ModeMaxIteration} {
		for budget := 1; budget <= 40; budget++ {
			cfg := suiteConfig(mode)
			cfg.MaxIterations = budget
			result := Trace(cfg, testPlaneField(), ray)
			if result.Iterations > budget {
				t.Fatalf("%s budget=%d used=%d", mode, budget, result.Iterations)
			}
			if result.Hit && math.IsNaN(result.DepthError) {
				t.Fatalf("%s budget=%d produced NaN error", mode, budget)
			}
		}
	}
}

func assertGeometryFallback(t *testing.T, result Result, ray Ray, expectedIterations int) {
	t.Helper()
	if result.Hit || !result.Fallback || result.Status != StatusFallbackGeometry {
		t.Fatalf("expected geometry fallback, got hit=%v status=%q reason=%q",
			result.Hit, result.Status, result.Reason)
	}
	if result.UV != ray.GeometryUV {
		t.Fatalf("fallback UV %+v is not geometry UV %+v", result.UV, ray.GeometryUV)
	}
	if result.UV == ray.PreviousUV && ray.PreviousUV != (Vec2{}) {
		t.Fatalf("fallback reused previous UV %+v", result.UV)
	}
	if result.Layer != -1 {
		t.Fatalf("fallback layer must be -1, got %d", result.Layer)
	}
	if expectedIterations > 0 && result.Iterations != expectedIterations {
		t.Fatalf("expected %d iterations, got %d", expectedIterations, result.Iterations)
	}
}
