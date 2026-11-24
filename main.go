package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory")
	r0 := flag.Float64("r0", 1, "fillet radius at v=0 (mm)")
	r1 := flag.Float64("r1", 5, "fillet radius at v=1 (mm)")
	length := flag.Float64("length", 30, "edge length (mm)")
	halfAngle := flag.Float64("half-angle", 30, "wedge half angle (degrees)")
	faceWidth := flag.Float64("face-width", 7, "parent face width (mm)")
	meshU := flag.Int("mesh-u", 80, "fillet samples across the arc")
	meshV := flag.Int("mesh-v", 220, "samples along the edge")
	panelW := flag.Int("panel-width", 860, "panel width in pixels")
	panelH := flag.Int("panel-height", 530, "panel height in pixels")
	ss := flag.Int("supersample", 2, "render supersampling factor")
	flag.Parse()

	cfg := DefaultConfig()
	cfg.RadiusStart, cfg.RadiusEnd = *r0, *r1
	cfg.Length, cfg.HalfAngle, cfg.FaceWidth = *length, *halfAngle, *faceWidth
	cfg.SamplesU, cfg.SamplesV = *meshU, *meshV
	cfg.PanelWidth, cfg.PanelHeight, cfg.SuperSample = *panelW, *panelH, *ss
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid configuration:", err)
		os.Exit(2)
	}

	start := time.Now()
	mesh := BuildMesh(cfg)
	report := buildReport(cfg)
	figure := renderFigure(mesh, cfg, report)

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "create output directory:", err)
		os.Exit(1)
	}
	pngPath := filepath.Join(*outDir, "fillet_diagnostics.png")
	jsonPath := filepath.Join(*outDir, "fillet_report.json")
	if err := writePNG(pngPath, figure); err != nil {
		fmt.Fprintln(os.Stderr, "write PNG:", err)
		os.Exit(1)
	}
	if err := writeJSON(jsonPath, report); err != nil {
		fmt.Fprintln(os.Stderr, "write report:", err)
		os.Exit(1)
	}

	maxPos, maxTangent, maxNormal, _ := maxSeamMetrics(report)
	fmt.Printf("radius: %.3f -> %.3f mm, %d x %d mesh\n", report.Radius.StartMM, report.Radius.EndMM, cfg.SamplesU, cfg.SamplesV)
	fmt.Printf("seams: G0 %.3e mm, tangent %.3e deg, normal %.3e deg\n", maxPos, maxTangent, maxNormal)
	fmt.Printf("acceptance: %s (%s)\n", passLabel(report.Acceptance.FilletParentSeamsPass), report.Acceptance.ContinuityClass)
	fmt.Printf("wrote %s and %s in %s\n", pngPath, jsonPath, time.Since(start).Round(time.Millisecond))
	if !report.Acceptance.FilletParentSeamsPass {
		os.Exit(1)
	}
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func writeJSON(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
