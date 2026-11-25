package main

import "math"

const (
	tau         = math.Pi * 2
	sampleCount = 520
	semiMajor   = 0.34
	semiMinor   = 0.17
)

func trefoilAt(t float64) Vec3 {
	s := math.Sin(t)
	c := math.Cos(t)
	return Vec3{
		X: s + 2*math.Sin(2*t),
		Y: c - 2*math.Cos(2*t),
		Z: -math.Sin(3 * t),
	}
}

func trefoilDerivative(t float64) Vec3 {
	s := math.Sin(t)
	c := math.Cos(t)
	return Vec3{
		X: c + 4*math.Cos(2*t),
		Y: -s + 4*math.Sin(2*t),
		Z: -3 * math.Cos(3*t),
	}
}

func trefoilSecond(t float64) Vec3 {
	s := math.Sin(t)
	c := math.Cos(t)
	return Vec3{
		X: -s - 8*math.Sin(2*t),
		Y: -c + 8*math.Cos(2*t),
		Z: 9 * math.Sin(3*t),
	}
}

func trefoilThird(t float64) Vec3 {
	s := math.Sin(t)
	c := math.Cos(t)
	return Vec3{
		X: -c - 16*math.Cos(2*t),
		Y: s - 16*math.Sin(2*t),
		Z: 27 * math.Cos(3*t),
	}
}

func sampleTrefoil(count int) ([]Station, float64) {
	const fine = 20000
	lengths := make([]float64, fine+1)
	for i := 1; i <= fine; i++ {
		t0 := tau * float64(i-1) / fine
		t1 := tau * float64(i) / fine
		lengths[i] = lengths[i-1] + trefoilAt(t1).Sub(trefoilAt(t0)).Norm()
	}
	total := lengths[fine]
	stations := make([]Station, count)

	index := 0
	for i := 0; i < count; i++ {
		target := total * float64(i) / float64(count)
		for index < fine && lengths[index+1] < target {
			index++
		}
		span := lengths[index+1] - lengths[index]
		u := 0.0
		if span > 0 {
			u = (target - lengths[index]) / span
		}
		t := tau * (float64(index) + u) / fine
		first := trefoilDerivative(t)
		speed := first.Norm()
		tangent := first.Normalized()
		second := trefoilSecond(t)
		third := trefoilThird(t)
		cross := first.Cross(second)
		curvature := cross.Norm() / (speed * speed * speed)
		torsion := cross.Dot(third) / cross.Dot(cross)

		stations[i] = Station{
			Param:     t,
			Point:     trefoilAt(t),
			Tangent:   tangent,
			Arc:       target,
			Curvature: curvature,
			Torsion:   torsion,
		}
	}
	return stations, total
}

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func angleAtan2(y, x float64) float64 {
	a := math.Atan2(y, x)
	if a < 0 {
		a += tau
	}
	return a
}

func wrapAngle(v float64) float64 {
	v = math.Mod(v, tau)
	if v < 0 {
		v += tau
	}
	return v
}
