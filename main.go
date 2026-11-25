package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	outDir := "artifacts"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}

	fmt.Println("building shared scene and rasterizing geometry buffer ...")
	sc := buildScene(20261125)
	gbuf := sc.rasterize()
	hair := 0
	scalp := 0
	for _, p := range gbuf.Pixel {
		switch p.Hit {
		case HitHair:
			hair++
		case HitScalp:
			scalp++
		}
	}
	fmt.Printf("fibers=%d gbuffer=%dx%d hairPixels=%d scalpPixels=%d\n",
		len(sc.Fibers), gbuf.W, gbuf.H, hair, scalp)

	fmt.Println("shading single-scatter / Kajiya-Kay / double-scatter ...")
	calibrateModels()
	results := renderAll(sc, gbuf)
	fileNames := map[string]string{
		"single":     "render_single_scatter.png",
		"kajiya_kay": "render_kajiya_kay.png",
		"double":     "render_double_scatter.png",
	}
	for _, res := range results {
		path := filepath.Join(outDir, fileNames[res.Name])
		writePNG(path, res.Image)
		fmt.Println("wrote", path)
	}

	fmt.Println("building highlight position overlay ...")
	overlay := buildOverlay(gbuf, results)
	annotateOverlay(overlay)
	writePNG(filepath.Join(outDir, "highlight_overlay.png"), overlay)

	fmt.Println("running grazing-angle anisotropy diagnostic ...")
	grazing, gstats := buildGrazingComposite()
	writePNG(filepath.Join(outDir, "grazing_diagnostic.png"), grazing)

	report := buildEnergyReport(gbuf, results)
	reportPath := filepath.Join(outDir, "energy_report.txt")
	writeReport(reportPath, report, gstats, len(sc.Fibers), hair)
	fmt.Println("wrote", reportPath)
	fmt.Println("done.")
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fatal(err)
	}
}

func annotateOverlay(img *image.RGBA) {
	entries := []struct {
		text string
		col  color.RGBA
	}{
		{"RED SINGLE", overlayColors["single"]},
		{"GREEN KAJIYA KAY", overlayColors["kajiya_kay"]},
		{"BLUE DOUBLE", overlayColors["double"]},
	}
	for i, e := range entries {
		drawText(img, 12+i*250, 12, 2, e.text, e.col)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
