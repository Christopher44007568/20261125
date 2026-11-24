package main

import "math"

type Params struct {
	Offset       float64
	CornerRadius float64
	ArmLength    float64
	Height       float64
	ArcSegments  int
}

func DefaultParams() Params {
	return Params{
		Offset:       2,
		CornerRadius: 1.25,
		ArmLength:    18,
		Height:       12,
		ArcSegments:  48,
	}
}

// Profile is the planar cross-section of the offset shell. The outer curve is a
// rounded 90 degree elbow; the inner curve is its exact normal offset.
type Profile struct {
	Params Params
	Outer  []Vec2
	Inner  []Vec2
	Normal []Vec2
}

func (p Params) valid() bool {
	return p.Offset >= 0 && p.CornerRadius > 0 && p.ArmLength > 0 &&
		p.Height > 0 && p.ArcSegments >= 2
}

func buildProfile(p Params) Profile {
	outer, normal := roundedElbow(p)
	inner := make([]Vec2, len(outer))
	for i := range outer {
		inner[i] = outer[i].Sub(normal[i].Scale(p.Offset))
	}
	return Profile{Params: p, Outer: outer, Inner: inner, Normal: normal}
}

// roundedElbow returns a quarter-circle elbow. Each straight arm has length
// ArmLength measured from its free end to the tangent point. The outward
// normal points to the convex side of the bend.
func roundedElbow(p Params) ([]Vec2, []Vec2) {
	n := p.ArcSegments
	r := p.CornerRadius
	a := p.ArmLength
	outer := make([]Vec2, 0, n+3)
	normal := make([]Vec2, 0, n+3)

	outer = append(outer, Vec2{-(a + r), 0})
	normal = append(normal, Vec2{0, -1})

	for i := 0; i <= n; i++ {
		theta := -0.5*math.Pi + 0.5*math.Pi*float64(i)/float64(n)
		outer = append(outer, Vec2{
			-r + r*math.Cos(theta),
			r + r*math.Sin(theta),
		})
		normal = append(normal, Vec2{math.Cos(theta), math.Sin(theta)})
	}

	outer = append(outer, Vec2{0, a + r})
	normal = append(normal, Vec2{1, 0})
	return outer, normal
}

// Polygon closes the wall: outer curve forward, inner curve backward.
func (p Profile) Polygon() []Vec2 {
	poly := make([]Vec2, 0, len(p.Outer)+len(p.Inner))
	poly = append(poly, p.Outer...)
	for i := len(p.Inner) - 1; i >= 0; i-- {
		poly = append(poly, p.Inner[i])
	}
	return poly
}

func (p Profile) Area() float64 {
	return math.Abs(signedArea2(p.Polygon()))
}

// AnalyticArea is the signed area swept by the smooth elbow. It remains a
// polynomial across the self-intersection threshold, which is the point of the
// continuity test: geometry is retained, only its orientation reverses.
func (p Profile) AnalyticArea(offset float64) float64 {
	return 2*p.Params.ArmLength*offset +
		0.5*math.Pi*(p.Params.CornerRadius*offset-0.5*offset*offset)
}

func (p Profile) AnalyticVolume(offset float64) float64 {
	return p.Params.Height * p.AnalyticArea(offset)
}
