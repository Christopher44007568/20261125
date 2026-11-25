package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"spectralresample"
)

func writeIlluminantSVG(p *spectralresample.Pipeline, outDir string) error {
	d65Norm := normalizeSeries(p.D65.Power)
	ledNorm := normalizeSeries(p.LED.Power)
	plot := spectralresample.SVGPlot{
		Width: 920, Height: 520,
		Title:  "ILLUMINANT SPECTRAL POWER DISTRIBUTIONS (NORMALISED PEAK = 1)",
		XLabel: "WAVELENGTH NM",
		YLabel: "RELATIVE POWER",
		XMin:   spectralresample.LambdaMin,
		XMax:   spectralresample.LambdaMax,
		YMin:   0,
		YMax:   1.05,
		XTicks: []float64{380, 430, 480, 530, 580, 630, 680, 730, 780},
		YTicks: []float64{0, 0.25, 0.5, 0.75, 1},
		Series: []spectralresample.SVGSeries{
			{Label: "D65 DAYLIGHT", Color: "#c0392b", X: p.D65.Wavelengths, Y: d65Norm, Width: 2},
			{Label: "NARROWBAND LED", Color: "#2471a3", X: p.LED.Wavelengths, Y: ledNorm, Width: 2.5},
		},
	}
	if err := plot.WriteSVG(filepath.Join(outDir, "illuminants_spd.svg")); err != nil {
		return err
	}
	raster := spectralresample.RenderRasterPlot(spectralresample.RasterPlotConfig{
		Width: 960, Height: 640,
		Title:  plot.Title,
		XLabel: plot.XLabel,
		YLabel: plot.YLabel,
		XMin:   plot.XMin, XMax: plot.XMax, YMin: plot.YMin, YMax: plot.YMax,
		XTicks: plot.XTicks, YTicks: plot.YTicks,
		Series: []spectralresample.RasterSeries{
			{Label: "D65 DAYLIGHT", Color: spectralresample.RGB{R: 192, G: 57, B: 43}, X: p.D65.Wavelengths, Y: d65Norm},
			{Label: "NARROWBAND LED", Color: spectralresample.RGB{R: 36, G: 113, B: 163}, X: p.LED.Wavelengths, Y: ledNorm},
		},
	})
	return raster.WritePPM(filepath.Join(outDir, "illuminants_spd.ppm"))
}

func writeFamilySpectra(p *spectralresample.Pipeline, outDir string) error {
	families := []struct {
		key  string
		file string
	}{
		{spectralresample.FamilyGrey, "spectra_grey.svg"},
		{spectralresample.FamilySkin, "spectra_skin.svg"},
		{spectralresample.FamilyOrange, "spectra_orange.svg"},
	}
	for _, family := range families {
		results := p.ByFamily(family.key)
		if err := writeFamilyPlot(p, results, family.key, filepath.Join(outDir, family.file)); err != nil {
			return err
		}
	}
	return nil
}

func writeFamilyPlot(p *spectralresample.Pipeline, results []spectralresample.PatchResult, family, path string) error {
	var series []spectralresample.SVGSeries
	colors := []string{"#1a5276", "#117864", "#7d3c98", "#b9770e", "#922b21", "#117a65"}
	for i, result := range results {
		color := colors[i%len(colors)]
		baseLabel := upper(family) + " " + itoa2(result.Index+1) + " BASE"
		metaLabel := upper(family) + " " + itoa2(result.Index+1) + " METAMER"
		series = append(series,
			spectralresample.SVGSeries{
				Label: baseLabel,
				Color: color,
				X:     result.Base.Wavelengths,
				Y:     result.Base.Values,
				Width: 1.8,
			},
			spectralresample.SVGSeries{
				Label:  metaLabel,
				Color:  color,
				X:      result.Metamer.Wavelengths,
				Y:      result.Metamer.Values,
				Width:  1.8,
				Dashed: true,
			},
		)
	}
	plot := spectralresample.SVGPlot{
		Width: 1100, Height: 760,
		Title:  fmt.Sprintf("REFLECTANCE SPECTRA - %s TARGETS (SOLID BASE, DASHED METAMER)", upper(family)),
		XLabel: "WAVELENGTH NM",
		YLabel: "REFLECTANCE",
		XMin:   spectralresample.LambdaMin,
		XMax:   spectralresample.LambdaMax,
		YMin:   0,
		YMax:   1,
		XTicks: []float64{380, 430, 480, 530, 580, 630, 680, 730, 780},
		YTicks: []float64{0, 0.25, 0.5, 0.75, 1},
		Series: series,
	}
	if err := plot.WriteSVG(path); err != nil {
		return err
	}
	rasterSeries := make([]spectralresample.RasterSeries, 0, len(plot.Series))
	for _, entry := range plot.Series {
		rasterSeries = append(rasterSeries, spectralresample.RasterSeries{
			Label:  entry.Label,
			Color:  parseHexColor(entry.Color),
			X:      entry.X,
			Y:      entry.Y,
			Dashed: entry.Dashed,
		})
	}
	raster := spectralresample.RenderRasterPlot(spectralresample.RasterPlotConfig{
		Width: 1120, Height: 840,
		Title:  plot.Title,
		XLabel: plot.XLabel,
		YLabel: plot.YLabel,
		XMin:   plot.XMin, XMax: plot.XMax, YMin: plot.YMin, YMax: plot.YMax,
		XTicks: plot.XTicks, YTicks: plot.YTicks,
		Series: rasterSeries,
	})
	return raster.WritePPM(strings.TrimSuffix(path, ".svg") + ".ppm")
}

func parseHexColor(hex string) spectralresample.RGB {
	if len(hex) != 7 || hex[0] != '#' {
		return spectralresample.RGB{}
	}
	r := hexPair(hex[1:3])
	g := hexPair(hex[3:5])
	b := hexPair(hex[5:7])
	return spectralresample.RGB{R: r, G: g, B: b}
}

func hexPair(pair string) uint8 {
	value := 0
	for _, c := range pair {
		value <<= 4
		switch {
		case c >= '0' && c <= '9':
			value += int(c - '0')
		case c >= 'a' && c <= 'f':
			value += int(c-'a') + 10
		}
	}
	return uint8(value)
}

func normalizeSeries(values []float64) []float64 {
	peak := 0.0
	for _, value := range values {
		if value > peak {
			peak = value
		}
	}
	out := make([]float64, len(values))
	for i, value := range values {
		out[i] = value / peak
	}
	return out
}

func upper(text string) string {
	out := make([]byte, len(text))
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func itoa2(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return string(rune('0'+value/10)) + string(rune('0'+value%10))
}
