package intersect

import (
	"math"

	"ruledintersect/geom"
)

type rawHit struct {
	a, b     geom.Vec3
	ta, tb   int
	pa0, pa1 geom.UV
	pb0, pb1 geom.UV
}

// Surfaces computes the intersection of two finite ruled strips. The returned
// Segments include duplicates; Valid excludes duplicates and Curves contains
// the endpoint-welded continuous chains.
func Surfaces(s1, s2 *geom.RuledSurface, opts Options) Result {
	if opts.WeldTolerance <= 0 {
		opts = DefaultOptions()
	}
	m1, m2 := geom.BuildMesh(s1), geom.BuildMesh(s2)
	b1, b2 := buildBVH(m1), buildBVH(m2)
	var raw []rawHit
	b1.pairs(b2, func(i, j int) {
		t1, t2 := m1.Triangles[i], m2.Triangles[j]
		p, q, ok := triangleTriangle(t1, t2)
		if !ok {
			return
		}
		raw = append(raw, rawHit{
			a: p, b: q, ta: t1.ID, tb: t2.ID,
			pa0: triangleUV(t1, p), pa1: triangleUV(t1, q),
			pb0: triangleUV(t2, p), pb1: triangleUV(t2, q),
		})
	})

	res := Result{RawCount: len(raw)}
	for _, hit := range raw {
		a, ga, oka := refineIntersection(s1, s2, hit.a, hit.pa0, hit.pb0, opts)
		b, gb, okb := refineIntersection(s1, s2, hit.b, hit.pa1, hit.pb1, opts)
		if !oka || !okb {
			res.RejectedCount++
			res.RefineFailures++
			continue
		}
		gapA, gapB := surfaceGap(s1, s2, a), surfaceGap(s1, s2, b)
		res.MaxSurfaceGap = math.Max(res.MaxSurfaceGap, math.Max(gapA, gapB))
		if math.Max(gapA, gapB) > opts.OnSurfaceTolerance ||
			math.Max(ga, gb) > opts.OnSurfaceTolerance*2 {
			res.RejectedCount++
			res.GapRejections++
			continue
		}
		if a.Dist(b) <= opts.MinSegmentLength {
			res.ShortRejections++
			continue
		}
		res.Segments = append(res.Segments, Segment{
			A: a, B: b, TriangleA: hit.ta, TriangleB: hit.tb,
		})
	}
	res.Segments = classifyDuplicates(res.Segments, opts.DuplicateTolerance)
	for _, s := range res.Segments {
		if s.Status == Duplicate {
			res.Duplicates = append(res.Duplicates, s)
		} else {
			res.Valid = append(res.Valid, s)
		}
	}
	res.Curves = stitchSegments(res.Valid, opts.WeldTolerance)
	res.DuplicateCurves = stitchSegments(res.Duplicates, opts.WeldTolerance)
	return res
}

func surfaceGap(s1, s2 *geom.RuledSurface, p geom.Vec3) float64 {
	return math.Max(s1.Distance(p), s2.Distance(p))
}
