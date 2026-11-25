package render

import (
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderComparisons(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		draw func(string) error
		w, h int
	}{
		{"wheel.png", ColorWheelPNG, 1500, 840},
		{"gradients.png", GradientsPNG, 1500, 900},
		{"skin.png", SkinTonesPNG, 1500, 870},
	}
	for _, tc := range cases {
		path := filepath.Join(dir, tc.name)
		if err := tc.draw(path); err != nil {
			t.Fatalf("render %s: %v", tc.name, err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		cfg, format, err := image.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", tc.name, err)
		}
		if format != "png" {
			t.Fatalf("%s format = %s, want png", tc.name, format)
		}
		if cfg.Width != tc.w || cfg.Height != tc.h {
			t.Fatalf("%s dimensions = %dx%d, want %dx%d",
				tc.name, cfg.Width, cfg.Height, tc.w, tc.h)
		}
	}
}
