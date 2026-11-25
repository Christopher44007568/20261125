package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SceneReport struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	English   string                `json:"english"`
	Range     string                `json:"range"`
	FlatColor string                `json:"flatColor"`
	Variants  map[string]ImageStats `json:"variants"`
}

type LabReport struct {
	Width      int           `json:"width"`
	Height     int           `json:"height"`
	FlatHeight int           `json:"flatHeight"`
	ColorSpace string        `json:"colorSpace"`
	Generated  string        `json:"generated"`
	Scenes     []SceneReport `json:"scenes"`
}

func exportLab(outDir string) (LabReport, error) {
	start := time.Now()
	report := LabReport{
		Width:      imageWidth,
		Height:     imageHeight,
		FlatHeight: flatHeight,
		ColorSpace: "8-bit sRGB display codes",
		Generated:  start.UTC().Format(time.RFC3339),
	}
	if err := os.MkdirAll(filepath.Join(outDir, "images"), 0o755); err != nil {
		return report, err
	}
	if err := writeWebAssets(outDir); err != nil {
		return report, err
	}

	embedded := make(map[string]map[string]string, len(scenes))
	for _, scene := range scenes {
		sr := SceneReport{
			ID:        scene.ID,
			Name:      scene.Name,
			English:   scene.English,
			Range:     scene.Range,
			FlatColor: fmt.Sprintf("rgb(%d %d %d)", int(scene.FlatColor.R), int(scene.FlatColor.G), int(scene.FlatColor.B)),
			Variants:  make(map[string]ImageStats, len(methods)),
		}
		embedded[scene.ID] = make(map[string]string, len(methods))
		for _, method := range methods {
			img, stats := renderScene(scene, method.Mode)
			name := fmt.Sprintf("%s_%s.png", scene.ID, method.File)
			if err := writePNG(filepath.Join(outDir, "images", name), img); err != nil {
				return report, fmt.Errorf("write %s: %w", name, err)
			}
			dataURL, err := encodeDataURL(img)
			if err != nil {
				return report, fmt.Errorf("encode %s: %w", name, err)
			}
			embedded[scene.ID][method.File] = dataURL
			sr.Variants[method.File] = stats
		}
		report.Scenes = append(report.Scenes, sr)
	}
	if err := writeEmbeddedAssets(outDir, embedded); err != nil {
		return report, err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return report, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(outDir, "report.json"), data, 0o644); err != nil {
		return report, err
	}
	return report, nil
}

func encodeDataURL(img image.Image) (string, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func writeEmbeddedAssets(outDir string, assets map[string]map[string]string) error {
	data, err := json.Marshal(assets)
	if err != nil {
		return err
	}
	script := append([]byte("window.DITHER_ASSETS = "), data...)
	script = append(script, ';', '\n')
	return os.WriteFile(filepath.Join(outDir, "assets.js"), script, 0o644)
}

func writeWebAssets(outDir string) error {
	entries, err := fs.ReadDir(webAssets, "web")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := webAssets.ReadFile("web/" + entry.Name())
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(entry.Name(), ".html") {
			mode = 0o755
		}
		if err := os.WriteFile(filepath.Join(outDir, entry.Name()), data, mode); err != nil {
			return err
		}
	}
	return nil
}
