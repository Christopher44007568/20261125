package main

// Scenario is a deterministic height-field/ray pair with the behavior the
// tests and report are expected to verify.
type Scenario struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Field       HeightField `json:"-"`
	Ray         Ray         `json:"ray"`
	WantStatus  Status      `json:"want_status"`
	WantLayer   int         `json:"want_layer"`
	WantDepth   float64     `json:"want_depth"`
	WantMinUVX  *float64    `json:"-"`
	WantMaxUVX  *float64    `json:"-"`
}

func suiteConfig(mode SearchMode) Config {
	cfg := DefaultConfig(mode)
	cfg.Layers = 8
	cfg.HitEpsilon = 1e-12
	switch mode {
	case ModeBinary:
		cfg.BinaryIterations = 10
		cfg.MaxIterations = 32
	case ModeMaxIteration:
		cfg.MaxIterations = 40
		cfg.Tolerance = 1e-9
	default:
		cfg.MaxIterations = 9
	}
	return cfg
}

// DefaultSuite contains plane, steep slope, reversed ray, invalid height,
// and a grazing ray that leaves the strict UV domain.
func DefaultSuite() []Scenario {
	planeField := BoundedFunc{
		Bounds: UnitRect,
		Fn:     func(Vec2) float64 { return 0.4 },
	}

	slopeField := BoundedFunc{
		Bounds: UnitRect,
		Fn: func(uv Vec2) float64 {
			return 0.8 + 4*(uv.X-0.5)
		},
	}

	invalidAfterStep := HeightFunc(func(uv Vec2) (float64, error) {
		if uv.X > 0.45 {
			return 1.25, nil
		}
		return 0.4, nil
	})

	return []Scenario{
		{
			Name:        "plane_forward",
			Description: "constant height plane with a forward view ray",
			Field:       planeField,
			Ray: Ray{
				GeometryUV: Vec2{X: 0.5, Y: 0.5},
				UVVelocity: Vec2{X: 0.8},
				MaxDepth:   1,
				PreviousUV: Vec2{X: 0.01, Y: 0.99},
			},
			WantStatus: StatusHit,
			WantLayer:  5,
			WantDepth:  0.6,
		},
		{
			Name:        "steep_slope",
			Description: "height gradient 4 intersects early in layer 1",
			Field:       slopeField,
			Ray: Ray{
				GeometryUV: Vec2{X: 0.5, Y: 0.5},
				UVVelocity: Vec2{X: 0.25},
				MaxDepth:   1,
				PreviousUV: Vec2{X: 0.9, Y: 0.1},
			},
			WantStatus: StatusHit,
			WantLayer:  1,
			WantDepth:  0.1,
		},
		{
			Name:        "reverse_view",
			Description: "negative UV velocity must march toward smaller U",
			Field:       planeField,
			Ray: Ray{
				GeometryUV: Vec2{X: 0.5, Y: 0.5},
				UVVelocity: Vec2{X: -0.8},
				MaxDepth:   1,
				PreviousUV: Vec2{X: 0.99, Y: 0.99},
			},
			WantStatus: StatusHit,
			WantLayer:  5,
			WantDepth:  0.6,
		},
		{
			Name:        "height_out_of_bounds",
			Description: "height 1.25 violates normalized [0,1] after a valid step",
			Field:       invalidAfterStep,
			Ray: Ray{
				GeometryUV: Vec2{X: 0.3, Y: 0.5},
				UVVelocity: Vec2{X: 0.8},
				MaxDepth:   1,
				PreviousUV: Vec2{X: 0.91, Y: 0.92},
			},
			WantStatus: StatusFallbackGeometry,
			WantLayer:  -1,
			WantDepth:  0,
		},
		{
			Name:        "grazing_uv_exit",
			Description: "grazing velocity leaves strict UV bounds before a hit",
			Field:       planeField,
			Ray: Ray{
				GeometryUV: Vec2{X: 0.5, Y: 0.5},
				UVVelocity: Vec2{X: 4},
				MaxDepth:   1,
				PreviousUV: Vec2{X: 0.123, Y: 0.456},
			},
			WantStatus: StatusFallbackGeometry,
			WantLayer:  -1,
			WantDepth:  0,
		},
	}
}

type Row struct {
	Scenario      string     `json:"scenario"`
	Description   string     `json:"description,omitempty"`
	Mode          SearchMode `json:"mode"`
	Status        Status     `json:"status"`
	HitLayer      int        `json:"hit_layer"`
	Iterations    int        `json:"iterations"`
	BinarySteps   int        `json:"binary_steps"`
	Depth         float64    `json:"depth"`
	ExpectedDepth float64    `json:"expected_depth"`
	DepthError    float64    `json:"depth_error"`
	Residual      float64    `json:"residual"`
	UV            Vec2       `json:"uv"`
	Fallback      bool       `json:"fallback"`
	Reason        string     `json:"reason,omitempty"`
}

// RunSuite executes every scenario with fixed, binary, and max-iteration search.
func RunSuite(scenarios []Scenario) []Row {
	modes := []SearchMode{ModeFixed, ModeBinary, ModeMaxIteration}
	rows := make([]Row, 0, len(scenarios)*len(modes))
	for _, scenario := range scenarios {
		for _, mode := range modes {
			cfg := suiteConfig(mode)
			result := Trace(cfg, scenario.Field, scenario.Ray)

			expectedDepth := scenario.WantDepth
			if result.Fallback {
				expectedDepth = 0
			}
			result.ScoreAgainst(expectedDepth)

			rows = append(rows, Row{
				Scenario:      scenario.Name,
				Description:   scenario.Description,
				Mode:          mode,
				Status:        result.Status,
				HitLayer:      result.Layer,
				Iterations:    result.Iterations,
				BinarySteps:   result.BinaryIterations,
				Depth:         result.Depth,
				ExpectedDepth: expectedDepth,
				DepthError:    result.DepthError,
				Residual:      result.Residual,
				UV:            result.UV,
				Fallback:      result.Fallback,
				Reason:        result.Reason,
			})
		}
	}
	return rows
}
