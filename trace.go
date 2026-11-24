package main

import (
	"fmt"
	"math"
)

type SearchMode string

const (
	// ModeFixed is uniform steep-parallax ray marching.
	ModeFixed SearchMode = "fixed"
	// ModeBinary adds bisection inside the final crossing bracket.
	ModeBinary SearchMode = "binary"
	// ModeMaxIteration bisects until Tolerance or the hard iteration cap.
	ModeMaxIteration SearchMode = "max-iteration"
)

type Status string

const (
	StatusHit              Status = "hit"
	StatusFallbackGeometry Status = "fallback_geometry"
)

// Ray is a linear ray in UV/depth space. Depth zero is the original
// interpolated geometry coordinate; UV(depth) = GeometryUV + UVVelocity*depth.
// A negative UVVelocity.X represents a reversed view ray.
type Ray struct {
	GeometryUV Vec2    `json:"geometry_uv"`
	UVVelocity Vec2    `json:"uv_velocity"`
	MaxDepth   float64 `json:"max_depth"`

	// PreviousUV exists to model frame-plumbing regressions. Trace never
	// reads it; a failed ray must return GeometryUV, not this value.
	PreviousUV Vec2 `json:"previous_uv"`
}

func (r Ray) uvAt(depth float64) Vec2 {
	return r.GeometryUV.Add(r.UVVelocity.Scale(depth))
}

type Config struct {
	Mode             SearchMode `json:"mode"`
	Layers           int        `json:"layers"`
	BinaryIterations int        `json:"binary_iterations"`
	MaxIterations    int        `json:"max_iterations"`
	Tolerance        float64    `json:"tolerance"`
	HitEpsilon       float64    `json:"hit_epsilon"`
	MinHeight        float64    `json:"min_height"`
	MaxHeight        float64    `json:"max_height"`
}

func DefaultConfig(mode SearchMode) Config {
	cfg := Config{
		Mode:          mode,
		Layers:        32,
		MaxIterations: 64,
		Tolerance:     1e-7,
		HitEpsilon:    1e-12,
		MinHeight:     0,
		MaxHeight:     1,
	}
	if mode == ModeBinary {
		cfg.BinaryIterations = 8
	}
	return cfg
}

func (c *Config) normalize() {
	if c.Mode == "" {
		c.Mode = ModeFixed
	}
	if c.Layers == 0 {
		c.Layers = 32
	}
	if c.MaxIterations == 0 {
		c.MaxIterations = 64
	}
	if c.Tolerance == 0 {
		c.Tolerance = 1e-7
	}
	if c.HitEpsilon == 0 {
		c.HitEpsilon = 1e-12
	}
	if c.MinHeight == 0 && c.MaxHeight == 0 {
		c.MaxHeight = 1
	}
}

func (c Config) Validate() error {
	if c.Layers < 1 {
		return fmt.Errorf("%w: layers must be positive", ErrInvalidConfig)
	}
	if c.MaxIterations < 1 {
		return fmt.Errorf("%w: max iterations must be positive", ErrInvalidConfig)
	}
	if c.BinaryIterations < 0 {
		return fmt.Errorf("%w: binary iterations must be non-negative", ErrInvalidConfig)
	}
	if c.Tolerance <= 0 {
		return fmt.Errorf("%w: tolerance must be positive", ErrInvalidConfig)
	}
	if c.HitEpsilon < 0 {
		return fmt.Errorf("%w: hit epsilon must be non-negative", ErrInvalidConfig)
	}
	if c.MaxHeight <= c.MinHeight {
		return fmt.Errorf("%w: height range is empty", ErrInvalidConfig)
	}
	return nil
}

type Result struct {
	Mode             SearchMode `json:"mode"`
	Status           Status     `json:"status"`
	Hit              bool       `json:"hit"`
	Fallback         bool       `json:"fallback"`
	UV               Vec2       `json:"uv"`
	Depth            float64    `json:"depth"`
	Layer            int        `json:"hit_layer"`
	Iterations       int        `json:"iterations"`
	BinaryIterations int        `json:"binary_iterations"`
	FieldHeight      float64    `json:"field_height"`
	SurfaceDepth     float64    `json:"surface_depth"`
	Residual         float64    `json:"residual"`
	BracketWidth     float64    `json:"bracket_width,omitempty"`
	ReferenceDepth   float64    `json:"reference_depth,omitempty"`
	DepthError       float64    `json:"depth_error"`
	Reason           string     `json:"reason,omitempty"`
}

type sample struct {
	depth        float64
	uv           Vec2
	height       float64
	surfaceDepth float64
	gap          float64
}

func (s sample) result(cfg Config, layer, iterations, binaryUsed int, bracketWidth float64) Result {
	return Result{
		Mode:             cfg.Mode,
		Status:           StatusHit,
		Hit:              true,
		UV:               s.uv,
		Depth:            s.depth,
		Layer:            layer,
		Iterations:       iterations,
		BinaryIterations: binaryUsed,
		FieldHeight:      s.height,
		SurfaceDepth:     s.surfaceDepth,
		Residual:         s.gap,
		BracketWidth:     bracketWidth,
	}
}

func fallbackResult(cfg Config, ray Ray, iterations int, reason string) Result {
	return Result{
		Mode:       cfg.Mode,
		Status:     StatusFallbackGeometry,
		Fallback:   true,
		UV:         ray.GeometryUV,
		Depth:      0,
		Layer:      -1,
		Iterations: iterations,
		Reason:     reason,
	}
}

func sampleRay(cfg Config, field HeightField, ray Ray, depth float64, iterations *int) (sample, error) {
	uv := ray.uvAt(depth)
	height, err := field.HeightAt(uv)
	*iterations++
	if err != nil {
		return sample{depth: depth, uv: uv}, err
	}
	if math.IsNaN(height) || math.IsInf(height, 0) || height < cfg.MinHeight || height > cfg.MaxHeight {
		err := fmt.Errorf("%w at uv=(%.6g,%.6g), height=%.6g", ErrHeightOutOfRange, uv.X, uv.Y, height)
		return sample{depth: depth, uv: uv, height: height}, err
	}
	surfaceDepth := (cfg.MaxHeight - height) / (cfg.MaxHeight - cfg.MinHeight)
	return sample{
		depth:        depth,
		uv:           uv,
		height:       height,
		surfaceDepth: surfaceDepth,
		gap:          depth - surfaceDepth,
	}, nil
}

// ScoreAgainst records depth error relative to an accepted expected depth.
func (r *Result) ScoreAgainst(expectedDepth float64) {
	r.ReferenceDepth = expectedDepth
	r.DepthError = math.Abs(r.Depth - expectedDepth)
}
