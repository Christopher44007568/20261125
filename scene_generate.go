package main

type sceneBlueprint struct {
	id       string
	title    string
	subtitle string
	gray     Rect
	sample   func(x, y float64) RGB
}

func grayRectFor(id string) Rect {
	switch id {
	case "window":
		return Rect{X0: 48, Y0: 260, X1: 148, Y1: 326}
	case "lights":
		return Rect{X0: 276, Y0: 252, X1: 364, Y1: 318}
	default:
		return Rect{X0: 270, Y0: 270, X1: 366, Y1: 334}
	}
}

func blueprints() []sceneBlueprint {
	return []sceneBlueprint{
		{
			id:       "window",
			title:    "HIGHLIGHT WINDOW",
			subtitle: "9000 NIT SKY / FRAME RETAINED",
			gray:     grayRectFor("window"),
			sample:   sampleWindowScene,
		},
		{
			id:       "lights",
			title:    "COLORED LAMPS",
			subtitle: "SATURATED HDR PRACTICALS",
			gray:     grayRectFor("lights"),
			sample:   sampleLightsScene,
		},
		{
			id:       "shadow",
			title:    "SHADOW INTERIOR",
			subtitle: "0.2-6 NIT ROOM DETAIL",
			gray:     grayRectFor("shadow"),
			sample:   sampleShadowScene,
		},
	}
}

func GenerateScenes() []*Scene {
	defs := blueprints()
	scenes := make([]*Scene, 0, len(defs))
	for _, def := range defs {
		scenes = append(scenes, generateBlueprint(def))
	}
	return scenes
}

func generateBlueprint(def sceneBlueprint) *Scene {
	scene := &Scene{
		ID:       def.id,
		Title:    def.title,
		Subtitle: def.subtitle,
		Gray:     def.gray,
		Width:    sceneW,
		Height:   sceneH,
		Pixels:   make([]RGB, sceneW*sceneH),
	}

	for y := 0; y < sceneH; y++ {
		for x := 0; x < sceneW; x++ {
			var sum RGB
			for sy := 0; sy < ssaa; sy++ {
				fy := (float64(sy) + 0.5) / ssaa
				for sx := 0; sx < ssaa; sx++ {
					fx := (float64(sx) + 0.5) / ssaa
					c := def.sample(float64(x)+fx, float64(y)+fy)
					c = paintCalibrationCard(c, float64(x)+fx, float64(y)+fy, def.gray)
					sum = sum.add(c)
				}
			}
			n := float64(ssaa * ssaa)
			idx := scene.index(x, y)
			scene.Pixels[idx] = sum.mul(1 / n)
		}
	}
	return scene
}

func paintCalibrationCard(c RGB, x, y float64, gray Rect) RGB {
	backing := Rect{X0: gray.X0 - 8, Y0: gray.Y0 - 8, X1: gray.X1 + 8, Y1: gray.Y1 + 8}
	c = paintRect(c, x, y, backing, RGB{1.4, 1.35, 1.25})

	stand := Rect{X0: gray.X0 + 40, Y0: gray.Y1 + 6, X1: gray.X1 - 40, Y1: gray.Y1 + 24}
	c = paintRect(c, x, y, stand, RGB{5.5, 4.2, 3.2})

	border := Rect{X0: gray.X0 - 4, Y0: gray.Y0 - 4, X1: gray.X1 + 4, Y1: gray.Y1 + 4}
	c = paintRect(c, x, y, border, RGB{2.2, 2.2, 2.15})
	c = paintRect(c, x, y, gray, RGB{grayNits, grayNits, grayNits})

	edge := Rect{X0: gray.X0, Y0: gray.Y0, X1: gray.X0 + 3, Y1: gray.Y1}
	c = paintRect(c, x, y, edge, RGB{grayNits * 1.08, grayNits * 1.08, grayNits * 1.08})
	return c
}
