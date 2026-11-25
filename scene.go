package main

import (
	"math"
	"sort"
)

type SurfaceFace struct {
	Poly   []Vec3
	Normal Vec3
	Center Vec3
	Error  float64
	U, V   float64
}

type ProjectedFace struct {
	Poly  []Vec2
	Depth float64
	Face  SurfaceFace
}

func ellipseInside(p Vec2, a, b float64) bool {
	return (p.X/a)*(p.X/a)+(p.Y/b)*(p.Y/b) <= 1
}

func edgeEllipseT(p, q Vec2, a, b float64) float64 {
	lo, hi := 0.0, 1.0
	pIn := ellipseInside(p, a, b)
	for i := 0; i < 32; i++ {
		mid := (lo + hi) * 0.5
		m := Vec2{p.X + (q.X-p.X)*mid, p.Y + (q.Y-p.Y)*mid}
		if ellipseInside(m, a, b) == pIn {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) * 0.5
}

func clipToEllipse(poly []Vec2, a, b float64) []Vec2 {
	if len(poly) == 0 {
		return nil
	}
	out := make([]Vec2, 0, len(poly)+4)
	for i := range poly {
		p, q := poly[i], poly[(i+1)%len(poly)]
		pIn, qIn := ellipseInside(p, a, b), ellipseInside(q, a, b)
		if pIn {
			out = append(out, p)
		}
		if pIn != qIn {
			t := edgeEllipseT(p, q, a, b)
			out = append(out, Vec2{p.X + (q.X-p.X)*t, p.Y + (q.Y-p.Y)*t})
		}
	}
	return out
}

func buildSurfaceFaces(surface *BSplineSurface, ref Reference, trimA, trimB float64, nu, nv int, offset float64) []SurfaceFace {
	faces := make([]SurfaceFace, 0, nu*nv)
	for iv := 0; iv < nv; iv++ {
		y0 := -trimB + float64(iv)/float64(nv)*2*trimB
		y1 := -trimB + float64(iv+1)/float64(nv)*2*trimB
		for iu := 0; iu < nu; iu++ {
			x0 := -trimA + float64(iu)/float64(nu)*2*trimA
			x1 := -trimA + float64(iu+1)/float64(nu)*2*trimA
			quad := []Vec2{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
			clip := clipToEllipse(quad, trimA, trimB)
			if len(clip) < 3 {
				continue
			}
			poly := make([]Vec3, len(clip))
			cx, cy := 0.0, 0.0
			for i, p := range clip {
				poly[i] = Vec3{p.X, p.Y, surface.Z(p.X, p.Y) + offset}
				cx += p.X
				cy += p.Y
			}
			cx /= float64(len(clip))
			cy /= float64(len(clip))
			normal := surface.Normal(cx, cy)
			if offset != 0 {
				// The normal is evaluated on the underlying surface; offsetting
				// only changes position, not the face orientation.
				normal = surface.Normal(cx, cy)
			}
			errSum := 0.0
			for _, p := range poly {
				errSum += angleDeg(surface.Normal(p.X, p.Y), ref.Normal(p.X, p.Y))
			}
			err := errSum / float64(len(poly))
			faces = append(faces, SurfaceFace{
				Poly: poly, Normal: normal, Center: Vec3{cx, cy, surface.Z(cx, cy)},
				Error: err, U: float64(iu) / float64(nu), V: float64(iv) / float64(nv),
			})
		}
	}
	return faces
}

func projectFaces(cam Camera, faces []SurfaceFace) []ProjectedFace {
	out := make([]ProjectedFace, 0, len(faces))
	_, _, view := cam.Basis()
	for _, face := range faces {
		if face.Normal.Dot(view) < -0.04 {
			continue
		}
		pf := ProjectedFace{Face: face}
		depth := 0.0
		for _, p := range face.Poly {
			x, y, d := cam.Project(p)
			pf.Poly = append(pf.Poly, Vec2{x, y})
			depth += d
		}
		pf.Depth = depth / float64(len(face.Poly))
		out = append(out, pf)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Depth > out[j].Depth })
	return out
}

func trimBoundary(surface *BSplineSurface, a, b float64, n int, offset float64) []Vec3 {
	out := make([]Vec3, 0, n+1)
	for i := 0; i <= n; i++ {
		t := 2 * math.Pi * float64(i) / float64(n)
		x, y := a*math.Cos(t), b*math.Sin(t)
		out = append(out, Vec3{x, y, surface.Z(x, y) + offset})
	}
	return out
}

func paramCurves(surface *BSplineSurface, a, b float64, n int, offset float64) [][]Vec3 {
	var lines [][]Vec3
	for k := 0; k <= n; k++ {
		fixed := -1 + 2*float64(k)/float64(n)
		for axis := 0; axis < 2; axis++ {
			var seg []Vec3
			for s := 0; s <= 220; s++ {
				t := -1 + 2*float64(s)/220
				x, y := fixed*a, t*b
				if axis == 1 {
					x, y = t*a, fixed*b
				}
				if !ellipseInside(Vec2{x, y}, a, b) {
					if len(seg) > 1 {
						lines = append(lines, seg)
					}
					seg = nil
					continue
				}
				seg = append(seg, Vec3{x, y, surface.Z(x, y) + offset})
			}
			if len(seg) > 1 {
				lines = append(lines, seg)
			}
		}
	}
	return lines
}

func isoCurve(surface *BSplineSurface, a, b, fixed float64, axis, offset int, samples int) []Vec3 {
	out := make([]Vec3, 0, samples)
	for i := 0; i <= samples; i++ {
		t := -1 + 2*float64(i)/float64(samples)
		x, y := fixed*a, t*b
		if axis == 1 {
			x, y = t*a, fixed*b
		}
		if ellipseInside(Vec2{x, y}, a, b) {
			out = append(out, Vec3{x, y, surface.Z(x, y) + float64(offset)*0.01})
		}
	}
	return out
}

func projectWorld(cam Camera, points []Vec3) []Vec2 {
	out := make([]Vec2, len(points))
	for i, p := range points {
		x, y, _ := cam.Project(p)
		out[i] = Vec2{x, y}
	}
	return out
}

func pointZ(surface *BSplineSurface, x, y float64) float64 {
	return surface.Z(clamp(x, FitXMin, FitXMax), clamp(y, FitYMin, FitYMax))
}
