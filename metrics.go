package main

import (
	"math"
	"sort"
)

type pixelStat struct {
	mean, m2, prev float64
	count          int
	flickerEvents  int
	ts             []float64
}

type EyeMetrics struct {
	Eye           string
	Frames        int
	FlickerPixels int
	PeriphFlicker int
	MaxLumVar     float64
	MeanLumVar    float64
	PeriphVar     float64
	RMSE          float64
	Sharpness     float64
	RelSharpness  float64
	TopPixels     []PixelSeries
}

type PixelSeries struct {
	X, Y int
	Lum  []float64
	Var  float64
}

type analyzer struct {
	cfg     Config
	eye     string
	stats   []pixelStat
	gt      []*Frame
	work    []*Frame
	sse     float64
	samples int
	grad    float64
	gradN   int
}

func newAnalyzer(cfg Config, eye string) *analyzer {
	n := cfg.W * cfg.H
	return &analyzer{
		cfg: cfg, eye: eye,
		stats: make([]pixelStat, n),
	}
}

func (a *analyzer) isPeriphery(x, y int) bool {
	cx := float64(a.cfg.W-1) / 2
	cy := float64(a.cfg.H-1) / 2
	return math.Hypot(float64(x)-cx, float64(y)-cy) >= a.cfg.midRadius()
}

func (a *analyzer) addFrame(frame int, gt, f *Frame) {
	a.gt = append(a.gt, gt)
	a.work = append(a.work, f)
	for y := 0; y < a.cfg.H; y++ {
		for x := 0; x < a.cfg.W; x++ {
			i := f.index(x, y)
			l := luminance(f.Pix[i])
			st := &a.stats[i]
			if st.count > 0 && math.Abs(l-st.prev) > a.cfg.Tau {
				st.flickerEvents++
			}
			st.prev = l
			st.count++
			delta := l - st.mean
			st.mean += delta / float64(st.count)
			st.m2 += delta * (l - st.mean)

			g := f.Pix[i]
			t := gt.Pix[i]
			d := Color{g.R - t.R, g.G - t.G, g.B - t.B}
			a.sse += d.R*d.R + d.G*d.G + d.B*d.B
			a.samples++
		}
	}
	for y := 1; y < a.cfg.H; y++ {
		for x := 1; x < a.cfg.W; x++ {
			g := math.Abs(luminance(f.Pix[f.index(x, y)]) - luminance(f.Pix[f.index(x-1, y)]))
			g += math.Abs(luminance(f.Pix[f.index(x, y)]) - luminance(f.Pix[f.index(x, y-1)]))
			a.grad += g
			a.gradN++
		}
	}
}

func (a *analyzer) varianceAt(i int) float64 {
	if a.stats[i].count < 2 {
		return 0
	}
	return a.stats[i].m2 / float64(a.stats[i].count-1)
}

func (a *analyzer) finalize(gtSharpness float64) EyeMetrics {
	m := EyeMetrics{Eye: a.eye, Frames: len(a.work)}
	var varSum, periphSum float64
	periphN := 0
	type ranked struct {
		i  int
		v  float64
		ev int
	}
	var ranks []ranked
	for y := 0; y < a.cfg.H; y++ {
		for x := 0; x < a.cfg.W; x++ {
			i := y*a.cfg.W + x
			v := a.varianceAt(i)
			varSum += v
			if v > m.MaxLumVar {
				m.MaxLumVar = v
			}
			if a.stats[i].flickerEvents > 0 {
				m.FlickerPixels++
				if a.isPeriphery(x, y) {
					m.PeriphFlicker++
				}
				ranks = append(ranks, ranked{i, v, a.stats[i].flickerEvents})
			}
			if a.isPeriphery(x, y) {
				periphSum += v
				periphN++
			}
		}
	}
	n := a.cfg.W * a.cfg.H
	m.MeanLumVar = varSum / float64(n)
	m.PeriphVar = periphSum / float64(max(periphN, 1))
	m.RMSE = math.Sqrt(a.sse / float64(max(a.samples, 1)))
	m.Sharpness = a.grad / float64(max(a.gradN, 1))
	if gtSharpness > 0 {
		m.RelSharpness = m.Sharpness / gtSharpness
	}

	sort.Slice(ranks, func(p, q int) bool { return ranks[p].v > ranks[q].v })
	for k := 0; k < len(ranks) && k < 8; k++ {
		r := ranks[k]
		ps := PixelSeries{X: r.i % a.cfg.W, Y: r.i / a.cfg.W, Var: r.v}
		for _, f := range a.work {
			ps.Lum = append(ps.Lum, luminance(f.Pix[r.i]))
		}
		m.TopPixels = append(m.TopPixels, ps)
	}
	return m
}

func interEyeResidual(cfg Config, lf, rf, lgt, rgt []*Frame) float64 {
	var sum float64
	n := 0
	for t := range lf {
		for i := range lf[t].Pix {
			dl := luminance(lf[t].Pix[i]) - luminance(lgt[t].Pix[i])
			dr := luminance(rf[t].Pix[i]) - luminance(rgt[t].Pix[i])
			sum += math.Abs(dl - dr)
			n++
		}
	}
	return sum / float64(max(n, 1))
}

func interEyeDivergence(cfg Config, lf, rf []*Frame) float64 {
	var sum float64
	n := 0
	for t := range lf {
		for i := range lf[t].Pix {
			sum += math.Abs(luminance(lf[t].Pix[i]) - luminance(rf[t].Pix[i]))
			n++
		}
	}
	return sum / float64(max(n, 1))
}

func blurFalseStability(strategy string, rmse, gtSharp, sharp, periphVar, baseVar float64) bool {
	if strategy != stratBlurred || gtSharp == 0 {
		return false
	}
	relSharp := sharp / gtSharp
	if relSharp >= 0.8 {
		return false
	}
	if rmse <= 0 {
		return false
	}
	return baseVar <= 0 || periphVar < baseVar
}

type SceneResult struct {
	Scene, Motion, Strategy string
	Eyes                    []EyeMetrics
	InterEyeResidual        float64
	FalseStability          bool
}
