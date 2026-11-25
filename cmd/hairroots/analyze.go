package main

import (
	"fmt"
	"math"
	"sort"

	"hairroots/internal/poisson"
	"hairroots/internal/render"
	"hairroots/internal/scalp"
)

// zone names follow the required anatomical regions. Classification uses
// smooth lobes only for reporting; the underlying density field itself is
// defined by continuous functions.
func zoneName(theta, phi float64) string {
	front := 0.5 + 0.5*math.Cos(phi)
	ear := math.Exp(-math.Pow(scalp.WrapAngle(math.Abs(phi)-math.Pi/2)/0.45, 2))
	switch {
	case theta < 0.55:
		return "crown"
	case ear > 0.6:
		return "behind_ear"
	case front > 0.65:
		return "forehead"
	default:
		return "other"
	}
}

func zoneArea(s *scalp.Scalp, name string) float64 {
	const nT, nP = 240, 480
	dTheta := (math.Pi / 2) / nT
	dPhi := 2 * math.Pi / nP
	area := 0.0
	for i := 0; i < nT; i++ {
		theta := (float64(i) + 0.5) * dTheta
		for j := 0; j < nP; j++ {
			phi := -math.Pi + (float64(j)+0.5)*dPhi
			if !s.Inside(theta, phi) || zoneName(theta, phi) != name {
				continue
			}
			area += s.AreaElement(theta, phi) * dTheta * dPhi
		}
	}
	return area
}

// Analyze produces the human-readable verification report.
func Analyze(s *scalp.Scalp, roots []poisson.Root, grid *render.DensityGrid) string {
	counts := map[string]int{}
	for _, r := range roots {
		counts[zoneName(r.Theta, r.Phi)]++
	}

	nn := nearestNeighbors(roots)
	sort.Float64s(nn)
	mean, sd := meanStd(nn)

	return fmt.Sprintf(`
Poisson-disc scalp report
=========================
roots            : %d
nearest neighbor : min %.4f  median %.4f  mean %.4f  max %.4f  sd %.4f
disc radius span : %.4f .. %.4f

regional density (roots per unit surface area)
----------------------------------------------
%s

lattice check
-------------
%s

continuity check (weight along phi at fixed theta, finite differences)
----------------------------------------------------------------------
%s
heat map bins    : %d theta x %d phi, peak density %.2f roots/area
`,
		len(roots),
		nn[0], nn[len(nn)/2], mean, nn[len(nn)-1], sd,
		roots[0].Radius, maxRadius(roots),
		zoneReport(s, counts),
		latticeReport(roots),
		continuityReport(s),
		grid.NTheta, grid.NPhi, grid.MaxDensity,
	)
}

func zoneReport(s *scalp.Scalp, counts map[string]int) string {
	out := ""
	for _, name := range []string{"crown", "forehead", "behind_ear", "other"} {
		area := zoneArea(s, name)
		density := float64(counts[name]) / area
		out += fmt.Sprintf("%-11s: %5d roots over %5.3f area -> %7.2f roots/area\n",
			name, counts[name], area, density)
	}
	return out
}

func nearestNeighbors(roots []poisson.Root) []float64 {
	// O(n^2) is acceptable at the target count and keeps the code dependency
	// free; distances are chord lengths on the unit-scale ellipsoid.
	out := make([]float64, len(roots))
	for i := range roots {
		best := math.Inf(1)
		pi := roots[i].Position
		for j := range roots {
			if i == j {
				continue
			}
			if d := pi.Sub(roots[j].Position).Length(); d < best {
				best = d
			}
		}
		out[i] = best
	}
	return out
}
