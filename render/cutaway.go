package render

import (
	"image"
	"image/color"
	"sort"

	"brepsection/brep"
	"brepsection/geom"
	"brepsection/section"
)

// Camera is an orthographic projection looking toward target from eye.
type Camera struct {
	eye, target, right, up, forward geom.Vec3
}

// NewCamera builds an orthographic camera from an eye position and target.
func NewCamera(eye, target geom.Vec3) Camera {
	fwd := target.Sub(eye).Normalized()
	ref := geom.V3(0, 0, 1)
	if absF(fwd.Dot(ref)) > 0.95 {
		ref = geom.V3(1, 0, 0)
	}
	right := fwd.Cross(ref).Normalized()
	up := right.Cross(fwd).Normalized()
	return Camera{eye: eye, target: target, right: right, up: up, forward: fwd}
}

func (cam Camera) project(p geom.Vec3, center image.Point, scale float64) image.Point {
	d := p.Sub(cam.target)
	x := center.X + int(d.Dot(cam.right)*scale)
	y := center.Y - int(d.Dot(cam.up)*scale)
	return image.Point{X: x, Y: y}
}

func (cam Camera) depth(p geom.Vec3) float64 { return p.Sub(cam.eye).Dot(cam.forward) }

type faceDraw struct {
	pts   []image.Point
	depth float64
	col   color.Color
}

// Cutaway renders the solid with every planar face shaded and sorted back to
// front, the section loops overlaid in a strong red and the cutting plane as a
// translucent quad so the cut relationship reads clearly.
func Cutaway(b *brep.BRep, rep *section.Report, pl geom.Plane, w, h int) *Canvas {
	c := NewCanvas(w, h, palette.Bg)
	cam := NewCamera(geom.V3(14, -16, 11), b.AABB().Center())
	center := image.Point{X: w / 2, Y: h / 2}
	scale := fitScale(b, cam, w, h)

	var faces []faceDraw
	for idx, f := range b.Faces {
		pts := make([]image.Point, len(f.Outer))
		var dSum float64
		for i, vi := range f.Outer {
			pts[i] = cam.project(b.Points[vi], center, scale)
			dSum += cam.depth(b.Points[vi])
		}
		base := palette.FaceA
		if idx%2 == 0 {
			base = palette.FaceB
		}
		faces = append(faces, faceDraw{
			pts: pts, depth: dSum / float64(len(f.Outer)), col: shade(base, f.Normal, cam),
		})
	}
	sort.SliceStable(faces, func(i, j int) bool {
		return faces[i].depth < faces[j].depth
	})
	for _, fd := range faces {
		c.Polygon(fd.pts, fd.col)
		for k := 0; k < len(fd.pts); k++ {
			a, b := fd.pts[k], fd.pts[(k+1)%len(fd.pts)]
			c.Line(a.X, a.Y, b.X, b.Y, 1, palette.Edge)
		}
	}

	drawCuttingPlane(c, b, pl, cam, center, scale)
	drawSection3D(c, rep, pl, cam, center, scale)

	c.DrawText(16, 14, 2, "CUTAWAY "+rep.PlaneName, palette.Text)
	return c
}

func fitScale(b *brep.BRep, cam Camera, w, h int) float64 {
	box := b.AABB()
	corners := []geom.Vec3{
		geom.V3(box.Min.X, box.Min.Y, box.Min.Z),
		geom.V3(box.Max.X, box.Min.Y, box.Min.Z),
		geom.V3(box.Min.X, box.Max.Y, box.Min.Z),
		geom.V3(box.Max.X, box.Max.Y, box.Min.Z),
		geom.V3(box.Min.X, box.Min.Y, box.Max.Z),
		geom.V3(box.Max.X, box.Min.Y, box.Max.Z),
		geom.V3(box.Min.X, box.Max.Y, box.Max.Z),
		geom.V3(box.Max.X, box.Max.Y, box.Max.Z),
	}
	minX, maxX := 1e18, -1e18
	minY, maxY := 1e18, -1e18
	for _, p := range corners {
		d := p.Sub(cam.target)
		x := d.Dot(cam.right)
		y := d.Dot(cam.up)
		minX, maxX = minF(minX, x), maxF(maxX, x)
		minY, maxY = minF(minY, y), maxF(maxY, y)
	}
	sx := float64(w-80) / (maxX - minX)
	sy := float64(h-80) / (maxY - minY)
	if sx < sy {
		return sx
	}
	return sy
}

func shade(base color.Color, n geom.Vec3, cam Camera) color.Color {
	light := geom.V3(-0.4, -0.6, 0.8).Normalized()
	l := 0.55 + 0.45*absF(n.Dot(light))
	r, g, bl, _ := base.RGBA()
	return color.RGBA{
		R: uint8(float64(r>>8) * l),
		G: uint8(float64(g>>8) * l),
		B: uint8(float64(bl>>8) * l),
		A: 255,
	}
}

func drawCuttingPlane(c *Canvas, b *brep.BRep, pl geom.Plane, cam Camera, center image.Point, scale float64) {
	box := b.AABB().Expand(boxDiag(b) * 0.08)
	// Build a quad on the cutting plane spanning the projected bounding box.
	corners := []geom.Vec3{
		geom.V3(box.Min.X, box.Min.Y, 0),
		geom.V3(box.Max.X, box.Min.Y, 0),
		geom.V3(box.Max.X, box.Max.Y, 0),
		geom.V3(box.Min.X, box.Max.Y, 0),
	}
	var onPlane []image.Point
	for _, q := range corners {
		// Project each XY corner onto the cutting plane along the normal.
		d := pl.SignedDistance(q)
		moved := q.Sub(pl.Normal.Scale(d))
		onPlane = append(onPlane, cam.project(moved, center, scale))
	}
	for _, px := range onPlane {
		c.Blend(px.X, px.Y, palette.Plane, 0.12)
	}
	// Translucent fill via per-pixel alpha inside the projected quad.
	blendPolygon(c, onPlane, palette.Plane, 0.16)
	for k := 0; k < 4; k++ {
		a, b2 := onPlane[k], onPlane[(k+1)%4]
		c.Line(a.X, a.Y, b2.X, b2.Y, 1, color.RGBA{180, 130, 40, 255})
	}
}

func blendPolygon(c *Canvas, pts []image.Point, col color.Color, alpha float64) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts[1:] {
		minY, maxY = minInt(minY, p.Y), maxInt(maxY, p.Y)
	}
	for y := minY; y <= maxY; y++ {
		var xs []int
		for i := 0; i < len(pts); i++ {
			a, b := pts[i], pts[(i+1)%len(pts)]
			if (a.Y > y) != (b.Y > y) {
				xs = append(xs, a.X+(y-a.Y)*(b.X-a.X)/(b.Y-a.Y))
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			lo, hi := xs[i], xs[i+1]
			if lo > hi {
				lo, hi = hi, lo
			}
			for x := lo; x <= hi; x++ {
				c.Blend(x, y, col, alpha)
			}
		}
	}
}

func drawSection3D(c *Canvas, rep *section.Report, pl geom.Plane, cam Camera, center image.Point, scale float64) {
	for _, l := range rep.Loops {
		for _, seg := range l.Segments3D {
			a := cam.project(seg.A, center, scale)
			b := cam.project(seg.B, center, scale)
			c.Line(a.X, a.Y, b.X, b.Y, 3, palette.Section)
		}
	}
}

func boxDiag(b *brep.BRep) float64 { return b.AABB().Diagonal() }

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
