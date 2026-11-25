package main

import "math"

func rectMask(x, y float64, r Rect, feather float64) float64 {
	if feather <= 0 {
		feather = 0.75
	}
	xf := smoothstep(r.X0-feather, r.X0+feather, x) *
		(1 - smoothstep(r.X1-feather, r.X1+feather, x))
	yf := smoothstep(r.Y0-feather, r.Y0+feather, y) *
		(1 - smoothstep(r.Y1-feather, r.Y1+feather, y))
	return xf * yf
}

func circleMask(x, y, cx, cy, radius, feather float64) float64 {
	if feather <= 0 {
		feather = 1
	}
	d := math.Hypot(x-cx, y-cy)
	return 1 - smoothstep(radius-feather, radius+feather, d)
}

func ringMask(x, y, cx, cy, radius, width, feather float64) float64 {
	d := math.Hypot(x-cx, y-cy)
	inner := radius - width/2
	outer := radius + width/2
	return smoothstep(inner-feather, inner+feather, d) *
		(1 - smoothstep(outer-feather, outer+feather, d))
}

func radialGlow(x, y, cx, cy, radius float64) float64 {
	d := math.Hypot(x-cx, y-cy)
	q := d / radius
	return math.Exp(-q * q * 2.4)
}

func warmGlow(x, y, cx, cy, radius float64, color RGB, energy float64) RGB {
	return color.mul(radialGlow(x, y, cx, cy, radius) * energy)
}

func vGradient(y, y0, y1 float64, top, bottom RGB) RGB {
	t := smoothstep(y0, y1, y)
	return mixRGB(top, bottom, t)
}

func trapezoidMask(x, y, y0, y1, topX0, topX1, bottomX0, bottomX1 float64) float64 {
	if y < y0-1 || y > y1+1 {
		return 0
	}
	t := clamp01((y - y0) / (y1 - y0))
	x0 := mix(topX0, bottomX0, t)
	x1 := mix(topX1, bottomX1, t)
	return smoothstep(x0-1, x0+1, x) * (1 - smoothstep(x1-1, x1+1, x)) *
		smoothstep(y0-1, y0+1, y) * (1 - smoothstep(y1-1, y1+1, y))
}

func paintRect(c RGB, x, y float64, r Rect, fill RGB) RGB {
	return mixRGB(c, fill, rectMask(x, y, r, 0.8))
}

func paintCircle(c RGB, x, y, cx, cy, radius float64, fill RGB) RGB {
	return mixRGB(c, fill, circleMask(x, y, cx, cy, radius, 1.0))
}

func addRectGlow(c RGB, x, y float64, r Rect, cx, cy, radius float64, color RGB, energy float64) RGB {
	if rectMask(x, y, r, 1.2) > 0 {
		return c.add(warmGlow(x, y, cx, cy, radius, color, energy))
	}
	return c
}
