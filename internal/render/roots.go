package render

import (
	"image/color"
	"math"

	"hairroots/internal/poisson"
	"hairroots/internal/scalp"
)

// DrawRoots stamps every visible root. Roots are depth-tested against the
// painted scalp, so back-side follicles stay hidden in both perspective and
// side views while local point spacing still reveals the density field.
func DrawRoots(c *Canvas, cam *Camera, s *scalp.Scalp, roots []poisson.Root) {
	hot := color.RGBA{R: 255, G: 168, B: 60, A: 255}
	cool := color.RGBA{R: 235, G: 96, B: 70, A: 255}
	for _, rt := range roots {
		// Back-face culling: a root is visible only when its outward normal
		// points toward the camera, which also removes roots on the far side.
		if s.Normal(rt.Theta, rt.Phi).Dot(cam.Forward) <= 0 {
			continue
		}
		// Visibility test: the ray from the camera must reach the root before
		// any other point of the ellipsoid, and the hit must stay inside the
		// hairline. This keeps roots glued to the rendered scalp patch.
		if !cam.RootVisible(rt.Position, rt.Theta, rt.Phi, s) {
			continue
		}
		x, y, z, ok := cam.Project(rt.Position, c)
		if !ok {
			continue
		}
		t := (rt.Weight - 0.55) / 0.85 // weight span seen in practice
		col := mixColor(cool, hot, clamp01(t))
		c.Splat(x, y, 2.6, z-1e-4, col)
	}
}

func mixColor(a, b color.RGBA, t float64) color.Color {
	return color.RGBA{
		R: uint8(float64(a.R) + t*float64(b.R-a.R)),
		G: uint8(float64(a.G) + t*float64(b.G-a.G)),
		B: uint8(float64(a.B) + t*float64(b.B-a.B)),
		A: 255,
	}
}

// DrawFlowGlyphs renders a downsampled set of hair direction vectors in a
// top-down orthographic view: each glyph is the projected world-space flow
// vector, scaled to a fixed pixel length for readability.
func DrawFlowGlyphs(c *Canvas, cam *Camera, s *scalp.Scalp, roots []poisson.Root, stride int) {
	head := color.RGBA{R: 238, G: 240, B: 245, A: 255}
	glyphLen := 13.0
	for k, rt := range roots {
		if k%stride != 0 {
			continue
		}
		// Keep only glyphs on reasonably frontal, directly visible surface so
		// projected arrows are not compressed near the silhouette.
		n := s.Normal(rt.Theta, rt.Phi)
		if n.Dot(cam.Forward) <= 0.35 || !cam.RootVisible(rt.Position, rt.Theta, rt.Phi, s) {
			continue
		}
		tip := rt.Position.Add(rt.FlowWorld.Scale(0.12))
		x0, y0, _, ok0 := cam.Project(rt.Position, c)
		x1, y1, _, ok1 := cam.Project(tip, c)
		if !ok0 || !ok1 {
			continue
		}
		dx, dy := x1-x0, y1-y0
		mag := math.Hypot(float64(dx), float64(dy))
		if mag < 1 {
			continue
		}
		x1 = x0 + int(math.Round(float64(dx)/mag*glyphLen))
		y1 = y0 + int(math.Round(float64(dy)/mag*glyphLen))
		c.Line(x0, y0, x1, y1, head)
		drawArrowHead(c, x0, y0, x1, y1, head)
	}
}

func drawArrowHead(c *Canvas, x0, y0, x1, y1 int, col color.Color) {
	angle := math.Atan2(float64(y1-y0), float64(x1-x0))
	for _, da := range []float64{2.6, -2.6} {
		ax := x1 + int(math.Round(4*math.Cos(angle+da)))
		ay := y1 + int(math.Round(4*math.Sin(angle+da)))
		c.Line(x1, y1, ax, ay, col)
	}
	_ = x0
	_ = y0
}
