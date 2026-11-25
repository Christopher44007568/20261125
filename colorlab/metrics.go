package colorlab

import (
	"fmt"
	"math"
	"sort"
)

// ChainMetrics records round-trip accuracy for one ordered space pair.
type ChainMetrics struct {
	From      Space
	To        Space
	PeakError float64
	Peak8Bit  int
	Samples   int
}

// MeasureChain converts the chart from -> to -> from and compares it with the
// chart represented directly in from. Continuous values are compared before
// any display clipping.
func MeasureChain(chart Chart, from, to Space) ChainMetrics {
	result := ChainMetrics{From: from, To: to}
	for _, row := range chart.Rows {
		for _, source := range row.Patches {
			reference := Convert(source, SRGB, from)
			got := Convert(Convert(reference, from, to), to, from)
			for _, delta := range channelDeltas(reference, got) {
				if delta > result.PeakError {
					result.PeakError = delta
				}
			}
			result.Peak8Bit = max(
				result.Peak8Bit,
				quantizedDelta(reference, got),
			)
			result.Samples++
		}
	}
	return result
}

func channelDeltas(a, b RGB) [3]float64 {
	return [3]float64{
		math.Abs(a.R - b.R),
		math.Abs(a.G - b.G),
		math.Abs(a.B - b.B),
	}
}

func quantizedDelta(a, b RGB) int {
	return max(
		absInt(int(quantize8(a.R))-int(quantize8(b.R))),
		absInt(int(quantize8(a.G))-int(quantize8(b.G))),
		absInt(int(quantize8(a.B))-int(quantize8(b.B))),
	)
}

// AllChainMetrics measures both directions of every unordered space pair.
func AllChainMetrics(chart Chart) []ChainMetrics {
	spaces := AllSpaces()
	var out []ChainMetrics
	for i := 0; i < len(spaces); i++ {
		for j := i + 1; j < len(spaces); j++ {
			out = append(out, MeasureChain(chart, spaces[i], spaces[j]))
			out = append(out, MeasureChain(chart, spaces[j], spaces[i]))
		}
	}
	return out
}

// MonotonicCheck is the result for one transfer curve sampled over [0,1],
// including points immediately around every piecewise boundary.
type MonotonicCheck struct {
	Name         string
	Samples      int
	Monotonic    bool
	MaxBacktrack float64
	ZeroOK       bool
	OneOK        bool
}

const monotonicTolerance = 1e-12

// CheckMonotonicity validates each decode and encode curve.
func CheckMonotonicity() []MonotonicCheck {
	type curve struct {
		name string
		fn   func(float64) float64
	}
	curves := []curve{
		{"sRGB decode", sRGBToLinear},
		{"sRGB encode", linearToSRGB},
		{"Rec.2020 decode", rec2020ToLinear},
		{"Rec.2020 encode", linearToRec2020},
		{"PQ decode", pqToLinear},
		{"PQ encode", linearToPQ},
		{"HLG decode", hlgToLinear},
		{"HLG encode", linearToHLG},
	}

	values := boundarySamples()
	out := make([]MonotonicCheck, 0, len(curves))
	for _, c := range curves {
		check := MonotonicCheck{
			Name:      c.name,
			Samples:   len(values),
			Monotonic: true,
			ZeroOK:    math.Abs(c.fn(0)) <= 1e-12,
			OneOK:     math.Abs(c.fn(1)-1) <= 1e-12,
		}
		previous := c.fn(values[0])
		for _, v := range values[1:] {
			current := c.fn(v)
			if math.IsNaN(current) || math.IsInf(current, 0) {
				check.Monotonic = false
				continue
			}
			if drop := previous - current; drop > check.MaxBacktrack {
				check.MaxBacktrack = drop
			}
			// libm pow/log implementations can differ by a few ulps at a
			// piecewise boundary. A material backward step still fails.
			if previous-current > monotonicTolerance {
				check.Monotonic = false
			}
			previous = current
		}
		out = append(out, check)
	}
	return out
}

func boundarySamples() []float64 {
	boundaries := []float64{
		srgbEncodedThreshold,
		srgbLinearThreshold,
		rec2020Beta,
		4.5 * rec2020Beta,
		1.0 / 12.0,
		0.5,
	}
	values := make([]float64, 0, 65536+len(boundaries)*3)
	for i := 0; i <= 65536; i++ {
		values = append(values, float64(i)/65536.0)
	}
	for _, b := range boundaries {
		values = append(values,
			math.Nextafter(b, math.Inf(-1)),
			b,
			math.Nextafter(b, math.Inf(1)),
		)
	}
	sort.Float64s(values)
	unique := values[:0]
	for _, v := range values {
		if v < 0 || v > 1 {
			continue
		}
		if len(unique) == 0 || v != unique[len(unique)-1] {
			unique = append(unique, v)
		}
	}
	return unique
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// SummaryLine formats a chain result for the terminal and report file.
func (m ChainMetrics) SummaryLine() string {
	return fmt.Sprintf("%s -> %s -> %s: peak=%.9g (8-bit=%d), samples=%d",
		m.From, m.To, m.From, m.PeakError, m.Peak8Bit, m.Samples)
}
