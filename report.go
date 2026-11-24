package main

import (
	"encoding/csv"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"
)

type PixelSummary struct {
	X          int        `json:"x"`
	Y          int        `json:"y"`
	Eye        int        `json:"eye"`
	Input      [5]float64 `json:"input_r_g_b_a_space"`
	Decoded    [4]float64 `json:"decoded_premul_r_g_b_a"`
	Composited [4]float64 `json:"composited_premul_r_g_b_a"`
}

type LayerReport struct {
	Name          string       `json:"name"`
	Eye           int          `json:"eye"`
	Space         string       `json:"space"`
	Premultiplied bool         `json:"premultiplied"`
	Probe         PixelSummary `json:"probe"`
	MaxAlpha      float64      `json:"max_alpha"`
	MeanAlpha     float64      `json:"mean_alpha"`
}

type ErrorStats struct {
	Eye              int     `json:"eye"`
	Mode             string  `json:"mode"`
	MeanLinearRMSE   float64 `json:"mean_linear_rmse"`
	MaxLinearRMSE    float64 `json:"max_linear_rmse"`
	MeanGammaDelta   float64 `json:"mean_gamma_delta_valid_pixels"`
	ValidGammaPixels int     `json:"valid_gamma_pixels"`
}

type Report struct {
	Width       int           `json:"width"`
	Height      int           `json:"height"`
	Background  string        `json:"background"`
	Layers      []LayerReport `json:"layers"`
	ErrorStats  []ErrorStats  `json:"gamma_error_stats"`
	OutputFiles []string      `json:"output_files"`
}

func f64(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Round(v*1e9) / 1e9
}

func writeFloat(w *csv.Writer, values ...float64) error {
	record := make([]string, len(values))
	for i, value := range values {
		record[i] = strconv.FormatFloat(f64(value), 'f', -1, 64)
	}
	return w.Write(record)
}

func writePremulCSV(path string, frame [][]PremulLinear) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	if err := w.Write([]string{"x", "y", "premul_r", "premul_g", "premul_b", "alpha"}); err != nil {
		return err
	}
	for y, row := range frame {
		for x, p := range row {
			if err := writeFloat(w, float64(x), float64(y), p.R, p.G, p.B, p.A); err != nil {
				return err
			}
		}
	}
	w.Flush()
	return w.Error()
}

func writeLayerCSV(path string, input [][]Sample, decoded [][]PremulLinear, composited [][]PremulLinear) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	if err := w.Write([]string{
		"x", "y", "input_r", "input_g", "input_b", "input_a",
		"decoded_premul_r", "decoded_premul_g", "decoded_premul_b", "decoded_a",
		"total_premul_r", "total_premul_g", "total_premul_b", "total_a",
	}); err != nil {
		return err
	}
	for y := range input {
		for x, sample := range input[y] {
			d := decoded[y][x]
			c := composited[y][x]
			if err := writeFloat(w,
				float64(x), float64(y),
				sample.R, sample.G, sample.B, sample.A,
				d.R, d.G, d.B, d.A,
				c.R, c.G, c.B, c.A,
			); err != nil {
				return err
			}
		}
	}
	w.Flush()
	return w.Error()
}

func frameImage(frame [][]PremulLinear, checker bool) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, len(frame[0]), len(frame)))
	for y, row := range frame {
		for x, p := range row {
			r, g, b, alpha := p.Straight()
			r, g, b = toneMap(r, g, b)
			r = linearTosRGB(r)
			g = linearTosRGB(g)
			b = linearTosRGB(b)

			var bg [3]float64
			if checker && (x/4+y/4)%2 == 0 {
				bg = [3]float64{0.18, 0.18, 0.18}
			} else if checker {
				bg = [3]float64{0.78, 0.78, 0.78}
			}

			ia := 1 - alpha
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(math.Round(255 * clamp01(r*alpha+bg[0]*ia))),
				G: uint8(math.Round(255 * clamp01(g*alpha+bg[1]*ia))),
				B: uint8(math.Round(255 * clamp01(b*alpha+bg[2]*ia))),
				A: 255,
			})
		}
	}
	return img
}

func encodedFrameImage(frame [][]PremulLinear) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, len(frame[0]), len(frame)))
	for y, row := range frame {
		for x, p := range row {
			r, g, b, alpha := p.Straight()
			r = clamp01(r)
			g = clamp01(g)
			b = clamp01(b)
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(math.Round(255 * r)),
				G: uint8(math.Round(255 * g)),
				B: uint8(math.Round(255 * b)),
				A: uint8(math.Round(255 * alpha)),
			})
		}
	}
	return img
}

func heatColor(e GammaError) color.NRGBA {
	if !e.Valid {
		return color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	}
	magnitude := clamp01((math.Abs(e.R) + math.Abs(e.G) + math.Abs(e.B)) / 3)
	if (e.R+e.G+e.B)/3 > 0 {
		return color.NRGBA{
			R: uint8(math.Round(255 * magnitude)),
			G: uint8(math.Round(42 + 60*(1-magnitude))),
			B: uint8(math.Round(60 + 40*(1-magnitude))),
			A: 255,
		}
	}
	return color.NRGBA{
		R: uint8(math.Round(40 + 40*(1-magnitude))),
		G: uint8(math.Round(70 + 50*(1-magnitude))),
		B: uint8(math.Round(255 * magnitude)),
		A: 255,
	}
}

func heatImage(errors [][]GammaError) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, len(errors[0]), len(errors)))
	for y, row := range errors {
		for x, e := range row {
			img.SetNRGBA(x, y, heatColor(e))
		}
	}
	return img
}

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
