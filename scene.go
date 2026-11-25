package main

import "math"

const (
	imageWidth  = 720
	imageHeight = 196
	flatHeight  = 40
)

// RGB is a floating-point sRGB triple in display-code space, 0..255.
type RGB struct {
	R float64
	G float64
	B float64
}

type Scene struct {
	ID        string
	Name      string
	English   string
	Detail    string
	Range     string
	Accent    string
	FlatColor RGB
	sample    func(t float64) RGB
}

var scenes = []Scene{
	{
		ID:        "gray",
		Name:      "灰阶",
		English:   "NEUTRAL RAMP",
		Detail:    "冷暖中性，暗部步进最容易形成等高线。",
		Range:     "L 2—38",
		Accent:    "#c9ff55",
		FlatColor: RGB{14, 14, 14},
		sample: func(t float64) RGB {
			v := 2 + 36*t
			return RGB{v, v, v}
		},
	},
	{
		ID:        "skin",
		Name:      "肤色",
		English:   "DEEP SKIN TONE",
		Detail:    "红绿蓝斜率不同，色带边缘会交错成彩色轮廓。",
		Range:     "R 9—87 / G 4—40",
		Accent:    "#ff9a78",
		FlatColor: RGB{30, 14, 11},
		sample: func(t float64) RGB {
			return RGB{
				R: 9 + 78*math.Pow(t, 1.04),
				G: 4 + 36*math.Pow(t, 0.96),
				B: 3 + 27*t,
			}
		},
	},
	{
		ID:        "chroma",
		Name:      "低亮度彩色",
		English:   "LOW-LIGHT CHROMA",
		Detail:    "青绿经紫罗兰过渡到暗玫瑰，专挑低亮度色阶。",
		Range:     "RGB 2—42",
		Accent:    "#7de9ff",
		FlatColor: RGB{10, 7, 19},
		sample: func(t float64) RGB {
			u := smoothStep(t)
			if u < 0.52 {
				k := u / 0.52
				return mixRGB(RGB{2, 10, 16}, RGB{14, 4, 31}, k)
			}
			k := (u - 0.52) / 0.48
			return mixRGB(RGB{14, 4, 31}, RGB{42, 8, 20}, k)
		},
	},
}

func sceneByID(id string) Scene {
	for _, s := range scenes {
		if s.ID == id {
			return s
		}
	}
	return scenes[0]
}

func (s Scene) At(x, y int) RGB {
	if y < flatHeight {
		return s.FlatColor
	}
	t := float64(x) / float64(imageWidth-1)
	return s.sample(clamp01(t))
}

func mixRGB(a, b RGB, t float64) RGB {
	return RGB{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
	}
}

func smoothStep(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
