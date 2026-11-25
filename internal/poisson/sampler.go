// Package poisson implements a variable-radius Bridson Poisson-disc sampler
// over the curved scalp domain. The exclusion radius follows the smooth
// density field, so spacing changes continuously without a lattice.
package poisson

import (
	"math"
	"math/rand"

	"hairroots/internal/scalp"
)

// Root is one accepted hair follicle.
type Root struct {
	ID             int
	Theta, Phi     float64
	Position       scalp.Vec3
	Weight, Radius float64
	FlowU, FlowV   float64 // unit direction in the local tangent frame
	FlowWorld      scalp.Vec3
}

type cellKey struct{ i, j, k int }

type cellEntry struct {
	index  int
	radius float64
}

// Sampler holds the spatial hash grid and active list.
type Sampler struct {
	s      *scalp.Scalp
	rng    *rand.Rand
	rMin   float64
	cell   float64
	grid   map[cellKey][]cellEntry
	active []int
	roots  []Root
}

// New builds a sampler. rMin is the disc radius in the densest (crown) zone.
func New(s *scalp.Scalp, rng *rand.Rand, rMin float64) *Sampler {
	return &Sampler{
		s:    s,
		rng:  rng,
		rMin: rMin,
		cell: rMin,
		grid: make(map[cellKey][]cellEntry),
	}
}

const (
	attemptsPerSeed = 32
	searchRadius    = 3 // cells; covers max pair radius (~2.6 * rMin)
	maxRoots        = 200000
)

// project maps an arbitrary 3D point back onto the ellipsoid by radial
// projection in scaled-unit space and returns its (theta, phi).
func (sm *Sampler) project(p scalp.Vec3) (theta, phi float64) {
	x, y, z := p.X/sm.s.A, p.Y/sm.s.B, p.Z/sm.s.C
	r := math.Sqrt(x*x + y*y + z*z)
	theta = math.Acos(clamp(y/r, -1, 1))
	phi = math.Atan2(z, x)
	return theta, phi
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (sm *Sampler) keyOf(p scalp.Vec3) cellKey {
	return cellKey{
		int(math.Floor(p.X / sm.cell)),
		int(math.Floor(p.Y / sm.cell)),
		int(math.Floor(p.Z / sm.cell)),
	}
}

// conflicts reports whether p is closer than the combined exclusion radius to
// any accepted root. The mean-radius criterion stays symmetric for pairs with
// different local densities.
func (sm *Sampler) conflicts(p scalp.Vec3, radius float64) bool {
	key := sm.keyOf(p)
	for di := -searchRadius; di <= searchRadius; di++ {
		for dj := -searchRadius; dj <= searchRadius; dj++ {
			for dk := -searchRadius; dk <= searchRadius; dk++ {
				for _, e := range sm.grid[cellKey{key.i + di, key.j + dj, key.k + dk}] {
					q := sm.roots[e.index].Position
					if p.Sub(q).Length() < 0.5*(radius+e.radius) {
						return true
					}
				}
			}
		}
	}
	return false
}

func (sm *Sampler) insert(r Root) {
	r.ID = len(sm.roots)
	sm.roots = append(sm.roots, r)
	key := sm.keyOf(r.Position)
	sm.grid[key] = append(sm.grid[key], cellEntry{index: r.ID, radius: r.Radius})
	sm.active = append(sm.active, r.ID)
}
