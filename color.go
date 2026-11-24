package main

import "math"

type RGB struct {
	R, G, B float64
}

func (a RGB) Lerp(b RGB, t float64) RGB {
	t = clamp(t, 0, 1)
	return RGB{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
	}
}

func (a RGB) Scale(s float64) RGB {
	return RGB{a.R * s, a.G * s, a.B * s}
}

func (a RGB) Add(b RGB) RGB {
	return RGB{a.R + b.R, a.G + b.G, a.B + b.B}
}

func (a RGB) Bytes() (uint8, uint8, uint8) {
	return uint8(clamp(a.R*255+0.5, 0, 255)),
		uint8(clamp(a.G*255+0.5, 0, 255)),
		uint8(clamp(a.B*255+0.5, 0, 255))
}

type paletteStop struct {
	t float64
	c RGB
}

var curvaturePalette = []paletteStop{
	{0.00, RGB{0.035, 0.045, 0.105}},
	{0.12, RGB{0.055, 0.135, 0.285}},
	{0.28, RGB{0.055, 0.295, 0.475}},
	{0.44, RGB{0.045, 0.455, 0.405}},
	{0.60, RGB{0.315, 0.595, 0.285}},
	{0.74, RGB{0.665, 0.725, 0.225}},
	{0.87, RGB{0.910, 0.600, 0.145}},
	{1.00, RGB{0.990, 0.965, 0.690}},
}

func paletteColor(stops []paletteStop, t float64) RGB {
	t = clamp(t, 0, 1)
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].t {
			u := (t - stops[i-1].t) / (stops[i].t - stops[i-1].t)
			return stops[i-1].c.Lerp(stops[i].c, u)
		}
	}
	return stops[len(stops)-1].c
}

func curvatureColor(k, maxK float64) RGB {
	t := clamp(k/maxK, 0, 1)
	const bands = 10
	band := int(t * bands)
	if band >= bands {
		band = bands - 1
	}
	bandT := (float64(band) + 0.45) / bands
	c := paletteColor(curvaturePalette, bandT)
	frac := t*bands - float64(band)
	if band > 0 && frac < 0.035 {
		c = c.Scale(0.55)
	}
	return c
}

func shadeClay(n Vec3, toEye Vec3) RGB {
	lUp := Vec3{-0.62, -0.30, 0.72}.Unit()
	lDown := Vec3{-0.54, -0.18, -0.82}.Unit()
	key := math.Max(n.Dot(lUp), 0)
	fill := math.Max(n.Dot(lDown), 0)
	bounce := 0.5 + 0.5*n.Z
	rim := math.Pow(1-math.Max(n.Dot(toEye), 0), 2.5)

	lum := 0.135 + 0.50*key + 0.31*fill + 0.08*bounce
	base := RGB{0.69, 0.73, 0.79}.Scale(lum)
	base = base.Add(RGB{0.12, 0.18, 0.28}.Scale(rim))
	return base
}

func normalSignColor(n Vec3) RGB {
	encode := func(x float64) float64 {
		if x >= 0 {
			return 0.18 + 0.82*math.Abs(x)
		}
		return 0.035
	}
	return RGB{encode(n.X), encode(n.Y), encode(n.Z)}
}

func zebraColor(n Vec3, incident Vec3) RGB {
	r := incident.Sub(n.Scale(2 * incident.Dot(n))).Unit()
	phase := 53*r.Z + 7.5*r.X + 2.3*r.Y
	s := 0.5 + 0.5*math.Tanh(3.2*math.Sin(phase))
	lum := 0.055 + 0.91*s
	return RGB{0.93, 0.96, 1.0}.Scale(lum)
}
