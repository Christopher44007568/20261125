package main

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestFontGeometry(t *testing.T) {
	for r, glyph := range font5x7 {
		if len(glyph) != 7 {
			t.Fatalf("glyph %q has %d rows", r, len(glyph))
		}
		for _, row := range glyph {
			if len(row) != 5 {
				t.Fatalf("glyph %q row %q is not 5 columns", r, row)
			}
		}
	}
	if GlyphFor('a') != GlyphFor('A') {
		t.Fatal("lowercase HUD glyph should fall back to uppercase")
	}
	for _, layer := range DefaultLayers() {
		for _, text := range layer.Texts {
			for _, r := range text.Text {
				if _, ok := font5x7[r]; !ok {
					t.Fatalf("layer %s text %q uses unsupported glyph %q", layer.ID, text.Text, r)
				}
			}
		}
	}
}

func TestOffAxisProjectionPrincipalPoint(t *testing.T) {
	eye := Eye{
		ID: LeftEye, Width: 100, Height: 100,
		FOV:  FOV{Left: 1, Right: 1, Up: 1, Down: 1},
		Pose: Pose{},
	}
	p := eye.Project(Vec3{X: 0, Y: 0, Z: 2})
	if math.Abs(p.Point.X-50) > 1e-9 || math.Abs(p.Point.Y-50) > 1e-9 {
		t.Fatalf("symmetric FOV center = %v, want (50,50)", p.Point)
	}

	eye.FOV.Left = 3
	p = eye.Project(Vec3{X: 0, Y: 0, Z: 2})
	if math.Abs(p.Point.X-75) > 1e-9 {
		t.Fatalf("asymmetric principal x = %.3f, want 75", p.Point.X)
	}
}

func TestNearHasLargestAngularDisparity(t *testing.T) {
	scenario := Scenarios()[0]
	eyes := scenario.BuildEyes(1280, 960, 0.064)
	layers := DefaultLayers()
	means := map[string]float64{}
	for _, layer := range layers {
		var sum float64
		var count float64
		for _, spec := range layer.Texts {
			geo := ProjectText(spec, layer, eyes)
			for _, glyph := range geo.Glyphs {
				sum += math.Abs(glyph.Parallax.AngularArcMin)
				count++
			}
		}
		means[layer.DepthClass] = sum / count
	}
	if !(means["near"] > means["mid"] && means["mid"] > means["system"]) {
		t.Fatalf("angular disparity ordering wrong: %+v", means)
	}
}

func TestRunProducesIndependentEyesAndReadableArtifacts(t *testing.T) {
	out := t.TempDir()
	report, err := Run(RunOptions{OutDir: out, Width: 1280, Height: 960, IPD: 0.064})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Summary.OverallPass {
		t.Fatalf("overall report did not pass: fail=%d warn=%d", report.Summary.FailCount, report.Summary.WarningCount)
	}
	if report.Summary.EyeFrameCount != 30 {
		t.Fatalf("eye frames = %d, want 30", report.Summary.EyeFrameCount)
	}

	required := []string{"report.json", "report.md", "parallax.csv", "boundary.csv", "readability.csv", "sharpness.csv"}
	for _, name := range required {
		info, err := os.Stat(filepath.Join(out, name))
		if err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s missing or empty: %v", name, err)
		}
	}

	for _, scenario := range report.Scenarios {
		if len(scenario.Layers) != 3 {
			t.Fatalf("scenario %s has %d layers", scenario.ID, len(scenario.Layers))
		}
		for _, layer := range scenario.Layers {
			for _, eyeID := range []EyeID{LeftEye, RightEye} {
				relative := layer.FrameFiles[eyeID]
				file, err := os.Open(filepath.Join(out, relative))
				if err != nil {
					t.Fatalf("missing frame %s: %v", relative, err)
				}
				config, err := png.DecodeConfig(file)
				file.Close()
				if err != nil {
					t.Fatalf("invalid PNG %s: %v", relative, err)
				}
				if config.Width != 1280 || config.Height != 960 {
					t.Fatalf("PNG %s size = %dx%d", relative, config.Width, config.Height)
				}
			}

			for _, glyph := range layer.Glyphs {
				if glyph.Parallax.CoordinatesCopied {
					t.Fatalf("copied coordinate for %s/%s", scenario.ID, layer.LayerID)
				}
				if glyph.Parallax.NaiveCopyErrorPx < 0.5 {
					t.Fatalf("copy error too small for %s/%s glyph %q", scenario.ID, layer.LayerID, glyph.Char)
				}
				if glyph.Left.View.Origin == glyph.Right.View.Origin {
					t.Fatalf("identical eye coordinates for %s/%s glyph %q", scenario.ID, layer.LayerID, glyph.Char)
				}
			}

			if layer.DepthClass == "system" {
				for _, element := range layer.Elements {
					for _, eyeID := range []EyeID{LeftEye, RightEye} {
						if sharpness, ok := element.Sharpness[eyeID]; ok {
							if !sharpness.Pass {
								t.Fatalf("system sharpness failed %s %s %s: %v", scenario.ID, element.Spec.ID, eyeID, sharpness.Failures)
							}
						}
					}
				}
			}
		}
	}
}

func TestMonoLostEyeUsesFallbackNotCopy(t *testing.T) {
	var scenario Scenario
	for _, candidate := range Scenarios() {
		if candidate.ID == "mono_tracking_lost" {
			scenario = candidate
		}
	}
	eyes := scenario.BuildEyes(1280, 960, 0.064)
	if eyes[1].Tracking.State != "lost_fallback" {
		t.Fatalf("right state = %q", eyes[1].Tracking.State)
	}
	world := Vec3{X: 0.04, Y: 0.03, Z: 0.8}
	left := eyes[0].Project(world)
	right := eyes[1].Project(world)
	if left.Point == right.Point {
		t.Fatal("lost eye coordinate was copied from tracked eye")
	}
	if right.ProjectionPath != "independent_eye_off_axis_frustum" {
		t.Fatalf("unexpected projection path %q", right.ProjectionPath)
	}
}

func TestBoundaryCullsEmptyRect(t *testing.T) {
	result := CheckBoundary(FloatRect{}, Eye{ID: LeftEye, Width: 100, Height: 100, Tracking: TrackingState{SafeInsetPx: 10}})
	if result.Status != "culled" {
		t.Fatalf("empty bounds status = %q", result.Status)
	}
}

func TestBoundaryClassifiesSafeEdgeAndClipped(t *testing.T) {
	eye := Eye{ID: LeftEye, Width: 100, Height: 100, Tracking: TrackingState{SafeInsetPx: 10}}
	cases := []struct {
		name string
		rect FloatRect
		want string
	}{
		{"visible", FloatRect{MinX: 20, MinY: 20, MaxX: 80, MaxY: 80}, "visible"},
		{"edge risk", FloatRect{MinX: 5, MinY: 20, MaxX: 80, MaxY: 80}, "edge_risk"},
		{"clipped", FloatRect{MinX: -10, MinY: 20, MaxX: 50, MaxY: 80}, "clipped"},
	}
	for _, tc := range cases {
		if got := CheckBoundary(tc.rect, eye).Status; got != tc.want {
			t.Fatalf("%s status = %q, want %q", tc.name, got, tc.want)
		}
	}
}
