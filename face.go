package main

import "math"

type ScreenVec struct{ X, Y float64 }

func screenOf(cam Camera, p Vec3, w, h int) (float64, float64, bool) {
	x, y, _, ok := cam.Project(p)
	if !ok {
		return 0, 0, false
	}
	return (x*0.5 + 0.5) * float64(w), (1 - (y*0.5 + 0.5)) * float64(h), true
}

func screenDelta(cam Camera, p, q Vec3, w, h int) (ScreenVec, bool) {
	x0, y0, ok0 := screenOf(cam, p, w, h)
	x1, y1, ok1 := screenOf(cam, q, w, h)
	if !ok0 || !ok1 {
		return ScreenVec{}, false
	}
	return ScreenVec{x1 - x0, y1 - y0}, true
}

func drawEllipse(c *Canvas, cam Camera, center, axisU, axisV Vec3, col RGB, alpha, feather float64) {
	cx, cy, ok := screenOf(cam, center, c.W, c.H)
	if !ok {
		return
	}
	u, okU := screenDelta(cam, center, center.Add(axisU), c.W, c.H)
	v, okV := screenDelta(cam, center, center.Add(axisV), c.W, c.H)
	if !okU || !okV {
		return
	}
	det := u.X*v.Y - u.Y*v.X
	if math.Abs(det) < 1e-6 {
		return
	}
	ext := math.Max(math.Hypot(u.X, u.Y), math.Hypot(v.X, v.Y)) + 2
	minX := max(0, int(cx-ext))
	maxX := min(c.W-1, int(cx+ext))
	minY := max(0, int(cy-ext))
	maxY := min(c.H-1, int(cy+ext))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5 - cx
			py := float64(y) + 0.5 - cy
			a := (px*v.Y - py*v.X) / det
			b := (u.X*py - u.Y*px) / det
			r := math.Hypot(a, b)
			cov := smoothstep(1+feather, 1-feather, r)
			if cov <= 0.001 {
				continue
			}
			i := y*c.W + x
			c.Pix[i] = over(c.Pix[i], premul(col, alpha*cov))
		}
	}
}

func drawTriangle(c *Canvas, cam Camera, p0, p1, p2 Vec3, col RGB, alpha, feather float64) {
	x0, y0, ok0 := screenOf(cam, p0, c.W, c.H)
	x1, y1, ok1 := screenOf(cam, p1, c.W, c.H)
	x2, y2, ok2 := screenOf(cam, p2, c.W, c.H)
	if !ok0 || !ok1 || !ok2 {
		return
	}
	minX := max(0, int(math.Floor(math.Min(x0, math.Min(x1, x2))-feather*3)))
	maxX := min(c.W-1, int(math.Ceil(math.Max(x0, math.Max(x1, x2))+feather*3)))
	minY := max(0, int(math.Floor(math.Min(y0, math.Min(y1, y2))-feather*3)))
	maxY := min(c.H-1, int(math.Ceil(math.Max(y0, math.Max(y1, y2))+feather*3)))
	area := (x1-x0)*(y2-y0) - (y1-y0)*(x2-x0)
	if math.Abs(area) < 1e-6 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			w0 := ((x1-px)*(y2-py) - (y1-py)*(x2-px)) / area
			w1 := ((x2-px)*(y0-py) - (y2-py)*(x0-px)) / area
			w2 := 1 - w0 - w1
			m := math.Min(w0, math.Min(w1, w2))
			cov := sat(m/math.Max(feather*0.08, 1e-4) + 0.5)
			if cov <= 0.001 {
				continue
			}
			i := y*c.W + x
			c.Pix[i] = over(c.Pix[i], premul(col, alpha*cov))
		}
	}
}

func drawFace(c *Canvas, cam Camera, scene *Scene) {
	head := scene.Prims[0]
	pow := math.Max(0.65, float64(c.H)/800)
	for _, idx := range []int{1, 2} {
		ear := scene.Prims[idx]
		center := ear.Center.Add(ear.Axes[2].Mul(ear.Radii.Z * 0.77))
		drawEllipse(c, cam,
			center, ear.Axes[0].Mul(0.132), ear.Axes[1].Mul(0.40),
			C(0.20, 0.075, 0.042), 0.67, 0.12)
		drawEllipse(c, cam,
			center.Add(ear.Axes[1].Mul(-0.035)), ear.Axes[0].Mul(0.074), ear.Axes[1].Mul(0.31),
			C(0.36, 0.13, 0.072), 0.48, 0.20)
	}

	eyeCenters := []Vec3{V(-0.345, 0.40, 0.815), V(0.345, 0.40, 0.815)}
	for _, center := range eyeCenters {
		n := center.Sub(head.Center).Norm()
		right := V(1, 0, 0).Sub(n.Mul(n.X)).Norm()
		up := n.Cross(right).Norm()
		view := cam.Pos.Sub(center).Norm()
		if n.Dot(view) < 0.08 {
			continue
		}
		drawEllipse(c, cam, center.Add(up.Mul(0.018)),
			right.Mul(0.128), up.Mul(0.098), C(0.035, 0.015, 0.008), 0.42, 0.34)
		drawEllipse(c, cam, center,
			right.Mul(0.112), up.Mul(0.083), C(0.012, 0.008, 0.006), 0.97, 0.09)
		drawEllipse(c, cam, center.Add(up.Mul(-0.006)),
			right.Mul(0.077), up.Mul(0.062), C(0.96, 0.42, 0.070), 0.98, 0.10)
		drawEllipse(c, cam, center.Add(up.Mul(-0.010)),
			right.Mul(0.038), up.Mul(0.050), C(0.008, 0.005, 0.003), 0.99, 0.10)
		drawEllipse(c, cam, center.Add(right.Mul(-0.035)).Add(up.Mul(0.035)),
			right.Mul(0.018*1.9), up.Mul(0.015*1.9), C(3.8, 3.2, 2.3), 0.96, 0.15)
		inner := center.Add(right.Mul(-0.078)).Add(up.Mul(-0.025))
		outer := center.Add(right.Mul(0.083)).Add(up.Mul(0.035))
		drawLineAA(c, cam, inner, outer, C(0.018, 0.010, 0.006), C(0.018, 0.010, 0.006), 0.75, 0.75, 2.3*pow)
	}

	nose := V(0, -0.175, 1.252)
	drawEllipse(c, cam, nose.Add(V(0, -0.035, -0.01)),
		V(0.145, 0, 0), V(0, 0.070, 0), C(0.04, 0.018, 0.011), 0.35, 0.35)
	p0 := nose.Add(V(-0.142, 0.045, 0))
	p1 := nose.Add(V(0.142, 0.045, 0))
	p2 := nose.Add(V(0, -0.105, 0.014))
	drawTriangle(c, cam, p0, p1, p2, C(0.028, 0.014, 0.011), 0.98, 1.2)
	drawEllipse(c, cam, nose.Add(V(-0.040, 0.022, 0.014)),
		V(0.048, 0, 0), V(0, 0.020, 0), C(0.46, 0.31, 0.23), 0.52, 0.26)
	drawLineAA(c, cam, nose.Add(V(0, -0.09, 0.014)), nose.Add(V(0, -0.155, 0.005)),
		C(0.045, 0.020, 0.012), C(0.045, 0.020, 0.012), 0.72, 0.45, 2.0*pow)
	drawLineAA(c, cam, nose.Add(V(0, -0.145, 0.005)), nose.Add(V(-0.075, -0.175, 0.002)),
		C(0.045, 0.020, 0.012), C(0.045, 0.020, 0.012), 0.35, 0.12, 1.5*pow)
	drawLineAA(c, cam, nose.Add(V(0, -0.145, 0.005)), nose.Add(V(0.075, -0.175, 0.002)),
		C(0.045, 0.020, 0.012), C(0.045, 0.020, 0.012), 0.35, 0.12, 1.5*pow)
}
