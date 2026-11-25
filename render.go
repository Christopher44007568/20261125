package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type ImageStats struct {
	MAE       float64 `json:"mae"`
	RMSE      float64 `json:"rmse"`
	MaxError  float64 `json:"maxError"`
	FlatError float64 `json:"flatMaxError"`
	UniqueRGB int     `json:"uniqueRGB"`
}

func renderScene(s Scene, mode Mode) (*image.NRGBA, ImageStats) {
	raw := renderRaw(s, mode)

	img := image.NewNRGBA(image.Rect(0, 0, imageWidth, imageHeight))
	stats := ImageStats{}
	var squared, absolute float64
	unique := make(map[uint32]struct{}, 4096)
	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			i := (y*imageWidth + x) * 3
			r, g, b := raw[i], raw[i+1], raw[i+2]
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
			unique[uint32(r)<<16|uint32(g)<<8|uint32(b)] = struct{}{}

			src := s.At(x, y)
			for _, channel := range [][2]float64{
				{float64(r), src.R}, {float64(g), src.G}, {float64(b), src.B},
			} {
				delta := math.Abs(channel[0] - channel[1])
				absolute += delta
				squared += delta * delta
				if delta > stats.MaxError {
					stats.MaxError = delta
				}
				if y < flatHeight && delta > stats.FlatError {
					stats.FlatError = delta
				}
			}
		}
	}
	channels := float64(imageWidth * imageHeight * 3)
	stats.MAE = absolute / channels
	stats.RMSE = math.Sqrt(squared / channels)
	stats.UniqueRGB = len(unique)
	return img, stats
}

func renderRaw(s Scene, mode Mode) []uint8 {
	switch mode {
	case ModeOrdered:
		return renderOrdered(s)
	case ModeDiffusion:
		return renderDiffusion(s)
	default:
		return renderNoDither(s)
	}
}

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
