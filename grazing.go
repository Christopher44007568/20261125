package main

import (
	"image"
	"image/color"
	"math"
	"sort"
)

const (
	GW   = 400
	GH   = 280
	GFib = 26 // number of horizontal fibers
)

// GrazingStats summarizes how elongated a highlight is along the fiber axis.
type GrazingStats struct {
	Model      string
	CentroidX  float64
	CentroidY  float64
	SigmaX     float64
	SigmaY     float64
	Anisotropy float64 // sigmaX / sigmaY: >1 means highlight runs along fiber
}

// grazingContext builds a synthetic context for horizontal fibers (tangent x)
// wrapped on a cylinder whose axis is x; the camera looks along -z, so the
// fiber axis maps to the horizontal screen axis.
func grazingContext(px, py int) (ShadeContext, bool) {
	fx := (float64(px)/GW - 0.5) * 2.6
	fy := (float64(py)/GH - 0.5) * 1.6
	rowPhase0 := float64(py) / float64(GH) * GFib
	row := math.Round(rowPhase0)
	phase := 1.7 * row
	ny := math.Sin(fy)
	nz := math.Cos(fy)
	normal := v(0, ny, nz).Normalize()
	view := v(0, 0, 1)
	// A gentle wave along the fiber makes the grazing highlight travel along
	// its length instead of staying uniform across the row.
	tangent := v(1, 0.35*math.Cos(2.2*fx+phase), 0).Normalize()
	light := v(0.55, 0.66, 0.55).Normalize()
	h := view.Add(light).Normalize()
	thetaI := math.Asin(clampf(light.Dot(tangent), -1, 1))
	thetaH := math.Asin(clampf(h.Dot(tangent), -1, 1))
	u := view.Sub(tangent.Scale(view.Dot(tangent))).Normalize()
	w := tangent.Cross(u).Normalize()
	lp := light.Sub(tangent.Scale(light.Dot(tangent)))
	vp := view.Sub(tangent.Scale(view.Dot(tangent)))
	phi := wrapPi(math.Atan2(lp.Dot(w), lp.Dot(u)) -
		math.Atan2(vp.Dot(w), vp.Dot(u)))
	c := ShadeContext{
		View: view, Light: light, Tangent: tangent, Normal: normal,
		Shell: 0.12, H: h, ThetaI: thetaI, ThetaH: thetaH, Phi: phi,
		NDotL: normal.Dot(light),
	}
	// Only rows near a fiber center carry a fiber; the rest is dark backdrop.
	onFiber := math.Abs(rowPhase0-row) < 0.16
	return c, onFiber
}

func renderGrazingPanel(model string) (*image.RGBA, []float64, []bool) {
	img := image.NewRGBA(image.Rect(0, 0, GW, GH))
	spec := make([]float64, GW*GH)
	hit := make([]bool, GW*GH)
	for py := 0; py < GH; py++ {
		for px := 0; px < GW; px++ {
			i := py*GW + px
			img.SetRGBA(px, py, color.RGBA{18, 20, 24, 255})
			c, onFiber := grazingContext(px, py)
			if !onFiber {
				continue
			}
			var col Vec3
			var specVal float64
			switch model {
			case "kajiya_kay":
				col = shadeKajiyaKay(nil, &c)
				specVal = kajiyaGain * kajiyaSpec(&c)
			case "single":
				col = shadeSingleScatter(nil, &c)
				specVal = luminance(componentMul(marschnerSpec(&c, 1.0), singleGain))
			case "double":
				col = shadeDoubleScatter(nil, &c)
				specVal = luminance(componentMul(marschnerSpec(&c, 1.0), singleGain)) +
					luminance(componentMul(doubleGlobal(&c), globalGain))
			case "plastic":
				col = blinnPlastic(&c)
				specVal = math.Pow(clamp01(c.Normal.Dot(c.H)), 90)
			}
			hit[i] = true
			spec[i] = specVal
			img.SetRGBA(px, py, toneMap(col))
		}
	}
	return img, spec, hit
}

// blinnPlastic is the isotropic control: a sphere-style Blinn-Phong reflection.
func blinnPlastic(c *ShadeContext) Vec3 {
	ndh := clamp01(c.Normal.Dot(c.H))
	spec := math.Pow(ndh, 90)
	diff := clamp01(c.Normal.Dot(c.Light))
	return HairBase.Scale(AmbientMix + 0.8*diff).Add(HairSpec.Scale(spec))
}

func grazingStats(model string, spec []float64, hit []bool) GrazingStats {
	vals := make([]float64, 0, len(spec))
	for i := range hit {
		if hit[i] {
			vals = append(vals, spec[i])
		}
	}
	sort.Float64s(vals)
	cut := vals[int(float64(len(vals))*0.90)]
	// Per-fiber highlight centroid along the fiber axis (screen x).
	rowCenters := map[int]float64{}
	rowWeights := map[int]float64{}
	var sw, sx, sy, sxx, syy float64
	for i := range hit {
		if !hit[i] || spec[i] < cut || spec[i] <= 0 {
			continue
		}
		x := float64(i % GW)
		y := float64(i / GW)
		weight := spec[i]
		row := int(y / (float64(GH) / GFib))
		rowCenters[row] += x * weight
		rowWeights[row] += weight
		sw += weight
		sx += x * weight
		sy += y * weight
		sxx += x * x * weight
		syy += y * y * weight
	}
	centers := make([]float64, 0, len(rowCenters))
	for row, num := range rowCenters {
		centers = append(centers, num/rowWeights[row])
	}
	var cm, cm2 float64
	for _, c := range centers {
		cm += c
		cm2 += c * c
	}
	cm /= float64(len(centers))
	centerSpread := math.Sqrt(math.Max(0, cm2/float64(len(centers))-cm*cm))
	cx, cy := sx/sw, sy/sw
	sigy := math.Sqrt(math.Max(0, syy/sw-cy*cy))
	return GrazingStats{
		Model: model, CentroidX: cx, CentroidY: cy,
		SigmaX: centerSpread, SigmaY: sigy,
		// centerSpread = travel of the highlight along the fiber between
		// neighboring fibers; large = anisotropic, ~0 = plastic locked band.
		Anisotropy: centerSpread,
	}
}

// buildGrazingComposite places four labeled panels in one row.
func buildGrazingComposite() (*image.RGBA, []GrazingStats) {
	models := []string{"single", "kajiya_kay", "double", "plastic"}
	gap, labelH := 12, 26
	out := image.NewRGBA(image.Rect(0, 0, 4*GW+5*gap, GH+labelH+gap))
	stats := make([]GrazingStats, 0, 4)
	for k, m := range models {
		panel, spec, hit := renderGrazingPanel(m)
		stats = append(stats, grazingStats(m, spec, hit))
		ox := gap + k*(GW+gap)
		for y := 0; y < GH; y++ {
			for x := 0; x < GW; x++ {
				out.Set(ox+x, labelH+y, panel.RGBAAt(x, y))
			}
		}
		col := color.RGBA{235, 235, 235, 255}
		if m == "plastic" {
			col = color.RGBA{255, 170, 90, 255}
		}
		drawText(out, ox+4, 6, 2, modelLabel(m), col)
	}
	return out, stats
}

func modelLabel(m string) string {
	switch m {
	case "single":
		return "SINGLE SCATTER"
	case "kajiya_kay":
		return "KAJIYA KAY"
	case "double":
		return "DOUBLE SCATTER"
	case "plastic":
		return "PLASTIC CONTROL"
	}
	return m
}
