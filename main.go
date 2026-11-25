package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

type Report struct {
	GeneratedAt    string             `json:"generated_at"`
	OutputDir      string             `json:"output_dir"`
	GrayAnchor     GrayAnchorReport   `json:"gray_anchor"`
	NativeGrayCode map[string]float64 `json:"native_gray_codes"`
	Scenes         []SceneReport      `json:"scenes"`
	AllChecksPass  bool               `json:"all_checks_pass"`
}

type GrayAnchorReport struct {
	HDRNits           float64 `json:"hdr_nits"`
	Reflectance       float64 `json:"reflectance"`
	ProofCode         float64 `json:"proof_srgb_code"`
	ProofLuma         float64 `json:"proof_linear_luma"`
	PositionStatement string  `json:"position_statement"`
}

type SceneReport struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	GrayRect      Rect           `json:"gray_rect_logical_pixels"`
	ComparisonPNG string         `json:"comparison_png"`
	DisplayPNGs   []string       `json:"display_pngs"`
	Metrics       []FrameMetrics `json:"metrics"`
}

func main() {
	outDir := flag.String("out", "artifacts", "directory for PNG and report output")
	flag.Parse()

	start := time.Now()
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	specs := DisplaySpecs()
	scenes := GenerateScenes()
	report := Report{
		GeneratedAt:    time.Now().Format(time.RFC3339),
		OutputDir:      *outDir,
		NativeGrayCode: NativeGrayCodes(),
		AllChecksPass:  true,
		GrayAnchor: GrayAnchorReport{
			HDRNits:     grayNits,
			Reflectance: 0.18,
			ProofCode:   grayProofCode,
			ProofLuma:   grayLinear,
			PositionStatement: fmt.Sprintf(
				"central pixels of each declared gray rectangle map to sRGB proof code %.0f; histogram marker is at bin %.0f",
				grayProofCode, grayProofCode),
		},
	}

	master := image.NewNRGBA(image.Rect(0, 0, sheetW, sheetH*len(scenes)))
	fillRect(master, 0, 0, sheetW, sheetH*len(scenes), colBackground)

	for sceneIndex, scene := range scenes {
		frames := make([]*DisplayFrame, 0, len(specs))
		sceneReport := SceneReport{
			ID:       scene.ID,
			Title:    scene.Title,
			GrayRect: scene.Gray,
		}

		for _, spec := range specs {
			frame := RenderDisplay(scene, spec)
			frames = append(frames, frame)
			metrics := AnalyzeFrame(scene, frame)
			if !metrics.Pass {
				report.AllChecksPass = false
			}
			sceneReport.Metrics = append(sceneReport.Metrics, metrics)

			name := fmt.Sprintf("hdr_%s_%s.png", scene.ID, spec.ID)
			path := filepath.Join(*outDir, name)
			if err := WritePNG(path, frame.Image); err != nil {
				fatal(err)
			}
			sceneReport.DisplayPNGs = append(sceneReport.DisplayPNGs, name)
		}

		sheet := BuildSceneSheet(scene, frames)
		comparisonName := fmt.Sprintf("hdr_%s_comparison.png", scene.ID)
		comparisonPath := filepath.Join(*outDir, comparisonName)
		if err := WritePNG(comparisonPath, sheet); err != nil {
			fatal(err)
		}
		sceneReport.ComparisonPNG = comparisonName

		draw.Draw(master, image.Rect(0, sceneIndex*sheetH, sheetW, (sceneIndex+1)*sheetH),
			sheet, image.Point{}, draw.Src)
		report.Scenes = append(report.Scenes, sceneReport)
	}

	masterPath := filepath.Join(*outDir, "hdr_three_scenes_all_displays.png")
	if err := WritePNG(masterPath, master); err != nil {
		fatal(err)
	}

	if err := writeJSONReport(filepath.Join(*outDir, "hdr_report.json"), report); err != nil {
		fatal(err)
	}
	if err := writeMarkdownReport(filepath.Join(*outDir, "hdr_report.md"), report); err != nil {
		fatal(err)
	}

	fmt.Printf("HDR display simulations complete in %s\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("contact sheets: %s/hdr_{window,lights,shadow}_comparison.png\n", *outDir)
	fmt.Printf("master sheet:   %s\n", masterPath)
	fmt.Printf("checks passed:  %v\n", report.AllChecksPass)
	if !report.AllChecksPass {
		os.Exit(1)
	}
}

func WritePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	return encoder.Encode(file, img)
}

func writeJSONReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
