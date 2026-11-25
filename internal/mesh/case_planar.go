package mesh

import (
	"math"
)

// NearPlanarPanel is a shallow forged cover. Its upper surface is intentionally
// within a few tenths of a millimetre of a plane while carrying three subtly
// different facet slopes. The weak normal breaks are the failure mode.
func NearPlanarPanel() *Mesh {
	m := &Mesh{}
	const sx, sz = 62.0, 44.0
	const nx, nz = 64, 46

	grid := make([][]Vec3, nz+1)
	for j := 0; j <= nz; j++ {
		grid[j] = make([]Vec3, nx+1)
		v := -1 + 2*float64(j)/nz
		z := v * sz
		for i := 0; i <= nx; i++ {
			u := -1 + 2*float64(i)/nx
			x := u * sx
			grid[j][i] = V(x, nearPlanarHeight(u, v), z)
		}
	}

	for j := 0; j < nz; j++ {
		for i := 0; i < nx; i++ {
			a, b, c, d := grid[j][i], grid[j][i+1], grid[j+1][i+1], grid[j+1][i]
			na, nb, nc, nd := topNormal(a), topNormal(b), topNormal(c), topNormal(d)
			m.AddQuad(a, d, c, b, na, nd, nc, nb, V(0, 1, 0), Plane)
		}
	}

	addDraftSides(m, grid, sx, sz, 0.075)
	m.AddQuad(
		V(-58, 0, -40), V(58, 0, -40), V(58, 0, 40), V(-58, 0, 40),
		V(0, -1, 0), V(0, -1, 0), V(0, -1, 0), V(0, -1, 0), V(0, -1, 0), Plane,
	)
	return m
}

func nearPlanarHeight(u, v float64) float64 {
	var h float64
	switch {
	case u < -0.35:
		h = 18.000 + 0.090*(u+1)
	case u < 0.35:
		h = 18.0585 - 0.070*(u+0.35)
	default:
		h = 18.0095 + 0.120*(u-0.35)
	}
	h += 0.025 * v
	h += 0.060 * math.Sin(2.2*math.Pi*u+0.3) * math.Cos(1.7*math.Pi*v)
	return h
}

func topNormal(p Vec3) Vec3 {
	const eps = 0.18
	u := p.X / 62
	v := p.Z / 44
	dx := (nearPlanarHeight(u+eps/62, v) - nearPlanarHeight(u-eps/62, v)) / (2 * eps)
	dz := (nearPlanarHeight(u, v+eps/44) - nearPlanarHeight(u, v-eps/44)) / (2 * eps)
	return V(-dx, 1, -dz).Norm()
}

func addDraftSides(m *Mesh, grid [][]Vec3, sx, sz, draft float64) {
	rows, cols := len(grid), len(grid[0])
	sideQuad := func(a, b Vec3, want Vec3) {
		ba, bb := draftPoint(a, sx, sz, draft), draftPoint(b, sx, sz, draft)
		n := want.Norm()
		m.AddQuad(a, b, bb, ba, n, n, n, n, n, Plane)
	}
	for i := 0; i < cols-1; i++ {
		a, b := grid[0][i], grid[0][i+1]
		sideQuad(a, b, V(0, 0, -1))
	}
	for i := 0; i < cols-1; i++ {
		a, b := grid[rows-1][i+1], grid[rows-1][i]
		sideQuad(a, b, V(0, 0, 1))
	}
	for j := 0; j < rows-1; j++ {
		a, b := grid[j+1][0], grid[j][0]
		sideQuad(a, b, V(-1, 0, 0))
	}
	for j := 0; j < rows-1; j++ {
		a, b := grid[j][cols-1], grid[j+1][cols-1]
		sideQuad(a, b, V(1, 0, 0))
	}
}

func draftPoint(p Vec3, sx, sz, draft float64) Vec3 {
	const bottomScale = 58.0 / 62.0
	_ = sx
	_ = sz
	return V(p.X*bottomScale, 0, p.Z*(40.0/44.0))
}
