// Package spectralresample implements wavelength-domain reflectance
// resampling and integration of fixed spectral responses (CIE 1931 2 degree
// standard observer colour matching functions) into CIE XYZ tristimulus
// values under arbitrary illuminant spectral power distributions.
package spectralresample

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
)

const (
	LambdaMin = 380.0
	LambdaMax = 780.0
	BandStep  = 5.0
	BandCount = 81
)

// Grid returns the shared 5 nm integration grid from 380 nm to 780 nm.
func Grid() []float64 {
	grid := make([]float64, BandCount)
	for i := range grid {
		grid[i] = LambdaMin + float64(i)*BandStep
	}
	return grid
}

// Spectrum is a tabulated spectral curve sampled at Wavelengths.
type Spectrum struct {
	Wavelengths []float64
	Values      []float64
}

// At evaluates the curve at lambda using piecewise linear interpolation and
// constant (edge) extrapolation, which keeps reflectance bounded outside the
// measured wavelength interval.
func (s Spectrum) At(lambda float64) float64 {
	if len(s.Values) == 0 {
		return 0
	}
	if lambda <= s.Wavelengths[0] {
		return s.Values[0]
	}
	last := len(s.Values) - 1
	if lambda >= s.Wavelengths[last] {
		return s.Values[last]
	}
	hi := 1
	for hi < len(s.Wavelengths) && s.Wavelengths[hi] < lambda {
		hi++
	}
	lo := hi - 1
	t := (lambda - s.Wavelengths[lo]) / (s.Wavelengths[hi] - s.Wavelengths[lo])
	return s.Values[lo] + t*(s.Values[hi]-s.Values[lo])
}

// Resample re-samples the curve onto the shared 5 nm grid. This is the
// multi-band resampling step: every downstream integration operates on the
// same wavelength basis instead of per-channel scalar gains.
func (s Spectrum) Resample() Spectrum {
	grid := Grid()
	out := Spectrum{Wavelengths: grid, Values: make([]float64, len(grid))}
	for i, lambda := range grid {
		out.Values[i] = s.At(lambda)
	}
	return out
}

// LoadColumnCSV reads a one-value-per-wavelength CSV with a named column.
func LoadColumnCSV(path, column string) (Spectrum, error) {
	records, header, err := readCSV(path)
	if err != nil {
		return Spectrum{}, err
	}
	col, err := columnIndex(header, column)
	if err != nil {
		return Spectrum{}, fmt.Errorf("%s: %w", path, err)
	}
	spec := Spectrum{}
	for row, record := range records {
		lambda, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			return Spectrum{}, fmt.Errorf("%s row %d: bad wavelength: %w", path, row+2, err)
		}
		value, err := strconv.ParseFloat(record[col], 64)
		if err != nil {
			return Spectrum{}, fmt.Errorf("%s row %d: bad value: %w", path, row+2, err)
		}
		spec.Wavelengths = append(spec.Wavelengths, lambda)
		spec.Values = append(spec.Values, value)
	}
	return spec, nil
}

// LoadReflectanceCSV reads every non-wavelength column as a reflectance
// patch, preserving column order.
func LoadReflectanceCSV(path string) ([]string, []Spectrum, error) {
	records, header, err := readCSV(path)
	if err != nil {
		return nil, nil, err
	}
	names := append([]string(nil), header[1:]...)
	patches := make([]Spectrum, len(names))
	for i := range patches {
		patches[i] = Spectrum{
			Wavelengths: make([]float64, 0, len(records)),
			Values:      make([]float64, 0, len(records)),
		}
	}
	for row, record := range records {
		lambda, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			return nil, nil, fmt.Errorf("%s row %d: bad wavelength: %w", path, row+2, err)
		}
		for i := range names {
			value, err := strconv.ParseFloat(record[i+1], 64)
			if err != nil {
				return nil, nil, fmt.Errorf("%s row %d: bad value: %w", path, row+2, err)
			}
			patches[i].Wavelengths = append(patches[i].Wavelengths, lambda)
			patches[i].Values = append(patches[i].Values, value)
		}
	}
	return names, patches, nil
}

func readCSV(path string) ([][]string, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	reader := csv.NewReader(bufio.NewReader(f))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(records) < 2 {
		return nil, nil, fmt.Errorf("%s: expected header plus rows", path)
	}
	return records[1:], records[0], nil
}

func columnIndex(header []string, column string) (int, error) {
	for i, name := range header {
		if name == column {
			return i, nil
		}
	}
	return 0, fmt.Errorf("column %q not found", column)
}

// Clamp01 bounds values into the closed unit interval.
func Clamp01(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}
