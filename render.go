package main

import (
	"math"
	"sort"
)

type RenderResult struct {
	Width       int
	Height      int
	Color       []Color
	Motion      []Vec2
	LayerMask   []Color
	MaxSpeed    float64
	LayerCount  int
	HairCount   int
	GlassCount  int
	FrameWeight float64
}

type FrameSample struct {
	X        int32
	Depth    float64
	LayerID  int32
	Kind     SurfaceKind
	Color    Color
	Alpha    float64
	Motion   Vec2
	Sequence uint32
}

type frameRows [][]FrameSample

type bestStamp struct {
	alpha float64
	depth float64
	u     float64
	vx    float64
	vy    float64
}

type renderer struct {
	scene    *Scene
	cfg      Config
	iw, ih   int
	ss       int
	bg       []Color
	glass    frameRows
	best     []bestStamp
	touched  []int
	tmp      []DeepSample
	timeSamp []float64
	weights  []float64
}

func Render(cfg Config) (*RenderResult, error) {
	scene := NewScene(cfg.Width, cfg.Height, cfg.Seed)
	r := &renderer{
		scene: scene, cfg: cfg,
		iw: cfg.Width * cfg.Supersample,
		ih: cfg.Height * cfg.Supersample,
		ss: cfg.Supersample,
	}
	r.bg = renderBackground(r.iw, r.ih, cfg.Width, cfg.Height)
	r.glass = r.buildGlassRows()
	r.best = make([]bestStamp, r.iw*r.ih)
	r.tmp = make([]DeepSample, 0, 64)
	r.makeTemporalSamples()

	outN := cfg.Width * cfg.Height
	colorAcc := make([]Color, outN)
	motionAcc := make([]Vec2, outN)
	weightAcc := make([]float64, outN)
	maskAcc := make([]Color, outN)
	maskWeight := make([]float64, outN)
	maxSpeed := 0.0
	centerIndex := -1

	for fi, t := range r.timeSamp {
		if centerIndex < 0 || math.Abs(t-cfg.FrameTime) < math.Abs(r.timeSamp[centerIndex]-cfg.FrameTime) {
			centerIndex = fi
		}
		rows := r.buildFrameRows(t)
		weight := r.weights[fi]
		for iy := 0; iy < r.ih; iy++ {
			hairPtr, glassPtr := 0, 0
			hairRow, glassRow := rows[iy], r.glass[iy]
			for ix := 0; ix < r.iw; ix++ {
				for hairPtr < len(hairRow) && int(hairRow[hairPtr].X) < ix {
					hairPtr++
				}
				for glassPtr < len(glassRow) && int(glassRow[glassPtr].X) < ix {
					glassPtr++
				}
				r.tmp = r.tmp[:0]
				for p := hairPtr; p < len(hairRow) && int(hairRow[p].X) == ix; p++ {
					r.tmp = append(r.tmp, toDeep(hairRow[p]))
				}
				for p := glassPtr; p < len(glassRow) && int(glassRow[p].X) == ix; p++ {
					r.tmp = append(r.tmp, toDeep(glassRow[p]))
				}
				sortDeep(r.tmp)
				col, motion := composite(r.tmp, r.bg[iy*r.iw+ix])
				oi := (iy/r.ss)*cfg.Width + ix/r.ss
				colorAcc[oi] = colorAcc[oi].Add(col.Mul(weight))
				weightAcc[oi] += weight
				if fi == centerIndex {
					motionAcc[oi] = motionAcc[oi].Add(motion)
					maskCol := compositeLayerMask(r.tmp, r.bg[iy*r.iw+ix])
					maskAcc[oi] = maskAcc[oi].Add(maskCol)
					maskWeight[oi]++
					if motion.Len() > maxSpeed {
						maxSpeed = motion.Len()
					}
				}
			}
		}
	}

	res := &RenderResult{
		Width: cfg.Width, Height: cfg.Height,
		Color: make([]Color, outN), Motion: make([]Vec2, outN),
		LayerMask: make([]Color, outN), MaxSpeed: maxSpeed,
		LayerCount: len(scene.Strands) + len(scene.Glass),
		HairCount:  len(scene.Strands), GlassCount: len(scene.Glass),
	}
	for i := 0; i < outN; i++ {
		if weightAcc[i] > 0 {
			res.Color[i] = colorAcc[i].Mul(1 / weightAcc[i])
		}
		if maskWeight[i] > 0 {
			res.LayerMask[i] = maskAcc[i].Mul(1 / maskWeight[i])
		}
		if centerIndex >= 0 {
			count := float64(cfg.Supersample * cfg.Supersample)
			res.Motion[i] = motionAcc[i].Mul(1 / count)
		}
	}
	return res, nil
}

func (r *renderer) makeTemporalSamples() {
	n := r.cfg.Samples
	if n < 1 {
		n = 1
	}
	r.timeSamp = make([]float64, n)
	r.weights = make([]float64, n)
	total := 0.0
	for i := 0; i < n; i++ {
		f := 0.0
		if n > 1 {
			f = float64(i) / float64(n-1)
		}
		t := r.cfg.FrameTime + (f-0.5)*r.cfg.Shutter
		r.timeSamp[i] = t
		w := 0.55 + 1.7*math.Pow(1-math.Abs(2*f-1), 2)
		r.weights[i] = w
		total += w
	}
	for i := range r.weights {
		r.weights[i] /= total
	}
}

func toDeep(s FrameSample) DeepSample {
	return DeepSample{
		Depth: s.Depth, LayerID: int(s.LayerID), Kind: s.Kind,
		Color: s.Color, Alpha: s.Alpha, Motion: s.Motion, Sequence: s.Sequence,
	}
}

func sortDeep(samples []DeepSample) {
	sort.SliceStable(samples, func(i, j int) bool {
		a, b := samples[i], samples[j]
		if a.Depth != b.Depth {
			return a.Depth < b.Depth
		}
		if a.Sequence != b.Sequence {
			return a.Sequence < b.Sequence
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.LayerID < b.LayerID
	})
}

func composite(samples []DeepSample, bg Color) (Color, Vec2) {
	col := bg
	motion := Vec2{}
	for _, s := range samples {
		a := clamp01(s.Alpha)
		if a <= 0 {
			continue
		}
		col = col.Lerp(s.Color, a)
		motion = motion.Lerp(s.Motion, a)
	}
	return col, motion
}
