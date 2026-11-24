package features

import (
	"testing"

	"eyediag/internal/model"
	"eyediag/internal/synth"
)

func TestHealthyEyeFeaturesAreQuiet(t *testing.T) {
	c, _ := synth.Generate(synth.ScenClean, 1)
	m := c.Markers[0]
	f := Extract(m, model.Left, nil, model.Defaults())

	if f.OrderMismatchOK || f.DepthAliasOK || !f.DepthOK || !f.CoverOK ||
		!f.SampleOK || !f.ParallaxOK {
		t.Fatalf("healthy eye produced active fault signals: %+v", f)
	}
	if f.ColorDist > model.Defaults().ColorEps {
		t.Fatalf("healthy color distance %.4f exceeds tolerance", f.ColorDist)
	}
}

func TestSortingFaultDrivesOrderSignals(t *testing.T) {
	c, _ := synth.Generate(synth.ScenSorting, 1)
	m := c.Markers[0]
	f := Extract(m, model.Right, nil, model.Defaults())

	if !f.OrderMismatchOK || f.OrderMismatch != 1 {
		t.Fatalf("order mismatch = %.2f, want 1", f.OrderMismatch)
	}
	if !f.DepthAliasOK {
		t.Fatal("equal-depth swapped layers must flag depth alias")
	}
}

func TestParallaxFaultDrivesVertexAndDisparity(t *testing.T) {
	c, _ := synth.Generate(synth.ScenParallax, 1)
	m := c.Markers[0]
	right := m.Eyes[model.Right]
	f := Extract(m, model.Left, &right, model.Defaults())

	if f.VertexDelta <= model.Defaults().DepthEps {
		t.Fatalf("vertex delta %.4f should exceed depth eps", f.VertexDelta)
	}
	if f.ParallaxOK || f.DisparityErr <= model.Defaults().ParallaxEps {
		t.Fatalf("parallax signals not raised: parallaxErr=%.2f disparity=%.2f",
			f.ParallaxErr, f.DisparityErr)
	}
}

func TestAlphaFaultIsCoverageNotGeometry(t *testing.T) {
	c, _ := synth.Generate(synth.ScenAlpha, 1)
	m := c.Markers[0]
	f := Extract(m, model.Left, nil, model.Defaults())

	if f.CoverOK || f.CoverageDelta < 0.1 {
		t.Fatalf("coverage delta %.4f too small", f.CoverageDelta)
	}
	if !f.DepthOK {
		t.Fatal("alpha fault must leave depth unchanged")
	}
	if f.AlphaLeakFit <= 0.5 {
		t.Fatalf("blend inversion fit %.2f should explain the color shift", f.AlphaLeakFit)
	}
}

func TestAAFaultDrivesSampleDeficitAndEdge(t *testing.T) {
	c, _ := synth.Generate(synth.ScenAA, 1)
	m := c.Markers[0]
	f := Extract(m, model.Left, nil, model.Defaults())

	if f.SampleOK || f.SampleDeficit < 0.5 {
		t.Fatalf("sample deficit %.2f should reflect 2/8 samples", f.SampleDeficit)
	}
	if f.EdgeDelta < 0.1 {
		t.Fatalf("edge delta %.2f should capture the jagged edge", f.EdgeDelta)
	}
}
