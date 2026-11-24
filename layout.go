package main

import "math"

type EyeGlyph struct {
	View       PlacedGlyph `json:"view"`
	Bounds     FloatRect   `json:"bounds"`
	Projection Projection  `json:"projection"`
	Valid      bool        `json:"valid"`
}

type GlyphParallax struct {
	HorizontalPx          float64 `json:"horizontalPxLeftMinusRight"`
	VerticalPx            float64 `json:"verticalPxLeftMinusRight"`
	AngularArcMin         float64 `json:"angularHorizontalArcMinLeftMinusRight"`
	VerticalAngularArcMin float64 `json:"verticalAngularArcMinLeftMinusRight"`
	NaiveCopyErrorPx      float64 `json:"naiveSameScreenCoordinateCopyErrorPx"`
	CoordinatesCopied     bool    `json:"coordinatesCopied"`
	ProjectionPath        string  `json:"projectionPath"`
}

type GlyphPair struct {
	ElementID string        `json:"elementId"`
	Index     int           `json:"index"`
	Char      string        `json:"char"`
	Left      EyeGlyph      `json:"left"`
	Right     EyeGlyph      `json:"right"`
	Parallax  GlyphParallax `json:"parallax"`
}

type ElementGeometry struct {
	Spec        TextSpec    `json:"spec"`
	Glyphs      []GlyphPair `json:"glyphs"`
	LeftBounds  FloatRect   `json:"leftBounds"`
	RightBounds FloatRect   `json:"rightBounds"`
}

func projectGlyph(spec TextSpec, layer Layer, index int, char rune, eye Eye) EyeGlyph {
	texelM := spec.Style.HeightM / 7.0
	advanceM := float64(index) * 6.0 * texelM
	originWorld := layer.worldPoint(spec.X+advanceM, spec.Top)
	rightWorld := originWorld.Add(Vec3{X: texelM})
	downWorld := originWorld.Add(Vec3{Y: -texelM})

	origin := eye.Project(originWorld)
	right := eye.Project(rightWorld)
	down := eye.Project(downWorld)
	valid := origin.Visible || (origin.DepthM > 0.02 && right.DepthM > 0.02 && down.DepthM > 0.02)
	if origin.DepthM <= 0.02 || right.DepthM <= 0.02 || down.DepthM <= 0.02 {
		valid = false
	}

	texelW := math.Abs(right.Point.X - origin.Point.X)
	texelH := math.Abs(down.Point.Y - origin.Point.Y)
	bounds := FloatRect{
		MinX: origin.Point.X,
		MinY: origin.Point.Y,
		MaxX: origin.Point.X + 5*texelW,
		MaxY: origin.Point.Y + 7*texelH,
	}
	return EyeGlyph{
		View: PlacedGlyph{
			Rune:   char,
			Origin: origin.Point,
			TexelW: texelW,
			TexelH: texelH,
		},
		Bounds:     bounds,
		Projection: origin,
		Valid:      valid,
	}
}

func ProjectText(spec TextSpec, layer Layer, eyes [2]Eye) ElementGeometry {
	geo := ElementGeometry{Spec: spec}
	geo.Glyphs = make([]GlyphPair, 0, len(spec.Text))

	for i, char := range spec.Text {
		left := projectGlyph(spec, layer, i, char, eyes[0])
		right := projectGlyph(spec, layer, i, char, eyes[1])
		horizontal := left.View.Origin.X - right.View.Origin.X
		vertical := left.View.Origin.Y - right.View.Origin.Y
		angleLeft := math.Atan(left.Projection.Tangent.X)
		angleRight := math.Atan(right.Projection.Tangent.X)
		angular := arcMinutes(angleLeft - angleRight)
		verticalAngular := arcMinutes(math.Atan(left.Projection.Tangent.Y) - math.Atan(right.Projection.Tangent.Y))
		copyError := math.Hypot(
			right.View.Origin.X-left.View.Origin.X,
			right.View.Origin.Y-left.View.Origin.Y,
		)
		path := left.Projection.ProjectionPath + ";" + right.Projection.ProjectionPath
		pair := GlyphPair{
			ElementID: spec.ID,
			Index:     i,
			Char:      string(char),
			Left:      left,
			Right:     right,
			Parallax: GlyphParallax{
				HorizontalPx:          horizontal,
				VerticalPx:            vertical,
				AngularArcMin:         angular,
				VerticalAngularArcMin: verticalAngular,
				NaiveCopyErrorPx:      copyError,
				CoordinatesCopied:     false,
				ProjectionPath:        path,
			},
		}
		geo.Glyphs = append(geo.Glyphs, pair)
		if left.Valid {
			geo.LeftBounds = unionRect(geo.LeftBounds, left.Bounds)
		}
		if right.Valid {
			geo.RightBounds = unionRect(geo.RightBounds, right.Bounds)
		}
	}
	return geo
}

func unionRect(a, b FloatRect) FloatRect {
	if a.Width() == 0 && a.Height() == 0 {
		return b
	}
	if b.Width() == 0 && b.Height() == 0 {
		return a
	}
	return FloatRect{
		MinX: math.Min(a.MinX, b.MinX),
		MinY: math.Min(a.MinY, b.MinY),
		MaxX: math.Max(a.MaxX, b.MaxX),
		MaxY: math.Max(a.MaxY, b.MaxY),
	}
}

func ProjectPanelRect(layer Layer, eye Eye) FloatRect {
	p := layer.Panel
	corners := []Vec3{
		layer.worldPoint(-p.Width/2, p.Height/2),
		layer.worldPoint(p.Width/2, p.Height/2),
		layer.worldPoint(-p.Width/2, -p.Height/2),
		layer.worldPoint(p.Width/2, -p.Height/2),
	}
	points := make([]ScreenPoint, 0, 4)
	for _, corner := range corners {
		pr := eye.Project(corner)
		if pr.DepthM > 0.02 {
			points = append(points, pr.Point)
		}
	}
	return RectFromPoints(points...)
}

func ProjectWorldRect(layer Layer, rect RectSpec, eye Eye) FloatRect {
	corners := []Vec3{
		layer.worldPoint(rect.X, rect.Top),
		layer.worldPoint(rect.X+rect.Width, rect.Top),
		layer.worldPoint(rect.X, rect.Top-rect.Height),
		layer.worldPoint(rect.X+rect.Width, rect.Top-rect.Height),
	}
	points := make([]ScreenPoint, 0, 4)
	for _, corner := range corners {
		pr := eye.Project(corner)
		if pr.DepthM > 0.02 {
			points = append(points, pr.Point)
		}
	}
	return RectFromPoints(points...)
}
