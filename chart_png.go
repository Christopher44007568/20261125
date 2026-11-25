package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type pngPlot struct {
	img *image.RGBA
}

func WriteMetricsPNG(path string, metrics []FrameMetric) error {
	const w = 1200
	const h = 760
	plot := &pngPlot{img: image.NewRGBA(image.Rect(0, 0, w, h))}
	plot.clear(color.RGBA{R: 255, G: 255, B: 255, A: 255})

	top := image.Rect(78, 64, 1140, 330)
	bottom := image.Rect(78, 430, 1140, 690)
	plot.drawBrightnessPanel(top, metrics)
	plot.drawTrianglePanel(bottom, metrics)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, plot.img)
}

func (p *pngPlot) clear(c color.RGBA) {
	for i := 0; i < len(p.img.Pix); i += 4 {
		p.img.Pix[i+0] = c.R
		p.img.Pix[i+1] = c.G
		p.img.Pix[i+2] = c.B
		p.img.Pix[i+3] = c.A
	}
}

func (p *pngPlot) pixel(x, y int, c color.RGBA) {
	b := p.img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	o := y*p.img.Stride + x*4
	p.img.Pix[o+0] = c.R
	p.img.Pix[o+1] = c.G
	p.img.Pix[o+2] = c.B
	p.img.Pix[o+3] = 255
}

func (p *pngPlot) rect(r image.Rectangle, c color.RGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			p.pixel(x, y, c)
		}
	}
}

func (p *pngPlot) line(x0, y0, x1, y1 int, c color.RGBA, thickness int) {
	dx := int(math.Abs(float64(x1 - x0)))
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -int(math.Abs(float64(y1 - y0)))
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	radius := thickness / 2
	for {
		for oy := -radius; oy <= radius; oy++ {
			for ox := -radius; ox <= radius; ox++ {
				if ox*ox+oy*oy <= radius*radius+radius {
					p.pixel(x0+ox, y0+oy, c)
				}
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (p *pngPlot) frame(r image.Rectangle) {
	axis := color.RGBA{R: 55, G: 65, B: 81, A: 255}
	for x := r.Min.X; x <= r.Max.X; x++ {
		p.pixel(x, r.Max.Y, axis)
		p.pixel(x, r.Min.Y, color.RGBA{R: 229, G: 231, B: 235, A: 255})
	}
	for y := r.Min.Y; y <= r.Max.Y; y++ {
		p.pixel(r.Min.X, y, axis)
		p.pixel(r.Max.X, y, color.RGBA{R: 229, G: 231, B: 235, A: 255})
	}
}

func xForMetric(r image.Rectangle, i, n int) int {
	if n <= 1 {
		return r.Min.X
	}
	return r.Min.X + int(float64(i)/float64(n-1)*float64(r.Dx()))
}

func transitionBandsPNG(p *pngPlot, r image.Rectangle, metrics []FrameMetric) {
	n := len(metrics)
	for i := 0; i < n; {
		if metrics[i].Blend <= 0 || metrics[i].LODFrom == metrics[i].LODTo {
			i++
			continue
		}
		start := i
		from := metrics[i].LODFrom
		for i < n && metrics[i].Blend > 0 && metrics[i].LODFrom == from {
			i++
		}
		end := i - 1
		x0 := xForMetric(r, start, n)
		x1 := xForMetric(r, end, n)
		step := 0
		if n > 1 {
			step = int(float64(r.Dx()) / float64(n-1))
		}
		band := color.RGBA{R: 255, G: 237, B: 213, A: 255}
		p.rect(image.Rect(x0-step/2, r.Min.Y, x1+step/2, r.Max.Y), band)
	}
}

var pngTextColor = color.RGBA{R: 31, G: 41, B: 55, A: 255}

func (p *pngPlot) text(x, y int, s string, scale int, c color.RGBA) {
	cursor := x
	for _, ch := range strings.ToUpper(s) {
		glyph, ok := font5x7[ch]
		if !ok {
			if ch == ' ' {
				cursor += 4 * scale
			}
			continue
		}
		for gy, row := range glyph {
			for gx, cell := range row {
				if cell != '1' {
					continue
				}
				p.rect(image.Rect(
					cursor+gx*scale, y+gy*scale,
					cursor+(gx+1)*scale, y+(gy+1)*scale,
				), c)
			}
		}
		cursor += 6 * scale
	}
}

var (
	blueCurve   = color.RGBA{R: 37, G: 99, B: 235, A: 255}
	orangeCurve = color.RGBA{R: 180, G: 83, B: 9, A: 255}
	barStable   = color.RGBA{R: 249, G: 115, B: 22, A: 255}
	barMorph    = color.RGBA{R: 245, G: 158, B: 11, A: 255}
	gridColor   = color.RGBA{R: 229, G: 231, B: 235, A: 255}
	tickColor   = color.RGBA{R: 107, G: 114, B: 128, A: 255}
)

func (p *pngPlot) drawBrightnessPanel(r image.Rectangle, metrics []FrameMetric) {
	p.text(r.Min.X, 24, "AVERAGE BRIGHTNESS  REC709 LUMA 0 TO 255", 2, pngTextColor)
	transitionBandsPNG(p, r, metrics)

	yFor := func(v float64) int {
		return r.Max.Y - int(clampFloat(v/255, 0, 1)*float64(r.Dy()))
	}
	for v := 0; v <= 255; v += 50 {
		y := yFor(float64(v))
		p.line(r.Min.X, y, r.Max.X, y, gridColor, 1)
		p.text(r.Min.X-38, y-7, fmt.Sprint(v), 1, tickColor)
	}

	n := len(metrics)
	for i := 1; i < n; i++ {
		x0 := xForMetric(r, i-1, n)
		x1 := xForMetric(r, i, n)
		y0 := yFor(metrics[i-1].MeanLuma)
		y1 := yFor(metrics[i].MeanLuma)
		p.line(x0, y0, x1, y1, blueCurve, 3)
		hy0 := yFor(metrics[i-1].HairLuma)
		hy1 := yFor(metrics[i].HairLuma)
		p.line(x0, hy0, x1, hy1, orangeCurve, 2)
	}
	p.frame(r)

	for i := 0; i < n; i += 12 {
		x := xForMetric(r, i, n)
		p.text(x-12, r.Max.Y+10, fmt.Sprintf("%03d", i), 1, tickColor)
	}
	p.rect(image.Rect(r.Max.X-250, 31, r.Max.X-222, 35), blueCurve)
	p.text(r.Max.X-214, 27, "MEAN", 1, pngTextColor)
	p.rect(image.Rect(r.Max.X-140, 31, r.Max.X-112, 35), orangeCurve)
	p.text(r.Max.X-104, 27, "HAIR", 1, pngTextColor)
}

func (p *pngPlot) drawTrianglePanel(r image.Rectangle, metrics []FrameMetric) {
	p.text(r.Min.X, 392, "HAIR TRIANGLE COUNT", 2, pngTextColor)
	transitionBandsPNG(p, r, metrics)
	if len(metrics) == 0 {
		return
	}

	maxTris := float64(metrics[0].LOD0Triangles) * 1.10
	yFor := func(v int) int {
		return r.Max.Y - int(clampFloat(float64(v)/maxTris, 0, 1)*float64(r.Dy()))
	}
	for t := 0.0; t <= 1.0001; t += 0.2 {
		v := int(maxTris * t)
		y := yFor(v)
		p.line(r.Min.X, y, r.Max.X, y, gridColor, 1)
		p.text(r.Min.X-48, y-7, formatTriangleTick(float64(v)), 1, tickColor)
	}

	n := len(metrics)
	slot := float64(r.Dx()) / float64(n)
	barW := int(math.Max(2, slot*0.66))
	for i, m := range metrics {
		xCenter := r.Min.X + int(float64(i)*slot+slot/2)
		y := yFor(m.HairTriangles)
		c := barStable
		if m.Blend > 0 && m.LODFrom != m.LODTo {
			c = barMorph
		}
		p.rect(image.Rect(xCenter-barW/2, y, xCenter+barW/2, r.Max.Y), c)
	}

	p.dashedHorizontal(yFor(metrics[0].LOD0Triangles), r, color.RGBA{R: 154, G: 52, B: 18, A: 255})
	p.dashedHorizontal(yFor(metrics[0].LOD1Triangles), r, color.RGBA{R: 180, G: 83, B: 9, A: 255})
	p.dashedHorizontal(yFor(metrics[0].LOD2Triangles), r, color.RGBA{R: 21, G: 128, B: 61, A: 255})
	p.frame(r)

	for i := 0; i < n; i += 12 {
		x := xForMetric(r, i, n)
		p.text(x-12, r.Max.Y+10, fmt.Sprintf("%03d", i), 1, tickColor)
	}
	p.rect(image.Rect(r.Max.X-250, 399, r.Max.X-222, 413), barStable)
	p.text(r.Max.X-214, 399, "STABLE", 1, pngTextColor)
	p.rect(image.Rect(r.Max.X-140, 399, r.Max.X-112, 413), barMorph)
	p.text(r.Max.X-104, 399, "MORPH", 1, pngTextColor)
}

func (p *pngPlot) dashedHorizontal(y int, r image.Rectangle, c color.RGBA) {
	for x := r.Min.X; x <= r.Max.X; x += 10 {
		p.line(x, y, minInt(x+5, r.Max.X), y, c, 1)
	}
}
