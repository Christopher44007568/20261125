package main

import "math"

type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

func RGB(r, g, b uint8) Color { return Color{R: r, G: g, B: b, A: 255} }

func (c Color) rgba() (uint32, uint32, uint32, uint32) {
	return uint32(c.R) * 257, uint32(c.G) * 257, uint32(c.B) * 257, uint32(c.A) * 257
}

func blendColor(a, b Color, t float64) Color {
	t = clampFloat(t, 0, 1)
	return Color{
		R: uint8(math.Round(float64(a.R)*(1-t) + float64(b.R)*t)),
		G: uint8(math.Round(float64(a.G)*(1-t) + float64(b.G)*t)),
		B: uint8(math.Round(float64(a.B)*(1-t) + float64(b.B)*t)),
		A: 255,
	}
}

func scaleColor(c Color, s float64) Color {
	return Color{
		R: uint8(clampFloat(float64(c.R)*s, 0, 255)),
		G: uint8(clampFloat(float64(c.G)*s, 0, 255)),
		B: uint8(clampFloat(float64(c.B)*s, 0, 255)),
		A: c.A,
	}
}

func heatColor(t float64) Color {
	t = clampFloat(t, 0, 1)
	stops := []Color{
		RGB(38, 70, 180),
		RGB(34, 150, 220),
		RGB(35, 210, 170),
		RGB(235, 220, 70),
		RGB(245, 130, 45),
		RGB(215, 35, 35),
	}
	x := t * float64(len(stops)-1)
	i := int(math.Floor(x))
	if i >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	return blendColor(stops[i], stops[i+1], x-float64(i))
}

func shadeNormal(n Vec3, base Color) Color {
	light := V(-0.38, 0.48, 0.79).Unit()
	key := math.Max(0, n.Dot(light))
	fill := math.Max(0, n.Dot(V(0.55, -0.25, 0.45).Unit())) * 0.18
	intensity := 0.40 + 0.58*key + fill
	return scaleColor(base, intensity)
}
