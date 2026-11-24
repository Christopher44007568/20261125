// Package model defines the marker-pixel input model and shared enums for the
// binocular transparent-material diagnostic classifier.
package model

// Eye identifies one rendered eye.
type Eye string

const (
	Left  Eye = "left"
	Right Eye = "right"
)

// RGBA is a linear-ish 0..1 color record. A is the material source alpha.
type RGBA struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
	A float64 `json:"a"`
}

// Observation is everything the capture pipeline can measure at one marker
// pixel for one eye. Ground-truth depth/color/coverage travel with the
// marker, so observed values can always be compared against expectations.
type Observation struct {
	Depth          float64 `json:"depth"`          // measured window-space depth
	Color          RGBA    `json:"color"`          // measured composite color
	Coverage       float64 `json:"coverage"`       // measured alpha coverage 0..1
	EdgeCoverage   float64 `json:"edgeCoverage"`   // edge-pixel fraction of coverage
	SampleCount    int     `json:"sampleCount"`    // MSAA samples actually used
	DrawOrder      []int   `json:"drawOrder"`      // transparent layer draw order, back to front
	VertexDepth    float64 `json:"vertexDepth"`    // depth interpolated at the vertex
	ParallaxOffset float64 `json:"parallaxOffset"` // per-eye UV/parallax offset applied
}

// Marker is a known-depth, known-color, known-coverage marker pixel plus the
// rendering trace for each eye.
type Marker struct {
	ID              string              `json:"id"`
	X               int                 `json:"x"`
	Y               int                 `json:"y"`
	LayerCount      int                 `json:"layerCount"`
	ExpectedOrder   []int               `json:"expectedOrder"`
	ExpectedDepth   float64             `json:"expectedDepth"`
	ExpectedColor   RGBA                `json:"expectedColor"`
	ExpectedCover   float64             `json:"expectedCoverage"`
	ExpectedSamples int                 `json:"expectedSamples"`
	Budget          BudgetStatus        `json:"budget"`
	Eyes            map[Eye]Observation `json:"eyes"`
}

// BudgetStatus records the renderer's transparent/AA budget for the frame.
type BudgetStatus struct {
	Degraded        bool    `json:"degraded"`
	DepthPrepassOff bool    `json:"depthPrepassOff"`
	AlphaToCoverage bool    `json:"alphaToCoverage"`
	MSAAHalved      bool    `json:"msaaHalved"`
	SampleBudget    int     `json:"sampleBudget"`
	OverdrawRatio   float64 `json:"overdrawRatio"`
}

// Case is one diagnostic input: many markers from one stereo frame pair.
type Case struct {
	Name    string   `json:"name"`
	Markers []Marker `json:"markers"`
}

// Cause is one of the four causal classes. Classification never collapses to
// a single aggregate pixel difference: every cause is scored independently
// from marker-local evidence.
type Cause string

const (
	Sorting  Cause = "sorting"  // 透明排序错误
	Parallax Cause = "parallax" // 视差/顶点深度错误
	Alpha    Cause = "alpha"    // Alpha 混合/覆盖率错误
	AA       Cause = "aa"       // 抗锯齿/采样错误
)

// Stage is where a confirmed cause is localized.
type Stage string

const (
	VertexStage Stage = "vertex" // 顶点阶段
	LayerStage  Stage = "layer"  // 图层阶段
	SampleStage Stage = "sample" // 采样阶段
)

// Thresholds controls classifier firing thresholds. Zero values fall back to
// Defaults.
type Thresholds struct {
	FireScore   float64 `json:"fireScore"`
	ColorEps    float64 `json:"colorEps"`
	DepthEps    float64 `json:"depthEps"`
	CoverEps    float64 `json:"coverEps"`
	ParallaxEps float64 `json:"parallaxEps"`
	OrderEps    float64 `json:"orderEps"`
}

// Defaults returns the empirically balanced threshold set.
func Defaults() Thresholds {
	return Thresholds{
		FireScore:   0.55,
		ColorEps:    0.04,
		DepthEps:    0.02,
		CoverEps:    0.05,
		ParallaxEps: 0.03,
		OrderEps:    0,
	}
}

// Resolve fills zero-valued fields with defaults.
func (t Thresholds) Resolve() Thresholds {
	d := Defaults()
	if t.FireScore == 0 {
		t.FireScore = d.FireScore
	}
	if t.ColorEps == 0 {
		t.ColorEps = d.ColorEps
	}
	if t.DepthEps == 0 {
		t.DepthEps = d.DepthEps
	}
	if t.CoverEps == 0 {
		t.CoverEps = d.CoverEps
	}
	if t.ParallaxEps == 0 {
		t.ParallaxEps = d.ParallaxEps
	}
	return t
}
