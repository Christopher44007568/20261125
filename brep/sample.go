package brep

import (
	"math"

	"brepsection/geom"
)

// SampleSolid builds the shared BRep used by every cutting direction.
//
// The profile deliberately contains the pathological cases named in the
// specification:
//   - a sharp wedge corner and a triangular through-hole (sharp corners);
//   - a faceted round bump whose first facet joins a straight wall in the
//     tangent direction, plus a straight edge split into two exactly
//     collinear pieces (coplanar / tangent face pair);
//   - a 40-facet circular through-hole (the oblique-cut via case).
func SampleSolid() *BRep {
	const (
		arcFacets   = 12
		circleSteps = 40
	)

	outer := []geom.Vec2{
		geom.V2(0, 0),
		geom.V2(5, 0), // split point: two coplanar side faces after extrusion
		geom.V2(10, 0),
		geom.V2(10, 3),
		geom.V2(8, 5), // wedge corner
		geom.V2(3.5, 5),
		geom.V2(3.5, 6.5),
	}
	// Upper semicircular bump, center (2.5,6.5), radius 1. Its start tangent
	// is vertical, matching the straight wall above.
	for k := 1; k <= arcFacets; k++ {
		theta := math.Pi * float64(k) / float64(arcFacets)
		outer = append(outer, geom.V2(2.5+math.Cos(theta), 6.5+math.Sin(theta)))
	}
	outer = append(outer,
		geom.V2(1.5, 5),
		geom.V2(0, 5),
	)

	// Circular through-hole, clockwise winding so parity fill treats it as a
	// hole. Vertex 20 lands exactly on x = 5 for the tangent-plane test.
	circle := make([]geom.Vec2, circleSteps)
	for i := 0; i < circleSteps; i++ {
		theta := -2 * math.Pi * float64(i) / float64(circleSteps)
		circle[i] = geom.V2(6+math.Cos(theta), 2.2+math.Sin(theta))
	}

	// Triangular through-hole with sharp corners, clockwise.
	triangle := []geom.Vec2{
		geom.V2(2, 1.5),
		geom.V2(2.6, 3.2),
		geom.V2(3.5, 1.8),
	}

	return Extrude("demo-block", Profile2D{
		Outer: outer,
		Holes: [][]geom.Vec2{circle, triangle},
	}, 0, 2.5)
}
