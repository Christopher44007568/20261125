package main

import "math"

const (
	HitNone = iota
	HitScalp
	HitHair
)

// Pixel stores everything required by every shading model so that geometry is
// rasterized exactly once and reused for all three comparisons.
type Pixel struct {
	Hit        uint8
	Depth      float64
	Pos        Vec3
	Normal     Vec3 // scalp normal (also hair root radial direction)
	Tangent    Vec3 // hair fiber tangent; zero on bare scalp
	ShellDepth float64
}

type GBuffer struct {
	W, H  int
	Pixel []Pixel
}

func newGBuffer(w, h int) *GBuffer {
	return &GBuffer{W: w, H: h, Pixel: make([]Pixel, w*h)}
}

func (g *GBuffer) idx(x, y int) int { return y*g.W + x }

// project maps a world point to pixel coordinates plus camera-space depth.
func (s *Scene) project(p Vec3) (sx, sy, depth float64) {
	rel := p.Sub(s.Cam.Eye)
	rx := rel.Dot(s.Cam.Right) / s.Cam.Half
	ry := rel.Dot(s.Cam.Up) / s.Cam.Half
	sx = float64(ImgW) * 0.5 * (1 + rx)
	sy = float64(ImgH) * 0.5 * (1 - ry)
	depth = rel.Dot(s.Cam.Forward)
	return
}

// shellThickness is the local thickness proxy of the hair volume above a scalp
// point with radial unit vector n. It drives the global (double-scattering) term.
func shellThickness(n Vec3) float64 {
	return HairLength * (0.72 + 0.28*clamp01(n.Y+0.25))
}

// rasterize fills the GBuffer: analytic scalp first, then hair polylines.
func (s *Scene) rasterize() *GBuffer {
	g := newGBuffer(ImgW, ImgH)
	s.rasterizeScalp(g)
	for fi := range s.Fibers {
		f := &s.Fibers[fi]
		rootN := f.Root.Normalize()
		shell := shellThickness(rootN)
		for k := 0; k < len(f.Points)-1; k++ {
			s.rasterizeSegment(g, f.Points[k], f.Points[k+1], f.Tangent[k+1], rootN, shell)
		}
	}
	return g
}

func (s *Scene) rasterizeScalp(g *GBuffer) {
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			ndx := g.idx(x, y)
			fx := (float64(x)+0.5)/float64(g.W)*2 - 1
			fy := 1 - (float64(y)+0.5)/float64(g.H)*2
			o := s.Cam.Eye.
				Add(s.Cam.Right.Scale(fx * s.Cam.Half)).
				Add(s.Cam.Up.Scale(fy * s.Cam.Half))
			// Ray o + t*forward against unit sphere centered at origin.
			b := 2 * o.Dot(s.Cam.Forward)
			c := o.Dot(o) - ScalpR*ScalpR
			disc := b*b - 4*c
			if disc < 0 {
				continue
			}
			sqrt := math.Sqrt(disc)
			t := (-b - sqrt) / 2
			if t < 0 {
				t = (-b + sqrt) / 2
			}
			if t < 0 {
				continue
			}
			pos := o.Add(s.Cam.Forward.Scale(t))
			if pos.Y < -0.25 { // only the scalp cap carries hair
				continue
			}
			n := pos.Normalize()
			g.Pixel[ndx] = Pixel{
				Hit:        HitScalp,
				Depth:      t,
				Pos:        pos,
				Normal:     n,
				ShellDepth: shellThickness(n),
			}
		}
	}
}

func (s *Scene) rasterizeSegment(g *GBuffer, p0, p1, tan, rootN Vec3, shell float64) {
	x0, y0, d0 := s.project(p0)
	x1, y1, d1 := s.project(p1)
	dx := x1 - x0
	dy := y1 - y0
	steps := int(math.Max(math.Abs(dx), math.Abs(dy)))
	if steps <= 0 {
		return
	}
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		px := int(math.Round(x0 + dx*u))
		py := int(math.Round(y0 + dy*u))
		if px < 0 || px >= g.W || py < 0 || py >= g.H {
			continue
		}
		pos := p0.Add(p1.Sub(p0).Scale(u))
		depth := d0 + (d1-d0)*u
		ndx := g.idx(px, py)
		cur := g.Pixel[ndx]
		if cur.Hit != HitNone && cur.Depth < depth {
			continue
		}
		g.Pixel[ndx] = Pixel{
			Hit:        HitHair,
			Depth:      depth,
			Pos:        pos,
			Normal:     rootN,
			Tangent:    tan,
			ShellDepth: shell,
		}
	}
}
