package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"spectralresample"
)

func writeCSVReport(p *spectralresample.Pipeline, outDir string) error {
	f, err := os.Create(filepath.Join(outDir, "report.csv"))
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, strings.Join([]string{
		"index", "label", "family",
		"d65_base_x", "d65_base_y", "d65_base_z",
		"d65_meta_x", "d65_meta_y", "d65_meta_z",
		"led_base_x", "led_base_y", "led_base_z",
		"led_meta_x", "led_meta_y", "led_meta_z",
		"de2000_d65", "de2000_led", "rgb_gain_distance",
		"rgb_gain_error_base", "rgb_gain_error_metamer",
		"spectral_rms_residual",
	}, ","))
	for _, r := range p.Results {
		fields := []string{
			fmt.Sprintf("%d", r.Index+1),
			r.Label,
			r.Family,
			formatXYZ(r.D65BaseXYZ),
			formatXYZ(r.D65MetaXYZ),
			formatXYZ(r.LEDBaseXYZ),
			formatXYZ(r.LEDMetaXYZ),
			fmt.Sprintf("%.4f", r.DeltaED65),
			fmt.Sprintf("%.4f", r.DeltaELED),
			fmt.Sprintf("%.4f", r.GainPairSplit),
			fmt.Sprintf("%.4f", r.DeltaEGainBase),
			fmt.Sprintf("%.4f", r.DeltaEGainMeta),
			fmt.Sprintf("%.5f", rmsResidual(r.Base.Values, r.Metamer.Values)),
		}
		fmt.Fprintln(w, strings.Join(fields, ","))
	}
	return w.Flush()
}

func formatXYZ(xyz [3]float64) string {
	return fmt.Sprintf("%.4f,%.4f,%.4f", xyz[0], xyz[1], xyz[2])
}

func rmsResidual(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(a)))
}

type familyStats struct {
	count        int
	meanD65      float64
	maxD65       float64
	meanLED      float64
	maxLED       float64
	meanGain     float64
	meanGainErr  float64
	meanSpectral float64
}

func computeStats(results []spectralresample.PatchResult) familyStats {
	var stats familyStats
	stats.count = len(results)
	for _, r := range results {
		stats.meanD65 += r.DeltaED65
		stats.maxD65 = math.Max(stats.maxD65, r.DeltaED65)
		stats.meanLED += r.DeltaELED
		stats.maxLED = math.Max(stats.maxLED, r.DeltaELED)
		stats.meanGain += r.GainPairSplit
		stats.meanGainErr += r.DeltaEGainBase
		stats.meanSpectral += rmsResidual(r.Base.Values, r.Metamer.Values)
	}
	if stats.count > 0 {
		stats.meanD65 /= float64(stats.count)
		stats.meanLED /= float64(stats.count)
		stats.meanGain /= float64(stats.count)
		stats.meanGainErr /= float64(stats.count)
		stats.meanSpectral /= float64(stats.count)
	}
	return stats
}

func writeMarkdownReport(p *spectralresample.Pipeline, outDir string) error {
	f, err := os.Create(filepath.Join(outDir, "report.md"))
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	fmt.Fprintln(w, "# Spectral Resampling Metamerism Report")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "- Wavelength grid: %.0f-%.0f nm at %.0f nm steps (%d bands)\n",
		spectralresample.LambdaMin, spectralresample.LambdaMax, spectralresample.BandStep, spectralresample.BandCount)
	fmt.Fprintf(w, "- Observer: CIE 1931 2 degree standard observer (trapezoidal spectral integration)\n")
	fmt.Fprintf(w, "- Reference illuminant: CIE D65 tabulated SPD\n")
	fmt.Fprintf(w, "- Test illuminant: %s\n", p.LED.Name)
	fmt.Fprintf(w, "- D65 white point XYZ: %.4f, %.4f, %.4f\n", p.D65White[0], p.D65White[1], p.D65White[2])
	fmt.Fprintf(w, "- LED white point XYZ: %.4f, %.4f, %.4f\n", p.LEDWhite[0], p.LEDWhite[1], p.LEDWhite[2])
	fmt.Fprintln(w)

	fmt.Fprintln(w, "## Family-level spectral mismatch")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| family | patches | mean dE2000 D65 | max dE2000 D65 | mean dE2000 LED | max dE2000 LED | gain pair split | gain vs spectral | mean spectral RMS |")
	fmt.Fprintln(w, "|---|---:|---:|---:|---:|---:|---:|---:|---:|")
	for _, family := range []string{spectralresample.FamilyGrey, spectralresample.FamilySkin, spectralresample.FamilyOrange, spectralresample.FamilyOther} {
		stats := computeStats(p.ByFamily(family))
		fmt.Fprintf(w, "| %s | %d | %.4f | %.4f | %.4f | %.4f | %.4f | %.4f | %.5f |\n",
			strings.ToUpper(family), stats.count,
			stats.meanD65, stats.maxD65, stats.meanLED, stats.maxLED,
			stats.meanGain, stats.meanGainErr, stats.meanSpectral)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "## Per-patch detail")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| # | label | family | dE2000 D65 | dE2000 LED | gain pair split | gain vs spectral | spectral RMS |")
	fmt.Fprintln(w, "|---:|---|---|---:|---:|---:|---:|---:|")
	for _, r := range p.Results {
		fmt.Fprintf(w, "| %d | %s | %s | %.4f | %.4f | %.4f | %.4f | %.5f |\n",
			r.Index+1, r.Label, r.Family, r.DeltaED65, r.DeltaELED,
			r.GainPairSplit, r.DeltaEGainBase, rmsResidual(r.Base.Values, r.Metamer.Values))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Interpretation")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The D65 column stays near zero by construction: each metamer pair shares identical")
	fmt.Fprintln(w, "tristimulus values under daylight. The LED column exposes the spectral mismatch that")
	fmt.Fprintln(w, "a single RGB colour-temperature multiplier cannot represent, with separate rows for")
	fmt.Fprintln(w, "grey, skin and orange targets.")
	return w.Flush()
}
