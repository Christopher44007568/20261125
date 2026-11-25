// Package scalp defines an ellipsoidal scalp parameterization and the smooth
// density field used by the Poisson-disc root sampler.
package scalp

import "math"

// Scalp is the upper half of an ellipsoid centered at the origin.
// Theta is the polar angle measured from the +Y (crown) axis; phi is the
// azimuth, with phi=0 pointing toward the face (+Z) and phi=+/-pi the back.
type Scalp struct {
	A, B, C float64 // semi-axes: X (left-right), Y (up), Z (face-back)
}

func New() *Scalp {
	return &Scalp{A: 1.0, B: 1.08, C: 1.15}
}

// Point maps (theta, phi) to a point on the ellipsoid surface.
func (s *Scalp) Point(theta, phi float64) Vec3 {
	st, ct := math.Sin(theta), math.Cos(theta)
	return Vec3{
		X: s.A * st * math.Cos(phi),
		Y: s.B * ct,
		Z: s.C * st * math.Sin(phi),
	}
}

// Normal returns the outward unit normal at (theta, phi).
func (s *Scalp) Normal(theta, phi float64) Vec3 {
	p := s.Point(theta, phi)
	return Vec3{p.X / (s.A * s.A), p.Y / (s.B * s.B), p.Z / (s.C * s.C)}.Normalize()
}

// Frame returns the orthonormal tangent frame (tTheta, tPhi, n) at the point.
func (s *Scalp) Frame(theta, phi float64) (tTheta, tPhi, n Vec3) {
	eps := 1e-6
	tTheta = s.Point(math.Min(theta+eps, math.Pi/2), phi).Sub(s.Point(math.Max(theta-eps, 0), phi)).Normalize()
	tPhi = s.Point(theta, phi+eps).Sub(s.Point(theta, phi-eps)).Normalize()
	n = s.Normal(theta, phi)
	// Re-orthogonalize to counter finite-difference noise.
	tTheta = tTheta.Sub(n.Scale(tTheta.Dot(n))).Normalize()
	tPhi = n.Cross(tTheta).Normalize()
	return tTheta, tPhi, n
}

// AreaElement returns dS/d(theta,phi), the physical surface area carried by
// one unit of parameter space. It is used to normalize the count heat map.
func (s *Scalp) AreaElement(theta, phi float64) float64 {
	eps := 1e-5
	p00 := s.Point(theta, phi)
	dTheta := s.Point(math.Min(theta+eps, math.Pi/2), phi).Sub(p00).Scale(1 / eps)
	dPhi := s.Point(theta, phi+eps).Sub(p00).Scale(1 / eps)
	return dTheta.Cross(dPhi).Length()
}

// ThetaMax is the polar angle of the hairline boundary at azimuth phi.
// The front hairline sits slightly higher than the nape, and smooth Gaussian
// notches pull the line upward behind both ears.
func (s *Scalp) ThetaMax(phi float64) float64 {
	const base = 1.52
	blend := 0.5 + 0.5*math.Cos(phi) // 1 at front (phi=0), 0 at back
	frontBack := -0.10 * blend
	ear := 0.18 * (gaussAngle(phi, math.Pi/2, 0.32) + gaussAngle(phi, -math.Pi/2, 0.32))
	return base + frontBack - ear
}

// Inside reports whether the parameter pair lies under the hairline.
func (s *Scalp) Inside(theta, phi float64) bool {
	return theta >= 0 && theta <= s.ThetaMax(phi)
}

// Weight is the smooth, lattice-free density field. It is C1 continuous in
// both parameters: one azimuth-independent crown lobe plus two symmetric
// behind-ear depressions, all built from smooth Gaussian/cosine functions.
// Because nothing is hard-thresholded, the crown -> forehead -> ear density
// changes are gradual and the phi seam has zero jump.
func (s *Scalp) Weight(theta, phi float64) float64 {
	const base = 0.95

	// Crown / vertex: the thickest region, centered on the polar axis so it
	// stays independent of azimuth (no seam artifacts at phi = +/-pi).
	crown := 0.52 * math.Exp(-(theta*theta)/(2*0.46*0.46))

	// Behind-ear thinning: symmetric smooth depressions, lowest near the ears.
	earMask := gaussAngle(phi, math.Pi/2, 0.40) + gaussAngle(phi, -math.Pi/2, 0.40)
	ear := -0.50 * earMask * math.Exp(-math.Pow((theta-1.02)/0.38, 2))

	return base + crown + ear
}

// Radius maps the local density weight to the Poisson-disc exclusion radius:
// denser regions accept smaller discs, and the mapping is continuous so the
// crown-to-ear spacing changes gradually rather than in bands.
func (s *Scalp) Radius(weight, rMin float64) float64 {
	return rMin / math.Sqrt(weight)
}
