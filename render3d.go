package main

import (
	"fmt"
	"math"
)

type Camera struct {
	Eye, Target, Up  Vec3
	Right, UpView    Vec3
	Forward          Vec3
	Focal            float64
	ScreenX, ScreenY float64
}

func newCamera(eye, target Vec3, fovDeg, screenX, screenY float64) Camera {
	forward := target.Sub(eye).Unit()
	right := forward.Cross(Vec3{0, 0, 1}).Unit()
	up := right.Cross(forward).Unit()
	focal := 380 / math.Tan(fovDeg*math.Pi/360)
	return Camera{
		Eye: eye, Target: target, Up: Vec3{0, 0, 1},
		Right: right, UpView: up, Forward: forward,
		Focal: focal, ScreenX: screenX, ScreenY: screenY,
	}
}

func (c Camera) Project(p Vec3) (Vec2, float64, bool) {
	rel := p.Sub(c.Eye)
	depth := rel.Dot(c.Forward)
	if depth < 0.05 {
		return Vec2{}, 0, false
	}
	x := rel.Dot(c.Right) / depth * c.Focal
	y := rel.Dot(c.UpView) / depth * c.Focal
	return Vec2{c.ScreenX + x, c.ScreenY - y}, depth, true
}

type SurfaceTriangle struct {
	P        [3]Vec3
	Inverted bool
}

func surfaceTriangles(profile Profile, inner bool) []SurfaceTriangle {
	path := profile.Outer
	if inner {
		path = profile.Inner
	}
	inv := make([]bool, len(path)-1)
	if inner {
		inv = innerWallInverted(profile)
	}
	z0, z1 := 0.0, profile.Params.Height
	out := make([]SurfaceTriangle, 0, 2*(len(path)-1))
	for i := 0; i+1 < len(path); i++ {
		a := Vec3{path[i].X, path[i].Y, z0}
		b := Vec3{path[i+1].X, path[i+1].Y, z0}
		c := Vec3{path[i+1].X, path[i+1].Y, z1}
		d := Vec3{path[i].X, path[i].Y, z1}
		out = append(out,
			SurfaceTriangle{P: [3]Vec3{a, b, c}, Inverted: inv[i]},
			SurfaceTriangle{P: [3]Vec3{a, c, d}, Inverted: inv[i]},
		)
	}
	return out
}

func renderSurface(profile Profile, inner bool) *Canvas {
	canvas := NewCanvas(1000, 760, 2)
	canvas.Fill(bgColor)
	center := Vec3{-9.6, 9.6, 5.8}
	eye := center.Add(Vec3{44, -44, 22})
	cam := newCamera(eye, center, 33, 515, 450)
	tris := surfaceTriangles(profile, inner)
	zbuf := make([]float64, canvas.iw()*canvas.ih())
	for i := range zbuf {
		zbuf[i] = math.Inf(1)
	}
	base := Color{55, 115, 188}
	if inner {
		base = Color{184, 112, 24}
	}
	for _, tri := range tris {
		buf := base
		if tri.Inverted {
			buf = redColor
		}
		rasterizeDepth(canvas, zbuf, cam, tri.P, buf)
	}
	drawSurfaceCard(canvas, inner, profile.Params)
	return canvas.Downsample()
}

func rasterizeDepth(canvas *Canvas, zbuf []float64, cam Camera, p [3]Vec3,
	base Color) {
	var screen [3]Vec2
	var depth [3]float64
	for i := range p {
		var ok bool
		screen[i], depth[i], ok = cam.Project(p[i])
		if !ok {
			return
		}
	}
	ss := float64(canvas.SS)
	var q [3]Vec2
	for i := range screen {
		q[i] = Vec2{screen[i].X * ss, screen[i].Y * ss}
	}
	area := cross2(q[0], q[1], q[2])
	if math.Abs(area) < 1e-8 {
		return
	}
	minX := int(math.Floor(math.Min(q[0].X, math.Min(q[1].X, q[2].X))))
	maxX := int(math.Ceil(math.Max(q[0].X, math.Max(q[1].X, q[2].X))))
	minY := int(math.Floor(math.Min(q[0].Y, math.Min(q[1].Y, q[2].Y))))
	maxY := int(math.Ceil(math.Max(q[0].Y, math.Max(q[1].Y, q[2].Y))))
	if maxX < 0 || maxY < 0 || minX >= canvas.iw() || minY >= canvas.ih() {
		return
	}

	normal := p[1].Sub(p[0]).Cross(p[2].Sub(p[0])).Unit()
	viewDir := p[0].Sub(cam.Eye).Unit()
	if normal.Dot(viewDir) > 0 {
		normal = normal.Scale(-1)
	}
	light := Vec3{-0.45, -0.25, 1}.Unit()
	diffuse := math.Abs(normal.Dot(light))
	shade := 0.76 + 0.32*diffuse
	col := shadeColor(base, shade)

	invZ := [3]float64{1 / depth[0], 1 / depth[1], 1 / depth[2]}
	for y := minY; y <= maxY; y++ {
		if y < 0 || y >= canvas.ih() {
			continue
		}
		for x := minX; x <= maxX; x++ {
			if x < 0 || x >= canvas.iw() {
				continue
			}
			pt := Vec2{float64(x) + 0.5, float64(y) + 0.5}
			w0 := cross2(q[1], q[2], pt) / area
			w1 := cross2(q[2], q[0], pt) / area
			w2 := cross2(q[0], q[1], pt) / area
			if w0 < -1e-7 || w1 < -1e-7 || w2 < -1e-7 {
				continue
			}
			iz := w0*invZ[0] + w1*invZ[1] + w2*invZ[2]
			if iz <= 0 {
				continue
			}
			z := 1 / iz
			idx := y*canvas.iw() + x
			if z < zbuf[idx] {
				zbuf[idx] = z
				canvas.SetInternal(x, y, col)
			}
		}
	}
}

func shadeColor(c Color, f float64) Color {
	return Color{
		R: uint8(clamp(float64(c.R)*f, 0, 255)),
		G: uint8(clamp(float64(c.G)*f, 0, 255)),
		B: uint8(clamp(float64(c.B)*f, 0, 255)),
	}
}

func drawSurfaceCard(c *Canvas, inner bool, p Params) {
	c.FillRect(24, 24, 624, 116, panelColor)
	swatch := outerColor
	title := "OUTER SURFACE"
	line2 := "CLEAR CONVEX OFFSET SKIN"
	if inner {
		swatch = innerColor
		title = "INNER SURFACE"
		line2 = "RED = INVERTED OFFSET FACES"
	}
	c.FillRect(48, 42, 88, 82, swatch)
	if inner {
		c.FillRect(94, 42, 126, 82, redColor)
	}
	c.Text(146, 36, title, 2, textColor)
	c.Text(146, 64, line2, 1, mutedColor)
	c.Text(146, 84, fmt.Sprintf("D=%.2f MM", p.Offset), 1, textColor)
}
