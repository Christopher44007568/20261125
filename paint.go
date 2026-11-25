package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
)

type Plot struct {
	W, H   int
	SS     int
	View   image.Rectangle
	Min    Vec2
	Max    Vec2
	hi     *image.RGBA
	scale  float64
	center Vec2
}

func NewPlot(w, h, ss int, view image.Rectangle, min, max Vec2) *Plot {
	if ss < 1 {
		ss = 1
	}
	hi := image.NewRGBA(image.Rect(0, 0, w*ss, h*ss))
	dx := max.X - min.X
	dy := max.Y - min.Y
	scale := math.Min(float64(view.Dx())/dx, float64(view.Dy())/dy)
	return &Plot{
		W: w, H: h, SS: ss, View: view, Min: min, Max: max,
		hi: hi, scale: scale,
		center: v2((min.X+max.X)/2, (min.Y+max.Y)/2),
	}
}

func (p *Plot) pixelX(x float64) float64 {
	cx := float64(p.View.Min.X + p.View.Dx()/2)
	return float64((p.View.Min.X)) + (cx - float64(p.View.Min.X)) + (x-p.center.X)*p.scale
}

func (p *Plot) X(x float64) float64 {
	return float64(p.SS) * (float64(p.View.Min.X) + float64(p.View.Dx())/2 + (x-p.center.X)*p.scale)
}

func (p *Plot) Y(y float64) float64 {
	return float64(p.SS) * (float64(p.View.Min.Y) + float64(p.View.Dy())/2 - (y-p.center.Y)*p.scale)
}

func (p *Plot) Clear(c color.RGBA) {
	for i := range p.hi.Pix {
		if i%4 == 3 {
			p.hi.Pix[i] = 255
		} else {
			p.hi.Pix[i] = c.R
			if i%4 == 1 {
				p.hi.Pix[i] = c.G
			} else if i%4 == 2 {
				p.hi.Pix[i] = c.B
			}
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, c color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	off := img.PixOffset(x, y)
	sa := uint32(c.A)
	ia := 255 - sa
	img.Pix[off+0] = uint8((uint32(c.R)*sa + uint32(img.Pix[off+0])*ia) / 255)
	img.Pix[off+1] = uint8((uint32(c.G)*sa + uint32(img.Pix[off+1])*ia) / 255)
	img.Pix[off+2] = uint8((uint32(c.B)*sa + uint32(img.Pix[off+2])*ia) / 255)
	img.Pix[off+3] = 255
}

func (p *Plot) FillPolygonWorld(poly []Vec2, fill color.RGBA, hatch *color.RGBA) {
	if len(poly) < 3 {
		return
	}
	ps := make([]Vec2, len(poly))
	for i, q := range poly {
		ps[i] = v2(p.X(q.X), p.Y(q.Y))
	}
	fillPixelPolygon(p.hi, ps, fill, hatch, p.SS)
}

func fillPixelPolygon(img *image.RGBA, ps []Vec2, fill color.RGBA, hatch *color.RGBA, ss int) {
	minY := math.Max(0, math.Floor(minVec(ps, 1))-1)
	maxY := math.Min(float64(img.Bounds().Dy()-1), math.Ceil(maxVec(ps, 1))+1)
	for y := int(minY); y <= int(maxY); y++ {
		yc := float64(y) + 0.5
		var xs []float64
		for i := range ps {
			a := ps[i]
			b := ps[(i+1)%len(ps)]
			if (a.Y <= yc && b.Y > yc) || (b.Y <= yc && a.Y > yc) {
				x := a.X + (yc-a.Y)*(b.X-a.X)/(b.Y-a.Y)
				xs = append(xs, x)
			}
		}
		sort.Float64s(xs)
		for k := 0; k+1 < len(xs); k += 2 {
			x0 := int(math.Ceil(xs[k] - 0.5))
			x1 := int(math.Floor(xs[k+1] - 0.5))
			for x := x0; x <= x1; x++ {
				blendPixel(img, x, y, fill)
				if hatch != nil {
					period := 13 * ss
					thick := 2 * ss
					mod := (x - y) % period
					if mod < 0 {
						mod += period
					}
					if mod < thick {
						blendPixel(img, x, y, *hatch)
					}
				}
			}
		}
	}
}

func minVec(ps []Vec2, coord int) float64 {
	m := ps[0].X
	if coord == 1 {
		m = ps[0].Y
	}
	for _, p := range ps[1:] {
		v := p.X
		if coord == 1 {
			v = p.Y
		}
		m = math.Min(m, v)
	}
	return m
}

func maxVec(ps []Vec2, coord int) float64 {
	m := ps[0].X
	if coord == 1 {
		m = ps[0].Y
	}
	for _, p := range ps[1:] {
		v := p.X
		if coord == 1 {
			v = p.Y
		}
		m = math.Max(m, v)
	}
	return m
}

func (p *Plot) LineWorld(a, b Vec2, width float64, c color.RGBA) {
	ax, ay := p.X(a.X), p.Y(a.Y)
	bx, by := p.X(b.X), p.Y(b.Y)
	dx, dy := bx-ax, by-ay
	length := math.Hypot(dx, dy)
	if length < 1e-6 {
		p.DiscWorld(a, width/2, c)
		return
	}
	hw := width * float64(p.SS) / 2
	nx, ny := -dy/length*hw, dx/length*hw
	quad := []Vec2{
		{X: ax + nx, Y: ay + ny},
		{X: bx + nx, Y: by + ny},
		{X: bx - nx, Y: by - ny},
		{X: ax - nx, Y: ay - ny},
	}
	fillPixelPolygon(p.hi, quad, c, nil, p.SS)
	p.DiscWorld(a, width/2, c)
	p.DiscWorld(b, width/2, c)
}

func (p *Plot) DiscWorld(center Vec2, radiusFinal float64, c color.RGBA) {
	cx, cy := p.X(center.X), p.Y(center.Y)
	r := radiusFinal * float64(p.SS)
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			if math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) <= r {
				blendPixel(p.hi, x, y, c)
			}
		}
	}
}

func (p *Plot) StrokeWorld(points []Vec2, closed bool, width float64, c color.RGBA) {
	if len(points) < 2 {
		return
	}
	for i := 0; i+1 < len(points); i++ {
		p.LineWorld(points[i], points[i+1], width, c)
	}
	if closed {
		p.LineWorld(points[len(points)-1], points[0], width, c)
	}
}

func (p *Plot) Finish() *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, p.W, p.H))
	for y := 0; y < p.H; y++ {
		for x := 0; x < p.W; x++ {
			var r, g, b, count uint32
			for sy := 0; sy < p.SS; sy++ {
				for sx := 0; sx < p.SS; sx++ {
					hx := x*p.SS + sx
					hy := y*p.SS + sy
					off := p.hi.PixOffset(hx, hy)
					r += uint32(p.hi.Pix[off+0])
					g += uint32(p.hi.Pix[off+1])
					b += uint32(p.hi.Pix[off+2])
					count++
				}
			}
			off := out.PixOffset(x, y)
			out.Pix[off+0] = uint8(r / count)
			out.Pix[off+1] = uint8(g / count)
			out.Pix[off+2] = uint8(b / count)
			out.Pix[off+3] = 255
		}
	}
	return out
}

func WritePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
