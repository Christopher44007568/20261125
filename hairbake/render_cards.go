package main

import "math"

// zBuffer tracks per-pixel camera depth for card rasterization.
type zBuffer struct{ d []float64 }

func newZBuffer(n int) *zBuffer {
	z := &zBuffer{d: make([]float64, n)}
	for i := range z.d {
		z.d[i] = math.Inf(1)
	}
	return z
}

// renderCards rasterizes the ribbon cards, sampling the baked atlases.
func renderCards(im *RGBA, cam Camera, cards []Card, atlas *Atlas, seamOverlay bool) *zBuffer {
	z := newZBuffer(im.W * im.H)
	for ci := range cards {
		card := &cards[ci]
		for j := 0; j < len(card.Center)-1; j++ {
			c0, c1 := card.Center[j], card.Center[j+1]
			a0 := card.Across[j]
			a1 := card.Across[j]
			w0, w1 := card.HalfW[j], card.HalfW[j+1]
			quad := [4]Vec3{
				c0.Add(a0.Scale(-w0)), c0.Add(a0.Scale(w0)),
				c1.Add(a1.Scale(w1)), c1.Add(a1.Scale(-w1)),
			}
			uvs := [4][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
			rasterQuad(im, z, cam, quad, uvs, card, atlas, j, seamOverlay)
		}
	}
	return z
}

func rasterQuad(im *RGBA, z *zBuffer, cam Camera, q [4]Vec3, uvs [4][2]float64,
	card *Card, atlas *Atlas, knot int, seamOverlay bool) {
	sx := [4]float64{}
	sy := [4]float64{}
	sz := [4]float64{}
	minX, minY, maxX, maxY := im.W, im.H, 0, 0
	for i := 0; i < 4; i++ {
		x, y, d := cam.project(q[i], im.W, im.H)
		if d < 0.05 {
			return
		}
		sx[i], sy[i], sz[i] = x, y, d
		minX = min(minX, int(math.Floor(x)))
		maxX = max(maxX, int(math.Ceil(x)))
		minY = min(minY, int(math.Floor(y)))
		maxY = max(maxY, int(math.Ceil(y)))
	}
	minX, minY = max(minX, 0), max(minY, 0)
	maxX, maxY = min(maxX, im.W-1), min(maxY, im.H-1)

	tan := card.Center[min(knot+1, len(card.Center)-1)].Sub(card.Center[knot]).Norm()
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			fx := float64(px) + 0.5
			fy := float64(py) + 0.5
			u, v, depth, ok := baryQuad(sx, sy, sz, uvs, fx, fy)
			if !ok {
				continue
			}
			pi := py*im.W + px
			if depth >= z.d[pi] {
				continue
			}
			tex := sampleAtlas(card.Rect, atlas, u, v)
			if tex[3] < 0.02 {
				continue
			}
			// Flow tangent perturbs the specular direction per pixel.
			flowT := decodeFlow(card, atlas, card.Rect, u, v, tan)
			pxShade := hairShade(flowT, cam.View.Neg(), keyLight,
				card.SpecMean*(0.6+0.8*tex[4]))
			col := [3]float64{tex[0] * pxShade, tex[1] * pxShade, tex[2] * pxShade}
			im.blend(px, py, [4]float64{col[0], col[1], col[2], tex[3] * 0.99})
			z.d[pi] = depth
		}
	}
}

// sampleAtlas returns rgb, alpha, spec from the two atlases at island uv.
func sampleAtlas(r AtlasRect, a *Atlas, u, v float64) [5]float64 {
	px := clampi(r.X+int(u*float64(r.W-1)), r.X, r.X+r.W-1)
	py := clampi(r.Y+int(v*float64(r.H-1)), r.Y, r.Y+r.H-1)
	ci := (py*a.Size + px) * 4
	return [5]float64{
		float64(a.RGBA[ci+0]) / 255,
		float64(a.RGBA[ci+1]) / 255,
		float64(a.RGBA[ci+2]) / 255,
		float64(a.RGBA[ci+3]) / 255,
		float64(a.Flow[ci+2]) / 255,
	}
}

func decodeFlow(card *Card, a *Atlas, r AtlasRect, u, v float64, base Vec3) Vec3 {
	px := clampi(r.X+int(u*float64(r.W-1)), r.X, r.X+r.W-1)
	py := clampi(r.Y+int(v*float64(r.H-1)), r.Y, r.Y+r.H-1)
	ci := (py*a.Size + px) * 4
	tx := float64(a.Flow[ci+0])/255*2 - 1
	j := clampi(int(v*float64(len(card.Center)-1)), 0, len(card.Center)-1)
	return card.Center[min(j+1, len(card.Center)-1)].Sub(card.Center[j]).Norm().
		Add(card.Across[j].Scale(tx * 0.35)).Norm()
}

// drawCardSeams projects the two long edges of every ribbon and dashes them,
// exposing where adjacent card islands meet on the silhouette.
func drawCardSeams(im *RGBA, cam Camera, cards []Card) {
	yellow := [3]float64{1, 0.8, 0.15}
	for ci := range cards {
		card := &cards[ci]
		left := make([][2]float64, 0, len(card.Center))
		right := make([][2]float64, 0, len(card.Center))
		for j := range card.Center {
			pl := card.Center[j].Add(card.Across[j].Scale(-card.HalfW[j]))
			pr := card.Center[j].Add(card.Across[j].Scale(card.HalfW[j]))
			xl, yl, zl := cam.project(pl, im.W, im.H)
			xr, yr, zr := cam.project(pr, im.W, im.H)
			if zl > 0.05 {
				left = append(left, [2]float64{xl, yl})
			}
			if zr > 0.05 {
				right = append(right, [2]float64{xr, yr})
			}
		}
		stroke := func(pts [][2]float64) {
			for k := 0; k+1 < len(pts); k++ {
				im.drawDashedLine(pts[k][0], pts[k][1], pts[k+1][0], pts[k+1][1], yellow, 0.55)
			}
		}
		stroke(left)
		stroke(right)
	}
}
