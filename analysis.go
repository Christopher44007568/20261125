package main

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// EnergyReport stores linear-radiance sums over the shared hair-pixel mask.
type EnergyReport struct {
	HairPixels int
	Total      map[string]Vec3
	SpecEnergy map[string]float64
	Relative   map[string]Vec3 // relative to single scattering
	PairwiseL1 map[string]float64
	Centroid   map[string][2]float64
	CentroidPx map[string][2]float64
}

func buildEnergyReport(g *GBuffer, results []*RenderResult) *EnergyReport {
	hair := 0
	for _, p := range g.Pixel {
		if p.Hit == HitHair {
			hair++
		}
	}
	r := &EnergyReport{
		HairPixels: hair,
		Total:      map[string]Vec3{},
		SpecEnergy: map[string]float64{},
		Relative:   map[string]Vec3{},
		PairwiseL1: map[string]float64{},
		Centroid:   map[string][2]float64{},
		CentroidPx: map[string][2]float64{},
	}
	byName := map[string]*RenderResult{}
	for _, res := range results {
		byName[res.Name] = res
		var sum Vec3
		var specSum float64
		for i, p := range g.Pixel {
			if p.Hit == HitHair {
				sum = sum.Add(res.Linear[i])
				specSum += res.Spec[i]
			}
		}
		r.Total[res.Name] = sum
		r.SpecEnergy[res.Name] = specSum
	}
	base := r.Total["single"]
	for name, sum := range r.Total {
		r.Relative[name] = v(divSafe(sum.X, base.X), divSafe(sum.Y, base.Y), divSafe(sum.Z, base.Z))
	}

	names := []string{"single", "kajiya_kay", "double"}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			a, b := byName[names[i]], byName[names[j]]
			var l1 float64
			for k, p := range g.Pixel {
				if p.Hit == HitHair {
					l1 += absVec(a.Linear[k].Sub(b.Linear[k]))
				}
			}
			r.PairwiseL1[names[i]+"-"+names[j]] = l1
		}
	}

	for _, res := range results {
		cx, cy := highlightCentroid(g, res)
		r.Centroid[res.Name] = [2]float64{cx, cy}
		r.CentroidPx[res.Name] = [2]float64{
			float64(g.W) * 0.5 * (1 + cx/OrthoHalf),
			float64(g.H) * 0.5 * (1 - cy/OrthoHalf),
		}
	}
	return r
}

func divSafe(a, b float64) float64 {
	if math.Abs(b) < 1e-9 {
		return 0
	}
	return a / b
}

func absVec(a Vec3) float64 {
	return math.Abs(a.X) + math.Abs(a.Y) + math.Abs(a.Z)
}

// highlightMask selects pixels whose specular strength is in the top 12%.
func highlightMask(g *GBuffer, res *RenderResult) []bool {
	vals := make([]float64, 0, g.W*g.H/4)
	for i, p := range g.Pixel {
		if p.Hit == HitHair {
			vals = append(vals, res.Spec[i])
		}
	}
	sort.Float64s(vals)
	cut := vals[int(float64(len(vals))*0.88)]
	mask := make([]bool, len(g.Pixel))
	for i := range g.Pixel {
		mask[i] = g.Pixel[i].Hit == HitHair && res.Spec[i] >= cut && res.Spec[i] > 0
	}
	return mask
}

func highlightCentroid(g *GBuffer, res *RenderResult) (float64, float64) {
	mask := highlightMask(g, res)
	var sw, sx, sy float64
	for i, p := range g.Pixel {
		if !mask[i] {
			continue
		}
		w := res.Spec[i]
		sw += w
		sx += p.Pos.X * w
		sy += p.Pos.Y * w
	}
	if sw < 1e-9 {
		return 0, 0
	}
	return sx / sw, sy / sw
}

var overlayColors = map[string]color.RGBA{
	"single":     {255, 64, 64, 255},
	"kajiya_kay": {64, 255, 128, 255},
	"double":     {90, 150, 255, 255},
}

// buildOverlay draws the three highlight masks as additive color channels on a
// darkened copy of the single-scattering render; overlaps blend toward white.
func buildOverlay(g *GBuffer, results []*RenderResult) *image.RGBA {
	base := results[0]
	out := image.NewRGBA(image.Rect(0, 0, g.W, g.H))
	for i := range g.Pixel {
		x, y := i%g.W, i/g.W
		rc := base.Image.RGBAAt(x, y)
		out.SetRGBA(x, y, color.RGBA{rc.R / 3, rc.G / 3, rc.B / 3, 255})
	}
	masks := map[string][]bool{}
	for _, res := range results {
		masks[res.Name] = highlightMask(g, res)
	}
	for _, name := range []string{"double", "kajiya_kay", "single"} {
		col := overlayColors[name]
		mask := masks[name]
		for i := range mask {
			if !mask[i] {
				continue
			}
			x, y := i%g.W, i/g.W
			cur := out.RGBAAt(x, y)
			cur.R = uint8(math.Min(255, float64(cur.R)+0.85*float64(col.R)))
			cur.G = uint8(math.Min(255, float64(cur.G)+0.85*float64(col.G)))
			cur.B = uint8(math.Min(255, float64(cur.B)+0.85*float64(col.B)))
			out.SetRGBA(x, y, cur)
		}
	}
	return out
}
