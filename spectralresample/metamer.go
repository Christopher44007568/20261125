package spectralresample

import (
	"math"
	"math/rand"
)

// MetamerPair is a base reflectance spectrum and an alternative reflectance
// that is metameric to it under the reference illuminant (identical
// tristimulus values) but spectrally different, so it diverges under the
// narrow-band LED.
type MetamerPair struct {
	Name       string
	Family     string
	Base       Spectrum
	Metamer    Spectrum
	NullWeight []float64
}

// MetamerConfig controls the null-space metamer construction.
type MetamerConfig struct {
	Harmonics  int
	Amplitude  float64
	Iterations int
	Seed       int64
}

// DefaultMetamerConfig gives smooth, bounded perturbations strong enough to
// remain visible in the error maps without driving reflectance outside
// [0, 1] for typical ColorChecker spectra.
func DefaultMetamerConfig() MetamerConfig {
	return MetamerConfig{Harmonics: 6, Amplitude: 0.22, Iterations: 60, Seed: 20261125}
}

// smoothBasis builds wavelength-domain sinusoidal basis vectors on the
// integration grid, starting with several full cycles across the visible
// range. Perturbations live in this smooth subspace, so generated spectra
// stay physically plausible rather than high-frequency noise.
func smoothBasis(harmonics int) [][]float64 {
	grid := Grid()
	n := len(grid)
	basis := make([][]float64, harmonics)
	for h := 0; h < harmonics; h++ {
		cycles := h + 2
		vector := make([]float64, n)
		for i, lambda := range grid {
			phase := float64(cycles) * math.Pi * (lambda - LambdaMin) / (LambdaMax - LambdaMin)
			vector[i] = math.Sin(phase)
		}
		normalize(vector)
		basis[h] = vector
	}
	return basis
}

// BuildMetamerPair constructs a smooth null-space perturbation of the base
// spectrum under the reference illuminant and alternates two projections:
//
//  1. project onto the tristimulus constraint B delta = 0,
//  2. clip the resulting reflectance into [0, 1].
//
// The fixed point preserves the D65 tristimulus match within numerical
// tolerance while producing a visibly different spectral curve.
func (c CMF) BuildMetamerPair(ill Illuminant, base Spectrum, name, family string, cfg MetamerConfig, rng *rand.Rand) MetamerPair {
	grid := Grid()
	n := len(grid)
	baseR := base.Resample()

	// Constraint matrix: rows are the three weighted CMF channels so that
	// B delta = 0 means the perturbation adds zero to X, Y and Z.
	bMatrix := make([][]float64, 3)
	for k, channel := range [3][]float64{c.Xbar, c.Ybar, c.Zbar} {
		row := make([]float64, n)
		for i := range row {
			row[i] = ill.Power[i] * channel[i]
		}
		bMatrix[k] = row
	}
	qr := NewQR(transpose(bMatrix))
	null := qr.NullSpace()

	// Express the smooth sinusoidal basis in null-space coordinates.
	smooth := smoothBasis(cfg.Harmonics)
	projected := make([][]float64, len(smooth))
	for i, vector := range smooth {
		projected[i] = projectToNull(vector, null)
		normalize(projected[i])
	}

	delta := make([]float64, n)
	weights := make([]float64, len(projected))
	for i := range weights {
		weights[i] = rng.NormFloat64()
	}
	normalize(weights)
	for i, vector := range projected {
		for j := range delta {
			delta[j] += cfg.Amplitude * weights[i] * vector[j]
		}
	}

	metamer := make([]float64, n)
	for iter := 0; iter < cfg.Iterations; iter++ {
		for i := range metamer {
			metamer[i] = baseR.Values[i] + delta[i]
		}
		for i := range metamer {
			metamer[i] = Clamp01(metamer[i])
		}
		currentDelta := make([]float64, n)
		for i := range currentDelta {
			currentDelta[i] = metamer[i] - baseR.Values[i]
		}
		delta = projectToNull(currentDelta, null)
	}
	for i := range metamer {
		metamer[i] = Clamp01(baseR.Values[i] + delta[i])
	}

	return MetamerPair{
		Name:       name,
		Family:     family,
		Base:       baseR,
		Metamer:    Spectrum{Wavelengths: grid, Values: metamer},
		NullWeight: weights,
	}
}

// projectToNull projects vector onto the span of the orthonormal null-space
// basis, guaranteeing B vector = 0 up to floating-point error.
func projectToNull(vector []float64, basis [][]float64) []float64 {
	out := make([]float64, len(vector))
	for _, direction := range basis {
		coefficient := dot(vector, direction)
		for i := range out {
			out[i] += coefficient * direction[i]
		}
	}
	return out
}

func transpose(a [][]float64) [][]float64 {
	m, n := len(a), len(a[0])
	out := make([][]float64, n)
	for j := range out {
		out[j] = make([]float64, m)
		for i := 0; i < m; i++ {
			out[j][i] = a[i][j]
		}
	}
	return out
}

// TristimulusResidual reports max |XYZ_pair - XYZ_base| under an
// illuminant, used to verify the metameric match numerically.
func (c CMF) TristimulusResidual(ill Illuminant, pair MetamerPair) float64 {
	base := c.Tristimulus(ill, pair.Base)
	meta := c.Tristimulus(ill, pair.Metamer)
	worst := 0.0
	for k := range base {
		worst = math.Max(worst, math.Abs(meta[k]-base[k]))
	}
	return worst
}
