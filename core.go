package main

import "math"

type Point struct{ X, Y float64 }

type Color struct{ R, G, B float64 }

func (c Color) Add(o Color) Color { return Color{c.R + o.R, c.G + o.G, c.B + o.B} }
func (c Color) Scale(s float64) Color {
	return Color{c.R * s, c.G * s, c.B * s}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (c Color) Clamped() Color {
	return Color{clamp01(c.R), clamp01(c.G), clamp01(c.B)}
}

func luminance(c Color) float64 {
	return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
}

type Frame struct {
	W, H int
	Pix  []Color
}

func newFrame(w, h int) *Frame {
	return &Frame{W: w, H: h, Pix: make([]Color, w*h)}
}

func (f *Frame) index(x, y int) int { return y*f.W + x }

type Config struct {
	W, H      int
	Frames    int
	GTSpp     int
	FoveaFrac float64
	MidFrac   float64
	EyeShift  float64
	Tau       float64
	Seed      uint64
	OutDir    string
}

func defaultConfig() Config {
	return Config{
		W: 120, H: 90, Frames: 24, GTSpp: 4,
		FoveaFrac: 0.18, MidFrac: 0.42,
		EyeShift: 1.5, Tau: 0.10,
		Seed: 20261125, OutDir: "artifacts/results",
	}
}

func testConfig() Config {
	c := defaultConfig()
	c.W, c.H, c.Frames = 48, 36, 8
	c.OutDir = ""
	return c
}

func (c Config) foveaRadius() float64 { return c.FoveaFrac * float64(c.W) }
func (c Config) midRadius() float64   { return c.MidFrac * float64(c.W) }

func (c Config) blockSize(x, y int) int {
	cx := float64(c.W-1) / 2
	cy := float64(c.H-1) / 2
	r := math.Hypot(float64(x)-cx, float64(y)-cy)
	switch {
	case r < c.foveaRadius():
		return 1
	case r < c.midRadius():
		return 2
	default:
		return 4
	}
}

type Motion struct {
	Name  string
	Shift func(frame int) (float64, float64)
}

func allMotions() []Motion {
	return []Motion{
		{"static", func(int) (float64, float64) { return 0, 0 }},
		{"slow-turn", func(t int) (float64, float64) {
			return 0.6 * float64(t), 0
		}},
		{"fast-saccade", func(t int) (float64, float64) {
			return 7.0 * float64(t), 0
		}},
	}
}

const (
	sceneTriangles   = "thin-triangles"
	sceneLeaves      = "translucent-leaves"
	sceneHighlights  = "specular-highlights"
	stratGT          = "ground-truth"
	stratIndependent = "foveated-independent"
	stratShared      = "foveated-shared"
	stratBlurred     = "foveated-blurred"
)

func allScenes() []string {
	return []string{sceneTriangles, sceneLeaves, sceneHighlights}
}

func allStrategies() []string {
	return []string{stratGT, stratIndependent, stratShared, stratBlurred}
}
