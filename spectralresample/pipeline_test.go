package spectralresample

import "testing"

func testPipeline(t *testing.T) *Pipeline {
	t.Helper()
	cfg := PipelineConfig{
		DataDir: "data",
		LEDName: "test led",
		LED: []LEDEmitter{
			{PeakNM: 450, FWHMNM: 22, PeakPower: 0.5},
			{PeakNM: 530, FWHMNM: 22, PeakPower: 0.7},
			{PeakNM: 630, FWHMNM: 20, PeakPower: 0.85},
		},
		Metamer: DefaultMetamerConfig(),
	}
	pipeline, err := NewPipeline(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return pipeline
}

func TestPipelineFamiliesPresent(t *testing.T) {
	pipeline := testPipeline(t)
	for _, family := range []string{FamilyGrey, FamilySkin, FamilyOrange} {
		if len(pipeline.ByFamily(family)) == 0 {
			t.Errorf("family %s has no patches", family)
		}
	}
}

func TestPipelineMetamerGuarantees(t *testing.T) {
	pipeline := testPipeline(t)
	ledMismatch := 0.0
	for _, result := range pipeline.Results {
		if result.DeltaED65 > 0.05 {
			t.Errorf("%s: D65 dE2000 %.4f exceeds metamer tolerance", result.Label, result.DeltaED65)
		}
		if result.GainPairSplit > 1e-9 {
			t.Errorf("%s: RGB gain split %.2e must be zero for a metamer pair", result.Label, result.GainPairSplit)
		}
		ledMismatch = maxFloat(ledMismatch, result.DeltaELED)
		for _, value := range result.Metamer.Values {
			if value < 0 || value > 1 {
				t.Errorf("%s: metamer reflectance %.4f outside [0,1]", result.Label, value)
			}
		}
	}
	if ledMismatch < 1.0 {
		t.Errorf("max LED dE2000 %.4f, expected visible spectral mismatch", ledMismatch)
	}
}

func TestFamilyStatistics(t *testing.T) {
	pipeline := testPipeline(t)
	grey := pipeline.ByFamily(FamilyGrey)
	if len(grey) != 6 {
		t.Fatalf("grey patches = %d, want 6 neutrals", len(grey))
	}
	skin := pipeline.ByFamily(FamilySkin)
	if len(skin) != 2 {
		t.Fatalf("skin patches = %d, want 2", len(skin))
	}
	orange := pipeline.ByFamily(FamilyOrange)
	if len(orange) != 2 {
		t.Fatalf("orange patches = %d, want 2", len(orange))
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
