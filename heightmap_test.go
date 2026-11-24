package main

import (
	"errors"
	"math"
	"testing"
)

func TestHeightMapBilinearAndBounds(t *testing.T) {
	samples := []float64{
		0, 0.25, 0.5,
		0.5, 0.75, 1,
	}
	heightMap, err := NewHeightMap(3, 2, samples)
	if err != nil {
		t.Fatal(err)
	}

	got, err := heightMap.HeightAt(Vec2{X: 0.25, Y: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-0.375) > 1e-12 {
		t.Fatalf("expected bilinear height 0.375, got %.12f", got)
	}

	if _, err := heightMap.HeightAt(Vec2{X: 1.01}); !errors.Is(err, ErrUVOutOfBounds) {
		t.Fatalf("expected UV bounds error, got %v", err)
	}

	heightMap.Repeat = true
	wrapped, err := heightMap.HeightAt(Vec2{X: 1.25, Y: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := heightMap.HeightAt(Vec2{X: 0.25, Y: 0.5})
	if wrapped != expected {
		t.Fatalf("repeat sampling gave %.3f, want %.3f", wrapped, expected)
	}
}

func TestTraceWithGridHeightMap(t *testing.T) {
	width, height := 33, 33
	samples := make([]float64, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			uv := Vec2{X: float64(x) / float64(width-1), Y: float64(y) / float64(height-1)}
			samples[y*width+x] = 0.4 + 0.0*uv.X
		}
	}
	heightMap, err := NewHeightMap(width, height, samples)
	if err != nil {
		t.Fatal(err)
	}

	result := MaxIterationSearch(heightMap, testPlaneRay(false), 16, 40, 1e-7)
	if !result.Hit {
		t.Fatalf("grid trace failed: %s", result.Reason)
	}
	result.ScoreAgainst(0.6)
	if result.DepthError > 1e-6 {
		t.Fatalf("grid depth error too large: %g", result.DepthError)
	}
}

func TestGrazingRayWithRepeatHeightMap(t *testing.T) {
	heightMap, err := NewHeightMap(2, 2, []float64{0.4, 0.4, 0.4, 0.4})
	if err != nil {
		t.Fatal(err)
	}
	heightMap.Repeat = true

	ray := Ray{
		GeometryUV: Vec2{X: 0.5, Y: 0.5},
		UVVelocity: Vec2{X: 8},
		MaxDepth:   1,
	}
	result := MaxIterationSearch(heightMap, ray, 8, 40, 1e-9)
	if !result.Hit {
		t.Fatalf("repeated height map should survive grazing ray: %s", result.Reason)
	}
	if result.Layer != 5 {
		t.Fatalf("grazing plane should hit layer 5, got %d", result.Layer)
	}
	result.ScoreAgainst(0.6)
	if result.DepthError > 1e-8 {
		t.Fatalf("grazing depth error too large: %g", result.DepthError)
	}
}
