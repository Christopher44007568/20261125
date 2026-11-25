package main

func compositeLayerMask(samples []DeepSample, bg Color) Color {
	col := bg.Mul(0.16).Add(RGB(0.006, 0.010, 0.020))
	for _, s := range samples {
		if s.Alpha <= 0 {
			continue
		}
		t := clamp01((s.Depth - 0.68) / (2.92 - 0.68))
		oc := orderPalette(t)
		if s.Alpha >= 0.985 {
			gray := RGB(0.055, 0.068, 0.090)
			col = col.Lerp(gray, 0.96)
			continue
		}
		jitter := 0.72 + 0.38*hash01(uint32(s.LayerID*2654435761))
		if s.Kind == KindGlass {
			jitter = 1.0
		}
		oc = oc.Mul(jitter)
		a := clamp01(0.22 + 0.72*s.Alpha)
		col = col.Lerp(oc, a)
	}
	return clampColor(col)
}

func orderPalette(t float64) Color {
	t = clamp01(t)
	type stop struct {
		at float64
		c  Color
	}
	stops := [...]stop{
		{0.00, RGB(0.12, 0.08, 0.34)},
		{0.16, RGB(0.08, 0.26, 0.78)},
		{0.34, RGB(0.04, 0.68, 0.82)},
		{0.52, RGB(0.24, 0.82, 0.50)},
		{0.68, RGB(0.78, 0.86, 0.22)},
		{0.82, RGB(1.00, 0.62, 0.10)},
		{0.92, RGB(1.00, 0.28, 0.12)},
		{1.00, RGB(1.00, 0.18, 0.52)},
	}
	for i := 0; i < len(stops)-1; i++ {
		if t <= stops[i+1].at {
			f := (t - stops[i].at) / (stops[i+1].at - stops[i].at)
			return stops[i].c.Lerp(stops[i+1].c, smoothstep(0, 1, f))
		}
	}
	return stops[len(stops)-1].c
}
