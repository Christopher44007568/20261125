package geom

import "math"

// Curve is a parametric curve on [0,1].
type Curve interface {
	Point(t float64) Vec3
}

// CurveFunc adapts a function to Curve.
type CurveFunc func(t float64) Vec3

func (f CurveFunc) Point(t float64) Vec3 { return f(t) }

// LineCurve is a straight segment. It is useful for planar ruled strips.
type LineCurve struct {
	A, B Vec3
}

func Line(a, b Vec3) LineCurve { return LineCurve{A: a, B: b} }

func (l LineCurve) Point(t float64) Vec3 { return l.A.Lerp(l.B, t) }

// Bezier3 is a cubic Bezier curve.
type Bezier3 struct {
	P [4]Vec3
}

func Cubic(a, b, c, d Vec3) Bezier3 {
	return Bezier3{P: [4]Vec3{a, b, c, d}}
}

func (b Bezier3) Point(t float64) Vec3 {
	u := 1 - t
	w0 := u * u * u
	w1 := 3 * u * u * t
	w2 := 3 * u * t * t
	w3 := t * t * t
	return b.P[0].Mul(w0).Add(b.P[1].Mul(w1)).
		Add(b.P[2].Mul(w2)).Add(b.P[3].Mul(w3))
}

// Derivatives returns d/du and d2/du2 in a finite-difference form. The
// intersection code only needs stable local frames, not an analytic basis.
func CurveDerivatives(c Curve, t, h float64) (Vec3, Vec3) {
	t0 := math.Max(0, t-h)
	t1 := math.Min(1, t+h)
	if t1-t0 < 1e-12 {
		return Vec3{}, Vec3{}
	}
	dt := t1 - t0
	p0, p1 := c.Point(t0), c.Point(t1)
	d1 := p1.Sub(p0).Div(dt)
	tm := 0.5 * (t0 + t1)
	tm0, tm1 := math.Max(0, tm-h), math.Min(1, tm+h)
	if tm1-tm0 < 1e-12 {
		return d1, Vec3{}
	}
	d2 := c.Point(tm1).Sub(c.Point(tm0)).Div(tm1 - tm0)
	return d1, d2.Sub(d1).Div(math.Max(tm-t, 1e-9))
}
