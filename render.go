package main

import (
	"image"
	"image/color"
	"math"
)

type ShaderFunc func(*Scene, *ShadeContext) Vec3

// RenderResult holds the display image together with linear-space diagnostics.
type RenderResult struct {
	Name   string
	Image  *image.RGBA
	Linear []Vec3 // HDR linear radiance per pixel
	Spec   []float64
}

var shaders = []struct {
	Name string
	Fn   ShaderFunc
}{
	{"single", shadeSingleScatter},
	{"kajiya_kay", shadeKajiyaKay},
	{"double", shadeDoubleScatter},
}

// specularOnly re-evaluates each model's pure specular scattering (ambient and
// diffuse floors removed) so overlays compare highlights rather than base color.
func specularOnly(sc *Scene, c *ShadeContext, model string) float64 {
	switch model {
	case "kajiya_kay":
		return kajiyaGain * kajiyaSpec(c)
	case "single":
		return luminance(componentMul(marschnerSpec(c, 1.0), singleGain))
	case "double":
		local := componentMul(marschnerSpec(c, 1.0), singleGain)
		return luminance(local) + luminance(componentMul(doubleGlobal(c), globalGain))
	}
	return 0
}

func luminance(c Vec3) float64 {
	return 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z
}

// toneMap applies soft filmic compression and converts linear radiance to sRGB.
func toneMap(c Vec3) color.RGBA {
	m := func(x float64) float64 {
		x = x * (1.6*x + 0.05) / (x*(1.6*x+0.55) + 0.14)
		return clamp01(math.Pow(clamp01(x), 1/2.2))
	}
	return color.RGBA{
		R: uint8(m(c.X) * 255),
		G: uint8(m(c.Y) * 255),
		B: uint8(m(c.Z) * 255),
		A: 255,
	}
}

func renderModel(sc *Scene, g *GBuffer, name string, fn ShaderFunc) *RenderResult {
	img := image.NewRGBA(image.Rect(0, 0, g.W, g.H))
	linear := make([]Vec3, g.W*g.H)
	spec := make([]float64, g.W*g.H)

	for i, p := range g.Pixel {
		var col Vec3
		if p.Hit == HitNone {
			col = backgroundAt(sc, g, i)
		} else {
			c := makeShadeContext(sc, &p)
			if p.Hit == HitScalp {
				col = scalpShade(sc, &c)
			} else {
				col = fn(sc, &c)
				spec[i] = math.Max(0, specularOnly(sc, &c, name))
			}
		}
		linear[i] = col
		img.SetRGBA(i%g.W, i/g.W, toneMap(col))
	}
	return &RenderResult{Name: name, Image: img, Linear: linear, Spec: spec}
}

func renderAll(sc *Scene, g *GBuffer) []*RenderResult {
	out := make([]*RenderResult, 0, len(shaders))
	for _, sh := range shaders {
		out = append(out, renderModel(sc, g, sh.Name, sh.Fn))
	}
	return out
}

// backgroundAt paints a subtle vertical vignette outside the scalp.
func backgroundAt(sc *Scene, g *GBuffer, i int) Vec3 {
	y := i / g.W
	t := float64(y) / float64(g.H)
	return v(0.045, 0.05, 0.06).Scale(1.2 - 0.5*t)
}
