package main

import (
	"encoding/json"
	"image"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExposureFusionRecoversLinearRadiance(t *testing.T) {
	radiance := RGB{R: 0.06, G: 0.05, B: 0.04}
	scales := []float64{0.125, 1.0, 6.0}
	bundle := makeTestBundle(8, 4, radiance, scales, scales)
	fused := fuseExposures(bundle)
	if len(fused.ExposureStats) != 3 {
		t.Fatalf("got %d exposure stats, want 3", len(fused.ExposureStats))
	}
	for p, got := range fused.Camera.Pix {
		if math.Abs(got.R-radiance.R) > 0.002 ||
			math.Abs(got.G-radiance.G) > 0.002 ||
			math.Abs(got.B-radiance.B) > 0.002 {
			t.Fatalf("pixel %d fused %+v, want %+v", p, got, radiance)
		}
	}
	if fused.MaxAbsDeviation > 0.06 {
		t.Fatalf("unexpected exposure deviation: %.4f EV", fused.MaxAbsDeviation)
	}
}

func TestExposureDeviationIsMeasured(t *testing.T) {
	radiance := RGB{R: 0.06, G: 0.05, B: 0.04}
	metadata := []float64{0.125, 1.0, 6.0}
	actual := []float64{0.125 * 0.90, 1.0, 6.0 * 1.10}
	bundle := makeTestBundle(8, 4, radiance, metadata, actual)
	fused := fuseExposures(bundle)
	if fused.MaxAbsDeviation < 0.10 {
		t.Fatalf("exposure mismatch not measured: %.4f EV", fused.MaxAbsDeviation)
	}
}

func makeTestBundle(width, height int, radiance RGB, metadata, actual []float64) *CaptureBundle {
	k := Intrinsics{
		FX: float64(width),
		FY: float64(width),
		CX: float64(width) / 2,
		CY: float64(height) / 2,
	}
	bundle := &CaptureBundle{
		Intrinsics: k,
		DepthM:     make([]float64, width*height),
		DepthValid: make([]bool, width*height),
	}
	for i := range bundle.DepthValid {
		bundle.DepthM[i] = 2
		bundle.DepthValid[i] = true
	}
	for e, scale := range metadata {
		frame := image.NewNRGBA(image.Rect(0, 0, width, height))
		for p := 0; p < width*height; p++ {
			offset := p * 4
			frame.Pix[offset+0] = srgbEncode8(radiance.R * actual[e])
			frame.Pix[offset+1] = srgbEncode8(radiance.G * actual[e])
			frame.Pix[offset+2] = srgbEncode8(radiance.B * actual[e])
			frame.Pix[offset+3] = 255
		}
		bundle.Exposures = append(bundle.Exposures, ExposureFrame{
			Scale:       scale,
			ActualScale: actual[e],
			Image:       frame,
		})
	}
	return bundle
}

func TestDepthOcclusionAndInvalidDepthPolicy(t *testing.T) {
	k := intrinsicsFor(320, 180, 58)
	bundle := captureBackground(320, 180, k, rand.New(rand.NewSource(7)))
	fused := fuseExposures(bundle)
	calibration := calibrateColor(fused.Camera, bundle.DepthValid, bundle.WhiteBalance)
	virtual, stats := renderVirtualLayer(bundle, calibration)

	if stats.VisiblePixels == 0 || stats.OccludedPixels == 0 || stats.InvalidDepthPixels == 0 {
		t.Fatalf("expected visible, occluded and invalid-depth pixels, got %+v", stats)
	}

	width, _ := k.pixelDimensions()
	invalidFound := 0
	occludedFound := 0
	for p, footprint := range virtual.Footprint {
		if !footprint {
			continue
		}
		if !bundle.DepthValid[p] {
			if virtual.Alpha[p] != 0 {
				t.Fatalf("invalid depth pixel %d rendered virtual alpha %.3f", p, virtual.Alpha[p])
			}
			invalidFound++
			continue
		}
		if virtual.Alpha[p] == 0 {
			x, y := p%width, p/width
			ray := k.rayForSample(x, y, 0.5, 0.5)
			z, _, ok := intersectSphere(ray, bundle.Sphere)
			if ok && z >= bundle.DepthM[p]-depthOcclusionBiasM {
				occludedFound++
			}
		}
	}
	if invalidFound == 0 {
		t.Fatal("no invalid-depth footprint pixels found")
	}
	if occludedFound == 0 {
		t.Fatal("no virtual pixels were depth-occluded by real geometry")
	}
}

func TestVirtualColorUsesCalibrationBeforeBlend(t *testing.T) {
	k := intrinsicsFor(192, 108, 58)
	bundle := captureBackground(192, 108, k, rand.New(rand.NewSource(11)))
	neutral, _ := renderVirtualLayer(bundle, ColorCalibration{Gains: RGB{R: 1, G: 1, B: 1}})
	warm, _ := renderVirtualLayer(bundle, ColorCalibration{Gains: RGB{R: 1.25, G: 1, B: 0.8}})

	for p, a := range neutral.Alpha {
		if a <= 0 || neutral.Pix[p].Luma() < 0.001 {
			continue
		}
		if math.Abs(warm.Pix[p].R/neutral.Pix[p].R-1.25) > 1e-9 ||
			math.Abs(warm.Pix[p].B/neutral.Pix[p].B-0.80) > 1e-9 {
			t.Fatalf("pixel %d was not color-aligned before blending", p)
		}
		return
	}
	t.Fatal("no visible virtual pixel found")
}

func TestLatencyStatistics(t *testing.T) {
	report := latencyReport([]time.Duration{
		5 * time.Millisecond,
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}, 16*time.Millisecond)
	if report.Samples != 4 || report.P95MS != 30 || report.WithinBudget {
		t.Fatalf("unexpected latency report: %+v", report)
	}
	if report.Measurement != "frame_available_to_composite" {
		t.Fatalf("unexpected latency definition: %s", report.Measurement)
	}
}

func TestRunWritesAllOutputsAndMetrics(t *testing.T) {
	outDir := t.TempDir()
	report, err := run(Config{
		Width:         192,
		Height:        108,
		Frames:        2,
		Seed:          23,
		OutDir:        outDir,
		LatencyBudget: defaultFrameBudget,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"passthrough.png", "virtual.png", "composite.png", "confidence.png", "depth.png", "metrics.json"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	for _, name := range []string{"passthrough.png", "virtual.png", "composite.png", "confidence.png"} {
		file, err := os.Open(filepath.Join(outDir, name))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		if img.Bounds().Dx() != 192 || img.Bounds().Dy() != 108 {
			t.Fatalf("%s has size %v", name, img.Bounds())
		}
	}
	if report.Depth.OccludedPixels == 0 || report.Depth.InvalidDepthInSpherePixels == 0 {
		t.Fatalf("depth report did not exercise occlusion: %+v", report.Depth)
	}
	if report.Exposure.FusionInputs != 3 || report.SystemLayer.Coverage <= 0 {
		t.Fatalf("incomplete metrics: %+v", report)
	}
	data, err := os.ReadFile(filepath.Join(outDir, "metrics.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("metrics JSON: %v", err)
	}
	if decoded.Color.BlendSpace != "linear-srgb" || !decoded.Color.AppliedBeforeMix {
		t.Fatalf("color alignment metadata missing: %+v", decoded.Color)
	}
}

func TestSystemLayerIsSemiTransparent(t *testing.T) {
	layer := makeSystemLayer(200, 100)
	maxAlpha := uint8(0)
	covered := 0
	for p := 3; p < len(layer.Pix); p += 4 {
		if layer.Pix[p] > maxAlpha {
			maxAlpha = layer.Pix[p]
		}
	}
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			if layer.NRGBAAt(x, y).A > 0 {
				covered++
			}
		}
	}
	if maxAlpha == 0 || maxAlpha == 255 || covered == 0 {
		t.Fatalf("system layer is not translucent: max alpha %d, covered %d", maxAlpha, covered)
	}
}
