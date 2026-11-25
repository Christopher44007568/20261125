package main

import "math"

func renderUndercoat(scene *Scene, cam Camera, w, h int) *Canvas {
	out := NewCanvas(w, h)
	center, bound := scene.Bounds()
	for y := 0; y < h; y++ {
		ny := 1 - 2*(float64(y)+0.5)/float64(h)
		for x := 0; x < w; x++ {
			nx := 2*(float64(x)+0.5)/float64(w) - 1
			ro, rd := cam.Ray(nx, ny)
			if !raySphere(ro, rd, center, bound) {
				continue
			}
			hit, ok := scene.Intersect(ro, rd)
			if !ok || !scene.Prims[hit.PrimIdx].Fur {
				continue
			}
			view := rd.Mul(-1)
			p := hit.Point
			col := furColor(hit.PrimIdx, 0.10, p, hit.Normal, view)
			fib := fbm3(p.Add(V(2.4, 8.1, 0.7)), 44, 2)
			col = col.Mul(0.52 + 0.48*(0.55+0.45*fib))
			key, fill, _ := surfaceLight(hit.Normal, view)
			edge := math.Pow(1-sat(hit.Normal.Dot(view)), 2.5)
			col = col.Add(furStyle(hit.PrimIdx).Mid.Mul(edge * (0.20 + key*0.30 + fill*0.12)))
			out.Pix[y*out.W+x] = premul(col, 1)
			out.Depth[y*out.W+x] = hit.T
		}
	}
	return out
}
