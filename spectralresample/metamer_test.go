package spectralresample

import (
	"math"
	"math/rand"
	"testing"
)

func TestQRNullSpace(t *testing.T) {
	// Rows of a are constraints: a null vector must satisfy A v = 0.
	a := [][]float64{
		{1, 0, 0, 2, 1},
		{0, 1, 0, -1, 3},
		{0, 0, 1, 1, -1},
	}
	qr := NewQR(transpose(a))
	basis := qr.NullSpace()
	if len(basis) != 2 {
		t.Fatalf("nullity = %d, want 2", len(basis))
	}
	for _, vector := range basis {
		for _, row := range a {
			residual := math.Abs(dot(row, vector))
			if residual > 1e-10 {
				t.Errorf("null vector violates constraint: residual %.2e", residual)
			}
		}
	}
}

func TestMetamerMatchUnderD65(t *testing.T) {
	cmf, err := LoadCMF("data")
	if err != nil {
		t.Fatal(err)
	}
	d65, err := LoadD65("data")
	if err != nil {
		t.Fatal(err)
	}
	_, patches, err := LoadReflectanceCSV("data/colorchecker_ohta_5nm.csv")
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultMetamerConfig()
	rng := rand.New(rand.NewSource(cfg.Seed))
	for i, base := range patches {
		pair := cmf.BuildMetamerPair(d65, base, "patch", "all", cfg, rng)
		residual := cmf.TristimulusResidual(d65, pair)
		if residual > 0.05 {
			t.Errorf("patch %d: D65 tristimulus residual %.4f exceeds 0.05", i, residual)
		}
		for j, value := range pair.Metamer.Values {
			if value < 0 || value > 1 {
				t.Errorf("patch %d band %d: metamer reflectance %.4f outside [0,1]", i, j, value)
			}
		}
	}
}

func TestMetamerDivergesUnderLED(t *testing.T) {
	cmf, err := LoadCMF("data")
	if err != nil {
		t.Fatal(err)
	}
	d65, err := LoadD65("data")
	if err != nil {
		t.Fatal(err)
	}
	led := NewGaussianLED("narrowband", []LEDEmitter{
		{PeakNM: 450, FWHMNM: 22, PeakPower: 1},
		{PeakNM: 530, FWHMNM: 22, PeakPower: 0.85},
		{PeakNM: 630, FWHMNM: 20, PeakPower: 0.7},
	})
	_ = led
	_, patches, err := LoadReflectanceCSV("data/colorchecker_ohta_5nm.csv")
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultMetamerConfig()
	rng := rand.New(rand.NewSource(cfg.Seed))
	maxResidual := 0.0
	for _, base := range patches {
		pair := cmf.BuildMetamerPair(d65, base, "patch", "all", cfg, rng)
		maxResidual = math.Max(maxResidual, cmf.TristimulusResidual(led, pair))
	}
	if maxResidual < 1.0 {
		t.Errorf("LED residual max %.4f, want spectral mismatch > 1 XYZ unit", maxResidual)
	}
}
