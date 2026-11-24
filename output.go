package main

import (
	"encoding/json"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	encodeErr := encoder.Encode(file, img)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func depthPreview(bundle *CaptureBundle) *image.Gray {
	width, height := bundle.Intrinsics.pixelDimensions()
	out := image.NewGray(image.Rect(0, 0, width, height))
	for p, depth := range bundle.DepthM {
		if !bundle.DepthValid[p] {
			out.Pix[p] = 0
			continue
		}
		near := 4.2 - depth
		near /= 3.0
		out.Pix[p] = uint8(math.Round(clamp01(near) * 255))
	}
	return out
}

func writeReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func outputPaths(outDir string) map[string]string {
	return map[string]string{
		"passthrough": filepath.Join(outDir, "passthrough.png"),
		"virtual":     filepath.Join(outDir, "virtual.png"),
		"composite":   filepath.Join(outDir, "composite.png"),
		"confidence":  filepath.Join(outDir, "confidence.png"),
		"depth":       filepath.Join(outDir, "depth.png"),
		"metrics":     filepath.Join(outDir, "metrics.json"),
	}
}
