package intersect

import (
	"testing"

	"ruledintersect/geom"
)

func TestTriangleUVAndRefineNearParallel(t *testing.T) {
	a := geom.NewRuledSurface("a",
		geom.CurveFunc(func(u float64) geom.Vec3 { return geom.V(0.04+0.92*u, 0.06, 0.3+0.12*u) }),
		geom.CurveFunc(func(u float64) geom.Vec3 { return geom.V(0, 0.82, 0.02) }))
	b := geom.NewRuledSurface("b",
		geom.CurveFunc(func(u float64) geom.Vec3 { return geom.V(0.04+0.92*u, 0.06, 0.305+0.122*u) }),
		geom.CurveFunc(func(u float64) geom.Vec3 { return geom.V(0, 0.82, -0.018) }))
	a.Nu, a.Nv = 8, 4
	b.Nu, b.Nv = 8, 4
	ma, mb := geom.BuildMesh(a), geom.BuildMesh(b)
	var hit rawHit
	found := false
	var foundA, foundB geom.MeshTriangle
	for _, ta := range ma.Triangles {
		for _, tb := range mb.Triangles {
			p, q, ok := triangleTriangle(ta, tb)
			if !ok {
				continue
			}
			hit = rawHit{a: p, b: q, ta: ta.ID, tb: tb.ID,
				pa0: triangleUV(ta, p), pa1: triangleUV(ta, q),
				pb0: triangleUV(tb, p), pb1: triangleUV(tb, q)}
			found = true
			foundA, foundB = ta, tb
			break
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("no raw planar hit")
	}
	t.Logf("tri A=%+v n=%+v", foundA.P, foundA.Normal())
	t.Logf("tri B=%+v n=%+v", foundB.P, foundB.Normal())
	t.Logf("hit a=%+v b=%+v pa0=%+v pb0=%+v", hit.a, hit.b, hit.pa0, hit.pb0)
	if d := a.Point(hit.pa0.U, hit.pa0.V).Dist(hit.a); d > 1e-9 {
		t.Fatalf("bad triangle UV map: %g", d)
	}
	q, gap, ok := refineIntersection(a, b, hit.a, hit.pa0, hit.pb0, DefaultOptions())
	t.Logf("refined q=%+v gap=%g ok=%v", q, gap, ok)
	if !ok || gap > 1e-8 {
		t.Fatal("refinement failed on exact planar seed")
	}
}
