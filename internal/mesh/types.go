package mesh

import "adaptivebezier/internal/geom"

type Edge struct {
	A, B int
}

type Triangle struct {
	A, B, C int
}

type Vertex struct {
	Param  geom.Vec2
	Screen geom.Vec2
	World  geom.Vec3
	Normal geom.Vec3
}

type Stats struct {
	ClippedTriangles int
	ClippedQuads     int
	ClippedPolygons  int
	Iterations       int
}

type Mesh struct {
	Vertices   []Vertex
	Triangles  []Triangle
	Boundary   []int
	TrimEdges  map[Edge]bool
	MaxChord   float64
	Contour    float64
	Crack      float64
	CrackEdges int
	Watertight bool
	Stats      Stats
}

type Options struct {
	Tolerance      float64
	BoundaryFactor float64
	GridN          int
	MaxDepth       int
	MaxIterations  int
	ErrorSampleN   int
	AuditSampleN   int
}

func DefaultOptions(tolerance float64) Options {
	return Options{
		Tolerance:      tolerance,
		BoundaryFactor: 0.50,
		GridN:          12,
		MaxDepth:       12,
		MaxIterations:  32,
		ErrorSampleN:   6,
		AuditSampleN:   12,
	}
}

func edgeOf(t Triangle, i int) (int, int) {
	switch i {
	case 0:
		return t.A, t.B
	case 1:
		return t.B, t.C
	default:
		return t.C, t.A
	}
}

func makeEdge(a, b int) Edge {
	if a > b {
		a, b = b, a
	}
	return Edge{A: a, B: b}
}

func distPointSegment(p, a, b geom.Vec2) float64 {
	ab := b.Sub(a)
	l2 := ab.Dot(ab)
	if l2 < 1e-20 {
		return p.Sub(a).Len()
	}
	t := geom.Clamp(p.Sub(a).Dot(ab)/l2, 0, 1)
	return p.Sub(a.Add(ab.Mul(t))).Len()
}

func triangleArea2(a, b, c geom.Vec2) float64 {
	return b.Sub(a).Cross(c.Sub(a))
}
