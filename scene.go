package main

import (
	"math"
	"math/rand"
)

type Skin struct {
	Center Vec3
	Radii  Vec3
	Albedo Vec3
}

type StrandPoint struct {
	P      Vec3
	Radius float64
}

type GuideStrand struct {
	ID     int
	Points []StrandPoint
	Albedo Vec3
	Tint   Vec3
}

type Segment struct {
	A, B      Vec3
	RA, RB    float64
	Tangent0  Vec3
	Tangent1  Vec3
	Tangent   Vec3
	StrandID  int
	SegmentID int
	Tint      Vec3
	Albedo    Vec3
}

type Scene struct {
	Width    int
	Height   int
	Skin     Skin
	Strands  []GuideStrand
	Segments []Segment

	LightDir      Vec3
	LightColor    Vec3
	KeyPower      float64
	AmbientSky    Vec3
	AmbientGround Vec3
}

func NewProceduralScene(width, height int, seed int64) *Scene {
	rng := rand.New(rand.NewSource(seed))
	scene := &Scene{
		Width:  width,
		Height: height,
		Skin: Skin{
			Center: Vec(0, -0.34, -0.18),
			Radii:  Vec(1.42, 1.62, 1.05),
			Albedo: srgbToLinear(Vec(0.96, 0.67, 0.52)),
		},
		LightDir:      Vec(-0.48, 0.78, 0.40).Normalized(),
		LightColor:    srgbToLinear(Vec(1.0, 0.91, 0.78)).Mul(2.35),
		KeyPower:      1.0,
		AmbientSky:    srgbToLinear(Vec(0.58, 0.68, 0.86)).Mul(0.42),
		AmbientGround: srgbToLinear(Vec(0.55, 0.34, 0.25)).Mul(0.30),
	}

	const guideCount = 64
	for i := 0; i < guideCount; i++ {
		u := float64(i) / float64(guideCount-1)
		x0 := lerp(-1.12, 1.12, u) + (rng.Float64()-0.5)*0.050
		edge := math.Abs(x0) / 1.12

		// A few long side guides frame the forehead. Central guides end in
		// uneven bang tips, leaving real skin gaps instead of a solid card.
		tipY := lerp(0.48, 0.16, rng.Float64())
		if edge > 0.72 {
			tipY = lerp(0.20, -0.12, rng.Float64())
		}
		if i%7 == 3 {
			tipY += 0.16
		}

		sweep := 0.10 + 0.16*rng.Float64()
		if rng.Intn(2) == 0 {
			sweep *= -1
		}
		xEnd := x0*(0.78+0.12*rng.Float64()) + sweep*(0.35+0.65*edge)
		layerZ := (rng.Float64() - 0.42) * 0.16
		if i%6 == 0 {
			layerZ += 0.08
		}

		root := Vec(x0, 1.27+rng.Float64()*0.17, 0.20+rng.Float64()*0.13+layerZ*0.25)
		control1 := Vec(x0+sweep*0.22, 1.20-rng.Float64()*0.08, 0.52+rng.Float64()*0.10+layerZ)
		control2 := Vec(xEnd-sweep*0.18, tipY+0.40+rng.Float64()*0.16, 0.88+rng.Float64()*0.08+layerZ)
		tip := Vec(xEnd, tipY, 0.91+rng.Float64()*0.08+layerZ)

		rootRadius := 0.015 + rng.Float64()*0.007
		if i%5 == 0 {
			rootRadius *= 1.28
		}
		strand := GuideStrand{
			ID:     i,
			Albedo: hairAlbedo(rng),
			Tint:   srgbToLinear(Vec(0.70, 0.43+rng.Float64()*0.08, 0.27)).Mul(0.82),
		}

		const pointCount = 15
		for j := 0; j < pointCount; j++ {
			t := float64(j) / float64(pointCount-1)
			p := cubicBezier(root, control1, control2, tip, t)
			// Tiny deterministic separation stops guides from reading as clones.
			p.X += math.Sin(t*math.Pi*2+float64(i)*1.71) * 0.010 * t
			radius := rootRadius * (1 - 0.78*smoothstep(0.58, 1.0, t))
			strand.Points = append(strand.Points, StrandPoint{P: p, Radius: radius})
		}
		scene.Strands = append(scene.Strands, strand)
	}

	for _, strand := range scene.Strands {
		for j := 0; j < len(strand.Points)-1; j++ {
			a := strand.Points[j]
			b := strand.Points[j+1]
			tangent0 := pointTangent(strand.Points, j)
			tangent1 := pointTangent(strand.Points, j+1)
			tangent := b.P.Sub(a.P).Normalized()
			scene.Segments = append(scene.Segments, Segment{
				A:         a.P,
				B:         b.P,
				RA:        a.Radius,
				RB:        b.Radius,
				Tangent0:  tangent0,
				Tangent1:  tangent1,
				Tangent:   tangent,
				StrandID:  strand.ID,
				SegmentID: j,
				Albedo:    strand.Albedo,
				Tint:      strand.Tint,
			})
		}
	}

	return scene
}

func pointTangent(points []StrandPoint, index int) Vec3 {
	if len(points) < 2 {
		return Vec(0, -1, 0)
	}
	if index == 0 {
		return points[1].P.Sub(points[0].P).Normalized()
	}
	if index == len(points)-1 {
		return points[index].P.Sub(points[index-1].P).Normalized()
	}
	return points[index+1].P.Sub(points[index-1].P).Normalized()
}

func hairAlbedo(rng *rand.Rand) Vec3 {
	base := Vec(0.075+rng.Float64()*0.035, 0.035+rng.Float64()*0.018, 0.018+rng.Float64()*0.012)
	if rng.Intn(7) == 0 {
		base = Vec(0.16, 0.075, 0.032)
	}
	return srgbToLinear(base)
}

func cubicBezier(p0, p1, p2, p3 Vec3, t float64) Vec3 {
	u := 1 - t
	a := u * u * u
	b := 3 * u * u * t
	c := 3 * u * t * t
	d := t * t * t
	return Vec3{
		X: a*p0.X + b*p1.X + c*p2.X + d*p3.X,
		Y: a*p0.Y + b*p1.Y + c*p2.Y + d*p3.Y,
		Z: a*p0.Z + b*p1.Z + c*p2.Z + d*p3.Z,
	}
}

// Intersect checks a camera ray against the ellipsoid used for the forehead.
func (s Skin) Intersect(origin, direction Vec3) (float64, Vec3, bool) {
	oo := origin.Sub(s.Center)
	scaledO := Vec(oo.X/s.Radii.X, oo.Y/s.Radii.Y, oo.Z/s.Radii.Z)
	scaledD := Vec(direction.X/s.Radii.X, direction.Y/s.Radii.Y, direction.Z/s.Radii.Z)

	a := scaledD.LenSqr()
	b := 2 * scaledO.Dot(scaledD)
	c := scaledO.LenSqr() - 1
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, Vec3{}, false
	}
	root := math.Sqrt(disc)
	t := (-b - root) / (2 * a)
	if t < 0.001 {
		t = (-b + root) / (2 * a)
	}
	if t < 0.001 {
		return 0, Vec3{}, false
	}
	p := origin.Add(direction.Mul(t))
	n := Vec(
		(p.X-s.Center.X)/s.Radii.X,
		(p.Y-s.Center.Y)/s.Radii.Y,
		(p.Z-s.Center.Z)/s.Radii.Z,
	).Normalized()
	return t, n, true
}
