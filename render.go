package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const (
	cell   = 64
	gap    = 4
	margin = 12
)

func toNRGBA(c Vec3) color.NRGBA {
	return color.NRGBA{
		R: uint8(math.Round(clamp01(c[0]) * 255)),
		G: uint8(math.Round(clamp01(c[1]) * 255)),
		B: uint8(math.Round(clamp01(c[2]) * 255)),
		A: 255,
	}
}

// heatColor maps a dE2000 value to a blue->green->yellow->red ramp,
// saturating at maxDE.
func heatColor(dE, maxDE float64) color.NRGBA {
	stops := []Vec3{
		{0.05, 0.10, 0.45}, // 0: deep blue
		{0.00, 0.65, 0.45}, // 1/3: green
		{0.95, 0.85, 0.10}, // 2/3: yellow
		{0.90, 0.10, 0.05}, // 1: red
	}
	t := clamp01(dE / maxDE)
	pos := t * float64(len(stops)-1)
	i := int(pos)
	if i >= len(stops)-1 {
		i = len(stops) - 2
	}
	f := pos - float64(i)
	mix := Vec3{
		stops[i][0] + f*(stops[i+1][0]-stops[i][0]),
		stops[i][1] + f*(stops[i+1][1]-stops[i][1]),
		stops[i][2] + f*(stops[i+1][2]-stops[i][2]),
	}
	return toNRGBA(mix)
}

func fillRect(img *image.NRGBA, x0, y0, w, h int, c color.NRGBA) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

func newCanvas(cols, rows int) *image.NRGBA {
	w := 2*margin + cols*cell + (cols-1)*gap
	h := 2*margin + rows*cell + (rows-1)*gap
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fillRect(img, 0, 0, w, h, color.NRGBA{24, 24, 28, 255})
	return img
}

func savePNG(path string, img *image.NRGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// RenderMosaic draws one row per color set (row 0 = originals, then one row
// per scenario) with one column per patch.
func RenderMosaic(path string, rows [][]Vec3) error {
	img := newCanvas(len(rows[0]), len(rows))
	for r, row := range rows {
		for c, col := range row {
			x := margin + c*(cell+gap)
			y := margin + r*(cell+gap)
			fillRect(img, x, y, cell, cell, toNRGBA(col))
		}
	}
	return savePNG(path, img)
}

// RenderHeatmap draws one row per scenario, one column per patch, colored by
// dE2000, plus a scale strip at the bottom.
func RenderHeatmap(path string, dEs [][]float64, maxDE float64) error {
	rows := len(dEs) + 1 // extra row for the scale strip
	img := newCanvas(len(dEs[0]), rows)
	for r, row := range dEs {
		for c, dE := range row {
			x := margin + c*(cell+gap)
			y := margin + r*(cell+gap)
			fillRect(img, x, y, cell, cell, heatColor(dE, maxDE))
		}
	}
	y := margin + len(dEs)*(cell+gap)
	for c := range dEs[0] {
		x := margin + c*(cell+gap)
		v := maxDE * float64(c) / float64(len(dEs[0])-1)
		fillRect(img, x, y, cell, cell, heatColor(v, maxDE))
	}
	return savePNG(path, img)
}
