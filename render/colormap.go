package render

import (
	"image/color"

	"brepsection/geom"
)

// jetColor maps t in [0,1] to a blue-cyan-yellow-red heat color.
func jetColor(t float64) color.RGBA {
	t = geom.Clamp(t, 0, 1)
	stops := []struct {
		at  float64
		rgb [3]float64
	}{
		{0.0, [3]float64{0, 0, 0.5}},
		{0.25, [3]float64{0, 0.6, 1}},
		{0.5, [3]float64{0.2, 1, 0.4}},
		{0.75, [3]float64{1, 0.9, 0}},
		{1.0, [3]float64{0.7, 0, 0}},
	}
	for i := 0; i+1 < len(stops); i++ {
		a, b := stops[i], stops[i+1]
		if t <= b.at {
			f := (t - a.at) / (b.at - a.at)
			mix := func(x, y float64) uint8 {
				return uint8(geom.Clamp((x+(y-x)*f)*255, 0, 255))
			}
			return color.RGBA{
				R: mix(a.rgb[0], b.rgb[0]),
				G: mix(a.rgb[1], b.rgb[1]),
				B: mix(a.rgb[2], b.rgb[2]),
				A: 255,
			}
		}
	}
	last := stops[len(stops)-1].rgb
	return color.RGBA{uint8(last[0] * 255), uint8(last[1] * 255), uint8(last[2] * 255), 255}
}

// palette holds the fixed drawing colors so every output shares one style.
var palette = struct {
	Bg, Panel, Text, Edge, Section, HoleFill, FaceA, FaceB, Plane color.Color
}{
	Bg:       color.RGBA{245, 245, 248, 255},
	Panel:    color.RGBA{255, 255, 255, 255},
	Text:     color.RGBA{30, 30, 40, 255},
	Edge:     color.RGBA{60, 60, 75, 255},
	Section:  color.RGBA{200, 40, 50, 255},
	HoleFill: color.RGBA{40, 70, 120, 255},
	FaceA:    color.RGBA{180, 195, 215, 255},
	FaceB:    color.RGBA{150, 168, 195, 255},
	Plane:    color.RGBA{230, 180, 90, 200},
}
