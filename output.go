package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func WriteOutputs(dir string, cfg Config, res *RenderResult) error {
	colorImg := colorImage(res.Color, res.Width, res.Height)
	if err := savePNG(filepath.Join(dir, "color.png"), colorImg); err != nil {
		return err
	}

	maskImg := colorImage(res.LayerMask, res.Width, res.Height)
	drawMaskLegend(maskImg)
	if err := savePNG(filepath.Join(dir, "layer_order.png"), maskImg); err != nil {
		return err
	}

	flowImg := motionImage(res, cfg)
	if err := savePNG(filepath.Join(dir, "motion_vectors.png"), flowImg); err != nil {
		return err
	}

	meta := buildMetadata(cfg, res)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0o644)
}

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func colorImage(pixels []Color, w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i, c := range pixels {
		img.Pix[i*4+0] = encodeSRGB(c.R)
		img.Pix[i*4+1] = encodeSRGB(c.G)
		img.Pix[i*4+2] = encodeSRGB(c.B)
		img.Pix[i*4+3] = 255
	}
	return img
}

func encodeSRGB(v float64) uint8 {
	v = clamp01(v)
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(clamp01(v)*255 + 0.5)
}

func motionImage(res *RenderResult, cfg Config) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, res.Width, res.Height))
	maxSpeed := res.MaxSpeed
	if maxSpeed < 300 {
		maxSpeed = 300
	}
	flowScale := math.Min(2800, math.Max(420, maxSpeed*0.62))
	for i, v := range res.Motion {
		c := res.Color[i]
		lum := 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
		base := RGB(c.R*0.10+lum*0.025, c.G*0.10+lum*0.030, c.B*0.10+lum*0.042)
		mag := v.Len()
		if mag > 2 {
			fc := flowColor(v, flowScale)
			norm := clamp01(mag / flowScale)
			a := (0.30 + 0.70*smoothstep(0.015, 0.16, norm)) *
				(1 - 0.58*smoothstep(0.10, 0.38, lum))
			base = base.Lerp(fc, a)
		}
		img.Pix[i*4+0] = encodeSRGB(base.R)
		img.Pix[i*4+1] = encodeSRGB(base.G)
		img.Pix[i*4+2] = encodeSRGB(base.B)
		img.Pix[i*4+3] = 255
	}
	drawMotionArrows(img, res, flowScale)
	drawFlowKey(img, flowScale, res)
	return img
}

func flowColor(v Vec2, scale float64) Color {
	angle := math.Atan2(v.Y, v.X)
	if angle < 0 {
		angle += 2 * math.Pi
	}
	hue := angle / (2 * math.Pi)
	mag := clamp01(v.Len() / scale)
	val := 0.42 + 0.58*math.Pow(mag, 0.42)
	sat := 0.68 + 0.32*clamp01(mag*3)
	c := hsvToRGB(hue, sat, val)
	return c
}

func hsvToRGB(h, s, v float64) Color {
	h = math.Mod(h, 1)
	if h < 0 {
		h++
	}
	i := int(math.Floor(h * 6))
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	switch i % 6 {
	case 0:
		return RGB(v, t, p)
	case 1:
		return RGB(q, v, p)
	case 2:
		return RGB(p, v, t)
	case 3:
		return RGB(p, q, v)
	case 4:
		return RGB(t, p, v)
	default:
		return RGB(v, p, q)
	}
}

func drawMotionArrows(img *image.NRGBA, res *RenderResult, flowScale float64) {
	const cell = 42
	for cy := cell / 2; cy < res.Height; cy += cell {
		for cx := cell / 2; cx < res.Width; cx += cell {
			best, bestIdx := 0.0, -1
			for y := maxInt(0, cy-cell/2); y < minInt(res.Height, cy+cell/2); y += 2 {
				for x := maxInt(0, cx-cell/2); x < minInt(res.Width, cx+cell/2); x += 2 {
					m := res.Motion[y*res.Width+x].Len()
					if m > best {
						best, bestIdx = m, y*res.Width+x
					}
				}
			}
			if bestIdx < 0 || best < 28 {
				continue
			}
			c := res.Color[bestIdx]
			lum := 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
			if lum > 0.16 {
				continue
			}
			v := res.Motion[bestIdx]
			length := v.Len()
			if length == 0 {
				continue
			}
			dir := v.Mul(1 / length)
			o := Vec2{X: float64(bestIdx % res.Width), Y: float64(bestIdx / res.Width)}
			n := clamp01(math.Sqrt(length / flowScale))
			l := 9 + 23*n
			tip := o.Add(dir.Mul(l))
			drawArrow(img, o.Add(Vec2{X: 1, Y: 1}), tip.Add(Vec2{X: 1, Y: 1}),
				color.NRGBA{R: 10, G: 15, B: 18, A: 190})
			drawArrow(img, o, tip, color.NRGBA{R: 245, G: 248, B: 235, A: 225})
		}
	}
}

func drawArrow(img *image.NRGBA, a, b Vec2, c color.NRGBA) {
	drawLine(img, a, b, c, 1)
	dir := b.Sub(a)
	l := dir.Len()
	if l < 1 {
		return
	}
	dir = dir.Mul(1 / l)
	left := Vec2{X: -dir.Y, Y: dir.X}
	head := b.Sub(dir.Mul(5))
	p1 := head.Add(left.Mul(2.8))
	p2 := head.Sub(left.Mul(2.8))
	drawLine(img, b, p1, c, 1)
	drawLine(img, b, p2, c, 1)
}

func drawLine(img *image.NRGBA, a, b Vec2, c color.NRGBA, radius int) {
	steps := int(math.Ceil(math.Max(math.Abs(b.X-a.X), math.Abs(b.Y-a.Y)))) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(a.X + (b.X-a.X)*t))
		y := int(math.Round(a.Y + (b.Y-a.Y)*t))
		for oy := -radius; oy <= radius; oy++ {
			for ox := -radius; ox <= radius; ox++ {
				blendPixel(img, x+ox, y+oy, c)
			}
		}
	}
}

func blendPixel(img *image.NRGBA, x, y int, c color.NRGBA) {
	if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
		return
	}
	i := img.PixOffset(x, y)
	a := float64(c.A) / 255
	img.Pix[i+0] = uint8(float64(img.Pix[i+0])*(1-a) + float64(c.R)*a)
	img.Pix[i+1] = uint8(float64(img.Pix[i+1])*(1-a) + float64(c.G)*a)
	img.Pix[i+2] = uint8(float64(img.Pix[i+2])*(1-a) + float64(c.B)*a)
	img.Pix[i+3] = 255
}

func fillPanel(img *image.NRGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	rect := image.Rect(x0, y0, x1, y1)
	draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Over)
}
