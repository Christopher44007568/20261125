package main

import (
	"image"
	"math"
	"time"
)

const (
	defaultWidth         = 640
	defaultHeight        = 360
	defaultFrames        = 8
	defaultFrameBudget   = 16667 * time.Microsecond
	depthOcclusionBiasM  = 0.012
	systemOverlayEnabled = true
)

type Config struct {
	Width         int
	Height        int
	Frames        int
	Seed          int64
	OutDir        string
	LatencyBudget time.Duration
}

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (v Vec3) Add(o Vec3) Vec3    { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3    { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Mul(s float64) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) Dot(o Vec3) float64 { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }
func (v Vec3) Length() float64    { return math.Sqrt(v.Dot(v)) }
func (v Vec3) Normalize() Vec3 {
	n := v.Length()
	if n < 1e-12 {
		return Vec3{}
	}
	return v.Mul(1 / n)
}

type RGB struct {
	R float64
	G float64
	B float64
}

func (c RGB) Add(o RGB) RGB        { return RGB{c.R + o.R, c.G + o.G, c.B + o.B} }
func (c RGB) Mul(o RGB) RGB        { return RGB{c.R * o.R, c.G * o.G, c.B * o.B} }
func (c RGB) Scale(s float64) RGB  { return RGB{c.R * s, c.G * s, c.B * s} }
func (c RGB) Luma() float64        { return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B }
func (c RGB) Max() float64         { return math.Max(c.R, math.Max(c.G, c.B)) }
func (c RGB) IsFinite() bool       { return isFinite(c.R) && isFinite(c.G) && isFinite(c.B) }
func clamp01(v float64) float64    { return math.Max(0, math.Min(1, v)) }
func isFinite(v float64) bool      { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func lerp(a, b, t float64) float64 { return a + (b-a)*t }
func smoothstep(a, b, x float64) float64 {
	t := clamp01((x - a) / (b - a))
	return t * t * (3 - 2*t)
}

type Intrinsics struct {
	FX float64
	FY float64
	CX float64
	CY float64
}

func intrinsicsFor(width, height int, verticalFOVDegrees float64) Intrinsics {
	fy := (float64(height) * 0.5) / math.Tan(verticalFOVDegrees*math.Pi/360)
	return Intrinsics{
		FX: fy,
		FY: fy,
		CX: float64(width) * 0.5,
		CY: float64(height) * 0.5,
	}
}

type Sphere struct {
	Center  Vec3
	Radius  float64
	AlbedoA RGB
	AlbedoB RGB
}

type ExposureFrame struct {
	Scale       float64
	ActualScale float64
	Image       *image.NRGBA
}

type CaptureBundle struct {
	Sequence     int
	CapturedAt   time.Time
	Intrinsics   Intrinsics
	Exposures    []ExposureFrame
	DepthM       []float64
	DepthValid   []bool
	SystemLayer  *image.NRGBA
	WhiteBalance RGB
	Sphere       Sphere
}

type LinearImage struct {
	Width  int
	Height int
	Pix    []RGB
}

func newLinearImage(width, height int) *LinearImage {
	return &LinearImage{Width: width, Height: height, Pix: make([]RGB, width*height)}
}

type VirtualLayer struct {
	Width     int
	Height    int
	Pix       []RGB
	Alpha     []float64
	Footprint []bool
}

func newVirtualLayer(width, height int) *VirtualLayer {
	return &VirtualLayer{
		Width:     width,
		Height:    height,
		Pix:       make([]RGB, width*height),
		Alpha:     make([]float64, width*height),
		Footprint: make([]bool, width*height),
	}
}

type ColorCalibration struct {
	Gains  RGB
	Method string
}
