package poisson

import (
	"math"
)

// candidate creates one trial point in the annulus [r, 2r] around an accepted
// root. The offset is built in the local tangent plane so candidates spread
// isotropically on the curved surface, then radially projected back.
func (sm *Sampler) candidate(base Root) (theta, phi float64, ok bool) {
	tTheta, tPhi, n := sm.s.Frame(base.Theta, base.Phi)
	r := base.Radius

	dist := r * (1 + sm.rng.Float64()) // uniform radius is fine with 32 tries
	angle := 2 * math.Pi * sm.rng.Float64()
	offset := tTheta.Scale(dist * math.Cos(angle)).
		Add(tPhi.Scale(dist * math.Sin(angle)))

	theta, phi = sm.project(base.Position.Add(offset))
	if !sm.s.Inside(theta, phi) {
		return 0, 0, false
	}

	// Reject points whose projection moved too far: near the parameter axis
	// radial projection is stable, but this guards degenerate tangents.
	if sm.s.Point(theta, phi).Sub(base.Position).Dot(n) > r {
		return 0, 0, false
	}
	return theta, phi, true
}

func (sm *Sampler) accept(theta, phi float64) Root {
	w := sm.s.Weight(theta, phi)
	rad := sm.s.Radius(w, sm.rMin)
	u, v := sm.s.FlowDirection(theta, phi)
	return Root{
		Theta:     theta,
		Phi:       phi,
		Position:  sm.s.Point(theta, phi),
		Weight:    w,
		Radius:    rad,
		FlowU:     u,
		FlowV:     v,
		FlowWorld: sm.s.FlowWorld(theta, phi),
	}
}

// Run executes the Bridson process until the active list can no longer grow.
func (sm *Sampler) Run(seeds int) []Root {
	// Uniformly spaced seeds across the parameter domain, themselves thinned
	// through the same conflict rule.
	for k := 0; k < seeds; k++ {
		phi := -math.Pi + 2*math.Pi*(float64(k)+0.5)/float64(seeds)
		theta := sm.s.ThetaMax(phi) * (0.15 + 0.7*sm.rng.Float64())
		if !sm.s.Inside(theta, phi) {
			continue
		}
		r := sm.accept(theta, phi)
		if !sm.conflicts(r.Position, r.Radius) {
			sm.insert(r)
		}
	}

	for len(sm.active) > 0 && len(sm.roots) < maxRoots {
		idx := sm.rng.Intn(len(sm.active))
		parent := sm.roots[sm.active[idx]]

		found := false
		for try := 0; try < attemptsPerSeed; try++ {
			theta, phi, ok := sm.candidate(parent)
			if !ok {
				continue
			}
			w := sm.s.Weight(theta, phi)
			rad := sm.s.Radius(w, sm.rMin)
			p := sm.s.Point(theta, phi)
			if sm.conflicts(p, rad) {
				continue
			}
			sm.insert(sm.accept(theta, phi))
			found = true
			break
		}

		if !found {
			last := len(sm.active) - 1
			sm.active[idx] = sm.active[last]
			sm.active = sm.active[:last]
		}
	}
	return sm.roots
}
