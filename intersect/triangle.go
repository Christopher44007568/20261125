package intersect

import (
	"math"

	"ruledintersect/geom"
)

const planeEpsilon = 2e-9

// triangleTriangle returns the non-zero overlap of two triangles. It clips the
// intersection of each triangle with the other plane and projects both clips
// onto the common plane-intersection line.
func triangleTriangle(a, b geom.MeshTriangle) (geom.Vec3, geom.Vec3, bool) {
	na, nb := a.Normal(), b.Normal()
	if na.Len2() < 0.5 || nb.Len2() < 0.5 {
		return geom.Vec3{}, geom.Vec3{}, false
	}
	da, db := signedDistances(a, nb, b.P[0]), signedDistances(b, na, a.P[0])
	if !straddlesPlane(da) || !straddlesPlane(db) {
		return geom.Vec3{}, geom.Vec3{}, false
	}
	dir := na.Cross(nb)
	dirLen := dir.Len()
	if dirLen <= 1e-8 {
		return coplanarIntersection(a, b, na, nb, da, db)
	}
	sa, oka := planeCut(a, da)
	sb, okb := planeCut(b, db)
	if !oka || !okb {
		return geom.Vec3{}, geom.Vec3{}, false
	}

	dir = dir.Div(dirLen)
	ca, cb := na.Dot(a.P[0]), nb.Dot(b.P[0])
	origin := nb.Cross(dir).Mul(ca).Add(dir.Cross(na).Mul(cb)).Div(dirLen)

	a0, a1 := projectRange(sa, origin, dir)
	b0, b1 := projectRange(sb, origin, dir)
	lo, hi := math.Max(a0, b0), math.Min(a1, b1)
	if hi-lo <= planeEpsilon {
		return geom.Vec3{}, geom.Vec3{}, false
	}
	return origin.Add(dir.Mul(lo)), origin.Add(dir.Mul(hi)), true
}

func signedDistances(t geom.MeshTriangle, n, origin geom.Vec3) [3]float64 {
	c := n.Dot(origin)
	return [3]float64{
		n.Dot(t.P[0]) - c,
		n.Dot(t.P[1]) - c,
		n.Dot(t.P[2]) - c,
	}
}

func straddlesPlane(d [3]float64) bool {
	pos, neg := false, false
	for _, x := range d {
		pos = pos || x > planeEpsilon
		neg = neg || x < -planeEpsilon
	}
	return pos && neg || abs(d[0]) <= planeEpsilon ||
		abs(d[1]) <= planeEpsilon || abs(d[2]) <= planeEpsilon
}

func planeCut(t geom.MeshTriangle, d [3]float64) ([2]geom.Vec3, bool) {
	pts := make([]geom.Vec3, 0, 3)
	for i := 0; i < 3; i++ {
		if math.Abs(d[i]) <= planeEpsilon {
			pts = append(pts, t.P[i])
		}
		j := (i + 1) % 3
		if (d[i] < -planeEpsilon && d[j] > planeEpsilon) ||
			(d[i] > planeEpsilon && d[j] < -planeEpsilon) {
			pts = append(pts, t.P[i].Lerp(t.P[j], d[i]/(d[i]-d[j])))
		}
	}
	pts = dedupPoints(pts, planeEpsilon*4)
	if len(pts) < 2 {
		return [2]geom.Vec3{}, false
	}
	best, bestD := [2]geom.Vec3{pts[0], pts[1]}, pts[0].Dist(pts[1])
	for i := 0; i < len(pts); i++ {
		for j := i + 1; j < len(pts); j++ {
			if dist := pts[i].Dist(pts[j]); dist > bestD {
				best, bestD = [2]geom.Vec3{pts[i], pts[j]}, dist
			}
		}
	}
	return best, bestD > planeEpsilon
}

func projectRange(seg [2]geom.Vec3, origin, dir geom.Vec3) (float64, float64) {
	t0 := seg[0].Sub(origin).Dot(dir)
	t1 := seg[1].Sub(origin).Dot(dir)
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	return t0, t1
}

func dedupPoints(pts []geom.Vec3, tol float64) []geom.Vec3 {
	out := make([]geom.Vec3, 0, len(pts))
	for _, p := range pts {
		seen := false
		for _, q := range out {
			if p.Dist(q) <= tol {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, p)
		}
	}
	return out
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// coplanarIntersection handles the rare case of overlapping triangles. The
// largest shared edge is sufficient for the rendering/stitching stage.
func coplanarIntersection(a, b geom.MeshTriangle, na, nb geom.Vec3, da, db [3]float64) (geom.Vec3, geom.Vec3, bool) {
	for _, x := range da {
		if math.Abs(x) > planeEpsilon*10 {
			return geom.Vec3{}, geom.Vec3{}, false
		}
	}
	for _, x := range db {
		if math.Abs(x) > planeEpsilon*10 {
			return geom.Vec3{}, geom.Vec3{}, false
		}
	}
	var pts []geom.Vec3
	for i := 0; i < 3; i++ {
		if pointInTriangle(a.P[i], b, planeEpsilon*20) {
			pts = append(pts, a.P[i])
		}
		if pointInTriangle(b.P[i], a, planeEpsilon*20) {
			pts = append(pts, b.P[i])
		}
	}
	if len(pts) < 2 {
		return geom.Vec3{}, geom.Vec3{}, false
	}
	p0, p1 := pts[0], pts[1]
	for i := 0; i < len(pts); i++ {
		for j := i + 1; j < len(pts); j++ {
			if pts[i].Dist(pts[j]) > p0.Dist(p1) {
				p0, p1 = pts[i], pts[j]
			}
		}
	}
	if p0.Dist(p1) <= planeEpsilon {
		return geom.Vec3{}, geom.Vec3{}, false
	}
	return p0, p1, true
}

func pointInTriangle(p geom.Vec3, t geom.MeshTriangle, eps float64) bool {
	v0, v1, v2 := t.P[1].Sub(t.P[0]), t.P[2].Sub(t.P[0]), p.Sub(t.P[0])
	d00, d01 := v0.Dot(v0), v0.Dot(v1)
	d11, d20, d21 := v1.Dot(v1), v2.Dot(v0), v2.Dot(v1)
	den := d00*d11 - d01*d01
	if math.Abs(den) < 1e-20 {
		return false
	}
	v := (d11*d20 - d01*d21) / den
	w := (d00*d21 - d01*d20) / den
	u := 1 - v - w
	areaScale := math.Max(1, math.Sqrt(den))
	e := eps / areaScale
	return u >= -e && v >= -e && w >= -e
}

func triangleUV(t geom.MeshTriangle, p geom.Vec3) geom.UV {
	q := closestOnTriangle(p, t)
	uv := geom.UV{}
	for i := 0; i < 3; i++ {
		uv.U += q.w[i] * t.UV[i].U
		uv.V += q.w[i] * t.UV[i].V
	}
	return uv
}

type barycentric struct {
	w [3]float64
	p geom.Vec3
}

func closestOnTriangle(p geom.Vec3, t geom.MeshTriangle) barycentric {
	// Ericson, Real-Time Collision Detection, closest point on triangle.
	a, b, c := t.P[0], t.P[1], t.P[2]
	ab, ac, ap := b.Sub(a), c.Sub(a), p.Sub(a)
	d1, d2 := ab.Dot(ap), ac.Dot(ap)
	if d1 <= 0 && d2 <= 0 {
		return barycentric{w: [3]float64{1, 0, 0}, p: a}
	}
	bp := p.Sub(b)
	d3, d4 := ab.Dot(bp), ac.Dot(bp)
	if d3 >= 0 && d4 <= d3 {
		return barycentric{w: [3]float64{0, 1, 0}, p: b}
	}
	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		v := d1 / (d1 - d3)
		return barycentric{w: [3]float64{1 - v, v, 0}, p: a.Add(ab.Mul(v))}
	}
	cp := p.Sub(c)
	d5, d6 := ab.Dot(cp), ac.Dot(cp)
	if d6 >= 0 && d5 <= d6 {
		return barycentric{w: [3]float64{0, 0, 1}, p: c}
	}
	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		w := d2 / (d2 - d6)
		return barycentric{w: [3]float64{1 - w, 0, w}, p: a.Add(ac.Mul(w))}
	}
	va := d3*d6 - d5*d4
	if va <= 0 && d4-d3 >= 0 && d5-d6 >= 0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return barycentric{w: [3]float64{0, 1 - w, w}, p: b.Add(c.Sub(b).Mul(w))}
	}
	den := 1 / (va + vb + vc)
	v, w := vb*den, vc*den
	u := 1 - v - w
	return barycentric{w: [3]float64{u, v, w}, p: a.Add(ab.Mul(v)).Add(ac.Mul(w))}
}
