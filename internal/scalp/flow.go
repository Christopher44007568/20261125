package scalp

import "math"

// Whorl position in parameter space (the crown, offset slightly backward).
const (
	WhorlTheta = 0.22
	WhorlPhi   = 2.35 // toward the back
)

// FlowDirection returns the unit hair-growth direction at (theta, phi),
// expressed in the local tangent frame as (u along +theta, v along +phi).
//
// The field blends two smooth tangent lobes: a backward component that sweeps
// hair from the forehead toward the nape, and a rotational component around
// the crown whorl. Both are smooth functions of the parameters, so direction
// never jumps between neighboring roots.
func (s *Scalp) FlowDirection(theta, phi float64) (u, v float64) {
	tTheta, tPhi, _ := s.Frame(theta, phi)

	// Backward sweep direction expressed in the tangent frame.
	sweepWorld := Vec3{X: 0, Y: -0.55, Z: 0.85}
	sweepU := sweepWorld.Dot(tTheta)
	sweepV := sweepWorld.Dot(tPhi)

	// Rotational lobe around the whorl: azimuthal tangent in parameter space.
	dTheta := theta - WhorlTheta
	dPhi := WrapAngle(phi - WhorlPhi)
	dist := math.Hypot(dTheta, dPhi)
	var swirlU, swirlV float64
	if dist > 1e-6 {
		// Unit azimuthal vector (-dPhi, dTheta)/dist around the whorl.
		swirlU = -dPhi / dist
		swirlV = dTheta / dist
	}
	// The swirl dominates close to the whorl and fades out smoothly.
	swirlGain := math.Exp(-(dist * dist) / (2 * 0.55 * 0.55))
	sweepGain := 1 - 0.55*swirlGain

	u = sweepGain*sweepU + swirlGain*swirlU
	v = sweepGain*sweepV + swirlGain*swirlV
	mag := math.Hypot(u, v)
	if mag < 1e-9 {
		return sweepU, sweepV
	}
	return u / mag, v / mag
}

// FlowWorld returns the hair direction in world coordinates.
func (s *Scalp) FlowWorld(theta, phi float64) Vec3 {
	tTheta, tPhi, _ := s.Frame(theta, phi)
	u, v := s.FlowDirection(theta, phi)
	return tTheta.Scale(u).Add(tPhi.Scale(v)).Normalize()
}
