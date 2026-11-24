package decallab

import "math"

type SurfaceKind string

const (
	SurfaceCurved            SurfaceKind = "curved"
	SurfaceTransparent       SurfaceKind = "transparent"
	SurfaceCurvedTransparent SurfaceKind = "curved_transparent"
)

type Material struct {
	ID          string  `json:"id"`
	Albedo      RGB     `json:"albedo"`
	Transparent bool    `json:"transparent"`
	Opacity     float64 `json:"opacity"`
	Roughness   float64 `json:"roughness"`
}

type RGB struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
}

func (c RGB) Scale(s float64) RGB {
	return RGB{R: c.R * s, G: c.G * s, B: c.B * s}
}

func (c RGB) Add(o RGB) RGB {
	return RGB{R: c.R + o.R, G: c.G + o.G, B: c.B + o.B}
}

func (c RGB) MaxChannel() float64 {
	return math.Max(c.R, math.Max(c.G, c.B))
}

type Hit struct {
	T      float64 `json:"t"`
	Point  Vec3    `json:"point"`
	Normal Vec3    `json:"normal"`
	Frame  Frame   `json:"frame"`
	Side   string  `json:"side"`
}

type Surface interface {
	ID() string
	Kind() SurfaceKind
	GetMaterial() Material
	Intersect(ray Ray) (Hit, bool)
	FrameAt(point Vec3) Frame
}

type baseSurface struct {
	SurfaceID string      `json:"id"`
	KindValue SurfaceKind `json:"kind"`
	Material  Material    `json:"material"`
}

func (s baseSurface) ID() string            { return s.SurfaceID }
func (s baseSurface) Kind() SurfaceKind     { return s.KindValue }
func (s baseSurface) GetMaterial() Material { return s.Material }

// Plane is a finite or infinite oriented plane. Width/Height <= 0 means unbounded.
type Plane struct {
	baseSurface
	Center Vec3    `json:"center"`
	Normal Vec3    `json:"normal"`
	AxisU  Vec3    `json:"axis_u"`
	AxisV  Vec3    `json:"axis_v"`
	Width  float64 `json:"width_m"`
	Height float64 `json:"height_m"`
}

func NewPlane(id string, kind SurfaceKind, center, normal, hint Vec3, width, height float64, material Material) *Plane {
	frame := MakeFrame(center, normal, hint)
	return &Plane{
		baseSurface: baseSurface{SurfaceID: id, KindValue: kind, Material: material},
		Center:      center,
		Normal:      frame.Normal,
		AxisU:       frame.Tangent,
		AxisV:       frame.Bitangent,
		Width:       width,
		Height:      height,
	}
}

func (p *Plane) FrameAt(point Vec3) Frame {
	return Frame{Origin: point, Normal: p.Normal, Tangent: p.AxisU, Bitangent: p.AxisV}
}

func (p *Plane) Intersect(ray Ray) (Hit, bool) {
	denom := ray.Direction.Dot(p.Normal)
	if math.Abs(denom) < 1e-10 {
		return Hit{}, false
	}
	t := p.Center.Sub(ray.Origin).Dot(p.Normal) / denom
	if t < 1e-6 {
		return Hit{}, false
	}
	point := ray.At(t)
	offset := point.Sub(p.Center)
	u := offset.Dot(p.AxisU)
	v := offset.Dot(p.AxisV)
	if p.Width > 0 && math.Abs(u) > p.Width/2+1e-7 {
		return Hit{}, false
	}
	if p.Height > 0 && math.Abs(v) > p.Height/2+1e-7 {
		return Hit{}, false
	}
	side := "front"
	normal := p.Normal
	if denom > 0 {
		side = "back"
	}
	return Hit{T: t, Point: point, Normal: normal, Frame: p.FrameAt(point), Side: side}, true
}

// Sphere is used as a reproducible curved decal/contact-shadow surface.
type Sphere struct {
	baseSurface
	Center Vec3    `json:"center"`
	Radius float64 `json:"radius_m"`
	Hint   Vec3    `json:"frame_hint"`
}

func NewSphere(id string, kind SurfaceKind, center Vec3, radius float64, material Material) *Sphere {
	return &Sphere{
		baseSurface: baseSurface{SurfaceID: id, KindValue: kind, Material: material},
		Center:      center,
		Radius:      radius,
		Hint:        Vec3{Y: 1},
	}
}

func (s *Sphere) FrameAt(point Vec3) Frame {
	normal := point.Sub(s.Center).Normalize()
	return MakeFrame(point, normal, s.Hint)
}

func (s *Sphere) Intersect(ray Ray) (Hit, bool) {
	oc := ray.Origin.Sub(s.Center)
	a := ray.Direction.Dot(ray.Direction)
	b := 2 * oc.Dot(ray.Direction)
	c := oc.Dot(oc) - s.Radius*s.Radius
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return Hit{}, false
	}
	root := math.Sqrt(discriminant)
	t := (-b - root) / (2 * a)
	if t < 1e-6 {
		t = (-b + root) / (2 * a)
	}
	if t < 1e-6 {
		return Hit{}, false
	}
	point := ray.At(t)
	normal := point.Sub(s.Center).Normalize()
	side := "front"
	if ray.Direction.Dot(normal) > 0 {
		side = "inside_or_back"
	}
	return Hit{T: t, Point: point, Normal: normal, Frame: s.FrameAt(point), Side: side}, true
}

// IntersectNearest returns the closest surface hit. Transparent surfaces remain
// eligible because the lab measures the decal layer on the glass itself.
func IntersectNearest(ray Ray, surfaces []Surface) (Hit, Surface, bool) {
	var best Hit
	var bestSurface Surface
	found := false
	for _, surface := range surfaces {
		hit, ok := surface.Intersect(ray)
		if !ok {
			continue
		}
		if !found || hit.T < best.T {
			best = hit
			bestSurface = surface
			found = true
		}
	}
	return best, bestSurface, found
}
