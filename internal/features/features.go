// Package features turns raw marker observations into per-eye evidence
// signals consumed by the causal classifier.
package features

import (
	"math"

	"eyediag/internal/model"
)

// EyeFeatures holds normalized 0..1 (or bool) signals for one eye of one
// marker. Signals are deliberately independent so the four causes can fire
// together instead of being forced into one bucket.
type EyeFeatures struct {
	Eye             model.Eye
	ColorDist       float64 // observed vs expected composite color
	DepthDelta      float64 // observed window depth vs expected
	VertexDelta     float64 // vertex/interpolated depth vs expected
	CoverageDelta   float64 // absolute coverage error
	EdgeDelta       float64 // edge coverage error
	SampleDeficit   float64 // 0 when full MSAA, ->1 when samples collapse
	OrderMismatch   float64 // fraction of transparent layers drawn out of order
	DepthAlias      float64 // 1 when swapped layers share near-identical depth
	ParallaxErr     float64 // per-eye offset deviation from expected parity
	DisparityErr    float64 // inter-eye depth disparity deviation
	AlphaLeakFit    float64 // how well an alpha-blend inversion explains color
	OverPremul      float64 // premultiplied-alpha error strength
	OrderMismatchOK bool
	DepthAliasOK    bool
	DepthOK         bool
	CoverOK         bool
	SampleOK        bool
	ParallaxOK      bool
}

// Extract builds the feature vector for one eye. The counterpart eye is
// needed for the inter-eye disparity signal; pass nil when the counterpart
// was not captured.
func Extract(m model.Marker, eye model.Eye, other *model.Observation, th model.Thresholds) EyeFeatures {
	obs := m.Eyes[eye]
	f := EyeFeatures{Eye: eye}

	f.ColorDist = colorDistance(obs.Color, m.ExpectedColor)
	f.DepthDelta = math.Abs(obs.Depth - m.ExpectedDepth)
	f.VertexDelta = math.Abs(obs.VertexDepth - m.ExpectedDepth)
	f.CoverageDelta = math.Abs(obs.Coverage - m.ExpectedCover)
	f.EdgeDelta = math.Abs(obs.EdgeCoverage - m.ExpectedCover*0.5)

	if m.ExpectedSamples > 0 {
		f.SampleDeficit = clamp(1 - float64(obs.SampleCount)/float64(m.ExpectedSamples))
	}

	f.OrderMismatch, f.DepthAlias = orderSignals(obs.DrawOrder, m.ExpectedOrder, m, obs)
	f.OrderMismatchOK = f.OrderMismatch > th.OrderEps
	f.DepthAliasOK = f.DepthAlias > 0.5
	f.DepthOK = f.DepthDelta <= th.DepthEps
	f.CoverOK = f.CoverageDelta <= th.CoverEps
	f.SampleOK = f.SampleDeficit <= 0.05

	// Parallax: expected per-eye offsets are mirrored around zero, so the
	// correct magnitude equals the expected coverage-weighted baseline.
	expectedOffset := m.ExpectedCover * 0.1
	if eye == model.Left {
		expectedOffset = -expectedOffset
	}
	f.ParallaxErr = math.Abs(obs.ParallaxOffset-expectedOffset) / 0.1
	f.ParallaxOK = f.ParallaxErr <= th.ParallaxEps
	if other != nil {
		wantDisparity := math.Abs(expectedOffset) * 2
		gotDisparity := math.Abs(obs.Depth - other.Depth)
		f.DisparityErr = clamp(math.Abs(gotDisparity-wantDisparity) / 0.1)
	}

	f.AlphaLeakFit = leakFit(obs, m)
	f.OverPremul = premulError(obs, m)
	return f
}

func orderSignals(got, want []int, m model.Marker, obs model.Observation) (mismatch, alias float64) {
	n := len(want)
	if n == 0 || len(got) != n {
		return 0, 0
	}
	wrong := 0
	for i := range want {
		if got[i] != want[i] {
			wrong++
		}
	}
	mismatch = float64(wrong) / float64(n)
	if wrong > 0 && fabs(obs.Depth-m.ExpectedDepth) <= 0.05 {
		alias = 1
	}
	return mismatch, alias
}

// leakFit inverts "over = src*a + dst*(1-a)": if the observed color is
// explainable by a coverage/blend leak while geometry stayed put, this
// approaches 1.
func leakFit(obs model.Observation, m model.Marker) float64 {
	if fabs(obs.Depth-m.ExpectedDepth) > 0.03 {
		return 0
	}
	blendErr := colorDistance(obs.Color, blend(m.ExpectedColor, obs.Coverage))
	return clamp(1 - blendErr/0.2)
}

func premulError(obs model.Observation, m model.Marker) float64 {
	expectedPremul := m.ExpectedColor.R * obs.Coverage
	if obs.Color.R < expectedPremul-0.02 {
		return clamp((expectedPremul - obs.Color.R) / 0.2)
	}
	return 0
}

func blend(src model.RGBA, coverage float64) model.RGBA {
	// Background proxy: 18% gray, the standard neutral backdrop.
	bg := model.RGBA{R: 0.18, G: 0.18, B: 0.18, A: 1}
	a := clamp(coverage)
	return model.RGBA{
		R: src.R*a + bg.R*(1-a),
		G: src.G*a + bg.G*(1-a),
		B: src.B*a + bg.B*(1-a),
		A: a,
	}
}

func colorDistance(a, b model.RGBA) float64 {
	dr, dg, db := a.R-b.R, a.G-b.G, a.B-b.B
	return math.Sqrt(dr*dr+dg*dg+db*db) / math.Sqrt(3)
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func fabs(v float64) float64 { return math.Abs(v) }
