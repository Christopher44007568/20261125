// Command gamut-viz renders Rec.2020 to sRGB gamut-mapping comparisons.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gamutmap/render"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for generated PNG files")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fail(err)
	}
	outputs := []struct {
		name string
		draw func(string) error
	}{
		{"rec2020-color-wheel.png", render.ColorWheelPNG},
		{"rec2020-gradients.png", render.GradientsPNG},
		{"skin-tone-comparison.png", render.SkinTonesPNG},
	}
	for _, output := range outputs {
		path := filepath.Join(*outDir, output.name)
		if err := output.draw(path); err != nil {
			fail(err)
		}
		fmt.Printf("wrote %s\n", path)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "gamut-viz: %v\n", err)
	os.Exit(1)
}
