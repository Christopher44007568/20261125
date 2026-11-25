package main

import (
	"math"
	"sort"
)

type Vec2 struct{ X, Y float64 }

type Audit struct {
	DenseSamples      int
	CurvatureSamples  int
	DenseRMSE         float64
	DenseMaxAbs       float64
	NormalRMSE        float64
	NormalP95         float64
	NormalMax         float64
	CurvatureRMS      float64
	CurvatureP95      float64
	CurvatureRelative float64
	RippleRMS         float64
	RippleRatio       float64
	SilhouetteIoU     float64
	SilhouetteChamfer float64
}

func angleDeg(a, b Vec3) float64 {
	d := clamp(a.Dot(b), -1, 1)
	return math.Acos(d) * 180 / math.Pi
}

func percentileSorted(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	t := pos - float64(lo)
	return sorted[lo]*(1-t) + sorted[hi]*t
}

func EvaluateFit(fit *BSplineSurface, trimA, trimB float64) Audit {
	ref := Reference{}
	var zSS, zMax float64
	var normals, means []float64
	denseCount := 0
	nx, ny := 239, 151
	for iy := 0; iy < ny; iy++ {
		y := FitYMin + (float64(iy)+0.5)*(FitYMax-FitYMin)/float64(ny)
		for ix := 0; ix < nx; ix++ {
			x := FitXMin + (float64(ix)+0.5)*(FitXMax-FitXMin)/float64(nx)
			r2 := trimRadius(x, y)
			if r2 > 0.985 {
				continue
			}
			zf := fit.Z(x, y)
			zt := ref.Z(x, y)
			d := zf - zt
			zSS += d * d
			zMax = math.Max(zMax, math.Abs(d))
			denseCount++
			if r2 <= 0.94 {
				nf, nt := fit.Normal(x, y), ref.Normal(x, y)
				normals = append(normals, angleDeg(nf, nt))
			}
		}
	}

	cnx, cny := 171, 109
	var meanSS, truthMeanSS float64
	for iy := 0; iy < cny; iy++ {
		y := FitYMin + (float64(iy)+0.5)*(FitYMax-FitYMin)/float64(cny)
		for ix := 0; ix < cnx; ix++ {
			x := FitXMin + (float64(ix)+0.5)*(FitXMax-FitXMin)/float64(cnx)
			if trimRadius(x, y) > 0.90 {
				continue
			}
			hm, _ := fit.Curvature(x, y)
			ht, _ := ref.Curvature(x, y)
			d := hm - ht
			meanSS += d * d
			truthMeanSS += ht * ht
			means = append(means, math.Abs(d))
		}
	}
	sort.Float64s(normals)
	sort.Float64s(means)

	curvCount := len(means)
	curvRMS := 0.0
	curvRel := 0.0
	if curvCount > 0 {
		curvRMS = math.Sqrt(meanSS / float64(curvCount))
		if truthMeanSS > 0 {
			curvRel = curvRMS / math.Sqrt(truthMeanSS/float64(curvCount)) * 100
		}
	}

	rippleRMS, rippleRatio := surfaceRipple(fit)
	iou, chamfer := silhouetteMetrics(fit, trimA, trimB)
	a := Audit{
		DenseSamples:      denseCount,
		CurvatureSamples:  curvCount,
		DenseRMSE:         math.Sqrt(zSS / math.Max(1, float64(denseCount))),
		DenseMaxAbs:       zMax,
		NormalRMSE:        rms(normals),
		NormalP95:         percentileSorted(normals, 0.95),
		NormalMax:         percentileSorted(normals, 1),
		CurvatureRMS:      curvRMS,
		CurvatureP95:      percentileSorted(means, 0.95),
		CurvatureRelative: curvRel,
		RippleRMS:         rippleRMS,
		RippleRatio:       rippleRatio,
		SilhouetteIoU:     iou,
		SilhouetteChamfer: chamfer,
	}
	return a
}

func rms(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range v {
		s += x * x
	}
	return math.Sqrt(s / float64(len(v)))
}

func surfaceRipple(fit *BSplineSurface) (absRMS, ratio float64) {
	const step = 4.0
	var diffSS, truthSS float64
	count := 0
	ref := Reference{}
	for iy := 1; ; iy++ {
		y := FitYMin + 4 + float64(iy)*step
		if y >= FitYMax-4 {
			break
		}
		for ix := 1; ; ix++ {
			x := FitXMin + 4 + float64(ix)*step
			if x >= FitXMax-4 {
				break
			}
			if trimRadius(x, y) > 0.94 {
				continue
			}
			lapFit := (fit.Z(x-step, y) - 2*fit.Z(x, y) + fit.Z(x+step, y)) / (step * step)
			lapFit += (fit.Z(x, y-step) - 2*fit.Z(x, y) + fit.Z(x, y+step)) / (step * step)
			lapTruth := (ref.Z(x-step, y) - 2*ref.Z(x, y) + ref.Z(x+step, y)) / (step * step)
			lapTruth += (ref.Z(x, y-step) - 2*ref.Z(x, y) + ref.Z(x, y+step)) / (step * step)
			d := lapFit - lapTruth
			diffSS += d * d
			truthSS += lapTruth * lapTruth
			count++
		}
	}
	if count == 0 {
		return 0, 0
	}
	countF := float64(count)
	absRMS = math.Sqrt(diffSS/countF) * 1000
	truthRMS := math.Sqrt(truthSS/countF) * 1000
	if truthSS > 0 {
		ratio = absRMS / truthRMS * 100
	}
	return
}

func silhouetteMetrics(fit *BSplineSurface, trimA, trimB float64) (iou, chamfer float64) {
	cam := cameraFor(-38, 24)
	const nx, ny = 201, 127
	var fitPts, truthPts []Vec2
	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	minY, maxY := math.MaxFloat64, -math.MaxFloat64
	for iy := 0; iy < ny; iy++ {
		y := FitYMin + (float64(iy)+0.5)*(FitYMax-FitYMin)/float64(ny)
		for ix := 0; ix < nx; ix++ {
			x := FitXMin + (float64(ix)+0.5)*(FitXMax-FitXMin)/float64(nx)
			if trimRadius(x, y) > 0.995 {
				continue
			}
			fx, fy := cam.ProjectMM(Vec3{x, y, fit.Z(x, y) + 0.25})
			tx, ty := cam.ProjectMM(Vec3{x, y, (Reference{}).Z(x, y) + 0.25})
			fitPts = append(fitPts, Vec2{fx, fy})
			truthPts = append(truthPts, Vec2{tx, ty})
			minX = math.Min(minX, math.Min(fx, tx))
			maxX = math.Max(maxX, math.Max(fx, tx))
			minY = math.Min(minY, math.Min(fy, ty))
			maxY = math.Max(maxY, math.Max(fy, ty))
		}
	}
	mw, mh := 300, 194
	mf := rasterMask(fitPts, minX, maxX, minY, maxY, mw, mh)
	mt := rasterMask(truthPts, minX, maxX, minY, maxY, mw, mh)
	inter, union := 0, 0
	for i := range mf {
		if mf[i] && mt[i] {
			inter++
		}
		if mf[i] || mt[i] {
			union++
		}
	}
	if union > 0 {
		iou = float64(inter) / float64(union)
	}

	const boundaryN = 720
	fitBoundary := make([]Vec2, 0, boundaryN)
	truthBoundary := make([]Vec2, 0, boundaryN)
	for i := 0; i < boundaryN; i++ {
		t := 2 * math.Pi * (float64(i) + 0.5) / boundaryN
		xf, yf := trimA*math.Cos(t), trimB*math.Sin(t)
		xt, yt := PatchA*math.Cos(t), PatchB*math.Sin(t)
		fx, fy := cam.ProjectMM(Vec3{xf, yf, fit.Z(xf, yf) + 0.25})
		tx, ty := cam.ProjectMM(Vec3{xt, yt, (Reference{}).Z(xt, yt) + 0.25})
		fitBoundary = append(fitBoundary, Vec2{fx, fy})
		truthBoundary = append(truthBoundary, Vec2{tx, ty})
	}
	chamfer = 0.5 * (directedChamfer(fitBoundary, truthBoundary) +
		directedChamfer(truthBoundary, fitBoundary))
	return
}

func rasterMask(points []Vec2, minX, maxX, minY, maxY float64, w, h int) []bool {
	m := make([]bool, w*h)
	if maxX <= minX || maxY <= minY {
		return m
	}
	for _, p := range points {
		ix := int((p.X - minX) / (maxX - minX) * float64(w))
		iy := int((p.Y - minY) / (maxY - minY) * float64(h))
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				x, y := ix+dx, iy+dy
				if x >= 0 && y >= 0 && x < w && y < h {
					m[y*w+x] = true
				}
			}
		}
	}
	return m
}

func directedChamfer(a, b []Vec2) float64 {
	sum := 0.0
	for _, p := range a {
		best := math.MaxFloat64
		for _, q := range b {
			dx, dy := p.X-q.X, p.Y-q.Y
			d := dx*dx + dy*dy
			if d < best {
				best = d
			}
		}
		sum += math.Sqrt(best)
	}
	return sum / math.Max(1, float64(len(a)))
}
