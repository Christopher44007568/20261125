package main

import "math"

// Strand is one guide hair: an ordered polyline from root to tip.
type Strand struct {
	ID        int
	Pts       []Vec3
	Root      Vec3
	Normal    Vec3
	Thickness float64
	Color     [3]float64
	Spec      float64
	Flyaway   bool
	Length    float64
}

// Groom bundles all strands plus the scalp they grow from.
type Groom struct {
	Strands []Strand
	Scalp   Ellipsoid
}

// Ellipsoid describes the scalp/head cap used for silhouette reference.
type Ellipsoid struct {
	Center Vec3
	Radii  Vec3
}

func (e Ellipsoid) surfaceNormal(p Vec3) Vec3 {
	d := p.Sub(e.Center)
	return v3(d.X/(e.Radii.X*e.Radii.X), d.Y/(e.Radii.Y*e.Radii.Y), d.Z/(e.Radii.Z*e.Radii.Z)).Norm()
}

const (
	strandCount  = 680
	segments     = 24
	flyawayRatio = 0.14
	groomSeed    = 20261125
	scalpCenterY = 9.0
	scalpRX      = 7.2
	scalpRY      = 8.2
	scalpRZ      = 6.6
)

// generateGroom builds a deterministic swept-back hair groom on an ellipsoid
// scalp. A minority of strands are deliberately noisy "flyaways".
func generateGroom() *Groom {
	rng := randSeed{groomSeed}
	scalp := Ellipsoid{Center: v3(0, scalpCenterY, 0), Radii: v3(scalpRX, scalpRY, scalpRZ)}
	g := &Groom{Scalp: scalp, Strands: make([]Strand, 0, strandCount)}

	for id := 0; id < strandCount; id++ {
		// Stratified-ish jittered sampling of the upper scalp cap.
		u := rng.float()
		v := rng.float()
		phi := 2 * math.Pi * u
		cosTheta := 1.0 - v*1.45 // keep the cap above the lower cutoff
		if cosTheta < -0.35 {
			cosTheta = -0.35
		}
		sinTheta := math.Sqrt(math.Max(0, 1-cosTheta*cosTheta))
		dir := v3(sinTheta*math.Cos(phi), cosTheta, sinTheta*math.Sin(phi))
		root := scalp.Center.Add(v3(scalp.Radii.X*dir.X, scalp.Radii.Y*dir.Y, scalp.Radii.Z*dir.Z))
		normal := scalp.surfaceNormal(root)

		flyaway := rng.float() < flyawayRatio
		s := Strand{
			ID:        id,
			Root:      root,
			Normal:    normal,
			Thickness: rng.rangeF(0.018, 0.034),
			Spec:      rng.rangeF(0.5, 1.0),
			Flyaway:   flyaway,
		}
		// Cool dark brown with subtle per-strand warmth variation.
		warm := rng.rangeF(-0.03, 0.05)
		s.Color = [3]float64{0.10 + warm, 0.075 + warm*0.6, 0.055}

		length := rng.rangeF(7.5, 11.5)
		if flyaway {
			length = rng.rangeF(5.0, 9.0)
		}
		s.Length = length

		// Flow field: mostly downward hair with a modest swept-back bias so
		// the groom reads as long hanging hair from the front view.
		flow := v3(0.22*dir.X, -0.92, -0.32).Norm()
		sweep := rng.rangeF(0.75, 1.15)
		dir0 := normal.Scale(0.55).Add(flow.Scale(sweep)).Norm()

		noiseAmp := 0.10
		if flyaway {
			noiseAmp = 0.55
		}
		n1 := rng.rangeF(-1, 1)
		n2 := rng.rangeF(-1, 1)
		n3 := rng.rangeF(-1, 1)
		noise := v3(n1, n2, n3).Scale(noiseAmp)

		s.Pts = make([]Vec3, segments+1)
		s.Pts[0] = root
		tan := dir0
		step := length / segments
		for k := 1; k <= segments; k++ {
			t := float64(k) / segments
			// Gravity bends the strand progressively toward -Y.
			gravity := v3(0, -1, 0).Scale(0.9 * t * t)
			kick := Vec3{}
			if flyaway {
				// Outward kick near the tip produces loose flyaway arcs.
				kick = normal.Scale(1.1 * t * t).Add(noise.Scale(t))
			} else {
				kick = noise.Scale(0.25 * t)
			}
			tan = tan.Add(gravity.Scale(0.09)).Add(kick.Scale(0.12)).Norm()
			s.Pts[k] = s.Pts[k-1].Add(tan.Scale(step))
		}
		g.Strands = append(g.Strands, s)
	}
	return g
}
