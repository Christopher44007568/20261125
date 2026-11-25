package main

import "sort"

// renderStrands draws the full strand groom (the ground-truth reference).
// Strands are depth sorted back->front and drawn with anisotropic shading.
func renderStrands(im *RGBA, cam Camera, g *Groom, highlight map[int]bool) {
	type item struct{ idx, depth int }
	items := make([]item, 0, len(g.Strands))
	depths := make([]float64, len(g.Strands))
	for i, s := range g.Strands {
		_, _, z := cam.project(s.Root, im.W, im.H)
		depths[i] = z
		items = append(items, item{i, int(z * 100)})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return depths[items[i].idx] > depths[items[j].idx]
	})
	for _, it := range items {
		s := &g.Strands[it.idx]
		col := s.Color
		if highlight != nil && highlight[s.ID] {
			col = [3]float64{0.95, 0.28, 0.2}
		}
		for j := 0; j < len(s.Pts)-1; j++ {
			p0, p1 := s.Pts[j], s.Pts[j+1]
			x0, y0, _ := cam.project(p0, im.W, im.H)
			x1, y1, _ := cam.project(p1, im.W, im.H)
			if x0 < 0 || x1 < 0 {
				continue
			}
			tan := p1.Sub(p0).Norm()
			shade := hairShade(tan, cam.View.Neg(), keyLight, s.Spec)
			c := [3]float64{
				col[0] * shade,
				col[1] * shade,
				col[2] * shade,
			}
			alpha := 0.72
			if s.Flyaway {
				alpha = 0.55
			}
			if highlight != nil && highlight[s.ID] {
				alpha = 0.95
			}
			im.drawLine(x0, y0, x1, y1, c, alpha, 1.3)
		}
	}
}

func (v Vec3) Neg() Vec3 { return Vec3{-v.X, -v.Y, -v.Z} }
