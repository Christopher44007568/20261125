package intersect

import "ruledintersect/geom"

type SegmentStatus uint8

const (
	Valid SegmentStatus = iota
	Duplicate
)

func (s SegmentStatus) String() string {
	if s == Duplicate {
		return "duplicate"
	}
	return "valid"
}

// Segment is one raw or stitched intersection piece. Refined points lie on
// both parametric surfaces within Options.OnSurfaceTolerance.
type Segment struct {
	A, B                 geom.Vec3
	Status               SegmentStatus
	TriangleA, TriangleB int
}

func (s Segment) Length() float64 { return s.A.Dist(s.B) }

func (s Segment) Midpoint() geom.Vec3 { return s.A.Lerp(s.B, 0.5) }

// Polyline is a continuously ordered chain. Statuses has len(Points)-1
// entries when present, so a chain can still show valid and duplicate pieces.
type Polyline struct {
	Points   []geom.Vec3
	Statuses []SegmentStatus
}

type Options struct {
	WeldTolerance       float64
	OnSurfaceTolerance  float64
	DuplicateTolerance  float64
	MinSegmentLength    float64
	MaxRefineIterations int
}

func DefaultOptions() Options {
	return Options{
		WeldTolerance:       2e-5,
		OnSurfaceTolerance:  2e-4,
		DuplicateTolerance:  2.5e-5,
		MinSegmentLength:    2e-6,
		MaxRefineIterations: 12,
	}
}

type Result struct {
	Segments        []Segment
	Valid           []Segment
	Duplicates      []Segment
	Curves          []Polyline
	DuplicateCurves []Polyline
	RawCount        int
	RejectedCount   int
	RefineFailures  int
	GapRejections   int
	ShortRejections int
	MaxSurfaceGap   float64
}
