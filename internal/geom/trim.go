package geom

import "math"

type SpanKind int

const (
	LineSpan SpanKind = iota
	ConicSpan
)

type Span struct {
	Kind   SpanKind
	Points [3]Vec2
	Weight float64
}

type TrimLoop struct {
	Spans []Span
}

func (s Span) Eval(t float64) Vec2 {
	if s.Kind == LineSpan {
		return s.Points[0].Add(s.Points[1].Sub(s.Points[0]).Mul(t))
	}
	p0, p1, p2 := s.Points[0], s.Points[1], s.Points[2]
	w0 := 1.0
	w1 := s.Weight
	w2 := 1.0
	a := 1 - t
	d0 := a * a * w0
	d1 := 2 * a * t * w1
	d2 := t * t * w2
	den := d0 + d1 + d2
	return Vec2{
		(p0.X*d0 + p1.X*d1 + p2.X*d2) / den,
		(p0.Y*d0 + p1.Y*d1 + p2.Y*d2) / den,
	}
}

func (s Span) Start() Vec2 { return s.Points[0] }
func (s Span) End() Vec2 {
	if s.Kind == LineSpan {
		return s.Points[1]
	}
	return s.Points[2]
}

func (t TrimLoop) Polyline(tolerance float64) []Vec2 {
	var out []Vec2
	for _, span := range t.Spans {
		sub := sampleSpan(span, tolerance)
		if len(out) > 0 {
			sub = sub[1:]
		}
		out = append(out, sub...)
	}
	if len(out) > 1 {
		first := out[0]
		last := out[len(out)-1]
		if first.Sub(last).Len() < 1e-10 {
			out = out[:len(out)-1]
		}
	}
	return out
}

func sampleSpan(span Span, tolerance float64) []Vec2 {
	n := 1
	if span.Kind == ConicSpan {
		n = 8
	}
	pts := make([]Vec2, n+1)
	for i := range pts {
		pts[i] = span.Eval(float64(i) / float64(n))
	}
	return pts
}

func (t TrimLoop) SignedArea(poly []Vec2) float64 {
	var area float64
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		area += a.Cross(b)
	}
	return area * 0.5
}

func PolylineIsConvex(poly []Vec2, eps float64) bool {
	if len(poly) < 3 {
		return false
	}
	area := 0.0
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		area += a.Cross(b)
	}
	sign := 1.0
	if area < 0 {
		sign = -1
	}
	for i := range poly {
		a := poly[i]
		b := poly[(i+1)%len(poly)]
		c := poly[(i+2)%len(poly)]
		if sign*(b.Sub(a).Cross(c.Sub(b))) < -eps {
			return false
		}
	}
	return true
}

func PolylineCentroid(poly []Vec2) Vec2 {
	var cx, cy, area float64
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		cross := a.Cross(b)
		area += cross
		cx += (a.X + b.X) * cross
		cy += (a.Y + b.Y) * cross
	}
	if math.Abs(area) < 1e-15 {
		var sum Vec2
		for _, p := range poly {
			sum = sum.Add(p)
		}
		return sum.Mul(1 / float64(len(poly)))
	}
	return Vec2{cx / (3 * area), cy / (3 * area)}
}
