package main

import (
	"fmt"
	"math"
	"sort"
)

type TrimCurve struct {
	ID       string
	Curve    NURBSCurve2
	Segments int
}

type Crossing struct {
	Point Vec2 `json:"point"`
	EdgeA int  `json:"edge_a"`
	EdgeB int  `json:"edge_b"`
}

type ExtractedLoop struct {
	ID        string     `json:"id"`
	CurveIDs  []string   `json:"curve_ids"`
	Points    []Vec2     `json:"points"`
	Closed    bool       `json:"closed"`
	Area      float64    `json:"area"`
	CCW       bool       `json:"ccw"`
	Crossings []Crossing `json:"crossings,omitempty"`
}

type SegmentHit struct {
	Point Vec2
	T     float64
	U     float64
	Kind  int
}

const (
	segNone = iota
	segProper
	segEndpoint
	segCollinear
)

func segmentIntersection(a, b, c, d Vec2) (SegmentHit, bool) {
	r := sub2(b, a)
	s := sub2(d, c)
	den := cross2(r, s)
	ca := sub2(c, a)
	if math.Abs(den) > geomEps {
		t := cross2(ca, s) / den
		u := cross2(ca, r) / den
		if t < -pointEps || t > 1+pointEps || u < -pointEps || u > 1+pointEps {
			return SegmentHit{}, false
		}
		t = math.Max(0, math.Min(1, t))
		u = math.Max(0, math.Min(1, u))
		kind := segProper
		if t <= pointEps || t >= 1-pointEps || u <= pointEps || u >= 1-pointEps {
			kind = segEndpoint
		}
		return SegmentHit{Point: add2(a, scale2(r, t)), T: t, U: u, Kind: kind}, true
	}
	if math.Abs(cross2(ca, r)) > 1e-7 {
		return SegmentHit{}, false
	}
	return SegmentHit{Kind: segCollinear}, true
}

func FindSelfCrossings(poly []Vec2) []Crossing {
	n := len(poly)
	if n < 4 {
		return nil
	}
	var out []Crossing
	for i := 0; i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		for j := i + 1; j < n; j++ {
			if j == i+1 || (i == 0 && j == n-1) {
				continue
			}
			c, d := poly[j], poly[(j+1)%n]
			hit, ok := segmentIntersection(a, b, c, d)
			if !ok || hit.Kind != segProper {
				continue
			}
			duplicate := false
			for _, old := range out {
				if approx2(old.Point, hit.Point, attachEps) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				out = append(out, Crossing{Point: hit.Point, EdgeA: i, EdgeB: j})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].EdgeA != out[j].EdgeA {
			return out[i].EdgeA < out[j].EdgeA
		}
		return out[i].EdgeB < out[j].EdgeB
	})
	return out
}

type sampledPiece struct {
	id     string
	points []Vec2
}

func ExtractLoops(curves []TrimCurve, tolerance float64) ([]ExtractedLoop, error) {
	if tolerance <= 0 {
		tolerance = attachEps
	}
	pieces := make([]sampledPiece, 0, len(curves))
	for _, tc := range curves {
		if err := tc.Curve.Validate(); err != nil {
			return nil, err
		}
		segments := tc.Segments
		if segments <= 0 {
			segments = 32
		}
		pts := dedupeAdjacent(tc.Curve.Sample(segments), tolerance)
		if len(pts) >= 2 {
			pieces = append(pieces, sampledPiece{id: tc.ID, points: pts})
		}
	}

	used := make([]bool, len(pieces))
	var loops []ExtractedLoop
	for seed := 0; seed < len(pieces); seed++ {
		if used[seed] {
			continue
		}
		used[seed] = true
		chain := append([]Vec2(nil), pieces[seed].points...)
		ids := []string{pieces[seed].id}
		closed := false

		for !closed {
			end := chain[len(chain)-1]
			start := chain[0]
			if approx2(end, start, tolerance) {
				closed = true
				break
			}
			found := -1
			reverse := false
			for k, piece := range pieces {
				if used[k] {
					continue
				}
				first := piece.points[0]
				last := piece.points[len(piece.points)-1]
				if approx2(first, end, tolerance) {
					found = k
					reverse = false
					break
				}
				if approx2(last, end, tolerance) {
					found = k
					reverse = true
					break
				}
			}
			if found < 0 {
				return loops, fmt.Errorf("open trim chain %v ends at %v", ids, end)
			}
			used[found] = true
			ids = append(ids, pieces[found].id)
			pts := pieces[found].points
			if reverse {
				reversed := append([]Vec2(nil), pts...)
				for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
					reversed[i], reversed[j] = reversed[j], reversed[i]
				}
				pts = reversed
			}
			chain = append(chain, pts[1:]...)
			if len(chain) > 1 && approx2(chain[len(chain)-1], chain[0], tolerance) {
				closed = true
			}
		}

		points := dedupeAdjacent(chain, tolerance)
		points = removeCollinear(points, tolerance*2)
		if len(points) < 3 {
			return loops, fmt.Errorf("trim loop %v has only %d effective vertices", ids, len(points))
		}
		area := signedArea(points)
		loop := ExtractedLoop{
			ID:        fmt.Sprintf("L%02d", len(loops)+1),
			CurveIDs:  ids,
			Points:    points,
			Closed:    true,
			Area:      area,
			CCW:       area > 0,
			Crossings: FindSelfCrossings(points),
		}
		loops = append(loops, loop)
	}
	return loops, nil
}

func pointOnSegmentEps(p, a, b Vec2, eps float64) bool {
	if math.Abs(orient2(a, b, p)) > eps*(math.Max(1, dist(a, b))) {
		return false
	}
	return p.X >= math.Min(a.X, b.X)-eps && p.X <= math.Max(a.X, b.X)+eps &&
		p.Y >= math.Min(a.Y, b.Y)-eps && p.Y <= math.Max(a.Y, b.Y)+eps
}

func PointInPolygon(p Vec2, poly []Vec2) bool {
	return pointInPolygon(p, poly, pointEps, true)
}

func PointInPolygonStrict(p Vec2, poly []Vec2) bool {
	if pointOnSegmentEps(p, poly[len(poly)-1], poly[0], pointEps) {
		return false
	}
	for i := 0; i < len(poly); i++ {
		if pointOnSegmentEps(p, poly[i], poly[(i+1)%len(poly)], pointEps) {
			return false
		}
	}
	return pointInPolygon(p, poly, pointEps, false)
}

func pointInPolygon(p Vec2, poly []Vec2, eps float64, boundaryInside bool) bool {
	for i := range poly {
		if pointOnSegmentEps(p, poly[i], poly[(i+1)%len(poly)], eps) {
			return boundaryInside
		}
	}
	inside := false
	for i := 0; i < len(poly); i++ {
		j := (i + 1) % len(poly)
		a, b := poly[i], poly[j]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			xCross := a.X + (p.Y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
			if xCross > p.X+eps {
				inside = !inside
			}
		}
	}
	return inside
}

func SplitOneCrossing(poly []Vec2, crossing Crossing) ([]Vec2, []Vec2, error) {
	n := len(poly)
	i := crossing.EdgeA
	j := crossing.EdgeB
	if i < 0 || j < 0 || i >= n || j >= n {
		return nil, nil, fmt.Errorf("invalid crossing edges %d,%d", i, j)
	}
	if i > j {
		i, j = j, i
	}
	x := crossing.Point
	loopA := []Vec2{x}
	for k := i + 1; k <= j; k++ {
		loopA = append(loopA, poly[k%n])
	}
	loopB := []Vec2{x}
	for k := j + 1; ; k++ {
		idx := k % n
		loopB = append(loopB, poly[idx])
		if idx == i {
			break
		}
	}
	loopA = dedupeAdjacent(loopA, attachEps)
	loopB = dedupeAdjacent(loopB, attachEps)
	if len(loopA) < 3 || len(loopB) < 3 {
		return nil, nil, fmt.Errorf("crossing split produced degenerate loops")
	}
	return loopA, loopB, nil
}

type TrimDomain struct {
	Outer             []Vec2          `json:"outer"`
	Holes             [][]Vec2        `json:"holes"`
	RawLoops          []ExtractedLoop `json:"raw_loops"`
	CrossingCount     int             `json:"crossing_count"`
	Repaired          bool            `json:"repaired"`
	RepairDescription string          `json:"repair_description,omitempty"`
}

func BuildTrimDomain(loops []ExtractedLoop) (TrimDomain, error) {
	if len(loops) == 0 {
		return TrimDomain{}, fmt.Errorf("no trim loops extracted")
	}
	domain := TrimDomain{RawLoops: loops}
	type sized struct {
		points []Vec2
		area   float64
	}
	var polygons []sized
	for _, loop := range loops {
		domain.CrossingCount += len(loop.Crossings)
		if len(loop.Crossings) == 0 {
			polygons = append(polygons, sized{loop.Points, math.Abs(loop.Area)})
			continue
		}
		if len(loop.Crossings) != 1 {
			return TrimDomain{}, fmt.Errorf("loop %s has %d self-crossings; this demo repairs one crossing", loop.ID, len(loop.Crossings))
		}
		a, b, err := SplitOneCrossing(loop.Points, loop.Crossings[0])
		if err != nil {
			return TrimDomain{}, err
		}
		polygons = append(polygons, sized{a, math.Abs(signedArea(a))}, sized{b, math.Abs(signedArea(b))})
		domain.Repaired = true
		domain.RepairDescription = "split one self-crossing at its intersection into two simple even-odd lobes"
	}

	outerIndex := 0
	for i := 1; i < len(polygons); i++ {
		if polygons[i].area > polygons[outerIndex].area {
			outerIndex = i
		}
	}
	domain.Outer = ensureOrientation(polygons[outerIndex].points, true)
	for i, poly := range polygons {
		if i == outerIndex {
			continue
		}
		if !PointInPolygon(poly.points[0], domain.Outer) {
			return TrimDomain{}, fmt.Errorf("loop %d is not inside selected outer contour", i)
		}
		domain.Holes = append(domain.Holes, ensureOrientation(poly.points, false))
	}
	return domain, nil
}

func InsideDomain(p Vec2, outer []Vec2, holes [][]Vec2) bool {
	if !PointInPolygon(p, outer) {
		return false
	}
	for _, hole := range holes {
		if PointInPolygon(p, hole) {
			return false
		}
	}
	return true
}
