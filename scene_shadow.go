package main

import "math"

func sampleShadowScene(x, y float64) RGB {
	wall := vGradient(y, 0, 252, RGB{2.0, 2.4, 3.5}, RGB{5.8, 4.9, 4.3})
	wall = wall.mul(0.82 + 0.34*fbm(x*0.04, y*0.035))
	c := wall.add(warmGlow(x, y, 128, 176, 210, RGB{17, 10.5, 4.8}, 1))
	c = c.add(RGB{2.2, 3.2, 6.5}.mul(radialGlow(x, y, 535, 102, 150)))

	floorT := smoothstep(246, 258, y)
	floor := vGradient(y, 252, sceneH, RGB{5.2, 3.7, 2.8}, RGB{1.1, 0.9, 0.8})
	floor = floor.mul(0.82 + 0.28*fbm(x*0.055, y*0.11))
	c = mixRGB(c, floor, floorT)

	rug := ellipseMask(x, y, 330, 326, 210, 42)
	c = mixRGB(c, RGB{7.5, 2.6, 2.4}, rug*0.82)

	c = paintBookshelf(c, x, y)
	c = paintMoonWindow(c, x, y)
	c = paintSofa(c, x, y)
	c = paintShadowLamp(c, x, y)

	table := Rect{X0: 206, Y0: 318, X1: 430, Y1: 350}
	c = paintRect(c, x, y, table, RGB{10.5, 6.2, 3.8})
	return c
}

func ellipseMask(x, y, cx, cy, rx, ry float64) float64 {
	q := ((x-cx)/rx)*((x-cx)/rx) + ((y-cy)/ry)*((y-cy)/ry)
	return 1 - smoothstep(0.92, 1.04, q)
}

func paintBookshelf(c RGB, x, y float64) RGB {
	frame := Rect{X0: 24, Y0: 62, X1: 214, Y1: 256}
	c = paintRect(c, x, y, frame, RGB{6.5, 4.2, 2.7})
	inside := Rect{X0: 34, Y0: 72, X1: 204, Y1: 246}
	c = paintRect(c, x, y, inside, RGB{2.2, 1.7, 1.3})

	for row := 0; row < 4; row++ {
		shelfY := 108 + float64(row)*38
		shelf := Rect{X0: 34, Y0: shelfY + 24, X1: 204, Y1: shelfY + 29}
		c = paintRect(c, x, y, shelf, RGB{8.5, 5.4, 3.2})
		for bx := 42.0; bx < 196; bx += 9 + math.Mod(bx, 5) {
			bh := 16 + 10*hash2(int64(bx*3), int64(row))
			book := Rect{X0: bx, Y0: shelfY + 25 - bh, X1: bx + 6.5, Y1: shelfY + 25}
			if book.contains(x, y) {
				h := hash2(int64(bx*17), int64(row*91))
				col := RGB{2 + 5*h, 2.2 + 3*hash2(int64(bx), 7), 2.5 + 5*(1-h)}
				c = mixRGB(c, col, 0.95)
			}
		}
	}
	return c
}

func paintMoonWindow(c RGB, x, y float64) RGB {
	frame := Rect{X0: 454, Y0: 34, X1: 618, Y1: 166}
	c = paintRect(c, x, y, frame, RGB{7.5, 6.2, 5.2})
	win := Rect{X0: 464, Y0: 44, X1: 608, Y1: 156}
	if rectMask(x, y, win, 1) > 0 {
		sky := vGradient(y, 44, 156, RGB{3.2, 5.2, 12}, RGB{8, 11, 20})
		sky = sky.add(warmGlow(x, y, 566, 76, 70, RGB{12, 16, 28}, 1))
		c = mixRGB(c, sky, rectMask(x, y, win, 1))
	}
	moon := circleMask(x, y, 566, 75, 15, 1.2)
	c = mixRGB(c, RGB{850, 900, 1050}, moon)
	for _, bar := range []Rect{
		{X0: 532, Y0: 44, X1: 538, Y1: 156},
		{X0: 464, Y0: 98, X1: 608, Y1: 104},
	} {
		c = paintRect(c, x, y, bar, RGB{5, 4.6, 4.2})
	}
	return c
}

func paintSofa(c RGB, x, y float64) RGB {
	back := trapezoidMask(x, y, 205, 306, 238, 454, 220, 470)
	c = mixRGB(c, RGB{4.2, 5.4, 9.2}, back)
	seat := Rect{X0: 214, Y0: 292, X1: 466, Y1: 326}
	c = paintRect(c, x, y, seat, RGB{6.2, 7.4, 12})
	for _, cx := range []float64{294, 386} {
		cushion := circleMask(x, y, cx, 260, 34, 2)
		c = mixRGB(c, RGB{7.5, 8.5, 14}, cushion*0.72)
	}
	armL := Rect{X0: 202, Y0: 260, X1: 226, Y1: 328}
	armR := Rect{X0: 454, Y0: 260, X1: 478, Y1: 328}
	c = paintRect(c, x, y, armL, RGB{5, 6.2, 10.5})
	c = paintRect(c, x, y, armR, RGB{5, 6.2, 10.5})
	return c
}

func paintShadowLamp(c RGB, x, y float64) RGB {
	table := Rect{X0: 72, Y0: 286, X1: 178, Y1: 326}
	c = paintRect(c, x, y, table, RGB{11, 7, 4.5})
	stem := Rect{X0: 122, Y0: 188, X1: 128, Y1: 288}
	c = paintRect(c, x, y, stem, RGB{16, 12, 8})
	shade := trapezoidMask(x, y, 150, 194, 96, 160, 78, 178)
	shadeColor := RGB{58, 35, 15}.add(warmGlow(x, y, 128, 176, 42, RGB{120, 70, 25}, 1))
	c = mixRGB(c, shadeColor, shade*0.96)
	bulb := circleMask(x, y, 128, 184, 8, 1)
	c = mixRGB(c, RGB{1500, 980, 430}, bulb)
	return c
}
