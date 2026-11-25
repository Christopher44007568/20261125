package main

// baryQuad tests a screen point against the two triangles of a quad and
// returns interpolated uv, camera depth. Quads are small so a screen-space
// barycentric interpolation is sufficient for this software renderer.
func baryQuad(sx, sy, sz [4]float64, uvs [4][2]float64, px, py float64) (
	u, v, depth float64, ok bool) {
	tris := [][3]int{{0, 1, 2}, {0, 2, 3}}
	for _, t := range tris {
		i0, i1, i2 := t[0], t[1], t[2]
		ax, ay := sx[i0], sy[i0]
		bx, by := sx[i1], sy[i1]
		cx, cy := sx[i2], sy[i2]
		den := (by-cy)*(ax-cx) + (cx-bx)*(ay-cy)
		if den == 0 {
			continue
		}
		l1 := ((by-cy)*(px-cx) + (cx-bx)*(py-cy)) / den
		l2 := ((cy-ay)*(px-cx) + (ax-cx)*(py-cy)) / den
		l3 := 1 - l1 - l2
		if l1 < -0.001 || l2 < -0.001 || l3 < -0.001 {
			continue
		}
		l1, l2, l3 = clampf(l1, 0, 1), clampf(l2, 0, 1), clampf(l3, 0, 1)
		u = l1*uvs[i0][0] + l2*uvs[i1][0] + l3*uvs[i2][0]
		v = l1*uvs[i0][1] + l2*uvs[i1][1] + l3*uvs[i2][1]
		depth = l1*sz[i0] + l2*sz[i1] + l3*sz[i2]
		return u, v, depth, true
	}
	return 0, 0, 0, false
}
