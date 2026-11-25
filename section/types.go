// Package section extracts planar section curves from a faceted BRep,
// stitches them into closed contours, and validates closure, nesting order,
// winding consistency and self-intersection behavior.
package section

import "brepsection/geom"

// Mode selects the arithmetic precision used for one full pipeline run.
type Mode struct {
	Name  string
	Use32 bool
}

// StandardModes are the two compared pipelines: native double precision and
// a float32-rounded stress run whose larger endpoint residuals exercise the
// closure tolerances and the endpoint error heat map.
var StandardModes = []Mode{
	{Name: "float64", Use32: false},
	{Name: "float32", Use32: true},
}

// Seg3 is a straight section segment in model coordinates.
type Seg3 struct {
	A, B geom.Vec3
}

// Loop is one stitched closed contour of a section.
type Loop struct {
	Index      int
	Nodes      []geom.Vec2 // closed walk, Nodes[len-1] joins Nodes[0]
	NodeIDs    []int       // endpoint cluster ids backing Nodes
	Segments3D []Seg3
	Level      int     // 0 = outer boundary, 1 = hole, 2 = island ...
	CCW        bool    // winding inside the cutting plane frame
	ClosureGap float64 // largest junction residual measured on this loop
	// Crossings lists self-intersection descriptions ("seg i x seg j").
	Crossings []string
}

// NodeErr pairs a junction position with its endpoint cluster residual.
type NodeErr struct {
	Pos geom.Vec2
	Err float64
}

// Report collects everything found while cutting one BRep with one plane.
type Report struct {
	PlaneName string
	ModeName  string

	RawSegments int
	Junctions   int
	Loops       []*Loop

	// MaxGap is the largest junction closure residual over the whole section.
	MaxGap float64
	// MaxF32Gap is the same statistic from the companion precision run,
	// filled in by Compare so renderers can show both color scales.
	MaxF32Gap float64
	Tolerance float64

	Closed          bool
	OrderConsistent bool
	SelfIntersect   bool
	// WindingConsistent is true when nesting alternates orientation, i.e.
	// all loops sharing a nesting level use the same 2D winding.
	WindingConsistent bool

	// NodeErrors feeds the endpoint error heat map.
	NodeErrors []NodeErr
}
