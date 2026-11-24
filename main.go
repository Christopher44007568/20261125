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
	cfg := DefaultConfig()
	outDir := flag.String("out", cfg.OutputDir, "output directory")
	segA := flag.Int("seg-a", cfg.SegAAlong, "coarse forward patch seam segments")
	segB := flag.Int("seg-b", cfg.SegBAlong, "fine reversed patch seam segments")
	bow := flag.Float64("bow", cfg.BowMM, "shared edge curvature/bow in mm")
	ss := flag.Int("supersample", 2, "PNG supersampling factor")
	crackSamples := flag.Int("crack-samples", 301, "dense crack measurement samples")
	flag.Parse()

	cfg.OutputDir = *outDir
	cfg.SegAAlong = *segA
	cfg.SegBAlong = *segB
	cfg.BowMM = *bow
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid config:", err)
		os.Exit(2)
	}
	if *ss < 1 || *ss > 4 {
		fmt.Fprintln(os.Stderr, "supersample must be 1..4")
		os.Exit(2)
	}

	started := time.Now()
	asm := BuildDemoPatches(cfg)
	cracks := MeasureCracks(asm, *crackSamples)

	st, stitchErr := StitchPatches(cfg, asm)
	afterMax, afterMean := MeasureRepairedCracks(st, *crackSamples)
	cracks.AfterMaxGapMM = afterMax
	cracks.AfterMeanMM = afterMean

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "create output directory:", err)
		os.Exit(1)
	}

	outputs := map[string]string{
		"normal_view":   filepath.Join(cfg.OutputDir, "normal_view.png"),
		"zoom_10x":      filepath.Join(cfg.OutputDir, "zoom_10x.png"),
		"crack_heatmap": filepath.Join(cfg.OutputDir, "crack_heatmap.png"),
		"report_json":   filepath.Join(cfg.OutputDir, "stitch_report.json"),
		"stitched_obj":  filepath.Join(cfg.OutputDir, "stitched_mesh.obj"),
	}

	images := map[string]*image.RGBA{
		"normal_view":   RenderNormalView(cfg, cracks, st),
		"zoom_10x":      RenderZoomView(cfg, asm, cracks, st),
		"crack_heatmap": RenderHeatmapView(cfg, asm, cracks, st),
	}
	for key, img := range images {
		if err := writePNG(outputs[key], img); err != nil {
			fmt.Fprintln(os.Stderr, "write image:", err)
			os.Exit(1)
		}
	}
	if err := WriteOBJ(outputs["stitched_obj"], st.Mesh); err != nil {
		fmt.Fprintln(os.Stderr, "write OBJ:", err)
		os.Exit(1)
	}

	report := BuildReport(cfg, asm, cracks, st, outputs)
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal report:", err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if err := os.WriteFile(outputs["report_json"], data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write report:", err)
		os.Exit(1)
	}

	fmt.Printf("before max crack: %.4f mm at q=%.3f\n", cracks.MaxGapMM, cracks.MaxAtQ)
	fmt.Printf("unified samples: %d, added triangles: A=%d B=%d total=%d\n",
		len(st.Samples), st.RefineA.NetAdded, st.RefineB.NetAdded,
		st.RefineA.NetAdded+st.RefineB.NetAdded)
	fmt.Printf("seam watertight: %v; sharp normal jump: %.1f deg; outer boundary edges: %d\n",
		st.Watertight.SeamWatertight, st.SharpNormalAngle, st.Watertight.OuterBoundaryEdges)
	fmt.Printf("wrote %s, %s, %s and report in %s\n",
		outputs["normal_view"], outputs["zoom_10x"], outputs["crack_heatmap"], time.Since(started).Round(time.Millisecond))

	if stitchErr != nil {
		fmt.Fprintln(os.Stderr, "stitch verification failed:", stitchErr)
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
