package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

// savePNG tone-maps linear radiance (simple Reinhard + gamma) to 8-bit PNG.
func savePNG(img *Image, path string, gain float64) error {
	out := image.NewRGBA(image.Rect(0, 0, img.W, img.H))
	for i, c := range img.Pix {
		to8 := func(v float64) uint8 {
			v = 1 - math.Exp(-v*gain) // filmic-ish exposure
			v = math.Pow(v, 1/2.2)
			if v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			return uint8(v*255 + 0.5)
		}
		out.SetRGBA(i%img.W, i/img.W, color.RGBA{to8(c.X), to8(c.Y), to8(c.Z), 255})
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, out)
}

// curveWriter accumulates named T-curves and writes them as one CSV with
// columns (series, t, T). Series from different strategies have different
// step spacings, so a long format avoids ragged columns.
type curveWriter struct{ rows [][]string }

func (c *curveWriter) add(series string, r MarchResult) {
	for i, t := range r.TTime {
		c.rows = append(c.rows, []string{
			series,
			strconv.FormatFloat(t, 'f', 4, 64),
			strconv.FormatFloat(r.TCurve[i], 'f', 6, 64),
		})
	}
}

func (c *curveWriter) save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"series", "t", "transmittance"}); err != nil {
		return err
	}
	return w.WriteAll(c.rows)
}

func writeReport(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := fmt.Fprintln(f, l); err != nil {
			return err
		}
	}
	return nil
}

func ensureDir(p string) error { return os.MkdirAll(p, 0o755) }

func outPath(dir, name string) string { return filepath.Join(dir, name) }
