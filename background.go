package main

import "math"

type bokeh struct {
	x, y, r float64
	color   Color
	alpha   float64
	ring    bool
}

func renderBackground(iw, ih, ow, oh int) []Color {
	target := make([]Color, iw*ih)
	top := RGB(0.007, 0.014, 0.029)
	bottom := RGB(0.003, 0.008, 0.017)
	glow := RGB(0.025, 0.16, 0.20)
	coolGlow := RGB(0.06, 0.08, 0.22)
	bokehs := []bokeh{
		{0.68, 0.35, 0.245, RGB(0.11, 0.38, 0.43), 0.17, true},
		{0.664, 0.346, 0.218, RGB(0.21, 0.55, 0.58), 0.035, true},
		{0.81, 0.18, 0.026, RGB(0.32, 0.68, 0.72), 0.13, false},
		{0.87, 0.28, 0.016, RGB(0.39, 0.78, 0.83), 0.10, false},
		{0.73, 0.14, 0.010, RGB(0.60, 0.86, 0.91), 0.13, false},
		{0.61, 0.20, 0.007, RGB(0.55, 0.78, 0.95), 0.11, false},
		{0.91, 0.43, 0.023, RGB(0.25, 0.52, 0.77), 0.08, false},
		{0.29, 0.22, 0.018, RGB(0.17, 0.27, 0.54), 0.08, false},
		{0.18, 0.37, 0.012, RGB(0.28, 0.43, 0.64), 0.07, false},
		{0.46, 0.13, 0.013, RGB(0.21, 0.34, 0.56), 0.06, false},
		{0.78, 0.58, 0.008, RGB(0.35, 0.62, 0.65), 0.08, false},
		{0.55, 0.72, 0.006, RGB(0.19, 0.34, 0.47), 0.07, false},
	}
	for i := 0; i < 9; i++ {
		fx := 0.60 + float64(i)*0.041
		fy := 0.10 + float64((i*7)%13)/31
		r := 0.007 + float64((i*11)%9)/1800
		a := 0.045 + float64((i*5)%7)/150
		bokehs = append(bokehs, bokeh{fx, fy, r, RGB(0.38, 0.72, 0.86), a, false})
	}

	for iy := 0; iy < ih; iy++ {
		y := (float64(iy) + 0.5) / float64(ih)
		for ix := 0; ix < iw; ix++ {
			x := (float64(ix) + 0.5) / float64(iw)
			base := top.Lerp(bottom, smoothstep(0, 1, y*0.9+0.08))

			dx, dy := x-0.69, y-0.36
			d := math.Hypot(dx, dy*0.82)
			nebula := math.Exp(-d * d / 0.075)
			base = base.Add(glow.Mul(nebula * 0.34))
			base = base.Add(coolGlow.Mul(math.Exp(-(dx*dx+dy*dy)/0.26) * 0.09))

			shaft := smoothstep(0.48, 0.0, math.Abs((x*0.62+y*0.78)-0.54))
			shaft *= smoothstep(0.95, 0.12, y) * 0.018
			base = base.Add(RGB(0.10, 0.24, 0.31).Mul(shaft))

			for _, b := range bokehs {
				bdx := (x - b.x) / b.r
				bdy := (y - b.y) / b.r
				br := math.Hypot(bdx, bdy)
				if b.ring {
					rim := 1 - smoothstep(0.86, 1.04, br)
					inner := 1 - smoothstep(0.0, 0.95, br)
					base = base.Add(b.color.Mul((rim + inner*0.18) * b.alpha))
				} else {
					disc := 1 - smoothstep(0.12, 1.0, br)
					edge := 1 - smoothstep(0.70, 1.0, br)
					base = base.Add(b.color.Mul(disc*disc*b.alpha + edge*0.012))
				}
			}

			h := uint32(ix*73856093) ^ uint32(iy*19349663) ^ 0x9e3779b9
			grain := (hash01(h) - 0.5) * 0.010
			base = base.Add(RGB(grain, grain, grain))

			flare := math.Pow(clamp01(1-math.Abs(x-0.80)*14), 6) * math.Pow(clamp01(1-math.Abs(y-0.20)*5), 2)
			base = base.Add(RGB(0.10, 0.24, 0.29).Mul(flare * 0.18))
			target[iy*iw+ix] = clampColor(base)
		}
	}
	return target
}

func clampColor(c Color) Color {
	return Color{R: clamp01(c.R), G: clamp01(c.G), B: clamp01(c.B)}
}
