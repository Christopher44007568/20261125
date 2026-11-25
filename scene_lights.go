package main

import "math"

type lamp struct {
	x     float64
	color RGB
	name  string
}

func lamps() []lamp {
	return []lamp{
		{x: 96, color: RGB{5200, 130, 70}},
		{x: 244, color: RGB{120, 4700, 190}},
		{x: 396, color: RGB{110, 260, 5400}},
		{x: 548, color: RGB{5400, 3400, 180}},
	}
}

func sampleLightsScene(x, y float64) RGB {
	base := vGradient(y, 0, sceneH, RGB{5.2, 4.6, 8.5}, RGB{1.4, 1.2, 2.2})
	noise := fbm(x*0.035, y*0.035)
	c := base.mul(0.78 + 0.38*noise)

	for _, l := range lamps() {
		pool := l.color.mul(0.010)
		c = c.add(warmGlow(x, y, l.x, 112, 225, pool, 1))
		floorGlow := warmGlow(x, y, l.x, 326, 150, l.color.mul(0.006), 1)
		c = c.add(floorGlow)
	}
	c = c.add(warmGlow(x, y, 320, 226, 175, RGB{22, 22, 24}, 1))

	floorMask := smoothstep(252, 262, y)
	floor := vGradient(y, 252, sceneH, RGB{8, 7, 11}, RGB{2.2, 2, 3.2})
	c = mixRGB(c, floor, floorMask*0.88)

	table := Rect{X0: 226, Y0: 318, X1: 414, Y1: 356}
	c = paintRect(c, x, y, table, RGB{12, 7.5, 4.8})

	for _, l := range lamps() {
		wire := Rect{X0: l.x - 1.5, Y0: 0, X1: l.x + 1.5, Y1: 58}
		c = paintRect(c, x, y, wire, RGB{8, 8, 9})
		socket := Rect{X0: l.x - 10, Y0: 52, X1: l.x + 10, Y1: 72}
		c = paintRect(c, x, y, socket, RGB{28, 25, 24})

		c = c.add(warmGlow(x, y, l.x, 84, 76, l.color.mul(0.20), 1))
		glass := circleMask(x, y, l.x, 84, 18, 1.5)
		glassColor := mixRGB(l.color.mul(0.32), RGB{1, 1, 1}.mul(grayNits), 0.04)
		c = mixRGB(c, glassColor, glass*0.95)
		core := circleMask(x, y, l.x, 84, 8.5, 1.0)
		c = mixRGB(c, l.color, core)

		highlight := circleMask(x, y, l.x-5, 78, 2.4, 0.6)
		c = mixRGB(c, RGB{9000, 9000, 8500}, highlight*0.85)
	}

	for bx := 42.0; bx < sceneW; bx += 54 {
		bokeh := circleMask(x, y, bx+17*math.Sin(bx), 190+28*math.Cos(bx), 3.2, 1.2)
		c = c.add(RGB{12, 12, 16}.mul(bokeh))
	}
	return c
}
