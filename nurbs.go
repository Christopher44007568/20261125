package main

import (
	"fmt"
	"math"
)

type NURBSCurve2 struct {
	ID      string
	Degree  int
	Knots   []float64
	Points  []Vec2
	Weights []float64
}

type NURBSSurface struct {
	DegreeU, DegreeV int
	KnotsU, KnotsV   []float64
	Points           [][]Vec3
	Weights          [][]float64
}

func clampedKnots(degree, controlCount int) []float64 {
	knotCount := controlCount + degree + 1
	k := make([]float64, knotCount)
	for i := 0; i <= degree; i++ {
		k[i] = 0
	}
	for i := controlCount; i < knotCount; i++ {
		k[i] = 1
	}
	internal := controlCount - degree - 1
	for i := 0; i < internal; i++ {
		k[degree+1+i] = float64(i+1) / float64(internal+1)
	}
	return k
}

func LineNURBS(id string, a, b Vec2) NURBSCurve2 {
	return NURBSCurve2{
		ID:     id,
		Degree: 1,
		Knots:  []float64{0, 0, 1, 1},
		Points: []Vec2{a, b},
	}
}

func QuarterArcNURBS(id string, center Vec2, radius, a0 float64) NURBSCurve2 {
	rot := func(p Vec2) Vec2 {
		c := math.Cos(a0)
		s := math.Sin(a0)
		return v2(center.X+radius*(p.X*c-p.Y*s), center.Y+radius*(p.X*s+p.Y*c))
	}
	w := math.Sqrt2 / 2
	return NURBSCurve2{
		ID:      id,
		Degree:  2,
		Knots:   []float64{0, 0, 0, 1, 1, 1},
		Points:  []Vec2{rot(v2(1, 0)), rot(v2(1, 1)), rot(v2(0, 1))},
		Weights: []float64{1, w, 1},
	}
}

func basisValue(i, degree int, u float64, knots []float64) float64 {
	lastKnot := knots[len(knots)-1]
	if degree == 0 {
		if knots[i] <= u && u < knots[i+1] {
			return 1
		}
		if u == lastKnot && i == len(knots)-2 && knots[i] <= u && u <= knots[i+1] {
			return 1
		}
		return 0
	}

	var left, right float64
	d1 := knots[i+degree] - knots[i]
	if d1 > 0 {
		left = (u - knots[i]) / d1 * basisValue(i, degree-1, u, knots)
	}
	d2 := knots[i+degree+1] - knots[i+1]
	if d2 > 0 {
		right = (knots[i+degree+1] - u) / d2 * basisValue(i+1, degree-1, u, knots)
	}
	return left + right
}

func (c NURBSCurve2) Validate() error {
	n := len(c.Points)
	if n < c.Degree+1 {
		return fmt.Errorf("curve %s: %d control points insufficient for degree %d", c.ID, n, c.Degree)
	}
	if len(c.Knots) != n+c.Degree+1 {
		return fmt.Errorf("curve %s: expected %d knots, got %d", c.ID, n+c.Degree+1, len(c.Knots))
	}
	if c.Weights != nil && len(c.Weights) != n {
		return fmt.Errorf("curve %s: expected %d weights, got %d", c.ID, n, len(c.Weights))
	}
	return nil
}

func (c NURBSCurve2) Eval(u float64) Vec2 {
	if err := c.Validate(); err != nil {
		panic(err)
	}
	u = math.Max(c.Knots[0], math.Min(c.Knots[len(c.Knots)-1], u))
	if u <= c.Knots[c.Degree] {
		return c.Points[0]
	}
	upper := c.Knots[len(c.Knots)-1-c.Degree]
	if u >= upper {
		return c.Points[len(c.Points)-1]
	}
	var wx, wy, wsum float64
	for i, p := range c.Points {
		b := basisValue(i, c.Degree, u, c.Knots)
		w := 1.0
		if c.Weights != nil {
			w = c.Weights[i]
		}
		f := b * w
		wx += p.X * f
		wy += p.Y * f
		wsum += f
	}
	if math.Abs(wsum) < geomEps {
		return c.Points[0]
	}
	return v2(wx/wsum, wy/wsum)
}

func (c NURBSCurve2) Sample(segments int) []Vec2 {
	if segments <= 0 {
		segments = 32
	}
	u0 := c.Knots[c.Degree]
	u1 := c.Knots[len(c.Knots)-1-c.Degree]
	if u1 <= u0 {
		u0, u1 = c.Knots[0], c.Knots[len(c.Knots)-1]
	}
	out := make([]Vec2, 0, segments+1)
	for i := 0; i <= segments; i++ {
		u := lerp(u0, u1, float64(i)/float64(segments))
		out = append(out, c.Eval(u))
	}
	return out
}

func (s NURBSSurface) EvalUV(u, v float64) Vec3 {
	var x, y, z, wsum float64
	for i := range s.Points {
		bu := basisValue(i, s.DegreeU, u, s.KnotsU)
		for j, p := range s.Points[i] {
			bv := basisValue(j, s.DegreeV, v, s.KnotsV)
			w := 1.0
			if s.Weights != nil && s.Weights[i] != nil {
				w = s.Weights[i][j]
			}
			f := bu * bv * w
			x += p.X * f
			y += p.Y * f
			z += p.Z * f
			wsum += f
		}
	}
	if math.Abs(wsum) < geomEps {
		return s.Points[0][0]
	}
	return Vec3{X: x / wsum, Y: y / wsum, Z: z / wsum}
}

func DemoPatch() NURBSSurface {
	degree := 2
	grid := [][]Vec3{
		{{X: 0, Y: 0, Z: 0}, {X: 0, Y: 50, Z: 1.4}, {X: 0, Y: 100, Z: 0}},
		{{X: 50, Y: 0, Z: 1.1}, {X: 50, Y: 50, Z: 3.2}, {X: 50, Y: 100, Z: 1.0}},
		{{X: 100, Y: 0, Z: 0}, {X: 100, Y: 50, Z: 1.5}, {X: 100, Y: 100, Z: 0}},
	}
	return NURBSSurface{
		DegreeU: degree,
		DegreeV: degree,
		KnotsU:  clampedKnots(degree, 3),
		KnotsV:  clampedKnots(degree, 3),
		Points:  grid,
	}
}
