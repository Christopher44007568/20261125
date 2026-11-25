package mesh

import "math"

type obliqueCylinder struct {
	axis, u, v Vec3
	center     Vec3
	radius     float64
	top, bot   float64
}

func (c obliqueCylinder) radial(phi float64) Vec3 {
	return c.u.Mul(math.Cos(phi)).Add(c.v.Mul(math.Sin(phi))).Norm()
}

func (c obliqueCylinder) cut(radiusDir Vec3, y float64) Vec3 {
	q := c.center
	p := q.Add(radiusDir.Mul(c.radius))
	if math.Abs(c.axis.Y) > 1e-8 {
		p = p.Add(c.axis.Mul((y - p.Y) / c.axis.Y))
	}
	return p
}

// ObliqueHole builds a plate with a tilted through-hole. The two opening loops
// are true oblique-cylinder/plane intersections, not circles mapped to an
// inclined normal.
func ObliqueHole() *Mesh {
	m := &Mesh{}
	const halfX, halfZ = 62.0, 44.0
	const topY, botY = 24.0, 0.0
	const radius = 8.4
	const segments = 80

	angle := 28 * math.Pi / 180
	axis := V(math.Sin(angle), -math.Cos(angle), 0.16*math.Sin(angle)).Norm()
	center := V(7, topY, -3)
	u := axis.Cross(V(0, 1, 0)).Norm()
	v := axis.Cross(u).Norm()
	cyl := obliqueCylinder{axis: axis, u: u, v: v, center: center, radius: radius, top: topY, bot: botY}

	topInner := make([]Vec3, segments)
	botInner := make([]Vec3, segments)
	topOuter := make([]Vec3, segments)
	botOuter := make([]Vec3, segments)
	topC := V(center.X, topY, center.Z)
	botShift := axis.Mul((botY - topY) / axis.Y)
	botC := center.Add(botShift)
	botC.Y = botY

	for i := 0; i < segments; i++ {
		phi := 2 * math.Pi * float64(i) / segments
		rd := cyl.radial(phi)
		topInner[i] = cyl.cut(rd, topY)
		botInner[i] = cyl.cut(rd, botY)
		dx, dz := topInner[i].X-topC.X, topInner[i].Z-topC.Z
		topOuter[i] = boxRay(topC.X, topC.Z, dx, dz, halfX, halfZ)
		topOuter[i].Y = topY
		dx, dz = botInner[i].X-botC.X, botInner[i].Z-botC.Z
		botOuter[i] = boxRay(botC.X, botC.Z, dx, dz, halfX, halfZ)
		botOuter[i].Y = botY
	}

	up, down := V(0, 1, 0), V(0, -1, 0)
	for i := 0; i < segments; i++ {
		j := (i + 1) % segments
		m.AddQuad(topInner[i], topInner[j], topOuter[j], topOuter[i],
			up, up, up, up, up, Plane)
		m.AddQuad(botInner[j], botInner[i], botOuter[i], botOuter[j],
			down, down, down, down, down, Plane)

		rd := cyl.radial(2 * math.Pi * (float64(i) + 0.5) / segments)
		m.AddQuad(topInner[i], topInner[j], botInner[j], botInner[i],
			rd.Mul(-1), rd.Mul(-1), rd.Mul(-1), rd.Mul(-1), rd.Mul(-1), Hole)

		oxT := V(topOuter[i].X-topC.X, 0, topOuter[i].Z-topC.Z)
		oxB := V(topOuter[j].X-topC.X, 0, topOuter[j].Z-topC.Z)
		wantSide := oxT.Add(oxB).Norm()
		m.AddQuad(topOuter[i], topOuter[j], botOuter[j], botOuter[i],
			wantSide, wantSide, wantSide, wantSide, wantSide, Plane)
	}
	addBoss(m, -38, 27, topY, 11, 7.5, 40)
	return m
}

func addBoss(m *Mesh, cx, cz, baseY, height, radius float64, segments int) {
	top := baseY + height
	for i := 0; i < segments; i++ {
		j := (i + 1) % segments
		a0 := 2 * math.Pi * float64(i) / float64(segments)
		a1 := 2 * math.Pi * float64(j) / float64(segments)
		p0 := V(cx+radius*math.Cos(a0), baseY, cz+radius*math.Sin(a0))
		p1 := V(cx+radius*math.Cos(a1), baseY, cz+radius*math.Sin(a1))
		t0 := V(p0.X, top, p0.Z)
		t1 := V(p1.X, top, p1.Z)
		mid := a0 + (a1-a0)/2
		want := V(math.Cos(mid), 0, math.Sin(mid))
		m.AddQuad(p0, p1, t1, t0, want, want, want, want, want, Cylinder)
	}
	ring := make([]Vec3, segments)
	for i := 0; i < segments; i++ {
		a := 2 * math.Pi * float64(i) / float64(segments)
		ring[i] = V(cx+radius*math.Cos(a), top, cz+radius*math.Sin(a))
	}
	m.AddFan(V(cx, top, cz), ring, V(0, 1, 0), Plane)
}

func boxRay(cx, cz, dx, dz, halfX, halfZ float64) Vec3 {
	if math.Abs(dx) < 1e-9 && math.Abs(dz) < 1e-9 {
		return V(cx, 0, cz)
	}
	t := math.Inf(1)
	if math.Abs(dx) > 1e-9 {
		tx := math.Copysign(halfX, dx)
		t = math.Min(t, (tx-cx)/dx)
	}
	if math.Abs(dz) > 1e-9 {
		tz := math.Copysign(halfZ, dz)
		t = math.Min(t, (tz-cz)/dz)
	}
	if math.IsInf(t, 1) {
		t = 0
	}
	return V(cx+dx*t, 0, cz+dz*t)
}
