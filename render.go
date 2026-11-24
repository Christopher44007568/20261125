package main

import "math"

const (
	tMax       = 8.0
	dtFull     = 0.02 // per-eye full sampling step
	dtCoarse   = 0.16 // shared low sampling step
	dtFovea    = 0.02 // foveal step for center-partitioned sampling
	dtPeriph   = 0.16 // peripheral step for center-partitioned sampling
	foveaAngle = 0.18 // radians around the gaze direction sampled finely
)

// MarchResult holds the accumulated radiance plus the transmittance
// curve T(t) sampled along the ray (for probe pixels).
type MarchResult struct {
	Color  Vec3
	TTime  []float64 // t of each curve sample (step midpoints)
	TCurve []float64 // transmittance after each step
}

// alphaFor converts density to per-step opacity for a given step length.
// This is the physically correct form; see CorrectAlpha.
func alphaFor(sigma, dt float64) float64 { return 1 - math.Exp(-sigma*dt) }

// CorrectAlpha rescales an opacity authored for step dtFrom to step dtTo:
// alpha2 = 1 - (1-alpha1)^(dtTo/dtFrom). Required whenever the volume
// step length changes, otherwise thin layers become too transparent and
// dense layers too opaque (or vice versa).
func CorrectAlpha(alpha, dtFrom, dtTo float64) float64 {
	return 1 - math.Pow(1-alpha, dtTo/dtFrom)
}

// march integrates one ray front-to-back with a per-sample step chosen by
// stepAt(t). record enables transmittance-curve capture.
func march(sc Scene, o, d Vec3, stepAt func(t float64) float64, record bool) MarchResult {
	T := 1.0
	L := Vec3{}
	var curve []float64
	var times []float64
	t := 0.0
	for t < tMax && T > 1e-4 {
		dt := stepAt(t)
		if t+dt > tMax {
			dt = tMax - t
		}
		p := o.Add(d.Scale(t + 0.5*dt))
		sigma := sc.Density(p)
		a := alphaFor(sigma, dt)
		e := sc.Emission(p)
		L = L.Add(e.Scale(T * dt))
		T *= 1 - a
		if record {
			times = append(times, t+0.5*dt)
			curve = append(curve, T)
		}
		t += dt
	}
	// Background sky contribution.
	bg := Vec3{0.05, 0.07, 0.10}
	L = L.Add(bg.Scale(T))
	return MarchResult{Color: L, TTime: times, TCurve: curve}
}

// stepFull is the per-eye full-sampling schedule.
func stepFull(float64) float64 { return dtFull }

// stepShared is the shared low-sampling schedule (one coarse march reused
// for both eyes -- the approach this tool shows to be wrong).
func stepShared(float64) float64 { return dtCoarse }

// fovealStep picks the step schedule for one pixel: fine inside the foveal
// cone around the gaze direction, coarse outside (center-partitioned
// sampling). A pinhole pixel has a fixed direction, so the partition is a
// per-pixel decision.
func fovealStep(d, gaze Vec3) func(t float64) float64 {
	cosF := math.Cos(foveaAngle)
	if d.Dot(gaze) >= cosF {
		return func(float64) float64 { return dtFovea }
	}
	return func(float64) float64 { return dtPeriph }
}
