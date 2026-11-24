package main

import (
	"math"
	"math/rand/v2"
)

type Blend int

const (
	BlendOver Blend = iota
	BlendAdd
)

type Shape interface {
	Shade(p Point) (Color, float64, Blend, bool)
}

type Triangle struct {
	A, B, C Point
	Col     Color
	Alpha   float64
}

func (t Triangle) sign(p, a, b Point) float64 {
	return (p.X-a.X)*(b.Y-a.Y) - (b.X-a.X)*(p.Y-a.Y)
}

func (t Triangle) Shade(p Point) (Color, float64, Blend, bool) {
	d1 := t.sign(p, t.A, t.B)
	d2 := t.sign(p, t.B, t.C)
	d3 := t.sign(p, t.C, t.A)
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	if hasNeg && hasPos {
		return Color{}, 0, BlendOver, false
	}
	return t.Col, t.Alpha, BlendOver, true
}

type Ellipse struct {
	Center      Point
	Rx, Ry, Rot float64
	Col         Color
	Alpha       float64
	Blend       Blend
}

func (e Ellipse) Shade(p Point) (Color, float64, Blend, bool) {
	cos, sin := math.Cos(e.Rot), math.Sin(e.Rot)
	dx, dy := p.X-e.Center.X, p.Y-e.Center.Y
	x := (cos*dx + sin*dy) / e.Rx
	y := (-sin*dx + cos*dy) / e.Ry
	d := x*x + y*y
	if d > 1 {
		return Color{}, 0, e.Blend, false
	}
	return e.Col, e.Alpha, e.Blend, true
}

type Highlight struct {
	Center Point
	Rx, Ry float64
	Peak   float64
}

func (h Highlight) Shade(p Point) (Color, float64, Blend, bool) {
	dx, dy := (p.X-h.Center.X)/h.Rx, (p.Y-h.Center.Y)/h.Ry
	d2 := dx*dx + dy*dy
	if d2 > 1 {
		return Color{}, 0, BlendAdd, false
	}
	v := h.Peak * math.Pow(1-d2, 3)
	return Color{v, v, v}, 1, BlendAdd, true
}

type Scene struct {
	Name       string
	Shapes     []Shape
	Offset     Point
	Background Color
}

func (s Scene) sample(p Point) Color {
	col := s.Background
	q := Point{p.X - s.Offset.X, p.Y - s.Offset.Y}
	for _, sh := range s.Shapes {
		c, a, blend, hit := sh.Shade(q)
		if !hit {
			continue
		}
		switch blend {
		case BlendAdd:
			col = col.Add(c.Scale(a))
		default:
			inv := 1 - a
			col = Color{
				c.R*a + col.R*inv,
				c.G*a + col.G*inv,
				c.B*a + col.B*inv,
			}
		}
	}
	return col.Clamped()
}

func rngFor(name string, seed uint64) *rand.Rand {
	h := seed
	for _, b := range []byte(name) {
		h = h*131 + uint64(b)
	}
	return rand.New(rand.NewPCG(h, h^0x9e3779b97f4a7c15))
}

func buildScene(name string, cfg Config) Scene {
	rng := rngFor(name, cfg.Seed)
	sc := Scene{Name: name, Background: Color{0.06, 0.07, 0.09}}
	place := func() Point {
		return Point{rng.Float64() * float64(cfg.W), rng.Float64() * float64(cfg.H)}
	}
	switch name {
	case sceneTriangles:
		for i := 0; i < 14; i++ {
			a := place()
			base := 4 + rng.Float64()*10
			thin := 0.35 + rng.Float64()*0.9
			ang := rng.Float64() * 2 * math.Pi
			b := Point{a.X + base*math.Cos(ang), a.Y + base*math.Sin(ang)}
			c := Point{a.X + thin*math.Cos(ang+math.Pi/2), a.Y + thin*math.Sin(ang+math.Pi/2)}
			col := Color{0.75 + rng.Float64()*0.2, 0.3 + rng.Float64()*0.3, 0.25 + rng.Float64()*0.2}
			sc.Shapes = append(sc.Shapes, Triangle{a, b, c, col, 0.85 + rng.Float64()*0.15})
		}
	case sceneLeaves:
		for i := 0; i < 16; i++ {
			cen := place()
			rx := 2.5 + rng.Float64()*5
			ry := 1.2 + rng.Float64()*2.5
			col := Color{0.15 + rng.Float64()*0.25, 0.45 + rng.Float64()*0.35, 0.12 + rng.Float64()*0.2}
			sc.Shapes = append(sc.Shapes, Ellipse{
				Center: cen, Rx: rx, Ry: ry, Rot: rng.Float64() * math.Pi,
				Col: col, Alpha: 0.25 + rng.Float64()*0.3, Blend: BlendOver,
			})
		}
	case sceneHighlights:
		for i := 0; i < 12; i++ {
			cen := place()
			sc.Shapes = append(sc.Shapes, Highlight{
				Center: cen,
				Rx:     1.0 + rng.Float64()*1.8,
				Ry:     0.8 + rng.Float64()*1.4,
				Peak:   0.85 + rng.Float64()*0.6,
			})
		}
	}
	return sc
}

type Eye struct {
	Name   string
	Offset Point
}

func bothEyes(cfg Config) []Eye {
	return []Eye{
		{"left", Point{-cfg.EyeShift, 0}},
		{"right", Point{cfg.EyeShift, 0}},
	}
}
