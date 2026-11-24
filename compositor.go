package main

import "math"

type LayerFunc func(x, y, eye int) Sample

type Layer struct {
	Name   string
	Space  Transfer
	Premul bool
	Render LayerFunc
}

type LayerFrame struct {
	Name       string
	Eye        int
	Input      [][]Sample
	Decoded    [][]PremulLinear
	Composited [][]PremulLinear
}

type Scene struct {
	Width  int
	Height int
	Layers []Layer
}

func newPremulFrame(width, height int) [][]PremulLinear {
	frame := make([][]PremulLinear, height)
	for y := range frame {
		frame[y] = make([]PremulLinear, width)
	}
	return frame
}

func newSampleFrame(width, height int) [][]Sample {
	frame := make([][]Sample, height)
	for y := range frame {
		frame[y] = make([]Sample, width)
	}
	return frame
}

func over(dst, src PremulLinear) PremulLinear {
	alpha := clampAlpha(src.A)
	invAlpha := 1 - alpha
	return PremulLinear{
		R: src.R + dst.R*invAlpha,
		G: src.G + dst.G*invAlpha,
		B: src.B + dst.B*invAlpha,
		A: alpha + dst.A*invAlpha,
	}
}

func (s Scene) CompositeEye(eye int) ([]LayerFrame, [][]PremulLinear) {
	current := newPremulFrame(s.Width, s.Height)
	records := make([]LayerFrame, 0, len(s.Layers))

	for _, layer := range s.Layers {
		input := newSampleFrame(s.Width, s.Height)
		decoded := newPremulFrame(s.Width, s.Height)
		composited := newPremulFrame(s.Width, s.Height)

		for y := 0; y < s.Height; y++ {
			for x := 0; x < s.Width; x++ {
				sample := layer.Render(x, y, eye)
				sample.Space = layer.Space
				sample.Premul = layer.Premul
				sample.A = clampAlpha(sample.A)

				input[y][x] = sample
				decoded[y][x] = decodePremultiplied(sample)
				current[y][x] = over(current[y][x], decoded[y][x])
				composited[y][x] = current[y][x]
			}
		}

		records = append(records, LayerFrame{
			Name:       layer.Name,
			Eye:        eye,
			Input:      input,
			Decoded:    decoded,
			Composited: composited,
		})
	}

	return records, current
}

func (s Scene) Composite() ([][]LayerFrame, [][][]PremulLinear) {
	allRecords := make([][]LayerFrame, 2)
	final := make([][][]PremulLinear, 2)
	for eye := 0; eye < 2; eye++ {
		allRecords[eye], final[eye] = s.CompositeEye(eye)
	}
	return allRecords, final
}

func wrongEncodedPremul(sample Sample) PremulLinear {
	alpha := clampAlpha(sample.A)
	if alpha == 0 {
		return PremulLinear{A: 0}
	}

	r, g, b := sample.R, sample.G, sample.B
	if sample.Premul {
		r = math.Max(0, r/alpha)
		g = math.Max(0, g/alpha)
		b = math.Max(0, b/alpha)
	}
	if sample.Space == Linear {
		r = linearTosRGB(clamp01(r))
		g = linearTosRGB(clamp01(g))
		b = linearTosRGB(clamp01(b))
	}

	return PremulLinear{
		R: clamp01(r) * alpha,
		G: clamp01(g) * alpha,
		B: clamp01(b) * alpha,
		A: alpha,
	}
}

func (s Scene) CompositeWrongEncodedEye(eye int) [][]PremulLinear {
	current := newPremulFrame(s.Width, s.Height)
	for _, layer := range s.Layers {
		for y := 0; y < s.Height; y++ {
			for x := 0; x < s.Width; x++ {
				sample := layer.Render(x, y, eye)
				sample.Space = layer.Space
				sample.Premul = layer.Premul
				current[y][x] = over(current[y][x], wrongEncodedPremul(sample))
			}
		}
	}
	return current
}

func (s Scene) CompositeDirectAdditionEye(eye int) [][]PremulLinear {
	current := newPremulFrame(s.Width, s.Height)
	for _, layer := range s.Layers {
		for y := 0; y < s.Height; y++ {
			for x := 0; x < s.Width; x++ {
				sample := layer.Render(x, y, eye)
				sample.Space = layer.Space
				sample.Premul = layer.Premul
				encoded := decodePremultiplied(sample)
				current[y][x] = PremulLinear{
					R: current[y][x].R + encoded.R,
					G: current[y][x].G + encoded.G,
					B: current[y][x].B + encoded.B,
					A: clampAlpha(current[y][x].A + encoded.A),
				}
			}
		}
	}
	return current
}

func correctEncoded(frame [][]PremulLinear) [][][3]float64 {
	out := make([][][3]float64, len(frame))
	for y, row := range frame {
		out[y] = make([][3]float64, len(row))
		for x, pixel := range row {
			r, g, b, _ := pixel.Straight()
			r, g, b = toneMap(r, g, b)
			out[y][x] = [3]float64{linearTosRGB(r), linearTosRGB(g), linearTosRGB(b)}
		}
	}
	return out
}

type GammaError struct {
	R, G, B    float64
	Valid      bool
	LinearRMSE float64
}

func gammaErrors(correct, wrong [][]PremulLinear, wrongSpace Transfer) [][]GammaError {
	encodedCorrect := correctEncoded(correct)
	wrongDisplay := encodedCorrect
	if wrongSpace == SRGB {
		wrongDisplay = make([][][3]float64, len(wrong))
		for y, row := range wrong {
			wrongDisplay[y] = make([][3]float64, len(row))
			for x, pixel := range row {
				r, g, b, _ := pixel.Straight()
				wrongDisplay[y][x] = [3]float64{clamp01(r), clamp01(g), clamp01(b)}
			}
		}
	} else {
		wrongDisplay = correctEncoded(wrong)
	}
	errors := make([][]GammaError, len(correct))

	for y := range correct {
		errors[y] = make([]GammaError, len(correct[y]))
		for x := range wrong[y] {
			correctPixel := encodedCorrect[y][x]
			wrongPixelDisplay := wrongDisplay[y][x]
			values := [][2]float64{
				{correctPixel[0], wrongPixelDisplay[0]},
				{correctPixel[1], wrongPixelDisplay[1]},
				{correctPixel[2], wrongPixelDisplay[2]},
			}

			metric := GammaError{R: math.NaN(), G: math.NaN(), B: math.NaN()}
			valid := true
			var sumSquared float64
			for channel, pair := range values {
				c, n := pair[0], pair[1]
				sumSquared += math.Pow(sRGBToLinear(n)-sRGBToLinear(c), 2)
				if c <= 1e-5 || n <= 1e-5 || c >= 0.999999 || n >= 0.999999 {
					valid = false
					continue
				}
				delta := math.Log(n)/math.Log(c) - 1
				switch channel {
				case 0:
					metric.R = delta
				case 1:
					metric.G = delta
				case 2:
					metric.B = delta
				}
			}

			metric.Valid = valid && !math.IsNaN(metric.R) && !math.IsNaN(metric.G) && !math.IsNaN(metric.B)
			metric.LinearRMSE = math.Sqrt(sumSquared / 3)
			errors[y][x] = metric
		}
	}
	return errors
}
