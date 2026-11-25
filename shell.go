package main

import "math"

const shellLayers = 64

const maxFurExpand = 0.220

func renderShell(scene *Scene, cam Camera, w, h int) *Canvas {
	out := NewCanvas(w, h)
	center, bound := scene.Bounds()
	bound += maxFurExpand + 0.02
	for y := 0; y < h; y++ {
		ny := 1 - 2*(float64(y)+0.5)/float64(h)
		for x := 0; x < w; x++ {
			nx := 2*(float64(x)+0.5)/float64(w) - 1
			ro, rd := cam.Ray(nx, ny)
			if !raySphere(ro, rd, center, bound) {
				continue
			}
			view := rd.Mul(-1)
			var acc HDR
			trans := 1.0
			depth := math.Inf(1)
			for k := 0; k < shellLayers; k++ {
				hShell := 1 - float64(k)/float64(shellLayers-1)
				expand := maxFurExpand * hShell
				hit, ok := scene.IntersectScaled(ro, rd, expand)
				if !ok {
					continue
				}
				idx := hit.PrimIdx
				if !scene.Prims[idx].Fur {
					continue
				}
				actual := furLength(idx, hit.Point)
				if expand > actual*1.015 && expand > 1e-4 {
					continue
				}
				hLocal := 0.0
				if actual > 1e-6 {
					hLocal = sat(expand / actual)
				}
				flow := groomFlow(idx, hit.Normal)
				a := shellLayerAlpha(idx, hLocal, hit.Point, hit.Normal, flow)
				if expand < 1e-4 {
					a = 0.985
					col := furColor(idx, 0.02, hit.Point, hit.Normal, view)
					col = col.Mul(0.72 + 0.28*fbm3(hit.Point, 35, 2))
					if hLocal > 0 {
						col = furColor(idx, hLocal, hit.Point, hit.Normal, view)
					}
					acc = acc.Add(premul(col.Mul(0.88), trans*a))
					trans *= 1 - a
					if depth == math.Inf(1) {
						depth = hit.T
					}
					break
				}
				if a < 0.002 {
					continue
				}
				if expand <= actual {
					a *= 0.55 + 0.45*smoothstep(1.0, 0.84, hLocal)
				}
				col := furColor(idx, hLocal, hit.Point, hit.Normal, view)
				col = col.Mul(0.88 + 0.16*smoothstep(0.72, 1.0, hLocal))
				acc = acc.Add(premul(col, trans*a))
				trans *= 1 - a
				if depth == math.Inf(1) {
					depth = hit.T
				}
				if trans < 0.006 {
					break
				}
			}
			if trans < 0.999 {
				out.Pix[y*out.W+x] = HDR{acc.R, acc.G, acc.B, 1 - trans}
				if depth < math.Inf(1) {
					out.Depth[y*out.W+x] = depth
				}
			}
		}
	}
	return out
}

func addBackground(fur *Canvas) *Canvas {
	out := NewCanvas(fur.W, fur.H)
	for y := 0; y < out.H; y++ {
		ny := 1 - 2*(float64(y)+0.5)/float64(out.H)
		for x := 0; x < out.W; x++ {
			nx := 2*(float64(x)+0.5)/float64(out.W) - 1
			bg := bgColor(nx, ny)
			p := fur.At(x, y)
			out.Pix[y*out.W+x] = over(HDR{bg.R, bg.G, bg.B, 1}, p)
		}
	}
	return out
}
