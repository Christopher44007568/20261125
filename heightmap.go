package main

import (
	"fmt"
	"math"
)

// HeightMap is a row-major normalized height field with bilinear sampling.
type HeightMap struct {
	Width   int
	Height  int
	Samples []float64
	// Repeat wraps UV coordinates into [0,1). When false, sampling outside
	// UnitRect returns ErrUVOutOfBounds and the tracer can safely fall back.
	Repeat bool
}

func NewHeightMap(width, height int, samples []float64) (*HeightMap, error) {
	if width < 2 || height < 2 {
		return nil, fmt.Errorf("height map dimensions must be at least 2x2")
	}
	if len(samples) != width*height {
		return nil, fmt.Errorf("height map has %d samples for %dx%d", len(samples), width, height)
	}
	return &HeightMap{Width: width, Height: height, Samples: samples}, nil
}

func (m *HeightMap) HeightAt(uv Vec2) (float64, error) {
	if m == nil || m.Width < 2 || m.Height < 2 || len(m.Samples) != m.Width*m.Height {
		return 0, ErrInvalidConfig
	}

	if !UnitRect.Contains(uv) {
		if !m.Repeat {
			return 0, fmt.Errorf("%w: uv=(%.6g,%.6g)", ErrUVOutOfBounds, uv.X, uv.Y)
		}
		uv.X = math.Mod(uv.X, 1)
		uv.Y = math.Mod(uv.Y, 1)
		if uv.X < 0 {
			uv.X++
		}
		if uv.Y < 0 {
			uv.Y++
		}
	}

	gx := uv.X * float64(m.Width-1)
	gy := uv.Y * float64(m.Height-1)
	x0 := int(math.Floor(gx))
	y0 := int(math.Floor(gy))
	x1 := minInt(m.Width-1, x0+1)
	y1 := minInt(m.Height-1, y0+1)
	tx := gx - float64(x0)
	ty := gy - float64(y0)

	h00 := m.at(x0, y0)
	h10 := m.at(x1, y0)
	h01 := m.at(x0, y1)
	h11 := m.at(x1, y1)
	top := h00*(1-tx) + h10*tx
	bottom := h01*(1-tx) + h11*tx
	return top*(1-ty) + bottom*ty, nil
}

func (m *HeightMap) at(x, y int) float64 {
	return m.Samples[y*m.Width+x]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
