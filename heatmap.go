package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func writePNG(path string, f *Frame) {
	img := image.NewRGBA(image.Rect(0, 0, f.W, f.H))
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			c := f.Pix[f.index(x, y)].Clamped()
			img.SetRGBA(x, y, color.RGBA{
				uint8(math.Round(c.R * 255)),
				uint8(math.Round(c.G * 255)),
				uint8(math.Round(c.B * 255)),
				255,
			})
		}
	}
	out, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		panic(err)
	}
}

func heatColor(v float64) color.RGBA {
	if v <= 0 {
		return color.RGBA{10, 12, 16, 255}
	}
	r := uint8(math.Round(math.Min(1, v) * 255))
	return color.RGBA{r, uint8(40 + (1-math.Min(1, v))*60), uint8(60), 255}
}

func writeFlickerHeatmap(path string, an *analyzer) {
	img := image.NewRGBA(image.Rect(0, 0, an.cfg.W, an.cfg.H))
	var maxV float64
	for i := range an.stats {
		if v := an.varianceAt(i); v > maxV {
			maxV = v
		}
	}
	for y := 0; y < an.cfg.H; y++ {
		for x := 0; x < an.cfg.W; x++ {
			i := y*an.cfg.W + x
			v := 0.0
			if maxV > 0 {
				v = an.varianceAt(i) / maxV
			}
			if an.stats[i].flickerEvents > 0 {
				v = math.Max(v, 0.25)
			}
			img.SetRGBA(x, y, heatColor(v))
		}
	}
	out, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		panic(err)
	}
}

func writeTimeseries(path string, cfg Config, an *analyzer) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fmt.Fprintln(f, "x,y,frame,luminance")
	for y := 0; y < cfg.H; y++ {
		for x := 0; x < cfg.W; x++ {
			i := y*cfg.W + x
			if an.stats[i].flickerEvents == 0 {
				continue
			}
			for t, fr := range an.work {
				fmt.Fprintf(f, "%d,%d,%d,%.6f\n", x, y, t, luminance(fr.Pix[i]))
			}
		}
	}
}
