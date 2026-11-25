package spectralresample

import (
	"math"
	"strings"
	"testing"
)

func approx(t *testing.T, got, want, tol float64, label string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %.6f, want %.6f (tol %.4f)", label, got, want, tol)
	}
}

func TestD65WhitePoint(t *testing.T) {
	cmf, err := LoadCMF("data")
	if err != nil {
		t.Fatal(err)
	}
	d65, err := LoadD65("data")
	if err != nil {
		t.Fatal(err)
	}
	white := cmf.WhitePoint(d65)
	approx(t, white[0], 95.047, 0.25, "D65 Xn")
	approx(t, white[1], 100.0, 1e-9, "D65 Yn")
	approx(t, white[2], 108.883, 0.25, "D65 Zn")
}

func TestPerfectWhite(t *testing.T) {
	cmf, err := LoadCMF("data")
	if err != nil {
		t.Fatal(err)
	}
	d65, err := LoadD65("data")
	if err != nil {
		t.Fatal(err)
	}
	grid := Grid()
	ones := Spectrum{Wavelengths: grid, Values: make([]float64, len(grid))}
	for i := range ones.Values {
		ones.Values[i] = 1
	}
	xyz := cmf.Tristimulus(d65, ones)
	white := cmf.WhitePoint(d65)
	for k := 0; k < 3; k++ {
		approx(t, xyz[k], white[k], 1e-6, "perfect white XYZ")
	}
}

func TestColorCheckerNeutralY(t *testing.T) {
	cmf, err := LoadCMF("data")
	if err != nil {
		t.Fatal(err)
	}
	d65, err := LoadD65("data")
	if err != nil {
		t.Fatal(err)
	}
	names, patches, err := LoadReflectanceCSV("data/colorchecker_ohta_5nm.csv")
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range names {
		if !strings.Contains(name, "neutral") && !strings.Contains(name, "white") && !strings.Contains(name, "black") {
			continue
		}
		xyz := cmf.Tristimulus(d65, patches[i])
		// For a near-flat neutral, Y must track the ybar-weighted mean
		// reflectance (the exact photopic-weighted prediction).
		resampled := patches[i].Resample()
		weighted, weight := 0.0, 0.0
		for i, value := range resampled.Values {
			weighted += value * d65.Power[i] * cmf.Ybar[i]
			weight += d65.Power[i] * cmf.Ybar[i]
		}
		approx(t, xyz[1], 100*weighted/weight, 0.5, name+" Y vs photopic mean")
	}
}

func TestDeltaE2000KnownPairs(t *testing.T) {
	// Sharma, Wu, Dalal (2005), Table 1 reference pairs.
	pairs := []struct {
		lab1, lab2 [3]float64
		want       float64
	}{
		{[3]float64{50, 2.6772, -79.7751}, [3]float64{50, 0, -82.7485}, 2.0425},
		{[3]float64{50, 3.1571, -77.2803}, [3]float64{50, 0, -82.7485}, 2.8615},
		{[3]float64{50, 2.8361, -74.0200}, [3]float64{50, 0, -82.7485}, 3.4412},
		{[3]float64{50, -1.3802, -84.2814}, [3]float64{50, 0, -82.7485}, 1.0000},
		{[3]float64{50, 2.5, 0}, [3]float64{73, 25, -18}, 27.1492},
		{[3]float64{50, 2.5, 0}, [3]float64{61, -5, 29}, 22.8977},
		{[3]float64{50, 2.5, 0}, [3]float64{56, -27, -3}, 31.9030},
		{[3]float64{50, 2.5, 0}, [3]float64{58, 24, 15}, 19.4535},
	}
	for i, pair := range pairs {
		got := DeltaE2000(pair.lab1, pair.lab2)
		approx(t, got, pair.want, 0.01, "CIEDE2000 pair "+string(rune('A'+i)))
	}
}

func TestResampleRoundTrip(t *testing.T) {
	grid := Grid()
	spec := Spectrum{
		Wavelengths: []float64{380, 480, 580, 680, 780},
		Values:      []float64{0.1, 0.4, 0.9, 0.5, 0.2},
	}
	r := spec.Resample()
	if len(r.Values) != BandCount {
		t.Fatalf("resampled length = %d, want %d", len(r.Values), BandCount)
	}
	if r.Wavelengths[0] != grid[0] || r.Wavelengths[BandCount-1] != grid[BandCount-1] {
		t.Fatal("resampled grid mismatch")
	}
	approx(t, r.At(480), 0.4, 1e-12, "node interpolation")
	approx(t, r.At(430), 0.25, 1e-12, "midpoint interpolation")
	approx(t, r.At(350), 0.1, 1e-12, "edge extrapolation low")
	approx(t, r.At(800), 0.2, 1e-12, "edge extrapolation high")
}
