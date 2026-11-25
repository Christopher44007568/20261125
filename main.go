package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func main() {
	outDir := "out"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}

	closeup := renderCloseup()
	if err := writePNG(filepath.Join(outDir, "closeup.png"), closeup); err != nil {
		fatal(err)
	}

	binStats := simulateFrames(ModeBinary)
	aaStats := simulateFrames(ModeAnalytic)
	thickStats := simulateFrames(ModeThick)

	film := renderFilmstrip(binStats, aaStats)
	if err := writePNG(filepath.Join(outDir, "filmstrip.png"), film); err != nil {
		fatal(err)
	}

	binAmp := flickerAmplitude(binStats)
	aaAmp := flickerAmplitude(aaStats)
	thickAmp := flickerAmplitude(thickStats)
	flicker := renderFlickerMap(binAmp, aaAmp)
	if err := writePNG(filepath.Join(outDir, "flicker.png"), flicker); err != nil {
		fatal(err)
	}

	if err := writeStatsCSV(filepath.Join(outDir, "stats.csv"),
		binStats, aaStats, thickStats, binAmp, aaAmp,
		thickAmp); err != nil {
		fatal(err)
	}

	fmt.Println("wrote out/closeup.png out/filmstrip.png out/flicker.png out/stats.csv")
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func writeStatsCSV(path string, bin, aa, thick *FrameStats,
	binAmp, aaAmp, thickAmp []float64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, "frame,shift_x,shift_y,binary_total,analytic_total,"+
		"thick_total,binary_lit,binary_min_cov,analytic_lit,analytic_min_cov,"+
		"thick_lit,thick_min_cov"); err != nil {
		return err
	}
	for i := 0; i < frames; i++ {
		dx, dy := frameShift(i)
		if _, err := fmt.Fprintf(f, "%d,%.3f,%.3f,%.4f,%.4f,%.4f,%d,%.4f,%d,%.4f,%d,%.4f\n",
			i, dx, dy,
			bin.total[i], aa.total[i], thick.total[i],
			bin.litCount[i], bin.minPixel[i],
			aa.litCount[i], aa.minPixel[i],
			thick.litCount[i], thick.minPixel[i]); err != nil {
			return err
		}
	}

	binMean, binMax := ampStats(binAmp)
	aaMean, aaMax := ampStats(aaAmp)
	thickMean, thickMax := ampStats(thickAmp)
	binToggles := sumInts(toggleCount(bin))
	aaToggles := sumInts(toggleCount(aa))

	if _, err := fmt.Fprintln(f); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, "metric,binary,analytic,thick"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "amplitude_mean,%.4f,%.4f,%.4f\n",
		binMean, aaMean, thickMean); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "amplitude_max,%.4f,%.4f,%.4f\n",
		binMax, aaMax, thickMax); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "frame_toggles,%d,%d,%d\n",
		binToggles, aaToggles, sumInts(toggleCount(thick))); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "total_coverage_cv,%.4f,%.4f,%.4f\n",
		coefficientOfVariation(bin.total),
		coefficientOfVariation(aa.total),
		coefficientOfVariation(thick.total)); err != nil {
		return err
	}
	return nil
}

func ampStats(amp []float64) (mean, max float64) {
	max = amp[0]
	for _, v := range amp {
		mean += v
		if v > max {
			max = v
		}
	}
	return mean / float64(len(amp)), max
}

func sumInts(vals []int) int {
	sum := 0
	for _, v := range vals {
		sum += v
	}
	return sum
}

func coefficientOfVariation(vals []float64) float64 {
	var mean float64
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))
	if mean == 0 {
		return 0
	}
	var sq float64
	for _, v := range vals {
		sq += (v - mean) * (v - mean)
	}
	return math.Sqrt(sq/float64(len(vals))) / mean
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "hairaa:", err)
	os.Exit(1)
}
