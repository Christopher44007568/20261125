package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type Canvas struct {
	W, H  int
	Pix   []HDR
	Depth []float64
}

func NewCanvas(w, h int) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]HDR, w*h), Depth: make([]float64, w*h)}
	for i := range c.Depth {
		c.Depth[i] = math.Inf(1)
	}
	return c
}

func (c *Canvas) Idx(x, y int) int { return y*c.W + x }

func (c *Canvas) Set(x, y int, v HDR) {
	if x >= 0 && y >= 0 && x < c.W && y < c.H {
		c.Pix[y*c.W+x] = v
	}
}

func (c *Canvas) At(x, y int) HDR {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return HDR{}
	}
	return c.Pix[y*c.W+x]
}

func (c *Canvas) DepthAt(x, y int) float64 {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return math.Inf(1)
	}
	return c.Depth[y*c.W+x]
}

func blendOver(dst HDR, src HDR) HDR { return over(dst, src) }

func bgColor(nx, ny float64) RGB {
	r := math.Hypot(nx*0.92, ny)
	deep := C(0.018, 0.030, 0.038)
	mid := C(0.070, 0.095, 0.105)
	warm := C(0.105, 0.075, 0.050)
	g := C(0, 0, 0).Lerp(mid, smoothstep(1.15, 0.05, r))
	g = g.Lerp(deep, smoothstep(0.35, 1.1, r))
	glow := math.Exp(-((nx+0.35)*(nx+0.35)+(ny-0.18)*(ny-0.18))*2.8) * 0.035
	return g.Add(warm.Mul(glow))
}

func (c *Canvas) Downsample(src *Canvas) {
	if src.W == c.W && src.H == c.H {
		copy(c.Pix, src.Pix)
		copy(c.Depth, src.Depth)
		return
	}
	sx := float64(src.W) / float64(c.W)
	sy := float64(src.H) / float64(c.H)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			var acc HDR
			n := 0
			for oy := 0; oy < 2; oy++ {
				for ox := 0; ox < 2; ox++ {
					px := int((float64(x) + 0.25 + float64(ox)*0.5) * sx)
					py := int((float64(y) + 0.25 + float64(oy)*0.5) * sy)
					acc = acc.Add(src.At(px, py))
					n++
				}
			}
			c.Pix[y*c.W+x] = acc.Scale(1 / float64(n))
			// The nearest subsample is a stable representative for LOD selection.
			c.Depth[y*c.W+x] = src.DepthAt(x*2, y*2)
		}
	}
}

func (h HDR) Add(o HDR) HDR {
	return HDR{h.R + o.R, h.G + o.G, h.B + o.B, h.A + o.A}
}

func (h HDR) Scale(s float64) HDR {
	return HDR{h.R * s, h.G * s, h.B * s, h.A * s}
}

func (c *Canvas) ToImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for i, p := range c.Pix {
		r := toSRGB(toneMap(p.R))
		g := toSRGB(toneMap(p.G))
		b := toSRGB(toneMap(p.B))
		img.Pix[i*4+0] = uint8(clamp(r, 0, 1)*255 + 0.5)
		img.Pix[i*4+1] = uint8(clamp(g, 0, 1)*255 + 0.5)
		img.Pix[i*4+2] = uint8(clamp(b, 0, 1)*255 + 0.5)
		img.Pix[i*4+3] = 255
	}
	return img
}

func savePNG(path string, c *Canvas) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(f, c.ToImage())
}

func saveImage(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(f, img)
}

func rgba(r, g, b float64, a float64) color.RGBA {
	return color.RGBA{
		R: uint8(clamp(r, 0, 1)*255 + 0.5),
		G: uint8(clamp(g, 0, 1)*255 + 0.5),
		B: uint8(clamp(b, 0, 1)*255 + 0.5),
		A: uint8(clamp(a, 0, 1)*255 + 0.5),
	}
}
