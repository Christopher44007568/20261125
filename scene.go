package main

import "math"

// Scene is an analytic participating medium: density sigma (1/m) and
// emitted radiance per unit length. It combines three features from the
// spec: a thin smoke layer, a fog wall, and a transparent glowing body.
type Scene struct{}

func (Scene) Density(p Vec3) float64 {
	return smokeDensity(p) + fogWallDensity(p) + glowDensity(p)
}

func (Scene) Emission(p Vec3) Vec3 {
	e := glowDensity(p)
	if e <= 0 {
		return Vec3{}
	}
	// Warm glow, emission proportional to local glow density.
	return Vec3{1.0, 0.55, 0.25}.Scale(e * 6.0)
}

// Thin smoke: a slab 0.25 m thick at z in [2.0, 2.25], density modulated
// by 2-octave value noise so per-eye rays diverge quickly.
func smokeDensity(p Vec3) float64 {
	const z0, z1 = 2.0, 2.25
	if p.Z < z0 || p.Z > z1 {
		return 0
	}
	n := 0.65*valueNoise(p.X*3.1, p.Y*3.1, p.Z*3.1) +
		0.35*valueNoise(p.X*7.3+11.0, p.Y*7.3, p.Z*7.3)
	edge := math.Min((p.Z-z0)/(z1-z0), (z1-p.Z)/(z1-z0)) * 4
	if edge > 1 {
		edge = 1
	}
	return 6.0 * n * edge
}

// Fog wall: a deep slab z in [4, 6] with slowly varying density.
func fogWallDensity(p Vec3) float64 {
	const z0, z1 = 4.0, 6.0
	if p.Z < z0 || p.Z > z1 {
		return 0
	}
	n := 0.7 + 0.3*valueNoise(p.X*1.2+5.0, p.Y*1.2, p.Z*1.2)
	return 0.9 * n
}

// Transparent glowing body: unit-ish sphere at (0.4, 0.1, 3.2).
func glowDensity(p Vec3) float64 {
	c := Vec3{0.4, 0.1, 3.2}
	d := p.Sub(c).Len()
	const r = 0.55
	if d >= r {
		return 0
	}
	fall := 1 - d/r
	return 1.5 * fall * fall
}

// valueNoise is deterministic trilinear value noise in [0,1], stdlib only.
func valueNoise(x, y, z float64) float64 {
	xi, yi, zi := math.Floor(x), math.Floor(y), math.Floor(z)
	xf, yf, zf := x-xi, y-yi, z-zi
	u := xf * xf * (3 - 2*xf)
	v := yf * yf * (3 - 2*yf)
	w := zf * zf * (3 - 2*zf)
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }
	c000 := hash3(xi, yi, zi)
	c100 := hash3(xi+1, yi, zi)
	c010 := hash3(xi, yi+1, zi)
	c110 := hash3(xi+1, yi+1, zi)
	c001 := hash3(xi, yi, zi+1)
	c101 := hash3(xi+1, yi, zi+1)
	c011 := hash3(xi, yi+1, zi+1)
	c111 := hash3(xi+1, yi+1, zi+1)
	return lerp(
		lerp(lerp(c000, c100, u), lerp(c010, c110, u), v),
		lerp(lerp(c001, c101, u), lerp(c011, c111, u), v), w)
}

func hash3(x, y, z float64) float64 {
	n := math.Sin(x*127.1+y*311.7+z*74.7) * 43758.5453123
	return n - math.Floor(n)
}
