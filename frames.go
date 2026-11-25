package main

import (
	"fmt"
	"math"
)

func rotateAround(v, axis Vec3, angle float64) Vec3 {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return v.Scale(c).Add(axis.Cross(v).Scale(s)).Add(axis.Scale(axis.Dot(v) * (1 - c)))
}

func tangentAngle(v, e1, e2 Vec3) float64 {
	return angleAtan2(v.Dot(e2), v.Dot(e1))
}

func unwrapFrameAngles(stations []Station) []float64 {
	angles := make([]float64, len(stations))
	for i := range stations {
		nextIndex := (i + 1) % len(stations)
		raw := tangentAngle(stations[nextIndex].E1, stations[i].E1, stations[i].E2)
		delta := raw
		if i > 0 {
			delta -= tangentAngle(stations[i].E1, stations[i-1].E1, stations[i-1].E2)
			for delta > math.Pi {
				delta -= tau
			}
			for delta < -math.Pi {
				delta += tau
			}
		}
		angles[i] = delta
	}
	return angles
}

func closedFrameTwist(stations []Station) float64 {
	total := 0.0
	for i := range stations {
		current := stations[i]
		next := stations[(i+1)%len(stations)]
		total += current.Tangent.Dot(current.E1.Cross(next.E1))
	}
	return total
}

func buildFrenet(base []Station) []Station {
	result := cloneStations(base)
	for i := range result {
		first := trefoilDerivative(result[i].Param)
		second := trefoilSecond(result[i].Param)
		binormal := first.Cross(second).Normalized()
		normal := binormal.Cross(result[i].Tangent).Normalized()
		result[i].Normal = normal
		result[i].Binormal = binormal
		result[i].E1 = normal
		result[i].E2 = binormal
	}
	return result
}

func buildParallelTransport(base []Station) ([]Station, float64, float64) {
	n := len(base)
	result := cloneStations(base)
	first := trefoilDerivative(base[0].Param)
	second := trefoilSecond(base[0].Param)
	e2 := first.Cross(second).Normalized()
	e1 := e2.Cross(base[0].Tangent).Normalized()
	result[0].E1 = e1
	result[0].E2 = e2
	result[0].Normal = e1
	result[0].Binormal = e2

	raw1 := make([]Vec3, n+1)
	raw2 := make([]Vec3, n+1)
	raw1[0], raw2[0] = e1, e2

	for i := 1; i <= n; i++ {
		prevT := base[i-1].Tangent
		nextT := base[0].Tangent
		if i < n {
			nextT = base[i].Tangent
		}
		axis := prevT.Cross(nextT)
		turn := axis.Norm()
		if turn > 1e-10 {
			axis = axis.Scale(1 / turn)
			angle := math.Asin(math.Min(1, turn))
			dot := clamp01((1 + prevT.Dot(nextT)) / 2)
			angle = 2 * math.Acos(math.Sqrt(dot))
			e1 = rotateAround(e1, axis, angle)
			e2 = rotateAround(e2, axis, angle)
		}
		raw1[i], raw2[i] = e1, e2
	}

	holonomy := wrapAngle(tangentAngle(raw1[n], raw1[0], raw2[0]))
	if holonomy > math.Pi {
		holonomy -= tau
	}

	for i := 0; i < n; i++ {
		correction := -holonomy * float64(i) / float64(n)
		result[i].E1 = rotateAround(raw1[i], base[i].Tangent, correction)
		result[i].E2 = rotateAround(raw2[i], base[i].Tangent, correction)
		result[i].Normal = result[i].E1
		result[i].Binormal = result[i].E2
	}
	correctedEnd1 := rotateAround(raw1[n], base[0].Tangent, -holonomy)
	correctedEnd2 := rotateAround(raw2[n], base[0].Tangent, -holonomy)
	closureResidual := math.Max(correctedEnd1.Sub(raw1[0]).Norm(), correctedEnd2.Sub(raw2[0]).Norm())
	return result, holonomy, closureResidual
}

func buildFixedFrame(base []Station) ([]Station, Vec3, float64) {
	bestRef := Vec3{0, 0, 1}
	bestMargin := -1.0
	const grid = 180
	for i := 0; i <= grid; i++ {
		for j := 1; j < grid; j++ {
			phi := tau * float64(i) / grid
			theta := math.Pi * float64(j) / grid
			ref := Vec3{
				math.Sin(theta) * math.Cos(phi),
				math.Sin(theta) * math.Sin(phi),
				math.Cos(theta),
			}
			margin := 1.0
			for _, s := range base {
				margin = math.Min(margin, math.Abs(s.Tangent.Dot(ref)))
			}
			if margin > bestMargin {
				bestMargin = margin
				bestRef = ref
			}
		}
	}
	if bestMargin < 1e-4 {
		panic(fmt.Sprintf("fixed reference is too close to a tangent: %.5f", bestMargin))
	}

	result := cloneStations(base)
	t0 := base[0].Tangent
	initial1 := bestRef.Sub(t0.Scale(bestRef.Dot(t0))).Normalized()
	initial2 := t0.Cross(initial1).Normalized()
	phase := tangentAngle(base[0].Normal, initial1, initial2)

	for i := range result {
		t := base[i].Tangent
		e1 := bestRef.Sub(t.Scale(bestRef.Dot(t))).Normalized()
		e2 := t.Cross(e1).Normalized()
		e1 = rotateAround(e1, t, phase)
		e2 = rotateAround(e2, t, phase)
		result[i].E1 = e1
		result[i].E2 = e2
		result[i].Normal = e1
		result[i].Binormal = e2
	}
	return result, bestRef, bestMargin
}

func cloneStations(base []Station) []Station {
	result := make([]Station, len(base))
	copy(result, base)
	return result
}
