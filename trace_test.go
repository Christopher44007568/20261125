package main

import (
	"math"
	"testing"
)

func testPlaneRay(reverse bool) Ray {
	velocity := 0.8
	if reverse {
		velocity = -0.8
	}
	return Ray{
		GeometryUV: Vec2{X: 0.5, Y: 0.5},
		UVVelocity: Vec2{X: velocity},
		MaxDepth:   1,
		PreviousUV: Vec2{X: 0.013, Y: 0.017},
	}
}

func testPlaneField() HeightField {
	return BoundedFunc{Bounds: UnitRect, Fn: func(Vec2) float64 { return 0.4 }}
}

func TestPlaneSearchAccuracyAndLayers(t *testing.T) {
	tests := []struct {
		mode          SearchMode
		maxDepthError float64
	}{
		{ModeFixed, 0.025 + 1e-12},
		{ModeBinary, 0.0002},
		{ModeMaxIteration, 1e-8},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			cfg := suiteConfig(tt.mode)
			result := Trace(cfg, testPlaneField(), testPlaneRay(false))
			result.ScoreAgainst(0.6)

			if !result.Hit || result.Status != StatusHit {
				t.Fatalf("expected hit, got status=%q reason=%q", result.Status, result.Reason)
			}
			if result.Layer != 5 {
				t.Fatalf("expected hit layer 5, got %d", result.Layer)
			}
			if math.Abs(result.Depth-0.6) > tt.maxDepthError {
				t.Fatalf("depth error too large: depth=%.12f error=%.12g", result.Depth, result.DepthError)
			}
			if result.DepthError > tt.maxDepthError {
				t.Fatalf("reported depth error %.12g exceeds %.12g", result.DepthError, tt.maxDepthError)
			}
			if result.Iterations <= 0 || result.Iterations > cfg.MaxIterations {
				t.Fatalf("iteration count %d outside [1,%d]", result.Iterations, cfg.MaxIterations)
			}
		})
	}
}

func TestSteepSlopeBinaryImprovesFixedResult(t *testing.T) {
	field := BoundedFunc{
		Bounds: UnitRect,
		Fn:     func(uv Vec2) float64 { return 0.8 + 4*(uv.X-0.5) },
	}
	ray := Ray{
		GeometryUV: Vec2{X: 0.5, Y: 0.5},
		UVVelocity: Vec2{X: 0.25},
		MaxDepth:   1,
	}

	fixed := Trace(suiteConfig(ModeFixed), field, ray)
	binary := Trace(suiteConfig(ModeBinary), field, ray)
	limited := Trace(suiteConfig(ModeMaxIteration), field, ray)

	for name, result := range map[string]Result{"fixed": fixed, "binary": binary, "max": limited} {
		if !result.Hit {
			t.Fatalf("%s: expected hit, got %q", name, result.Reason)
		}
		if result.Layer != 1 {
			t.Fatalf("%s: expected layer 1, got %d", name, result.Layer)
		}
		result.ScoreAgainst(0.1)
		if result.DepthError > 0.025+1e-12 {
			t.Fatalf("%s: unexpected depth error %g", name, result.DepthError)
		}
	}

	fixed.ScoreAgainst(0.1)
	binary.ScoreAgainst(0.1)
	limited.ScoreAgainst(0.1)
	if binary.DepthError >= fixed.DepthError {
		t.Fatalf("binary error %g should improve fixed error %g", binary.DepthError, fixed.DepthError)
	}
	if limited.DepthError >= binary.DepthError {
		t.Fatalf("max-iteration error %g should improve binary error %g", limited.DepthError, binary.DepthError)
	}
	if math.Abs(limited.UV.X-0.525) > 1e-8 {
		t.Fatalf("expected refined U=0.525, got %.10f", limited.UV.X)
	}
}

func TestReverseViewRayMarchesToNegativeU(t *testing.T) {
	for _, mode := range []SearchMode{ModeFixed, ModeBinary, ModeMaxIteration} {
		t.Run(string(mode), func(t *testing.T) {
			result := Trace(suiteConfig(mode), testPlaneField(), testPlaneRay(true))
			if !result.Hit {
				t.Fatalf("expected hit: %s", result.Reason)
			}
			if result.Layer != 5 {
				t.Fatalf("expected layer 5, got %d", result.Layer)
			}
			if result.UV.X >= 0.5 {
				t.Fatalf("reverse ray must move below geometry U, got %.6f", result.UV.X)
			}

			result.ScoreAgainst(0.6)
			switch mode {
			case ModeFixed:
				if result.DepthError > 0.025+1e-12 {
					t.Fatalf("fixed reverse depth error %g", result.DepthError)
				}
			case ModeBinary:
				if result.DepthError > 0.0002 {
					t.Fatalf("binary reverse depth error %g", result.DepthError)
				}
			case ModeMaxIteration:
				if result.DepthError > 1e-8 {
					t.Fatalf("max reverse depth error %g", result.DepthError)
				}
				if math.Abs(result.UV.X-0.02) > 1e-8 {
					t.Fatalf("expected reverse hit U≈0.02, got %.10f", result.UV.X)
				}
			}
		})
	}
}
