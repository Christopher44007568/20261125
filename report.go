package main

type RunOptions struct {
	OutDir string  `json:"outDir"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
	IPD    float64 `json:"ipdMeters"`
}

type ReportSummary struct {
	ScenarioCount       int               `json:"scenarioCount"`
	LayerCount          int               `json:"layerCount"`
	EyeFrameCount       int               `json:"eyeFrameCount"`
	GlyphPairCount      int               `json:"glyphPairCount"`
	ElementEyeChecks    int               `json:"elementEyeChecks"`
	PassCount           int               `json:"passCount"`
	WarningCount        int               `json:"warningCount"`
	FailCount           int               `json:"failCount"`
	OverallPass         bool              `json:"overallPass"`
	CopyGuard           string            `json:"copyGuard"`
	SystemSharpnessPass bool              `json:"systemSharpnessPass"`
	Artifacts           map[string]string `json:"artifacts"`
}

type ParallaxSummary struct {
	GlyphCount            int      `json:"glyphCount"`
	MeanHorizontalPx      float64  `json:"meanHorizontalPxLeftMinusRight"`
	MeanAngularArcMin     float64  `json:"meanAngularArcMin"`
	MeanCopyErrorPx       float64  `json:"meanNaiveCopyErrorPx"`
	MinCopyErrorPx        float64  `json:"minNaiveCopyErrorPx"`
	MaxAbsVerticalPx      float64  `json:"maxAbsVerticalPx"`
	MaxAbsVerticalArcMin  float64  `json:"maxAbsVerticalArcMin"`
	CoordinatesCopied     bool     `json:"coordinatesCopied"`
	IndependentProjection bool     `json:"independentProjection"`
	ProjectionPath        string   `json:"projectionPath"`
	Status                string   `json:"status"`
	Warnings              []string `json:"warnings,omitempty"`
	Failures              []string `json:"failures,omitempty"`
}

type ElementReport struct {
	Spec        TextSpec                    `json:"spec"`
	Status      string                      `json:"status"`
	Parallax    ParallaxSummary             `json:"parallax"`
	Boundary    map[EyeID]BoundaryResult    `json:"boundary"`
	Readability map[EyeID]ReadabilityResult `json:"readability"`
	Sharpness   map[EyeID]SharpnessResult   `json:"sharpness,omitempty"`
}

type LayerReport struct {
	LayerID     string              `json:"layerId"`
	NameZH      string              `json:"nameZh"`
	DepthClass  string              `json:"depthClass"`
	FrameFiles  map[EyeID]string    `json:"frameFiles"`
	PanelBounds map[EyeID]FloatRect `json:"panelBounds"`
	Glyphs      []GlyphPair         `json:"glyphs"`
	Elements    []ElementReport     `json:"elements"`
	Status      string              `json:"status"`
}

type ScenarioReport struct {
	Scenario
	Eyes   [2]Eye        `json:"eyes"`
	Layers []LayerReport `json:"layers"`
}

type Report struct {
	Schema      string           `json:"schema"`
	GeneratedAt string           `json:"generatedAt"`
	Generator   string           `json:"generator"`
	Options     RunOptions       `json:"options"`
	Summary     ReportSummary    `json:"summary"`
	Scenarios   []ScenarioReport `json:"scenarios"`
}
