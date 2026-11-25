package render

import (
	"image/color"
	"math"

	"hairroots/internal/scalp"
)

// Camera looks at the scalp from Eye, with screen Up defining orientation.
// Ortho switches between perspective and orthographic (side view) projection.
type Camera struct {
	Eye, Up, Right scalp.Vec3
	Forward        scalp.Vec3 // unit, points from the target toward the eye
	Focal          float64
	Ortho          bool
	Scale          float64 // orthographic world-to-pixel scale factor
	Zoom           float64 // screen-space magnification applied at project
}

// NewCamera builds a camera aimed at the origin.
func NewCamera(eye, up scalp.Vec3, focal float64, ortho bool) *Camera {
	fwd := eye.Normalize()
	right := up.Cross(fwd).Normalize()
	up = fwd.Cross(right).Normalize()
	return &Camera{
		Eye:     eye,
		Up:      up,
		Right:   right,
		Forward: fwd,
		Focal:   focal,
		Ortho:   ortho,
		Scale:   1,
		Zoom:    1,
	}
}

// Project maps a world point to pixel coordinates and a camera-space depth.
// ok is false when the point lies behind the camera.
func (cam *Camera) Project(p scalp.Vec3, c *Canvas) (x, y int, depth float64, ok bool) {
	rel := p.Sub(cam.Eye)
	viewDist := rel.Dot(cam.Forward) // negative in front of the eye
	if viewDist > 0 {
		return 0, 0, 0, false
	}
	// Positive "closer is smaller" depth, matching the Canvas convention.
	depth = -viewDist

	sx := rel.Dot(cam.Right)
	sy := rel.Dot(cam.Up)
	var nx, ny float64
	if cam.Ortho {
		nx, ny = sx*cam.Scale, sy*cam.Scale
	} else {
		// Pinhole projection: screen offset is the tangent-plane coordinate
		// scaled by focal length over view distance. This matches pixelRay.
		nx = cam.Focal * sx / depth
		ny = cam.Focal * sy / depth
	}

	unit := math.Min(float64(c.W), float64(c.H)) * 0.34 * cam.Zoom
	x = c.W/2 + int(math.Round(nx*unit))
	y = c.H/2 - int(math.Round(ny*unit))
	return x, y, depth, true
}

// PaintScalp rasterizes the visible ellipsoid surface into the depth buffer
// by ray-casting every pixel, shaded by a headlight plus a vertical gradient.
// Ray casting avoids the scan-line gaps that parameter-grid splatting leaves
// on steep parts of the ellipsoid.
func PaintScalp(c *Canvas, cam *Camera, s *scalp.Scalp, base [3]float64) {
	c.resetDepth()
	unit := math.Min(float64(c.W), float64(c.H)) * 0.34 * cam.Zoom

	for py := 0; py < c.H; py++ {
		for px := 0; px < c.W; px++ {
			nx := (float64(px-c.W/2) + 0.5) / unit
			ny := (float64(c.H/2-py) - 0.5) / unit

			origin, dir, hitOK := cam.pixelRay(nx, ny)
			if !hitOK {
				continue
			}
			t, point, normal, intersect := rayEllipsoid(origin, dir, s)
			if !intersect {
				continue
			}
			theta := math.Acos(clampf(point.Y/s.B, -1, 1))
			phi := math.Atan2(point.Z/s.C, point.X/s.A)
			if !s.Inside(theta, phi) {
				continue
			}
			depth := cam.Eye.Sub(point).Dot(cam.Forward)
			headlight := 0.70 + 0.30*math.Max(0, normal.Dot(cam.Forward))
			vertical := 0.92 + 0.08*point.Y/s.B
			shade := headlight * vertical
			col := colorColor(base[0]*shade, base[1]*shade, base[2]*shade)
			c.Set(px, py, depth, col)
			_ = t
		}
	}
}

// pixelRay returns the world-space ray (origin, unit direction) for a
// normalized screen coordinate under perspective or orthographic projection.
func (cam *Camera) pixelRay(nx, ny float64) (origin, dir scalp.Vec3, ok bool) {
	if cam.Ortho {
		origin = cam.Eye.
			Add(cam.Right.Scale(nx)).
			Add(cam.Up.Scale(ny))
		return origin, cam.Forward.Scale(-1), true
	}
	origin = cam.Eye
	dir = cam.Forward.Scale(-cam.Focal).
		Add(cam.Right.Scale(nx)).
		Add(cam.Up.Scale(ny)).
		Normalize()
	return origin, dir, true
}

// rootVisible reports whether the root at p (with known parameters) is the
// first ellipsoid hit along its camera ray. Hairline membership is taken
// from the root's own parameters, which the sampler guaranteed at creation.
func (cam *Camera) RootVisible(p scalp.Vec3, theta, phi float64, s *scalp.Scalp) bool {
	if !s.Inside(theta, phi) {
		return false
	}
	var origin, dir scalp.Vec3
	if cam.Ortho {
		// Ray parallel to the view axis, passing through p.
		forward := cam.Forward.Scale(-1)
		origin = p.Sub(forward.Scale(cam.Eye.Length() + 4))
		dir = forward
	} else {
		origin = cam.Eye
		dir = p.Sub(cam.Eye).Normalize()
	}
	_, hit, _, ok := rayEllipsoid(origin, dir, s)
	if !ok {
		return false
	}
	// The expected root must coincide with the first surface entry point.
	return hit.Sub(p).Length() < 0.005
}

// rayEllipsoid intersects a ray with the axis-aligned ellipsoid and returns
// the nearest entry point together with its outward unit normal.
func rayEllipsoid(origin, dir scalp.Vec3, s *scalp.Scalp) (t float64, point, normal scalp.Vec3, ok bool) {
	ox, oy, oz := origin.X/s.A, origin.Y/s.B, origin.Z/s.C
	dx, dy, dz := dir.X/s.A, dir.Y/s.B, dir.Z/s.C

	a := dx*dx + dy*dy + dz*dz
	b := 2 * (ox*dx + oy*dy + oz*dz)
	cc := ox*ox + oy*oy + oz*oz - 1
	disc := b*b - 4*a*cc
	if disc < 0 {
		return 0, scalp.Vec3{}, scalp.Vec3{}, false
	}
	sqrt := math.Sqrt(disc)
	t1 := (-b - sqrt) / (2 * a)
	t2 := (-b + sqrt) / (2 * a)
	switch {
	case t1 > 1e-6:
		t = t1
	case t2 > 1e-6:
		t = t2
	default:
		return 0, scalp.Vec3{}, scalp.Vec3{}, false
	}
	point = origin.Add(dir.Scale(t))
	normal = scalp.Vec3{
		X: point.X / (s.A * s.A),
		Y: point.Y / (s.B * s.B),
		Z: point.Z / (s.C * s.C),
	}.Normalize()
	return t, point, normal, true
}

func colorColor(r, g, b float64) color.Color {
	return color.RGBA{
		uint8(clamp255(r * 255)),
		uint8(clamp255(g * 255)),
		uint8(clamp255(b * 255)),
		255,
	}
}

func clamp255(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
