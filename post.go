package main

import "math"

func boxBlur(src *Canvas, radius int) *Canvas {
	tmp := NewCanvas(src.W, src.H)
	out := NewCanvas(src.W, src.H)
	if radius < 1 {
		copy(out.Pix, src.Pix)
		return out
	}
	inv := 1 / float64(radius*2+1)
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			var acc HDR
			for k := -radius; k <= radius; k++ {
				acc = acc.Add(src.At(x+k, y))
			}
			tmp.Pix[y*src.W+x] = acc.Scale(inv)
		}
	}
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			var acc HDR
			for k := -radius; k <= radius; k++ {
				acc = acc.Add(tmp.At(x, y+k))
			}
			out.Pix[y*src.W+x] = acc.Scale(inv)
		}
	}
	return out
}

func applyBloom(c *Canvas) {
	bright := NewCanvas(c.W, c.H)
	for i, p := range c.Pix {
		lum := 0.2126*p.R + 0.7152*p.G + 0.0722*p.B
		f := smoothstep(0.70, 1.65, lum)
		if f <= 0 {
			continue
		}
		bright.Pix[i] = HDR{R: p.R * f, G: p.G * f, B: p.B * f, A: 1}
	}
	soft := boxBlur(bright, 7)
	wide := boxBlur(bright, 19)
	tint := C(1.04, 0.86, 0.62)
	for i := range c.Pix {
		a, b := soft.Pix[i], wide.Pix[i]
		c.Pix[i].R += (a.R*0.20 + b.R*0.13) * tint.R
		c.Pix[i].G += (a.G*0.20 + b.G*0.13) * tint.G
		c.Pix[i].B += (a.B*0.20 + b.B*0.13) * tint.B
	}
}

func applyFinish(c *Canvas) {
	for y := 0; y < c.H; y++ {
		ny := 1 - 2*(float64(y)+0.5)/float64(c.H)
		for x := 0; x < c.W; x++ {
			nx := 2*(float64(x)+0.5)/float64(c.W) - 1
			r := math.Hypot(nx*0.84, ny)
			vig := 0.66 + 0.34*smoothstep(1.18, 0.18, r)
			i := y*c.W + x
			n := (rand01(x*7+13, y*9+29) - 0.5) * 0.012
			c.Pix[i].R = c.Pix[i].R*vig + n
			c.Pix[i].G = c.Pix[i].G*vig + n*0.9
			c.Pix[i].B = c.Pix[i].B*vig + n*0.75
			c.Pix[i].A = 1
		}
	}
}
