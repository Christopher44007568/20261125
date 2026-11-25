package mesh

import (
	"fmt"
	"math"

	"adaptivebezier/internal/geom"
)

type builder struct {
	patch     geom.Patch
	project   geom.Projector
	tol       float64
	opts      Options
	vertices  []Vertex
	triangles []Triangle
	keys      map[[2]int64]int
	stats     Stats
	trimEdges map[Edge]bool
	boundary  []int
	trimPoly  []geom.Vec2
}

func Build(patch geom.Patch, trim geom.TrimLoop, project geom.Projector, tolerance float64) (*Mesh, error) {
	return BuildWithOptions(patch, trim, project, DefaultOptions(tolerance))
}

func BuildWithOptions(patch geom.Patch, trim geom.TrimLoop, project geom.Projector, opts Options) (*Mesh, error) {
	if opts.Tolerance <= 0 {
		return nil, fmt.Errorf("tolerance must be positive")
	}
	if opts.GridN < 2 {
		return nil, fmt.Errorf("grid resolution must be at least 2")
	}
	boundaryTol := opts.Tolerance * opts.BoundaryFactor
	sampler := boundarySampler{
		patch: patch, trim: trim, project: project,
		tol: boundaryTol, maxDepth: opts.MaxDepth,
	}
	points := sampler.run()
	if len(points) < 8 {
		return nil, fmt.Errorf("trim loop produced too few boundary samples")
	}
	poly := make([]geom.Vec2, len(points))
	for i := range points {
		poly[i] = points[i].Param
	}
	area := 0.0
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		area += a.Cross(b)
	}
	if math.Abs(area) < 1e-10 {
		return nil, fmt.Errorf("trim loop has zero area")
	}
	if area < 0 {
		for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
			points[i], points[j] = points[j], points[i]
			poly[i], poly[j] = poly[j], poly[i]
		}
	}
	if !geom.PolylineIsConvex(poly, 1e-10) {
		return nil, fmt.Errorf("trim loop must be convex for the clipped-cell builder")
	}

	b := &builder{
		patch: patch, project: project, tol: opts.Tolerance, opts: opts,
		keys: make(map[[2]int64]int), trimEdges: make(map[Edge]bool),
		trimPoly: poly,
	}
	for i := range points {
		b.boundary = append(b.boundary, b.vertexFor(points[i].Param))
	}

	n := opts.GridN
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			x0, x1 := float64(i)/float64(n), float64(i+1)/float64(n)
			y0, y1 := float64(j)/float64(n), float64(j+1)/float64(n)
			cell := []geom.Vec2{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
			clipped := clipConvex(cell, poly)
			if len(clipped) < 3 {
				continue
			}
			changed := false
			for _, q := range clipped {
				found := false
				for _, c := range cell {
					if q.Sub(c).Len() < 1e-9 {
						found = true
						break
					}
				}
				if !found {
					changed = true
					break
				}
			}
			if changed {
				switch len(clipped) {
				case 3:
					b.stats.ClippedTriangles++
				case 4:
					b.stats.ClippedQuads++
				default:
					b.stats.ClippedPolygons++
				}
			}
			b.triangulatePolygon(clipped)
		}
	}
	if len(b.triangles) == 0 {
		return nil, fmt.Errorf("clipping produced no triangles")
	}
	if err := b.refine(); err != nil {
		return nil, err
	}
	m := b.finish(&sampler, points)
	return m, nil
}

func (b *builder) vertexKey(p geom.Vec2) [2]int64 {
	const q = 1e9
	return [2]int64{int64(math.Round(p.X * q)), int64(math.Round(p.Y * q))}
}

func (b *builder) vertexFor(p geom.Vec2) int {
	k := b.vertexKey(p)
	if idx, ok := b.keys[k]; ok {
		return idx
	}
	world := b.patch.Eval(p.X, p.Y)
	v := Vertex{
		Param: p, Screen: b.project.Project(world), World: world,
		Normal: b.patch.Normal(p.X, p.Y),
	}
	idx := len(b.vertices)
	b.vertices = append(b.vertices, v)
	b.keys[k] = idx
	return idx
}

func (b *builder) triangulatePolygon(poly []geom.Vec2) {
	if len(poly) < 3 {
		return
	}
	a := poly[0]
	area := 0.0
	for i := 1; i+1 < len(poly); i++ {
		area += triangleArea2(a, poly[i], poly[i+1])
	}
	if math.Abs(area) < 1e-14 {
		return
	}
	if area < 0 {
		poly = append([]geom.Vec2(nil), poly...)
		for i, j := 0, len(poly)-1; i < j; i, j = i+1, j-1 {
			poly[i], poly[j] = poly[j], poly[i]
		}
	}
	idx := make([]int, len(poly))
	for i := range poly {
		idx[i] = b.vertexFor(poly[i])
	}
	for i := range idx {
		a, c := idx[i], idx[(i+1)%len(idx)]
		if a == c {
			continue
		}
		pa, pb := b.vertices[a].Param, b.vertices[c].Param
		mid := geom.Vec2{X: (pa.X + pb.X) * 0.5, Y: (pa.Y + pb.Y) * 0.5}
		if b.onTrimBoundary(mid) {
			b.trimEdges[makeEdge(a, c)] = true
		}
	}
	centroid := geom.PolylineCentroid(poly)
	center := b.vertexFor(centroid)
	for i := range idx {
		a, c := idx[i], idx[(i+1)%len(idx)]
		if math.Abs(triangleArea2(b.vertices[center].Param, b.vertices[a].Param, b.vertices[c].Param)) < 1e-14 {
			continue
		}
		b.triangles = append(b.triangles, Triangle{A: center, B: a, C: c})
	}
}

func (b *builder) onTrimBoundary(p geom.Vec2) bool {
	for i := range b.trimPoly {
		a := b.trimPoly[i]
		c := b.trimPoly[(i+1)%len(b.trimPoly)]
		if distPointSegment(p, a, c) < 1e-7 {
			return true
		}
	}
	return false
}
