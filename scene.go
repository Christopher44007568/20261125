package main

import (
	"math"
	"math/rand"
)

type SurfaceKind uint8

const (
	KindHair SurfaceKind = iota + 1
	KindGlass
	KindSkin
	KindInk
)

type DeepSample struct {
	Depth    float64
	LayerID  int
	Kind     SurfaceKind
	Color    Color
	Alpha    float64
	Motion   Vec2
	Sequence uint32
}

type Strand struct {
	ID         int
	Root       Vec2
	C1         Vec2
	C2         Vec2
	Tip        Vec2
	Width      float64
	TipWidth   float64
	Depth      float64
	DepthSlope float64
	Alpha      float64
	BaseColor  Color
	TipColor   Color
	Shine      Color
	Lag        float64
	Lift       float64
	Phase      float64
	Wobble     float64
	Segments   int
}

type GlassBlade struct {
	ID     int
	Points []Vec2
	Depth  float64
	Alpha  float64
	Fill   Color
	Edge   Color
	EdgeW  float64
	Tilt   float64
}

type OpaqueShape struct {
	ID        int
	Points    []Vec2
	Color     Color
	Depth     float64
	Alpha     float64
	Kind      SurfaceKind
	HeadSpace bool
}

type Feature struct {
	ID     int
	Points []Vec2
	Color  Color
	Depth  float64
	Width  float64
	Alpha  float64
	Kind   SurfaceKind
}

type Scene struct {
	Width   float64
	Height  float64
	Pivot   Vec2
	Strands []Strand
	Glass   []GlassBlade
	Opaque  []OpaqueShape
	Ink     []Feature
}

const (
	turnStart = 0.545
	turnEnd   = 0.775
	turnAngle = 0.62
)

func headYaw(t float64) float64 {
	return turnAngle * smoothstep(turnStart, turnEnd, t)
}

func headYawVelocity(t float64) float64 {
	const dt = 0.001
	return (headYaw(t+dt) - headYaw(t-dt)) / (2 * dt)
}

func projectHead(p Vec2, yaw float64, pivot Vec2) Vec2 {
	c, s := math.Cos(yaw), math.Sin(yaw)
	dx := p.X - pivot.X
	dy := p.Y - pivot.Y
	return Vec2{
		X: pivot.X + dx*(0.94+0.06*c) + 255*s,
		Y: pivot.Y + dy + 12*s + 0.018*dx*s,
	}
}

func projectHeadAt(p Vec2, t float64, pivot Vec2) Vec2 {
	return projectHead(p, headYaw(t), pivot)
}

func NewScene(w, h int, seed int64) *Scene {
	s := &Scene{
		Width:  float64(w),
		Height: float64(h),
		Pivot:  Vec2{X: 0.45 * float64(w), Y: 0.55 * float64(h)},
	}
	sx, sy := float64(w)/1280, float64(h)/800
	rng := rand.New(rand.NewSource(seed))
	s.buildOpaque(sx, sy)
	s.buildHair(rng, sx, sy)
	s.buildGlass(sx, sy)
	return s
}

func scalePts(pts []Vec2, sx, sy float64) []Vec2 {
	out := make([]Vec2, len(pts))
	for i, p := range pts {
		out[i] = Vec2{X: p.X * sx, Y: p.Y * sy}
	}
	return out
}

func (s *Scene) buildOpaque(sx, sy float64) {
	s.Opaque = append(s.Opaque,
		OpaqueShape{
			ID: 1,
			Points: chaikinClosed(scalePts([]Vec2{
				{145, 800}, {230, 590}, {340, 520}, {455, 487},
				{548, 505}, {676, 512}, {810, 590}, {925, 800},
			}, sx, sy), 2),
			Color: RGB(0.017, 0.029, 0.048), Depth: 0.92, Kind: KindSkin,
		},
		OpaqueShape{
			ID: 2,
			Points: chaikinClosed(scalePts([]Vec2{
				{496, 375}, {554, 377}, {578, 405}, {589, 466},
				{596, 516}, {488, 524}, {476, 462}, {480, 405},
			}, sx, sy), 2),
			Color: RGB(0.285, 0.165, 0.175), Depth: 1.33, Kind: KindSkin,
			HeadSpace: true,
		},
		OpaqueShape{
			ID: 3,
			Points: chaikinClosed(scalePts([]Vec2{
				{547, 145}, {590, 146}, {621, 176}, {638, 224},
				{639, 280}, {626, 338}, {596, 386}, {566, 397},
				{548, 380}, {564, 334}, {573, 276}, {572, 224},
				{559, 178},
			}, sx, sy), 2),
			Color: RGB(0.035, 0.045, 0.075), Depth: 1.48, Kind: KindInk,
			HeadSpace: true,
		},
		OpaqueShape{
			ID: 4,
			Points: chaikinClosed(scalePts([]Vec2{
				{529, 154}, {500, 160}, {477, 181}, {462, 207},
				{447, 227}, {436, 249}, {424, 274}, {414, 294},
				{419, 307}, {432, 313}, {436, 324}, {429, 334},
				{426, 342}, {435, 349}, {439, 358}, {437, 368},
				{451, 381}, {470, 397}, {495, 410}, {523, 414},
				{547, 398}, {554, 370}, {553, 332}, {548, 291},
				{548, 247}, {542, 206}, {535, 174},
			}, sx, sy), 2),
			Color: RGB(0.52, 0.315, 0.285), Depth: 1.42, Kind: KindSkin,
			HeadSpace: true,
		},
	)

	s.Opaque = append(s.Opaque,
		OpaqueShape{
			ID: 5,
			Points: chaikinClosed(scalePts([]Vec2{
				{536, 151}, {568, 141}, {603, 153}, {626, 181},
				{640, 220}, {639, 280}, {626, 337}, {600, 375},
				{573, 359}, {577, 303}, {577, 247}, {562, 190},
			}, sx, sy), 2),
			Color: RGB(0.055, 0.075, 0.12), Depth: 1.52, Kind: KindHair,
			HeadSpace: true,
		},
		OpaqueShape{
			ID: 6,
			Points: chaikinClosed(scalePts([]Vec2{
				{505, 159}, {529, 150}, {556, 147}, {584, 154},
				{606, 171}, {608, 196}, {598, 218}, {581, 231},
				{564, 218}, {551, 195}, {532, 181}, {512, 182},
			}, sx, sy), 2),
			Color: RGB(0.033, 0.044, 0.082), Depth: 1.56, Kind: KindHair,
			HeadSpace: true,
		},
		OpaqueShape{
			ID: 7,
			Points: chaikinClosed(scalePts([]Vec2{
				{461, 303}, {478, 321}, {490, 352}, {483, 381},
				{466, 397}, {453, 390}, {458, 365}, {454, 330},
			}, sx, sy), 2),
			Color: RGB(0.22, 0.10, 0.12), Depth: 1.425, Alpha: 0.28, Kind: KindSkin,
			HeadSpace: true,
		},
		OpaqueShape{
			ID: 8,
			Points: chaikinClosed(scalePts([]Vec2{
				{486, 416}, {520, 420}, {545, 405}, {551, 419},
				{559, 463}, {563, 503}, {500, 509}, {492, 463},
			}, sx, sy), 2),
			Color: RGB(0.14, 0.065, 0.075), Depth: 1.335, Alpha: 0.30, Kind: KindSkin,
			HeadSpace: true,
		},
	)

	s.Ink = append(s.Ink,
		Feature{
			ID: 10, Kind: KindInk, Depth: 1.535, Width: 3.2 * sx,
			Alpha: 0.95, Color: RGB(0.08, 0.045, 0.06),
			Points: scalePts([]Vec2{{462, 269}, {475, 265}, {489, 268}, {497, 273}}, sx, sy),
		},
		Feature{
			ID: 11, Kind: KindInk, Depth: 1.532, Width: 1.5 * sx,
			Alpha: 0.8, Color: RGB(0.13, 0.075, 0.07),
			Points: scalePts([]Vec2{{455, 247}, {472, 239}, {490, 240}, {504, 244}}, sx, sy),
		},
		Feature{
			ID: 12, Kind: KindSkin, Depth: 1.538, Width: 2.0 * sx,
			Alpha: 0.72, Color: RGB(0.72, 0.31, 0.31),
			Points: scalePts([]Vec2{{428, 349}, {438, 346}, {447, 351}, {441, 359}}, sx, sy),
		},
		Feature{
			ID: 13, Kind: KindSkin, Depth: 1.536, Width: 1.4 * sx,
			Alpha: 0.32, Color: RGB(0.92, 0.58, 0.48),
			Points: scalePts([]Vec2{{436, 318}, {425, 302}, {413, 296}}, sx, sy),
		},
	)
}

func (st *Strand) restPoint(u float64) Vec2 {
	return cubicBezier(st.Root, st.C1, st.C2, st.Tip, u)
}

func (st *Strand) pointAt(u, t float64, pivot Vec2) Vec3 {
	p := st.restPoint(u)
	phase := st.Phase + u*5.4
	wobble := st.Wobble * math.Pow(u, 1.35)
	p.X += math.Sin(phase+t*9.0) * wobble
	p.Y += math.Cos(phase*0.83+t*7.3) * wobble * 0.48

	rigid := projectHead(p, headYaw(t), pivot)
	omega := headYawVelocity(t)
	lag := -st.Lag * omega * math.Pow(u, 1.65)
	lift := st.Lift * omega * omega * u * u
	rigid.X += lag
	rigid.Y += lift
	return Vec3{
		X: rigid.X,
		Y: rigid.Y,
		Z: st.Depth + st.DepthSlope*u,
	}
}

func (st *Strand) motionAt(u, t float64, pivot Vec2) Vec2 {
	const dt = 0.0008
	a := st.pointAt(u, t-dt, pivot)
	b := st.pointAt(u, t+dt, pivot)
	return Vec2{X: (b.X - a.X) / (2 * dt), Y: (b.Y - a.Y) / (2 * dt)}
}

func (st *Strand) widthAt(u float64) float64 {
	rootBlend := smoothstep(0, 0.14, u)
	tipBlend := 1 - smoothstep(0.62, 1, u)
	w := st.Width * (0.32 + 0.68*rootBlend)
	w = w*(0.55+0.45*tipBlend) + st.TipWidth*(1-tipBlend)
	return math.Max(0.35, w)
}

func (st *Strand) colorAt(u float64) Color {
	base := st.BaseColor.Lerp(st.TipColor, smoothstep(0.18, 1, u))
	band := math.Pow(math.Sin(math.Pi*clamp01(u*0.93+0.04)), 5)
	return base.Lerp(st.Shine, 0.38*band)
}

func addStrand(dst []Strand, rng *rand.Rand, nextID *int, sx, sy float64,
	root, c1, c2, tip Vec2, width, tipWidth, depth, slope, alpha float64,
	base, tipColor, shine Color, lag, lift, wobble float64) []Strand {
	st := Strand{
		ID: *nextID, Root: Vec2{root.X * sx, root.Y * sy},
		C1: Vec2{c1.X * sx, c1.Y * sy}, C2: Vec2{c2.X * sx, c2.Y * sy},
		Tip: Vec2{tip.X * sx, tip.Y * sy}, Width: width * sx,
		TipWidth: tipWidth * sx, Depth: depth, DepthSlope: slope,
		Alpha: alpha, BaseColor: base, TipColor: tipColor, Shine: shine,
		Lag: lag * sx, Lift: lift * sy, Phase: rng.Float64() * math.Pi * 2,
		Wobble: wobble * sx, Segments: 30,
	}
	*nextID++
	return append(dst, st)
}

func (s *Scene) buildHair(rng *rand.Rand, sx, sy float64) {
	nextID := 100
	darkBase := RGB(0.025, 0.034, 0.072)
	darkTip := RGB(0.075, 0.145, 0.285)
	darkShine := RGB(0.22, 0.44, 0.76)
	violetBase := RGB(0.042, 0.030, 0.078)
	violetTip := RGB(0.135, 0.072, 0.215)
	violetShine := RGB(0.44, 0.25, 0.60)
	inkBase := RGB(0.018, 0.022, 0.048)
	inkTip := RGB(0.055, 0.095, 0.19)
	inkShine := RGB(0.16, 0.31, 0.52)

	for i := 0; i < 30; i++ {
		u := (float64(i) + rng.Float64()*0.8) / 30
		root := Vec2{520 + 72*u + rng.NormFloat64()*7, 160 + 220*u + rng.NormFloat64()*9}
		tip := Vec2{760 + 245*u + rng.NormFloat64()*40, 700 + 85*u + rng.NormFloat64()*32}
		c1 := Vec2{root.X + 95 + 60*u + rng.NormFloat64()*16, root.Y + 95 + 40*u}
		c2 := Vec2{tip.X - 130 - 35*u + rng.NormFloat64()*25, tip.Y - 78 - 55*u}
		base := inkBase.Lerp(darkBase, 0.38+rng.Float64()*0.3)
		tc := inkTip.Lerp(darkTip, 0.25+rng.Float64()*0.25)
		sh := inkShine.Lerp(darkShine, 0.35+rng.Float64()*0.35)
		s.Strands = addStrand(s.Strands, rng, &nextID, sx, sy, root, c1, c2, tip,
			4.6+rng.Float64()*5.1, 0.35+rng.Float64()*0.60,
			1.13+rng.Float64()*0.18, 0.06+rng.Float64()*0.08,
			0.70+rng.Float64()*0.18, base, tc, sh,
			9+rng.Float64()*6, 1.8+rng.Float64()*2.4, 0.8+rng.Float64()*1.2)
	}

	for i := 0; i < 46; i++ {
		u := (float64(i) + rng.Float64()*0.9) / 46
		root := Vec2{505 + 86*u + rng.NormFloat64()*8, 154 + 190*u + rng.NormFloat64()*8}
		tip := Vec2{870 + 380*u + rng.NormFloat64()*55, 430 + 335*math.Sin(u*1.35) + rng.NormFloat64()*36}
		c1 := Vec2{root.X + 120 + 80*u + rng.NormFloat64()*22, root.Y + 75 + 130*u}
		c2 := Vec2{tip.X - 120 - 75*u + rng.NormFloat64()*35, tip.Y - 110 + 70*u}
		base := darkBase.Lerp(violetBase, 0.12+rng.Float64()*0.32)
		tc := darkTip.Lerp(violetTip, 0.10+rng.Float64()*0.40)
		sh := darkShine.Lerp(violetShine, rng.Float64()*0.34)
		s.Strands = addStrand(s.Strands, rng, &nextID, sx, sy, root, c1, c2, tip,
			2.6+rng.Float64()*4.2, 0.28+rng.Float64()*0.70,
			1.47+rng.Float64()*0.28, -0.10+rng.Float64()*0.28,
			0.62+rng.Float64()*0.25, base, tc, sh,
			11+rng.Float64()*8, 2.4+rng.Float64()*3.8, 1.1+rng.Float64()*2.1)
	}

	for i := 0; i < 30; i++ {
		u := (float64(i) + rng.Float64()*0.9) / 30
		root := Vec2{495 + 87*u + rng.NormFloat64()*7, 158 + 165*u + rng.NormFloat64()*7}
		tip := Vec2{690 + 365*u + rng.NormFloat64()*60, 315 + 340*u + rng.NormFloat64()*55}
		c1 := Vec2{root.X + 95 + 70*u + rng.NormFloat64()*20, root.Y + 75 + 105*u}
		c2 := Vec2{tip.X - 110 - 60*u + rng.NormFloat64()*35, tip.Y - 45 - 35*u}
		base := violetBase.Lerp(darkBase, 0.25+rng.Float64()*0.30)
		tc := violetTip.Lerp(darkTip, 0.15+rng.Float64()*0.40)
		sh := violetShine.Lerp(darkShine, 0.18+rng.Float64()*0.42)
		s.Strands = addStrand(s.Strands, rng, &nextID, sx, sy, root, c1, c2, tip,
			1.3+rng.Float64()*2.4, 0.20+rng.Float64()*0.50,
			1.88+rng.Float64()*0.26, -0.18+rng.Float64()*0.22,
			0.60+rng.Float64()*0.28, base, tc, sh,
			14+rng.Float64()*10, 3.4+rng.Float64()*4.2, 1.5+rng.Float64()*2.6)
	}

	for i := 0; i < 16; i++ {
		u := (float64(i) + rng.Float64()*0.8) / 16
		root := Vec2{495 + 92*u + rng.NormFloat64()*8, 150 + 150*u + rng.NormFloat64()*7}
		tip := Vec2{565 + 455*u + rng.NormFloat64()*55, 235 + 295*u + rng.NormFloat64()*60}
		c1 := Vec2{root.X + 90 + 80*u + rng.NormFloat64()*25, root.Y + 35 + 85*u}
		c2 := Vec2{tip.X - 75 - 60*u + rng.NormFloat64()*45, tip.Y - 15 + 20*u}
		base := violetBase.Lerp(inkBase, 0.20+rng.Float64()*0.25)
		tc := violetTip.Lerp(inkTip, 0.15+rng.Float64()*0.30)
		sh := violetShine.Lerp(darkShine, 0.25+rng.Float64()*0.5)
		s.Strands = addStrand(s.Strands, rng, &nextID, sx, sy, root, c1, c2, tip,
			0.55+rng.Float64()*0.9, 0.16+rng.Float64()*0.22,
			2.12+rng.Float64()*0.35, -0.06+rng.Float64()*0.2,
			0.62+rng.Float64()*0.24, base, tc, sh,
			18+rng.Float64()*12, 4.5+rng.Float64()*4.0, 2.0+rng.Float64()*3.4)
	}

	for i := 0; i < 18; i++ {
		u := (float64(i) + rng.Float64()*0.8) / 18
		root := Vec2{505 + 64*u + rng.NormFloat64()*5, 157 + 27*u + rng.NormFloat64()*4}
		tip := Vec2{610 + 165*u + rng.NormFloat64()*45, 245 + 235*u + rng.NormFloat64()*35}
		c1 := Vec2{root.X + 66 + 35*u + rng.NormFloat64()*14, root.Y + 24 + 36*u}
		c2 := Vec2{tip.X - 45 - 40*u + rng.NormFloat64()*24, tip.Y - 18 - 32*u}
		base := inkBase.Lerp(violetBase, 0.18+rng.Float64()*0.25)
		tc := inkTip.Lerp(violetTip, 0.10+rng.Float64()*0.25)
		sh := inkShine.Lerp(darkShine, 0.20+rng.Float64()*0.35)
		s.Strands = addStrand(s.Strands, rng, &nextID, sx, sy, root, c1, c2, tip,
			0.9+rng.Float64()*1.7, 0.14+rng.Float64()*0.30,
			1.58+rng.Float64()*0.18, -0.02+rng.Float64()*0.10,
			0.48+rng.Float64()*0.24, base, tc, sh,
			6+rng.Float64()*6, 1.0+rng.Float64()*2.0, 0.6+rng.Float64()*1.4)
	}

	for i := 0; i < 10; i++ {
		u := float64(i) / 9
		root := Vec2{493 + 38*u + rng.NormFloat64()*4, 165 + 75*u + rng.NormFloat64()*5}
		tip := Vec2{382 + 80*u + rng.NormFloat64()*24, 555 + 110*u + rng.NormFloat64()*35}
		c1 := Vec2{root.X - 92 - 20*u, root.Y + 60 + 10*u}
		c2 := Vec2{tip.X + 30 - 35*u, tip.Y - 130}
		base := inkBase.Lerp(violetBase, 0.35)
		tc := inkTip.Lerp(violetTip, 0.20)
		sh := inkShine.Lerp(violetShine, 0.25)
		s.Strands = addStrand(s.Strands, rng, &nextID, sx, sy, root, c1, c2, tip,
			1.0+rng.Float64()*1.7, 0.16+rng.Float64()*0.3,
			1.72+rng.Float64()*0.19, -0.10+rng.Float64()*0.12,
			0.45+rng.Float64()*0.22, base, tc, sh,
			9+rng.Float64()*7, 1.5+rng.Float64()*2.0, 0.8+rng.Float64()*1.5)
	}
}

func (s *Scene) buildGlass(sx, sy float64) {
	type bladeDef struct {
		pts   []Vec2
		depth float64
		alpha float64
		fill  Color
		edge  Color
		tilt  float64
	}
	defs := []bladeDef{
		{
			pts:   []Vec2{{714, -55}, {736, -55}, {816, -16}, {650, 758}, {545, 790}, {566, 748}},
			depth: 0.74, alpha: 0.11, fill: RGB(0.11, 0.43, 0.53), edge: RGB(0.33, 0.88, 0.95), tilt: 0.4,
		},
		{
			pts:   []Vec2{{974, 52}, {1006, 52}, {1063, 84}, {890, 638}, {832, 604}, {846, 570}},
			depth: 1.10, alpha: 0.13, fill: RGB(0.16, 0.50, 0.58), edge: RGB(0.48, 0.94, 0.98), tilt: -0.3,
		},
		{
			pts:   []Vec2{{867, 110}, {896, 109}, {935, 122}, {732, 643}, {690, 621}, {705, 588}},
			depth: 1.25, alpha: 0.10, fill: RGB(0.18, 0.35, 0.55), edge: RGB(0.44, 0.72, 1.0), tilt: 0.2,
		},
		{
			pts:   []Vec2{{627, -60}, {646, -62}, {690, -32}, {568, 666}, {526, 692}, {507, 654}},
			depth: 2.48, alpha: 0.14, fill: RGB(0.15, 0.55, 0.64), edge: RGB(0.54, 1.0, 1.0), tilt: 0.6,
		},
		{
			pts:   []Vec2{{1092, 122}, {1117, 118}, {1175, 158}, {1038, 744}, {970, 770}, {933, 730}},
			depth: 2.62, alpha: 0.12, fill: RGB(0.20, 0.46, 0.65), edge: RGB(0.58, 0.92, 1.0), tilt: -0.5,
		},
		{
			pts:   []Vec2{{349, 86}, {369, 86}, {414, 119}, {322, 670}, {265, 704}, {244, 661}},
			depth: 2.78, alpha: 0.105, fill: RGB(0.21, 0.40, 0.60), edge: RGB(0.54, 0.84, 1.0), tilt: 0.3,
		},
		{
			pts:   []Vec2{{748, 302}, {766, 299}, {797, 318}, {735, 554}, {690, 568}, {681, 539}},
			depth: 1.72, alpha: 0.13, fill: RGB(0.29, 0.61, 0.65), edge: RGB(0.68, 1.0, 0.94), tilt: -0.6,
		},
		{
			pts:   []Vec2{{482, 605}, {501, 606}, {534, 633}, {458, 788}, {416, 793}, {419, 765}},
			depth: 2.08, alpha: 0.10, fill: RGB(0.30, 0.50, 0.64), edge: RGB(0.61, 0.84, 1.0), tilt: 0.5,
		},
	}
	for i, d := range defs {
		s.Glass = append(s.Glass, GlassBlade{
			ID: 300 + i, Points: scalePts(d.pts, sx, sy), Depth: d.depth,
			Alpha: d.alpha, Fill: d.fill, Edge: d.edge, EdgeW: 1.35 * sx, Tilt: d.tilt,
		})
	}
}
