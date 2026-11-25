package render

import (
	"image/color"
	"math"
	"sort"

	"hairroots/internal/poisson"
	"hairroots/internal/scalp"
)

// turbo maps t in [0,1] to a blue -> cyan -> green -> yellow -> red ramp.
func turbo(t float64) color.Color {
	t = clamp01(t)
	r := clamp01(1.5 - math.Abs(4*t-3))
	g := clamp01(1.5 - math.Abs(4*t-2))
	b := clamp01(1.5 - math.Abs(4*t-1))
	return color.RGBA{R: uint8(255 * r), G: uint8(255 * g), B: uint8(255 * b), A: 255}
}

func clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// DensityGrid accumulates root counts over a (theta, phi) grid and divides by
// the physical surface area of each bin, giving roots per unit area instead of
// a raw count that would be distorted by the parameterization.
type DensityGrid struct {
	NTheta, NPhi int
	Counts       []int
	Density      []float64
	MaxDensity   float64
	ColorMax     float64
	Valid        []bool
}

func BuildDensityGrid(s *scalp.Scalp, roots []poisson.Root, nTheta, nPhi int) *DensityGrid {
	g := &DensityGrid{
		NTheta:  nTheta,
		NPhi:    nPhi,
		Counts:  make([]int, nTheta*nPhi),
		Density: make([]float64, nTheta*nPhi),
		Valid:   make([]bool, nTheta*nPhi),
	}
	for _, rt := range roots {
		it := int(rt.Theta / (math.Pi / 2) * float64(nTheta))
		ip := int((rt.Phi + math.Pi) / (2 * math.Pi) * float64(nPhi))
		it = min(it, nTheta-1)
		ip = min(max(ip, 0), nPhi-1)
		g.Counts[it*nPhi+ip]++
	}

	dTheta := (math.Pi / 2) / float64(nTheta)
	dPhi := 2 * math.Pi / float64(nPhi)
	for it := 0; it < nTheta; it++ {
		theta := (float64(it) + 0.5) * dTheta
		for ip := 0; ip < nPhi; ip++ {
			phi := -math.Pi + (float64(ip)+0.5)*dPhi
			area := s.AreaElement(theta, phi) * dTheta * dPhi
			// Mask the two innermost polar rings: the theta/phi Jacobian
			// tends to zero at the pole, so one root there would imply a
			// huge, unstable per-area density. The crown density is still
			// visible one ring lower where the area element is conditioned.
			if it < 2 {
				continue
			}
			d := float64(g.Counts[it*nPhi+ip]) / area
			g.Density[it*nPhi+ip] = d
			g.Valid[it*nPhi+ip] = true
			if d > g.MaxDensity {
				g.MaxDensity = d
			}
		}
	}
	// Robust color scale: use the 95th percentile so a single root landing in
	// a tiny polar bin cannot flatten the whole map to one color.
	values := make([]float64, 0, len(g.Density))
	for i, d := range g.Density {
		if d > 0 && g.Valid[i] {
			values = append(values, d)
		}
	}
	sort.Float64s(values)
	if len(values) > 0 {
		g.ColorMax = values[int(float64(len(values))*0.95)]
	}
	g.smooth()
	return g
}

// smooth applies two separable 3x3 averaging passes over the density field,
// suppressing single-bin count noise while preserving the smooth lobes of
// the underlying weight function. Invalid polar bins take no part.
func (g *DensityGrid) smooth() {
	for pass := 0; pass < 2; pass++ {
		tmp := make([]float64, len(g.Density))
		for it := 0; it < g.NTheta; it++ {
			for ip := 0; ip < g.NPhi; ip++ {
				if !g.Valid[it*g.NPhi+ip] {
					continue
				}
				sum, n := 0.0, 0.0
				for d := -1; d <= 1; d++ {
					jp := (ip + d + g.NPhi) % g.NPhi
					if !g.Valid[it*g.NPhi+jp] {
						continue
					}
					sum += g.Density[it*g.NPhi+jp]
					n++
				}
				tmp[it*g.NPhi+ip] = sum / n
			}
		}
		for it := 0; it < g.NTheta; it++ {
			for ip := 0; ip < g.NPhi; ip++ {
				if !g.Valid[it*g.NPhi+ip] {
					continue
				}
				sum, n := 0.0, 0.0
				for d := -1; d <= 1; d++ {
					jt := it + d
					if jt < 0 || jt >= g.NTheta || !g.Valid[jt*g.NPhi+ip] {
						continue
					}
					sum += tmp[jt*g.NPhi+ip]
					n++
				}
				g.Density[it*g.NPhi+ip] = sum / n
			}
		}
	}
}

// DrawHeatMap lays the normalized density grid into an image. The horizontal
// axis is phi (front in the middle, back at the edges) and the vertical axis
// is theta from crown (top) to hairline (bottom).
func (g *DensityGrid) DrawHeatMap(c *Canvas, s *scalp.Scalp) {
	for y := 0; y < c.H; y++ {
		it := min(nThetaScaled(y, c.H, g.NTheta), g.NTheta-1)
		theta := (float64(it) + 0.5) * (math.Pi / 2) / float64(g.NTheta)
		for x := 0; x < c.W; x++ {
			ip := min(nPhiScaled(x, c.W, g.NPhi), g.NPhi-1)
			phi := -math.Pi + (float64(ip)+0.5)*(2*math.Pi)/float64(g.NPhi)
			col := color.RGBA{R: 24, G: 26, B: 32, A: 255}
			if s.Inside(theta, phi) && g.Valid[it*g.NPhi+ip] && g.ColorMax > 0 {
				col = turbo(g.Density[it*g.NPhi+ip] / g.ColorMax).(color.RGBA)
			}
			c.Set2D(x, y, col)
		}
	}
}

func nThetaScaled(y, h, n int) int { return min(int(float64(y)/float64(h)*float64(n)), n-1) }
func nPhiScaled(x, w, n int) int   { return min(int(float64(x)/float64(w)*float64(n)), n-1) }
