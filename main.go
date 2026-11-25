package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for the three PNG passes")
	width := flag.Int("width", 720, "output image width in pixels")
	height := flag.Int("height", 900, "output image height in pixels")
	lightSize := flag.Int("light-size", 512, "square deep-opacity map resolution")
	seed := flag.Int64("seed", 20261125, "procedural guide-strand seed")
	deepMapPath := flag.String("deep-map", "", "optional path for a light-space deep-opacity diagnostic PNG")
	flag.Parse()

	if err := run(*outDir, *width, *height, *lightSize, *seed, *deepMapPath); err != nil {
		log.Fatal(err)
	}
}

func run(outDir string, width, height, lightSize int, seed int64, deepMapPath string) error {
	if width < 32 || height < 32 || lightSize < 16 {
		return fmt.Errorf("dimensions are too small")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	started := time.Now()
	scene := NewProceduralScene(width, height, seed)
	deep := BuildDeepOpacityMap(scene, lightSize)
	renderer := NewRenderer(scene, deep, width, height)
	result := renderer.Render()

	beautyPath := filepath.Join(outDir, "beauty.png")
	shadowPath := filepath.Join(outDir, "shadow_only.png")
	transmissionPath := filepath.Join(outDir, "hair_transmission.png")

	if err := WriteBeautyPNG(beautyPath, result.Beauty, width, height); err != nil {
		return err
	}
	if err := WriteLinearVecPNG(shadowPath, result.Shadow, width, height, 1.0); err != nil {
		return err
	}
	if err := WriteLinearVecPNG(transmissionPath, result.Transmission, width, height, 1.08); err != nil {
		return err
	}
	if deepMapPath != "" {
		if err := WriteDeepTransmissionPNG(deepMapPath, deep); err != nil {
			return err
		}
	}

	fmt.Printf("rendered %d guide strands (%d segments), deep map %dx%d in %s\n",
		len(scene.Strands), len(scene.Segments), lightSize, lightSize, time.Since(started).Round(time.Millisecond))
	fmt.Printf("ordinary lighting: %s\n", beautyPath)
	fmt.Printf("shadow only:       %s\n", shadowPath)
	fmt.Printf("hair transmission: %s\n", transmissionPath)
	return nil
}
