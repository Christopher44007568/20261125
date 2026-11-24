package main

import (
	"fmt"
	"math"
)

const (
	degToRad = math.Pi / 180
	radToDeg = 180 / math.Pi
)

type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64         { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Unit() Vec3 {
	l := a.Len()
	if l == 0 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

type Config struct {
	RadiusStart float64
	RadiusEnd   float64
	Length      float64
	HalfAngle   float64
	FaceWidth   float64
	SamplesU    int
	SamplesV    int
	PanelWidth  int
	PanelHeight int
	SuperSample int
}

func DefaultConfig() Config {
	return Config{
		RadiusStart: 1,
		RadiusEnd:   5,
		Length:      30,
		HalfAngle:   30,
		FaceWidth:   7,
		SamplesU:    80,
		SamplesV:    220,
		PanelWidth:  860,
		PanelHeight: 530,
		SuperSample: 2,
	}
}

func (c Config) Validate() error {
	switch {
	case c.RadiusStart <= 0 || c.RadiusEnd <= 0:
		return fmt.Errorf("radii must be positive")
	case c.Length <= 0 || c.FaceWidth <= 0:
		return fmt.Errorf("length and face width must be positive")
	case c.HalfAngle <= 2 || c.HalfAngle >= 88:
		return fmt.Errorf("half angle must be between 2 and 88 degrees")
	case c.SamplesU < 8 || c.SamplesV < 8:
		return fmt.Errorf("mesh sampling must be at least 8 by 8")
	case c.PanelWidth < 200 || c.PanelHeight < 160:
		return fmt.Errorf("panel size is too small")
	case c.SuperSample < 1 || c.SuperSample > 4:
		return fmt.Errorf("supersampling must be in [1,4]")
	}
	return nil
}

// RadiusProfile is a cosine-eased 1D radius law. Its derivative is zero at
// both longitudinal ends, which prevents an artificial tangent kink there.
func (c Config) Radius(v float64) float64 {
	v = clamp(v, 0, 1)
	s := 0.5 - 0.5*math.Cos(math.Pi*v)
	return c.RadiusStart + (c.RadiusEnd-c.RadiusStart)*s
}

func (c Config) RadiusDerivatives(v float64) (float64, float64) {
	d := c.RadiusEnd - c.RadiusStart
	r1 := d * 0.5 * math.Pi * math.Sin(math.Pi*v)
	r2 := d * 0.5 * math.Pi * math.Pi * math.Cos(math.Pi*v)
	return r1, r2
}

func (c Config) yAt(v float64) float64 {
	return (v - 0.5) * c.Length
}

type SurfaceSample struct {
	P  Vec3
	Pu Vec3
	Pv Vec3
	N  Vec3
	UU Vec3
	UV Vec3
	VV Vec3
}

type FilletSurface struct {
	Config
	sinA float64
	cosA float64
}

func NewFilletSurface(c Config) FilletSurface {
	a := c.HalfAngle * degToRad
	return FilletSurface{Config: c, sinA: math.Sin(a), cosA: math.Cos(a)}
}

// Eval evaluates the variable-radius fillet. u runs across the arc from the
// upper parent (u=0) to the lower parent (u=1), while v runs along the edge.
func (f FilletSurface) Eval(u, v float64) SurfaceSample {
	r := f.Radius(v)
	r1, r2 := f.RadiusDerivatives(v)
	delta := math.Pi - 2*f.HalfAngle*degToRad
	ang := math.Pi/2 + f.HalfAngle*degToRad + u*delta
	ca, sa := math.Cos(ang), math.Sin(ang)
	y := f.yAt(v)
	centerX := r / f.sinA

	p := Vec3{centerX + r*ca, y, r * sa}
	pu := Vec3{-r * delta * sa, 0, r * delta * ca}
	radial := Vec3{ca, 0, sa}
	q := Vec3{1/f.sinA + ca, 0, sa}
	pv := q.Scale(r1).Add(Vec3{Y: f.Length})
	n := pv.Cross(pu).Unit()

	return SurfaceSample{
		P:  p,
		Pu: pu,
		Pv: pv,
		N:  n,
		UU: radial.Scale(-r * delta * delta),
		UV: Vec3{-r1 * delta * sa, 0, r1 * delta * ca},
		VV: q.Scale(r2),
	}
}

// ParentSurface is one of the two planar faces. side=+1 selects the upper
// face and side=-1 the lower face. s=0 is the fillet contact and s=1 is the
// far edge of the parent strip.
type ParentSurface struct {
	Config
	Side int
	sinA float64
	cosA float64
}

func NewParentSurface(c Config, side int) ParentSurface {
	a := c.HalfAngle * degToRad
	return ParentSurface{Config: c, Side: side, sinA: math.Sin(a), cosA: math.Cos(a)}
}

func (p ParentSurface) Eval(s, v float64) SurfaceSample {
	r := p.Radius(v)
	r1, r2 := p.RadiusDerivatives(v)
	tangent := Vec3{p.cosA, 0, float64(p.Side) * p.sinA}
	contact := tangent.Scale(r * p.cotHalfAngle())
	point := contact.Add(tangent.Scale(s * p.FaceWidth))
	point.Y = p.yAt(v)

	pv := tangent.Scale(r1 * p.cotHalfAngle())
	pv.Y = p.Length
	normal := Vec3{-p.sinA, 0, float64(p.Side) * p.cosA}
	return SurfaceSample{
		P:  point,
		Pu: tangent.Scale(p.FaceWidth),
		Pv: pv,
		N:  normal,
		VV: tangent.Scale(r2 * p.cotHalfAngle()),
	}
}

func (p ParentSurface) cotHalfAngle() float64 {
	return p.cosA / p.sinA
}

func (f FilletSurface) Contact(v float64, upper bool) Vec3 {
	if upper {
		return f.Eval(0, v).P
	}
	return f.Eval(1, v).P
}

type Curvatures struct {
	K1, K2 float64
	Mean   float64
	Gauss  float64
}

func (s SurfaceSample) Curvatures() Curvatures {
	n := s.N
	E, F, G := s.Pu.Dot(s.Pu), s.Pu.Dot(s.Pv), s.Pv.Dot(s.Pv)
	e, f, g := n.Dot(s.UU), n.Dot(s.UV), n.Dot(s.VV)
	den := E*G - F*F
	if math.Abs(den) < 1e-16 {
		return Curvatures{}
	}
	mean := (e*G + g*E - 2*f*F) / (2 * den)
	gauss := (e*g - f*f) / den
	disc := math.Max(0, mean*mean-gauss)
	root := math.Sqrt(disc)
	return Curvatures{
		K1:    mean + root,
		K2:    mean - root,
		Mean:  mean,
		Gauss: gauss,
	}
}

func (c Curvatures) MaxAbs() float64 {
	return math.Max(math.Abs(c.K1), math.Abs(c.K2))
}
