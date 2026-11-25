package main

import (
	"math"
	"math/rand"
)

// Shared scene configuration. Every shading model consumes the exact same
// geometry buffer, camera and light produced from these constants.
const (
	ImgW       = 760
	ImgH       = 760
	ScalpR     = 1.0
	FiberCount = 19000 // over-sample the sphere; the cap filter leaves ~12k
	FiberSteps = 5
	HairLength = 0.17

	OrthoHalf = 1.42

	FiberRadius = 0.0045
)

// Fiber is a polyline in world space, grown outward from the scalp.
type Fiber struct {
	Points  []Vec3
	Tangent []Vec3
	Root    Vec3
}

// Camera describes an orthographic camera.
type Camera struct {
	Eye                Vec3
	Forward, Right, Up Vec3
	Half               float64
}

// Scene bundles the shared inputs.
type Scene struct {
	Cam        Camera
	LightDir   Vec3 // unit vector pointing *toward* the light
	LightColor Vec3
	Ambient    Vec3
	Fibers     []Fiber
	Rng        *rand.Rand
}

func buildScene(seed int64) *Scene {
	rng := rand.New(rand.NewSource(seed))

	eye := v(0.0, 0.18, 3.5)
	target := v(0.0, -0.08, 0.0)
	forward := target.Sub(eye).Normalize()
	right := v(1, 0, 0).Cross(forward).Normalize()
	up := forward.Cross(right).Normalize()
	cam := Camera{Eye: eye, Forward: forward, Right: right, Up: up, Half: OrthoHalf}

	// Key light: upper front-right, giving an oblique angle to the fiber field.
	lightDir := v(-0.45, 0.78, 0.62).Normalize()
	lightColor := v(1.0, 0.96, 0.9).Scale(1.15)
	ambient := v(0.10, 0.11, 0.13)

	s := &Scene{
		Cam:        cam,
		LightDir:   lightDir,
		LightColor: lightColor,
		Ambient:    ambient,
		Rng:        rng,
	}
	s.Fibers = generateFibers(rng)
	return s
}

// generateFibers distributes roots over the upper sphere cap with a jittered
// Fibonacci lattice and integrates each strand as a slightly curling polyline.
func generateFibers(rng *rand.Rand) []Fiber {
	fibers := make([]Fiber, 0, FiberCount)
	golden := math.Pi * (3 - math.Sqrt(5))
	back := v(0, 0, -1)

	for i := 0; i < FiberCount; i++ {
		// Uniform sphere sampling, then keep the scalp cap (y > -0.22).
		z := 1 - 2*(float64(i)+0.5)/float64(FiberCount)
		rr := math.Sqrt(math.Max(0, 1-z*z))
		theta := golden * float64(i)
		dir := v(rr*math.Cos(theta), z, rr*math.Sin(theta))
		if dir.Y < -0.22 {
			continue
		}

		normal := dir.Normalize()
		// Jitter the root inside a small tangent-space disk.
		t1 := v(1, 0, 0).Cross(normal).Normalize()
		t2 := normal.Cross(t1).Normalize()
		jitter := rng.NormFloat64() * 0.018
		root := normal.Scale(ScalpR).
			Add(t1.Scale(jitter)).
			Add(t2.Scale(rng.NormFloat64() * 0.018))

		// Initial growth direction: outward plus a slight backward flow.
		tan := normal.Add(back.Scale(0.28 + rng.NormFloat64()*0.05)).Normalize()
		bend := v(0, -1, -0.35).Normalize().Scale(0.85)

		length := HairLength * (0.85 + 0.3*rng.Float64())
		ds := length / FiberSteps
		pts := make([]Vec3, FiberSteps+1)
		tans := make([]Vec3, FiberSteps+1)
		pos := root
		pts[0] = pos
		tans[0] = tan
		for k := 1; k <= FiberSteps; k++ {
			tan = tan.Add(bend.Scale(ds)).Normalize()
			pos = pos.Add(tan.Scale(ds))
			pts[k] = pos
			tans[k] = tan
		}
		fibers = append(fibers, Fiber{Points: pts, Tangent: tans, Root: root})
	}
	return fibers
}
