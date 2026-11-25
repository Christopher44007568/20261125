package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBayerMatrixIsPermutation(t *testing.T) {
	matrix := bayerMatrix(8)
	if len(matrix) != 8 || len(matrix[0]) != 8 {
		t.Fatalf("matrix size = %dx%d, want 8x8", len(matrix), len(matrix[0]))
	}
	seen := make(map[int]bool, 64)
	for _, row := range matrix {
		for _, value := range row {
			if value < 0 || value >= 64 || seen[value] {
				t.Fatalf("invalid or duplicate Bayer value %d", value)
			}
			seen[value] = true
		}
	}
}

func TestOrderedChannelPreservesIntegersAndFractions(t *testing.T) {
	for _, test := range []struct {
		value float64
		mean  float64
	}{
		{0, 0},
		{0.25, 0.25},
		{17, 17},
		{17.25, 17.25},
		{17.5, 17.5},
		{17.75, 17.75},
	} {
		sum := 0.0
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				sum += orderedChannel(test.value, x, y)
			}
		}
		if math.Abs(sum/64-test.mean) > 1e-9 {
			t.Errorf("ordered mean for %.2f = %.6f, want %.6f", test.value, sum/64, test.mean)
		}
	}
}

func TestFlatReferenceIsUntouched(t *testing.T) {
	for _, scene := range scenes {
		for _, method := range methods {
			img, stats := renderScene(scene, method.Mode)
			want := color.NRGBA{
				R: uint8(scene.FlatColor.R),
				G: uint8(scene.FlatColor.G),
				B: uint8(scene.FlatColor.B),
				A: 255,
			}
			for y := 0; y < flatHeight; y++ {
				for x := 0; x < imageWidth; x++ {
					if got := img.NRGBAAt(x, y); got != want {
						t.Fatalf("%s/%s flat pixel (%d,%d) = %v, want %v",
							scene.ID, method.Mode, x, y, got, want)
					}
				}
			}
			if stats.FlatError != 0 {
				t.Errorf("%s/%s flat error = %v, want 0", scene.ID, method.Mode, stats.FlatError)
			}
		}
	}
}

func TestNoDitherRoundsEachChannel(t *testing.T) {
	scene := sceneByID("skin")
	img, _ := renderScene(scene, ModeNone)
	for y := flatHeight; y < imageHeight; y += 17 {
		for x := 0; x < imageWidth; x += 19 {
			src := scene.At(x, y)
			got := img.NRGBAAt(x, y)
			want := [3]uint8{
				uint8(quantize(src.R)),
				uint8(quantize(src.G)),
				uint8(quantize(src.B)),
			}
			if [3]uint8{got.R, got.G, got.B} != want {
				t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestDitheringBreaksHardBands(t *testing.T) {
	scene := sceneByID("gray")
	count := func(mode Mode) int {
		raw := renderRaw(scene, mode)
		changes := 0
		y := 120
		for x := 1; x < imageWidth; x++ {
			a := raw[(y*imageWidth+x)*3]
			b := raw[(y*imageWidth+x-1)*3]
			if a != b {
				changes++
			}
		}
		return changes
	}
	none := count(ModeNone)
	ordered := count(ModeOrdered)
	diffusion := count(ModeDiffusion)
	if none > 50 {
		t.Fatalf("no-dither row has %d transitions, want isolated bands", none)
	}
	if ordered < none*5 || diffusion < none*5 {
		t.Fatalf("dither transitions: none=%d ordered=%d diffusion=%d", none, ordered, diffusion)
	}
}

func TestDitherVariantsAreDistinct(t *testing.T) {
	scene := sceneByID("chroma")
	ordered := renderRaw(scene, ModeOrdered)
	diffusion := renderRaw(scene, ModeDiffusion)
	different := 0
	for i := range ordered {
		if ordered[i] != diffusion[i] {
			different++
		}
	}
	if different < len(ordered)/10 {
		t.Fatalf("only %d of %d channel values differ", different, len(ordered))
	}
}

func TestExportProducesNinePNGsAndPage(t *testing.T) {
	out := t.TempDir()
	report, err := exportLab(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Scenes) != 3 {
		t.Fatalf("scene count = %d, want 3", len(report.Scenes))
	}
	for _, scene := range report.Scenes {
		for _, method := range methods {
			path := filepath.Join(out, "images", scene.ID+"_"+method.File+".png")
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(file)
			file.Close()
			if err != nil {
				t.Fatalf("decode %s: %v", path, err)
			}
			if img.Bounds() != image.Rect(0, 0, imageWidth, imageHeight) {
				t.Fatalf("%s bounds = %v", path, img.Bounds())
			}
			if stats := scene.Variants[method.File]; stats.FlatError != 0 {
				t.Fatalf("%s flat error = %v", path, stats.FlatError)
			}
		}
	}
	for _, name := range []string{"index.html", "style.css", "app.js", "assets.js", "report.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(out, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded LabReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded.FlatHeight != flatHeight || len(decoded.Scenes) != 3 {
		t.Fatalf("unexpected report: %+v", decoded)
	}
}

func TestPageExposesAllComparisons(t *testing.T) {
	data, err := webAssets.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	for _, needle := range []string{
		`data-scene="gray"`, `data-scene="skin"`, `data-scene="chroma"`,
		`data-mode="none"`, `data-mode="ordered"`, `data-mode="diffusion"`,
		`BAYER 8×8`, `FLOYD–STEINBERG`, `id="canvas-none"`, `id="canvas-ordered"`,
		`id="canvas-diffusion"`, `data-region="flat"`, `data-view="enhanced"`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("index.html missing %q", needle)
		}
	}
}
