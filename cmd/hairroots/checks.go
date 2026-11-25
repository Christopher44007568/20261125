package main

import (
	"fmt"
	"math"

	"hairroots/internal/poisson"
	"hairroots/internal/scalp"
)

func meanStd(xs []float64) (mean, sd float64) {
	for _, x := range xs {
		mean += x
	}
	mean /= float64(len(xs))
	for _, x := range xs {
		sd += (x - mean) * (x - mean)
	}
	return mean, math.Sqrt(sd / float64(len(xs)))
}

func maxRadius(roots []poisson.Root) float64 {
	m := 0.0
	for _, r := range roots {
		if r.Radius > m {
			m = r.Radius
		}
	}
	return m
}

// latticeReport measures directional anisotropy of nearest-neighbor links.
// A crystalline lattice shows one or two dominant directions (high resultant
// length); a Poisson pattern spreads energy uniformly around the ring.
func latticeReport(roots []poisson.Root) string {
	const bins = 24
	hist := make([]float64, bins)
	count := 0
	for i := range roots {
		best := math.Inf(1)
		bestJ := -1
		pi := roots[i].Position
		for j := range roots {
			if i == j {
				continue
			}
			if d := pi.Sub(roots[j].Position).Length(); d < best {
				best, bestJ = d, j
			}
		}
		if bestJ < 0 {
			continue
		}
		v := roots[bestJ].Position.Sub(pi)
		angle := math.Atan2(v.Y, v.X)
		hist[int((angle+math.Pi)/(2*math.Pi)*bins)%bins]++
		count++
	}

	// Resultant length of the circular histogram (both order 1 and order 2;
	// square lattices are detected most clearly by the 2nd harmonic).
	var c1, s1, c2, s2 float64
	for k, h := range hist {
		a := 2 * math.Pi * float64(k) / bins
		c1 += h * math.Cos(a)
		s1 += h * math.Sin(a)
		c2 += h * math.Cos(2*a)
		s2 += h * math.Sin(2*a)
	}
	n := float64(count)
	r1 := math.Hypot(c1, s1) / n
	r2 := math.Hypot(c2, s2) / n
	verdict := "isotropic (no dominant lattice direction)"
	if r1 > 0.12 || r2 > 0.12 {
		verdict = "WARNING: directional bias detected"
	}
	return fmt.Sprintf("neighbor links=%d  R1=%.4f  R2=%.4f  -> %s", count, r1, r2, verdict)
}

// continuityReport numerically verifies that the weight field has no jumps
// along a full azimuth ring and reports the maximum finite difference.
func continuityReport(s *scalp.Scalp) string {
	const theta = 0.9
	const steps = 720
	dPhi := 2 * math.Pi / steps
	maxJump := 0.0
	prev := s.Weight(theta, -math.Pi)
	for k := 1; k <= steps; k++ {
		phi := -math.Pi + float64(k)*dPhi
		w := s.Weight(theta, phi)
		if d := math.Abs(w-prev) / dPhi; d > maxJump {
			maxJump = d
		}
		prev = w
	}
	wrapJump := math.Abs(s.Weight(theta, math.Pi) - s.Weight(theta, -math.Pi))
	return fmt.Sprintf("theta=%.2f ring: max |dWeight/dphi|=%.4f, seam jump=%.2e (both near zero => smooth)\n",
		theta, maxJump, wrapJump)
}
