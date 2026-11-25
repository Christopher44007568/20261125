package mesh

import (
	"math"
	"sort"

	"adaptivebezier/internal/geom"
)

func clipConvex(subject, clip []geom.Vec2) []geom.Vec2 {
	if len(subject) < 3 || len(clip) < 3 {
		return nil
	}
	const eps = 1e-9
	candidates := make([]geom.Vec2, 0, len(subject)+len(clip)+8)
	for _, p := range subject {
		if pointInConvex(p, clip, eps) {
			candidates = append(candidates, p)
		}
	}
	for _, p := range clip {
		if pointInConvex(p, subject, eps) {
			candidates = append(candidates, p)
		}
	}
	for i := range subject {
		a := subject[i]
		b := subject[(i+1)%len(subject)]
		for j := range clip {
			c := clip[j]
			d := clip[(j+1)%len(clip)]
			if p, ok := segmentIntersect(a, b, c, d, eps); ok {
				candidates = append(candidates, p)
			}
		}
	}
	candidates = dedupeClosed(candidates, 1e-9)
	if len(candidates) < 3 {
		return nil
	}
	var center geom.Vec2
	for _, p := range candidates {
		center = center.Add(p)
	}
	center = center.Mul(1 / float64(len(candidates)))
	sort.Slice(candidates, func(i, j int) bool {
		ai := math.Atan2(candidates[i].Y-center.Y, candidates[i].X-center.X)
		aj := math.Atan2(candidates[j].Y-center.Y, candidates[j].X-center.X)
		if math.Abs(ai-aj) < 1e-14 {
			return candidates[i].Sub(center).Len() < candidates[j].Sub(center).Len()
		}
		return ai < aj
	})
	return dedupeClosed(candidates, 1e-9)
}

func pointInConvex(p geom.Vec2, poly []geom.Vec2, eps float64) bool {
	sign := 1.0
	area := 0.0
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		area += a.Cross(b)
	}
	if area < 0 {
		sign = -1
	}
	for i := range poly {
		a := poly[i]
		b := poly[(i+1)%len(poly)]
		cross := sign * b.Sub(a).Cross(p.Sub(a))
		if cross < -eps*max(1, p.Sub(a).Len()) {
			return false
		}
	}
	return true
}

func segmentIntersect(p, q, a, b geom.Vec2, eps float64) (geom.Vec2, bool) {
	r := q.Sub(p)
	s := b.Sub(a)
	den := r.Cross(s)
	if math.Abs(den) < 1e-15 {
		return geom.Vec2{}, false
	}
	t := a.Sub(p).Cross(s) / den
	u := a.Sub(p).Cross(r) / den
	if t < -eps || t > 1+eps || u < -eps || u > 1+eps {
		return geom.Vec2{}, false
	}
	return p.Add(r.Mul(geom.Clamp(t, 0, 1))), true
}

func dedupeClosed(poly []geom.Vec2, eps float64) []geom.Vec2 {
	if len(poly) < 2 {
		return poly
	}
	out := make([]geom.Vec2, 0, len(poly))
	for _, p := range poly {
		if len(out) == 0 || p.Sub(out[len(out)-1]).Len() > eps {
			out = append(out, p)
		}
	}
	for len(out) > 1 && out[0].Sub(out[len(out)-1]).Len() <= eps {
		out = out[:len(out)-1]
	}
	return out
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
