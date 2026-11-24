package geom

import "math"

// RuledSurface is the finite rule strip
//
//	P(u,v) = Base(u) + v*Dir(u),  u in [U0,U1], v in [V0,V1].
//
// Both Base and Dir are allowed to curve with u, so the strip need not be
// planar.
type RuledSurface struct {
	Name   string
	Base   Curve
	Dir    Curve
	U0, U1 float64
	V0, V1 float64
	Nu, Nv int
}

func NewRuledSurface(name string, base, dir Curve) *RuledSurface {
	return &RuledSurface{
		Name: name,
		Base: base,
		Dir:  dir,
		U0:   0,
		U1:   1,
		V0:   0,
		V1:   1,
		Nu:   72,
		Nv:   36,
	}
}

func (s *RuledSurface) Point(u, v float64) Vec3 {
	return s.Base.Point(u).Add(s.Dir.Point(u).Mul(v))
}

func (s *RuledSurface) Derivatives(u, v float64) (du, dv Vec3) {
	base, dbase := CurveDerivatives(s.Base, u, 1e-5)
	dir, ddir := CurveDerivatives(s.Dir, u, 1e-5)
	_ = base
	return dbase.Add(ddir.Mul(v)), dir
}

func (s *RuledSurface) Normal(u, v float64) Vec3 {
	du, dv := s.Derivatives(u, v)
	n := du.Cross(dv).Unit()
	if n.Len2() > 0.5 {
		return n
	}
	// Degenerate parabolas can make du parallel to dv; use a nearby section.
	for _, h := range []float64{1e-3, -1e-3, 3e-3, -3e-3} {
		nu := Clamp(u+h, s.U0, s.U1)
		du2, dv2 := s.Derivatives(nu, v)
		if n2 := du2.Cross(dv2).Unit(); n2.Len2() > 0.5 {
			return n2
		}
	}
	return Vec3{}
}

func (s *RuledSurface) optimalV(u float64, p Vec3) float64 {
	base := s.Base.Point(u)
	dir := s.Dir.Point(u)
	den := dir.Len2()
	if den < 1e-20 {
		return Clamp(0, s.V0, s.V1)
	}
	return Clamp(p.Sub(base).Dot(dir)/den, s.V0, s.V1)
}

func (s *RuledSurface) distanceAtU(u float64, p Vec3) float64 {
	v := s.optimalV(u, p)
	return s.Point(u, v).Dist(p)
}

// Project returns the closest point on the finite ruled strip. A coarse scan
// brackets the global minimum, followed by golden-section refinement.
func (s *RuledSurface) Project(p Vec3) (u, v float64, q Vec3) {
	const samples = 96
	bestI, bestD := 0, math.Inf(1)
	for i := 0; i <= samples; i++ {
		ui := s.U0 + (s.U1-s.U0)*float64(i)/samples
		if d := s.distanceAtU(ui, p); d < bestD {
			bestI, bestD = i, d
		}
	}
	step := (s.U1 - s.U0) / samples
	a := math.Max(s.U0, s.U0+(float64(bestI)-1)*step)
	b := math.Min(s.U1, s.U0+(float64(bestI)+1)*step)
	gr := (math.Sqrt(5) - 1) / 2
	c := b - gr*(b-a)
	d := a + gr*(b-a)
	fc, fd := s.distanceAtU(c, p), s.distanceAtU(d, p)
	for i := 0; i < 48 && b-a > 1e-10; i++ {
		if fc < fd {
			b, d, fd = d, c, fc
			c = b - gr*(b-a)
			fc = s.distanceAtU(c, p)
		} else {
			a, c, fc = c, d, fd
			d = a + gr*(b-a)
			fd = s.distanceAtU(d, p)
		}
	}
	u = 0.5 * (a + b)
	v = s.optimalV(u, p)
	return u, v, s.Point(u, v)
}

func (s *RuledSurface) Distance(p Vec3) float64 {
	_, _, q := s.Project(p)
	return q.Dist(p)
}

func (s *RuledSurface) SampleGrid() ([]Vec3, []UV) {
	nu, nv := s.Nu, s.Nv
	if nu < 1 {
		nu = 1
	}
	if nv < 1 {
		nv = 1
	}
	points := make([]Vec3, 0, (nu+1)*(nv+1))
	params := make([]UV, 0, (nu+1)*(nv+1))
	for j := 0; j <= nv; j++ {
		v := s.V0 + (s.V1-s.V0)*float64(j)/float64(nv)
		for i := 0; i <= nu; i++ {
			u := s.U0 + (s.U1-s.U0)*float64(i)/float64(nu)
			points = append(points, s.Point(u, v))
			params = append(params, UV{U: u, V: v})
		}
	}
	return points, params
}

type UV struct {
	U, V float64
}
