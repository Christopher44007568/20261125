package main

import "math"

type Prim struct {
	Center Vec3
	Radii  Vec3
	Axes   [3]Vec3
	Fur    bool
	Seed   float64
	SeedI  int
}

func NewPrim(center, radii Vec3, seed int) Prim {
	return Prim{
		Center: center, Radii: radii,
		Axes: [3]Vec3{V(1, 0, 0), V(0, 1, 0), V(0, 0, 1)},
		Fur:  true, Seed: float64(seed), SeedI: seed,
	}
}

func (p Prim) RotZ(a float64) Prim {
	c, s := math.Cos(a), math.Sin(a)
	p.Axes[0] = V(c, s, 0)
	p.Axes[1] = V(-s, c, 0)
	return p
}

func (p Prim) ToLocal(x Vec3) Vec3 {
	d := x.Sub(p.Center)
	return V(d.Dot(p.Axes[0]), d.Dot(p.Axes[1]), d.Dot(p.Axes[2]))
}

func (p Prim) ToWorld(x Vec3) Vec3 {
	return p.Axes[0].Mul(x.X).Add(p.Axes[1].Mul(x.Y)).Add(p.Axes[2].Mul(x.Z))
}

func (p Prim) Scaled(expand float64) Prim {
	p.Radii = V(math.Max(0.01, p.Radii.X+expand), math.Max(0.01, p.Radii.Y+expand), math.Max(0.01, p.Radii.Z+expand))
	return p
}

type Hit struct {
	T       float64
	Point   Vec3
	Normal  Vec3
	UV      Vec3
	Local   Vec3
	PrimIdx int
}

func (p Prim) Intersect(ro, rd Vec3) (float64, bool) {
	o := p.ToLocal(ro)
	d := V(rd.Dot(p.Axes[0]), rd.Dot(p.Axes[1]), rd.Dot(p.Axes[2]))
	r := p.Radii
	ox, oy, oz := o.X/r.X, o.Y/r.Y, o.Z/r.Z
	dx, dy, dz := d.X/r.X, d.Y/r.Y, d.Z/r.Z
	a := dx*dx + dy*dy + dz*dz
	b := 2 * (ox*dx + oy*dy + oz*dz)
	c := ox*ox + oy*oy + oz*oz - 1
	disc := b*b - 4*a*c
	if disc < 0 || math.Abs(a) < 1e-14 {
		return 0, false
	}
	s := math.Sqrt(disc)
	t0, t1 := (-b-s)/(2*a), (-b+s)/(2*a)
	if t1 < 0 {
		return 0, false
	}
	if t0 >= 0 {
		return t0, true
	}
	return t1, true
}

func (p Prim) HitAt(ro, rd Vec3, t float64) Hit {
	pt := ro.Add(rd.Mul(t))
	local := p.ToLocal(pt)
	nlocal := V(local.X/(p.Radii.X*p.Radii.X), local.Y/(p.Radii.Y*p.Radii.Y), local.Z/(p.Radii.Z*p.Radii.Z)).Norm()
	n := p.Axes[0].Mul(nlocal.X).Add(p.Axes[1].Mul(nlocal.Y)).Add(p.Axes[2].Mul(nlocal.Z)).Norm()
	u := math.Atan2(local.Y/p.Radii.Y, local.X/p.Radii.X)/(2*math.Pi) + 0.5
	v := 0.5 + math.Asin(clamp(local.Z/p.Radii.Z, -1, 1))/math.Pi
	return Hit{T: t, Point: pt, Normal: n, UV: V(u, v, 0), Local: local}
}

func (s *Scene) Intersect(ro, rd Vec3) (Hit, bool) {
	best := math.Inf(1)
	bestIdx := -1
	for i := range s.Prims {
		t, ok := s.Prims[i].Intersect(ro, rd)
		if ok && t < best {
			best, bestIdx = t, i
		}
	}
	if bestIdx < 0 {
		return Hit{}, false
	}
	h := s.Prims[bestIdx].HitAt(ro, rd, best)
	h.PrimIdx = bestIdx
	return h, true
}

type Scene struct {
	Prims []Prim
}

func (s *Scene) Bounds() (Vec3, float64) {
	c := V(0, 0.15, 0.1)
	r := 0.0
	for _, p := range s.Prims {
		d := p.Center.Sub(c).Len() + math.Max(p.Radii.X, math.Max(p.Radii.Y, p.Radii.Z))
		r = math.Max(r, d)
	}
	return c, r
}

func raySphere(ro, rd, center Vec3, radius float64) bool {
	oc := ro.Sub(center)
	b := oc.Dot(rd)
	c := oc.Dot(oc) - radius*radius
	disc := b*b - c
	if disc < 0 {
		return false
	}
	sq := math.Sqrt(disc)
	return -b+sq >= 0
}

func buildScene() Scene {
	s := Scene{}
	head := NewPrim(V(0, 0.24, 0), V(1.08, 0.98, 0.90), 11)
	s.Prims = append(s.Prims, head)

	earL := NewPrim(V(-0.63, 1.15, -0.04), V(0.33, 0.62, 0.24), 23).RotZ(0.27)
	earR := NewPrim(V(0.63, 1.15, -0.04), V(0.33, 0.62, 0.24), 29).RotZ(-0.27)
	s.Prims = append(s.Prims, earL, earR)

	muzzle := NewPrim(V(0, -0.20, 0.78), V(0.52, 0.42, 0.48), 37)
	s.Prims = append(s.Prims, muzzle)
	return s
}

func (s *Scene) IntersectScaled(ro, rd Vec3, expand float64) (Hit, bool) {
	best := math.Inf(1)
	bestIdx := -1
	for i := range s.Prims {
		if !s.Prims[i].Fur {
			continue
		}
		p := s.Prims[i]
		p.Radii = p.Radii.Add(V(expand, expand, expand))
		t, ok := p.Intersect(ro, rd)
		if ok && t < best {
			best, bestIdx = t, i
		}
	}
	if bestIdx < 0 {
		return Hit{}, false
	}
	p := s.Prims[bestIdx]
	p.Radii = p.Radii.Add(V(expand, expand, expand))
	h := p.HitAt(ro, rd, best)
	h.PrimIdx = bestIdx
	return h, true
}
