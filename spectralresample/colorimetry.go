package spectralresample

import "math"

// CMF holds the CIE 1931 2 degree standard observer colour matching functions
// xbar, ybar and zbar on the shared wavelength grid.
type CMF struct {
	Wavelengths []float64
	Xbar        []float64
	Ybar        []float64
	Zbar        []float64
}

// LoadCMF reads the extracted CIE 1931 table and re-samples it onto the
// shared grid.
func LoadCMF(dir string) (CMF, error) {
	x, err := LoadColumnCSV(dir+"/cie1931_2deg_5nm.csv", "xbar")
	if err != nil {
		return CMF{}, err
	}
	y, err := LoadColumnCSV(dir+"/cie1931_2deg_5nm.csv", "ybar")
	if err != nil {
		return CMF{}, err
	}
	z, err := LoadColumnCSV(dir+"/cie1931_2deg_5nm.csv", "zbar")
	if err != nil {
		return CMF{}, err
	}
	xr, yr, zr := x.Resample(), y.Resample(), z.Resample()
	return CMF{Wavelengths: xr.Wavelengths, Xbar: xr.Values, Ybar: yr.Values, Zbar: zr.Values}, nil
}

// Illuminant is a spectral power distribution sampled on the shared grid.
// Values carry arbitrary relative power units; tristimulus normalisation
// removes the absolute scale.
type Illuminant struct {
	Name        string
	Wavelengths []float64
	Power       []float64
}

// LoadD65 reads the tabulated CIE D65 SPD and re-samples it.
func LoadD65(dir string) (Illuminant, error) {
	spec, err := LoadColumnCSV(dir+"/d65_5nm.csv", "power")
	if err != nil {
		return Illuminant{}, err
	}
	r := spec.Resample()
	return Illuminant{Name: "D65", Wavelengths: r.Wavelengths, Power: r.Values}, nil
}

// GaussianLED builds a narrow-band LED SPD: one or more Gaussian emitters
// centred at peak wavelengths (nm) with full width at half maximum widths
// (nm) and relative peak powers.
type LEDEmitter struct {
	PeakNM    float64
	FWHMNM    float64
	PeakPower float64
}

// NewGaussianLED constructs the LED illuminant from its emitter list.
func NewGaussianLED(name string, emitters []LEDEmitter) Illuminant {
	grid := Grid()
	power := make([]float64, len(grid))
	for i, lambda := range grid {
		for _, emitter := range emitters {
			sigma := emitter.FWHMNM / (2 * math.Sqrt(2*math.Ln2))
			d := lambda - emitter.PeakNM
			power[i] += emitter.PeakPower * math.Exp(-(d*d)/(2*sigma*sigma))
		}
	}
	return Illuminant{Name: name, Wavelengths: grid, Power: power}
}

// trapezoid integrates y over x with the composite trapezoidal rule. The
// grid is uniformly 5 nm, but the rule is written generically.
func trapezoid(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}
	sum := 0.0
	for i := 1; i < len(x); i++ {
		sum += (x[i] - x[i-1]) * (y[i-1] + y[i]) / 2
	}
	return sum
}

// WhitePoint integrates the illuminant against each CMF channel and
// normalises Y to 100, returning the reference white XYZ.
func (c CMF) WhitePoint(ill Illuminant) [3]float64 {
	n := len(c.Wavelengths)
	channels := [3][]float64{c.Xbar, c.Ybar, c.Zbar}
	var white [3]float64
	for k := 0; k < 3; k++ {
		products := make([]float64, n)
		for i := 0; i < n; i++ {
			products[i] = ill.Power[i] * channels[k][i]
		}
		white[k] = trapezoid(ill.Wavelengths, products)
	}
	denominator := white[1]
	for k := range white {
		white[k] *= 100 / denominator
	}
	return white
}

// Tristimulus integrates the fixed spectral response against the product of
// illuminant SPD and surface reflectance, then normalises to the illuminant
// white point so Y equals 100 for a perfect white reflector.
func (c CMF) Tristimulus(ill Illuminant, reflectance Spectrum) [3]float64 {
	r := reflectance.Resample()
	n := len(c.Wavelengths)
	channels := [3][]float64{c.Xbar, c.Ybar, c.Zbar}
	var xyz [3]float64
	denominator := make([]float64, n)
	for i := 0; i < n; i++ {
		denominator[i] = ill.Power[i] * c.Ybar[i]
	}
	norm := trapezoid(ill.Wavelengths, denominator)
	for k := 0; k < 3; k++ {
		product := make([]float64, n)
		for i := 0; i < n; i++ {
			product[i] = ill.Power[i] * r.Values[i] * channels[k][i]
		}
		xyz[k] = 100 * trapezoid(ill.Wavelengths, product) / norm
	}
	return xyz
}
