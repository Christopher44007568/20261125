package sim

import "math"

// lumaAt is a deterministic continuous scene: layered sine gradients plus a
// soft moving blob. It is defined in normalized UV space so any resolution
// samples the same underlying scene, which makes cross-resolution frame
// jumps directly comparable.
func lumaAt(u, v, t float64) float64 {
	base := 0.315 + 0.08*math.Sin(6.2831853*(u*1.7+t*0.10)) +
		0.07*math.Cos(6.2831853*(v*2.3-t*0.12))
	du := u - (0.5 + 0.16*math.Sin(t*0.09))
	dv := v - (0.45 + 0.10*math.Cos(t*0.07))
	blob := 0.26 * math.Exp(-(du*du*9.0 + dv*dv*12.0))
	return clamp01(base + blob)
}

// frameImage stores per-eye render output at a given size.
type frameImage struct {
	size Size
	// luma is row-major normalized brightness, one entry per pixel.
	luma []float64
}

func renderFrame(size Size, t float64) *frameImage {
	img := &frameImage{size: size, luma: make([]float64, size.pixels())}
	for y := 0; y < size.Height; y++ {
		v := (float64(y) + 0.5) / float64(size.Height)
		row := y * size.Width
		for x := 0; x < size.Width; x++ {
			u := (float64(x) + 0.5) / float64(size.Width)
			img.luma[row+x] = lumaAt(u, v, t)
		}
	}
	return img
}

// meanLuma returns the average brightness over the full image.
func (f *frameImage) meanLuma() float64 {
	sum := 0.0
	for _, v := range f.luma {
		sum += v
	}
	return sum / float64(len(f.luma))
}

// gridMean samples the scene on the fixed 160x90 UV grid from an existing
// image, so frames of different resolutions are compared on equal footing.
func (f *frameImage) gridMean() float64 {
	const gw, gh = 160, 90
	sum := 0.0
	for gy := 0; gy < gh; gy++ {
		v := (float64(gy) + 0.5) / gh
		sy := v * float64(f.size.Height)
		y0 := int(math.Floor(sy - 0.5))
		wy := sy - 0.5 - float64(y0)
		for gx := 0; gx < gw; gx++ {
			u := (float64(gx) + 0.5) / gw
			sx := u * float64(f.size.Width)
			x0 := int(math.Floor(sx - 0.5))
			wx := sx - 0.5 - float64(x0)
			sum += f.bilinear(x0, y0, wx, wy)
		}
	}
	return sum / float64(gw*gh)
}

func (f *frameImage) bilinear(x0, y0 int, wx, wy float64) float64 {
	x1 := x0 + 1
	y1 := y0 + 1
	clampX := func(x int) int {
		if x < 0 {
			return 0
		}
		if x >= f.size.Width {
			return f.size.Width - 1
		}
		return x
	}
	clampY := func(y int) int {
		if y < 0 {
			return 0
		}
		if y >= f.size.Height {
			return f.size.Height - 1
		}
		return y
	}
	p00 := f.luma[clampY(y0)*f.size.Width+clampX(x0)]
	p10 := f.luma[clampY(y0)*f.size.Width+clampX(x1)]
	p01 := f.luma[clampY(y1)*f.size.Width+clampX(x0)]
	p11 := f.luma[clampY(y1)*f.size.Width+clampX(x1)]
	top := p00*(1-wx) + p10*wx
	bot := p01*(1-wx) + p11*wx
	return top*(1-wy) + bot*wy
}
