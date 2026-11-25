package main

import "math"

func sampleWindowScene(x, y float64) RGB {
	wall := vGradient(y, 0, sceneH, RGB{3.2, 3.6, 4.8}, RGB{8.4, 6.9, 5.8})
	n := fbm(x*0.025, y*0.025)
	wall = wall.mul(0.82 + 0.36*n)
	wall = wall.add(warmGlow(x, y, 414, 132, 285, RGB{23, 21, 18}, 1.0))

	floorT := smoothstep(244, 252, y)
	floor := vGradient(y, 244, sceneH, RGB{16, 9.5, 5.8}, RGB{3.6, 2.5, 2.0})
	plank := 0.86 + 0.14*math.Sin(y*0.55)
	floor = floor.mul(plank * (0.85 + 0.3*fbm(x*0.04, y*0.08)))
	c := mixRGB(wall, floor, floorT)

	baseboard := Rect{X0: 0, Y0: 238, X1: sceneW, Y1: 250}
	c = paintRect(c, x, y, baseboard, RGB{28, 25, 22})

	c = paintWindowArchitecture(c, x, y)
	c = paintCurtains(c, x, y)
	c = paintPlant(c, x, y, 202, 304)
	c = paintChair(c, x, y)
	return c
}

func paintWindowArchitecture(c RGB, x, y float64) RGB {
	casing := Rect{X0: 226, Y0: 28, X1: 600, Y1: 238}
	c = paintRect(c, x, y, casing, RGB{34, 29, 24})

	win := Rect{X0: 244, Y0: 42, X1: 582, Y1: 222}
	mask := rectMask(x, y, win, 1.0)
	if mask > 0 {
		t := clamp01((y - win.Y0) / (win.Y1 - win.Y0))
		sky := mixRGB(RGB{850, 1250, 2500}, RGB{2800, 3500, 5400}, smoothstep(0, 0.42, t))
		sky = mixRGB(sky, RGB{7200, 5700, 3500}, smoothstep(0.38, 0.82, t))
		sky = mixRGB(sky, RGB{620, 760, 1050}, smoothstep(0.84, 1, t)*0.55)
		sky = sky.add(warmGlow(x, y, 505, 94, 165, RGB{4200, 3500, 2300}, 1))
		sky = paintWindowBuildings(sky, x, y)
		c = mixRGB(c, sky, mask)
	}

	sun := circleMask(x, y, 506, 94, 27, 1.4)
	c = mixRGB(c, RGB{9200, 8400, 6700}, sun)

	mullions := []Rect{
		{X0: 407, Y0: 42, X1: 419, Y1: 222},
		{X0: 244, Y0: 126, X1: 582, Y1: 137},
		{X0: 244, Y0: 42, X1: 252, Y1: 222},
		{X0: 574, Y0: 42, X1: 582, Y1: 222},
	}
	for _, bar := range mullions {
		c = paintRect(c, x, y, bar, RGB{13, 12.5, 12})
	}
	return c
}

func paintWindowBuildings(c RGB, x, y float64) RGB {
	buildings := []Rect{
		{X0: 252, Y0: 151, X1: 326, Y1: 221},
		{X0: 330, Y0: 128, X1: 402, Y1: 221},
		{X0: 424, Y0: 158, X1: 492, Y1: 221},
		{X0: 500, Y0: 137, X1: 574, Y1: 221},
	}
	colors := []RGB{{62, 72, 96}, {42, 52, 76}, {88, 74, 70}, {48, 62, 88}}
	for i, b := range buildings {
		if b.contains(x, y) {
			c = mixRGB(c, colors[i], 0.92)
			wx := math.Mod(x-b.X0, 15)
			wy := math.Mod(y-b.Y0, 17)
			if wx > 4 && wx < 9 && wy > 5 && wy < 10 {
				c = c.add(RGB{150, 118, 58}.mul(0.65 + 0.7*hash2(int64(x/7), int64(y/9))))
			}
		}
	}
	return c
}

func paintCurtains(c RGB, x, y float64) RGB {
	left := trapezoidMask(x, y, 24, 246, 204, 238, 190, 246)
	right := trapezoidMask(x, y, 24, 246, 594, 636, 604, 650)
	fold := 0.72 + 0.28*math.Sin(x*0.42)
	cloth := RGB{19 * fold, 4.2 * fold, 5.2 * fold}
	c = mixRGB(c, cloth, math.Max(left, right))
	rod := Rect{X0: 194, Y0: 22, X1: 642, Y1: 29}
	c = paintRect(c, x, y, rod, RGB{42, 29, 17})
	return c
}

func paintPlant(c RGB, x, y, px, py float64) RGB {
	pot := trapezoidMask(x, y, py, py+48, px-24, px+24, px-31, px+31)
	c = mixRGB(c, RGB{24, 10, 6}, pot)
	leaves := []struct{ dx, dy, r float64 }{
		{-22, -28, 25}, {4, -42, 31}, {27, -25, 24}, {-4, -18, 28}, {-38, -8, 18}, {38, -5, 18},
	}
	for _, leaf := range leaves {
		m := circleMask(x, y, px+leaf.dx, py+leaf.dy, leaf.r, 1.4)
		leafColor := RGB{2.2 + 1.5*fbm(x*.1, y*.1), 8.5, 4.2}
		c = mixRGB(c, leafColor, m)
	}
	return c
}

func paintChair(c RGB, x, y float64) RGB {
	back := trapezoidMask(x, y, 238, 326, 492, 628, 474, 638)
	c = mixRGB(c, RGB{5.5, 7.5, 12}, back*0.96)
	seat := Rect{X0: 462, Y0: 318, X1: 636, Y1: 350}
	c = paintRect(c, x, y, seat, RGB{7.5, 9.5, 15})
	return c
}
