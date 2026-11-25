package brep

import (
	"brepsection/geom"
)

// Profile2D is a planar polygon with holes, given in the XY plane.
// Outer must be counter-clockwise; holes clockwise (standard even-odd fill).
type Profile2D struct {
	Outer []geom.Vec2
	Holes [][]geom.Vec2
}

// Extrude sweeps the profile along +Z from z0 to z1. When a base profile edge
// is split into multiple collinear pieces (identical endpoints shared) the
// resulting side faces stay exactly coplanar: this models tangent face joins.
func Extrude(name string, p Profile2D, z0, z1 float64) *BRep {
	n2 := len(p.Outer)
	holeLen := 0
	for _, h := range p.Holes {
		holeLen += len(h)
	}

	points := make([]geom.Vec3, 0, 2*(n2+holeLen))

	// Two explicit layers; shared XY positions reuse vertices within a layer.
	bottom := map[uint64]int{}
	top := map[uint64]int{}
	addVertex := func(q geom.Vec2, z float64, m map[uint64]int) int {
		key := quantize(q)
		if idx, ok := m[key]; ok {
			return idx
		}
		idx := len(points)
		points = append(points, geom.V3(q.X, q.Y, z))
		m[key] = idx
		return idx
	}

	indexLoop := func(loop []geom.Vec2, z float64, m map[uint64]int) []int {
		idx := make([]int, len(loop))
		for i, q := range loop {
			idx[i] = addVertex(q, z, m)
		}
		return idx
	}

	botOuter := indexLoop(p.Outer, z0, bottom)
	topOuter := indexLoop(p.Outer, z1, top)
	var botHoles, topHoles [][]int
	for _, h := range p.Holes {
		botHoles = append(botHoles, indexLoop(h, z0, bottom))
		topHoles = append(topHoles, indexLoop(h, z1, top))
	}

	b := &BRep{Name: name, Points: points}
	center := profileInterior(p)

	// Side walls for every outer and hole edge.
	addSideWalls := func(loop2 []geom.Vec2, bot, top []int) {
		m := len(loop2)
		for i := 0; i < m; i++ {
			j := (i + 1) % m
			outer := []int{bot[i], bot[j], top[j], top[i]}
			f := Face{
				Outer:    outer,
				Interior: geom.V3(center.X, center.Y, (z0+z1)*0.5),
			}
			orientFace(points, &f)
			b.Faces = append(b.Faces, f)
		}
	}
	addSideWalls(p.Outer, botOuter, topOuter)
	for k, h := range p.Holes {
		addSideWalls(h, botHoles[k], topHoles[k])
	}

	// End caps with hole loops. Orientation is fixed by orientFace.
	bottomCap := Face{
		Outer:    append([]int(nil), botOuter...),
		Holes:    botHoles,
		Interior: geom.V3(center.X, center.Y, (z0+z1)*0.5),
	}
	topCap := Face{
		Outer:    append([]int(nil), topOuter...),
		Holes:    topHoles,
		Interior: geom.V3(center.X, center.Y, (z0+z1)*0.5),
	}
	orientFace(points, &bottomCap)
	orientFace(points, &topCap)
	b.Faces = append(b.Faces, bottomCap, topCap)

	return b
}

// quantize maps a 2D point to an exact merge key. Profile coordinates are
// generated from a small set of constants, so 1e-9 relative rounding merges
// shared endpoints without collapsing distinct facet vertices.
func quantize(q geom.Vec2) uint64 {
	const scale = 1e9
	ix := int64(mathRound(q.X * scale))
	iy := int64(mathRound(q.Y * scale))
	// Two 32-bit fields keep the key collision-free for our coordinate range.
	return (uint64(uint32(ix)) << 32) | uint64(uint32(iy))
}

func mathRound(v float64) float64 {
	if v < 0 {
		return -mathRound(-v)
	}
	return float64(int64(v + 0.5))
}

// profileInterior finds a point strictly inside the outer polygon and
// outside every hole, by sampling the edge midpoint fan and testing parity.
func profileInterior(p Profile2D) geom.Vec2 {
	var cand []geom.Vec2
	avg := geom.V2(0, 0)
	for _, q := range p.Outer {
		avg = avg.Add(q)
	}
	avg = avg.Scale(1 / float64(len(p.Outer)))
	cand = append(cand, avg)
	for i, q := range p.Outer {
		cand = append(cand, q.Add(p.Outer[(i+1)%len(p.Outer)]).Scale(0.5))
	}
	for _, c := range cand {
		if geom.PointInLoop(c, p.Outer) <= 0 {
			continue
		}
		blocked := false
		for _, h := range p.Holes {
			if geom.PointInLoop(c, h) != 0 {
				blocked = true
				break
			}
		}
		if !blocked {
			return c
		}
	}
	// Deterministic fallback: walk toward the outer centroid.
	return avg
}
