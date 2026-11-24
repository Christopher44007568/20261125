package main

import (
	"errors"
	"math"
)

// Vec2 is a tangent-space UV coordinate or velocity.
type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vec2) Scale(scale float64) Vec2 {
	return Vec2{X: v.X * scale, Y: v.Y * scale}
}

func (v Vec2) Distance(other Vec2) float64 {
	return math.Hypot(v.X-other.X, v.Y-other.Y)
}

// Rect is an inclusive UV sampling domain.
type Rect struct {
	Min Vec2 `json:"min"`
	Max Vec2 `json:"max"`
}

var UnitRect = Rect{Min: Vec2{}, Max: Vec2{X: 1, Y: 1}}

func (r Rect) Contains(uv Vec2) bool {
	return uv.X >= r.Min.X && uv.X <= r.Max.X &&
		uv.Y >= r.Min.Y && uv.Y <= r.Max.Y
}

var (
	ErrUVOutOfBounds    = errors.New("uv coordinate is outside height field bounds")
	ErrHeightOutOfRange = errors.New("height sample is outside normalized range")
	ErrInvalidRay       = errors.New("invalid parallax ray")
	ErrInvalidConfig    = errors.New("invalid search configuration")
)

// HeightField returns normalized elevation in [0,1]. Higher values are closer
// to the viewer; SurfaceDepth is consequently 1-HeightAt.
type HeightField interface {
	HeightAt(uv Vec2) (float64, error)
}

// HeightFunc adapts a function to HeightField. A non-nil error stops the
// search and triggers the geometric-surface fallback.
type HeightFunc func(uv Vec2) (float64, error)

func (f HeightFunc) HeightAt(uv Vec2) (float64, error) {
	return f(uv)
}

// BoundedFunc rejects UV coordinates outside Bounds before invoking Fn.
type BoundedFunc struct {
	Bounds Rect
	Fn     func(uv Vec2) float64
}

func (f BoundedFunc) HeightAt(uv Vec2) (float64, error) {
	if f.Bounds == (Rect{}) {
		f.Bounds = UnitRect
	}
	if !f.Bounds.Contains(uv) {
		return 0, ErrUVOutOfBounds
	}
	return f.Fn(uv), nil
}
