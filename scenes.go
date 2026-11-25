package main

import "math"

type Rect struct {
	MinX, MinY, MaxX, MaxY float64
}

type SceneSpec struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Curves      []TrimCurve `json:"-"`
}

func RoundedRectCurves(prefix string, r Rect, radius float64) []TrimCurve {
	if radius > (r.MaxX-r.MinX)/2 {
		radius = (r.MaxX - r.MinX) / 2
	}
	if radius > (r.MaxY-r.MinY)/2 {
		radius = (r.MaxY - r.MinY) / 2
	}
	arcSeg := 18
	return []TrimCurve{
		{ID: prefix + "-bottom", Segments: 1, Curve: LineNURBS(prefix+"-bottom", v2(r.MinX+radius, r.MinY), v2(r.MaxX-radius, r.MinY))},
		{ID: prefix + "-br", Segments: arcSeg, Curve: QuarterArcNURBS(prefix+"-br", v2(r.MaxX-radius, r.MinY+radius), radius, -math.Pi/2)},
		{ID: prefix + "-right", Segments: 1, Curve: LineNURBS(prefix+"-right", v2(r.MaxX, r.MinY+radius), v2(r.MaxX, r.MaxY-radius))},
		{ID: prefix + "-tr", Segments: arcSeg, Curve: QuarterArcNURBS(prefix+"-tr", v2(r.MaxX-radius, r.MaxY-radius), radius, 0)},
		{ID: prefix + "-top", Segments: 1, Curve: LineNURBS(prefix+"-top", v2(r.MaxX-radius, r.MaxY), v2(r.MinX+radius, r.MaxY))},
		{ID: prefix + "-tl", Segments: arcSeg, Curve: QuarterArcNURBS(prefix+"-tl", v2(r.MinX+radius, r.MaxY-radius), radius, math.Pi/2)},
		{ID: prefix + "-left", Segments: 1, Curve: LineNURBS(prefix+"-left", v2(r.MinX, r.MaxY-radius), v2(r.MinX, r.MinY+radius))},
		{ID: prefix + "-bl", Segments: arcSeg, Curve: QuarterArcNURBS(prefix+"-bl", v2(r.MinX+radius, r.MinY+radius), radius, math.Pi)},
	}
}

func CircleCurves(prefix string, center Vec2, radius float64) []TrimCurve {
	angles := []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2}
	out := make([]TrimCurve, 0, 4)
	for i, a := range angles {
		id := prefix + "-arc-" + string(rune('1'+i))
		out = append(out, TrimCurve{
			ID:       id,
			Segments: 24,
			Curve:    QuarterArcNURBS(id, center, radius, a),
		})
	}
	return out
}

func PolygonCurves(prefix string, points []Vec2) []TrimCurve {
	if signedArea(points) < 0 {
		points = ensureOrientation(points, true)
	}
	out := make([]TrimCurve, 0, len(points))
	for i := range points {
		j := (i + 1) % len(points)
		id := prefix + "-edge-" + string(rune('a'+i))
		out = append(out, TrimCurve{
			ID:       id,
			Segments: 1,
			Curve:    LineNURBS(id, points[i], points[j]),
		})
	}
	return out
}

func reorderCurves(curves []TrimCurve) []TrimCurve {
	order := make([]TrimCurve, 0, len(curves))
	for i := 0; i < len(curves); i += 2 {
		order = append(order, curves[i])
	}
	for i := 1; i < len(curves); i += 2 {
		order = append(order, curves[i])
	}
	return order
}

func DemoScenes() []SceneSpec {
	outer := RoundedRectCurves("outer", Rect{MinX: 6, MinY: 12, MaxX: 94, MaxY: 88}, 9)

	narrow := append([]TrimCurve{}, outer...)
	narrow = append(narrow, RoundedRectCurves("slot", Rect{MinX: 15, MinY: 46, MaxX: 85, MaxY: 54}, 4)...)

	multi := append([]TrimCurve{}, outer...)
	multi = append(multi, CircleCurves("hole-circle", v2(24, 50), 11)...)
	multi = append(multi, RoundedRectCurves("hole-slot", Rect{MinX: 44, MinY: 29, MaxX: 56, MaxY: 71}, 6)...)
	multi = append(multi, PolygonCurves("hole-tri", []Vec2{
		v2(68, 34), v2(87, 48), v2(70, 69),
	})...)

	self := append([]TrimCurve{}, outer...)
	self = append(self, PolygonCurves("cross-line", []Vec2{
		v2(22, 30), v2(78, 70), v2(24, 70), v2(76, 30),
	})...)

	return []SceneSpec{
		{
			ID:          "narrow_slot",
			Title:       "NARROW SLOT",
			Description: "long 8-unit slot sampled from rational NURBS arcs",
			Curves:      reorderCurves(narrow),
		},
		{
			ID:          "multiple_holes",
			Title:       "MULTIPLE HOLES",
			Description: "NURBS circle, vertical slot, and triangular hole",
			Curves:      reorderCurves(multi),
		},
		{
			ID:          "self_intersecting",
			Title:       "SELF-INTERSECTING TRIM",
			Description: "crossing split into two simple even-odd lobes",
			Curves:      reorderCurves(self),
		},
	}
}
