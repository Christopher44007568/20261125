package main

import (
	"path/filepath"

	"spectralresample"
)

func swatches(results []spectralresample.PatchResult, pick func(spectralresample.PatchResult) spectralresample.Swatch) []spectralresample.Swatch {
	out := make([]spectralresample.Swatch, 0, len(results))
	for _, result := range results {
		out = append(out, pick(result))
	}
	return out
}

func writePatchCharts(p *spectralresample.Pipeline, outDir string) error {
	baseD65 := swatches(p.Results, func(r spectralresample.PatchResult) spectralresample.Swatch {
		return spectralresample.Swatch{Label: r.Label, Family: r.Family, RGB: r.D65BaseRGB}
	})
	metaD65 := swatches(p.Results, func(r spectralresample.PatchResult) spectralresample.Swatch {
		return spectralresample.Swatch{Label: r.Label, Family: r.Family, RGB: r.D65MetaRGB}
	})
	baseLED := swatches(p.Results, func(r spectralresample.PatchResult) spectralresample.Swatch {
		return spectralresample.Swatch{Label: r.Label, Family: r.Family, RGB: r.LEDBaseRGB}
	})
	metaLED := swatches(p.Results, func(r spectralresample.PatchResult) spectralresample.Swatch {
		return spectralresample.Swatch{Label: r.Label, Family: r.Family, RGB: r.LEDMetaRGB}
	})

	charts := []struct {
		name     string
		title    string
		swatches []spectralresample.Swatch
	}{
		{"patches_d65_base.ppm", "D65 BASE REFLECTANCE", baseD65},
		{"patches_d65_metamer.ppm", "D65 METAMER (MATCHED XYZ)", metaD65},
		{"patches_led_base.ppm", "LED BASE REFLECTANCE", baseLED},
		{"patches_led_metamer.ppm", "LED METAMER", metaLED},
	}
	for _, chart := range charts {
		canvas := spectralresample.RenderPatchChart(chart.title, chart.swatches, 6)
		if err := canvas.WritePPM(filepath.Join(outDir, chart.name)); err != nil {
			return err
		}
	}
	return nil
}

func writeErrorMaps(p *spectralresample.Pipeline, outDir string) error {
	d65 := swatches(p.Results, func(r spectralresample.PatchResult) spectralresample.Swatch {
		return spectralresample.Swatch{Label: r.Label, Family: r.Family, DeltaE: r.DeltaED65}
	})
	led := swatches(p.Results, func(r spectralresample.PatchResult) spectralresample.Swatch {
		return spectralresample.Swatch{Label: r.Label, Family: r.Family, DeltaE: r.DeltaELED}
	})
	gain := swatches(p.Results, func(r spectralresample.PatchResult) spectralresample.Swatch {
		return spectralresample.Swatch{Label: r.Label, Family: r.Family, RGB: r.GainBaseRGB}
	})

	maps := []struct {
		name     string
		title    string
		swatches []spectralresample.Swatch
	}{
		{"errormap_d65.ppm", "METAMERISM ERROR UNDER D65 (DE2000)", d65},
		{"errormap_led.ppm", "METAMERISM ERROR UNDER LED (DE2000)", led},
	}
	for _, heat := range maps {
		canvas := spectralresample.RenderErrorMap(heat.title, heat.swatches, 6)
		if err := canvas.WritePPM(filepath.Join(outDir, heat.name)); err != nil {
			return err
		}
	}

	prediction := spectralresample.RenderPatchChart("NAIVE RGB GAIN LED PREDICTION", gain, 6)
	if err := prediction.WritePPM(filepath.Join(outDir, "patches_rgb_gain_prediction.ppm")); err != nil {
		return err
	}
	gainError := swatches(p.Results, func(r spectralresample.PatchResult) spectralresample.Swatch {
		return spectralresample.Swatch{Label: r.Label, Family: r.Family, DeltaE: r.DeltaEGainBase}
	})
	canvas := spectralresample.RenderErrorMap("RGB GAIN VS SPECTRAL LED ERROR", gainError, 6)
	return canvas.WritePPM(filepath.Join(outDir, "errormap_rgb_gain.ppm"))
}
