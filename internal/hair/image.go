package hair

import (
	"image"
	"image/color"
	"math"
	"sort"
)

type Palette int

const (
	PaletteHair Palette = iota
	PaletteGuides
	PaletteLOD
)

func ColorizeField(field *DensityField, palette Palette) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, field.Width, field.Height))
	for y := 0; y < field.Height; y++ {
		bg := background(y, field.Height)
		for x := 0; x < field.Width; x++ {
			a := AlphaAt(field, x, y)
			v := math.Pow(clamp(a, 0, 1), 0.70)
			tint := hairTint(palette, v)
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(clamp(float64(bg.R)*(1-v)+float64(tint.R)*v, 0, 255)),
				G: uint8(clamp(float64(bg.G)*(1-v)+float64(tint.G)*v, 0, 255)),
				B: uint8(clamp(float64(bg.B)*(1-v)+float64(tint.B)*v, 0, 255)),
				A: 255,
			})
		}
	}
	return img
}

func background(y, height int) color.RGBA {
	t := 0.0
	if height > 1 {
		t = float64(y) / float64(height-1)
	}
	return color.RGBA{
		R: uint8(lerp(11, 4, t)),
		G: uint8(lerp(16, 7, t)),
		B: uint8(lerp(25, 12, t)),
		A: 255,
	}
}

func hairTint(palette Palette, v float64) color.RGBA {
	highlight := 0.0
	if v > 0.72 {
		highlight = (v - 0.72) / 0.28
	}
	switch palette {
	case PaletteGuides:
		return color.RGBA{
			R: byteClamp(42 + 110*v + 70*highlight),
			G: byteClamp(125 + 115*v + 15*highlight),
			B: byteClamp(138 + 95*v + 18*highlight),
			A: 255,
		}
	case PaletteLOD:
		return color.RGBA{
			R: byteClamp(52 + 203*v + 18*highlight),
			G: byteClamp(45 + 145*v + 28*highlight),
			B: byteClamp(38 + 84*v + 40*highlight),
			A: 255,
		}
	default:
		return color.RGBA{
			R: byteClamp(50 + 199*v + 24*highlight),
			G: byteClamp(42 + 143*v + 30*highlight),
			B: byteClamp(34 + 79*v + 44*highlight),
			A: 255,
		}
	}
}

func byteClamp(v float64) uint8 {
	return uint8(clamp(math.Round(v), 0, 255))
}

type ErrorReport struct {
	MeanAbs       float64  `json:"meanAbs"`
	P95Abs        float64  `json:"p95Abs"`
	MaxAbs        float64  `json:"maxAbs"`
	CoverageRatio float64  `json:"coverageRatio"`
	IoU           float64  `json:"iou"`
	FullCoverage  Coverage `json:"fullCoverage"`
	LODCoverage   Coverage `json:"lodCoverage"`
}

func ErrorHeatmap(full, lod *DensityField) (*image.RGBA, ErrorReport) {
	report := ErrorReport{
		FullCoverage: full.Coverage(0.02),
		LODCoverage:  lod.Coverage(0.02),
	}
	if report.FullCoverage.Pixels > 0 {
		report.CoverageRatio = float64(report.LODCoverage.Pixels) / float64(report.FullCoverage.Pixels)
	}
	fullMask := full.Occupancy(0.02)
	lodMask := lod.Occupancy(0.02)
	report.IoU = maskIoU(fullMask, lodMask)

	fullSmooth := blurField(full, 2)
	lodSmooth := blurField(lod, 2)
	img := image.NewRGBA(image.Rect(0, 0, full.Width, full.Height))
	hist := make([]int, 256)
	var sum float64
	maxErr := 0.0
	for y := 0; y < full.Height; y++ {
		bg := background(y, full.Height)
		for x := 0; x < full.Width; x++ {
			err := math.Abs(AlphaAt(fullSmooth, x, y) - AlphaAt(lodSmooth, x, y))
			sum += err
			if err > maxErr {
				maxErr = err
			}
			hist[int(clamp(err*255, 0, 255))]++
			t := math.Sqrt(clamp(err/0.55, 0, 1))
			c := heatColor(t)
			blendAmount := clamp(t*0.92, 0, 1)
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(lerp(float64(bg.R), float64(c.R), blendAmount)),
				G: uint8(lerp(float64(bg.G), float64(c.G), blendAmount)),
				B: uint8(lerp(float64(bg.B), float64(c.B), blendAmount)),
				A: 255,
			})
		}
	}
	pixels := full.Width * full.Height
	report.MeanAbs = sum / float64(pixels)
	report.MaxAbs = maxErr
	report.P95Abs = percentileFromHistogram(hist, pixels, 0.95)
	return img, report
}

func blurField(field *DensityField, radius int) *DensityField {
	if radius < 1 {
		return field
	}
	tmp := &DensityField{Width: field.Width, Height: field.Height, Data: make([]float64, len(field.Data))}
	out := NewDensityField(field.Width, field.Height)
	for y := 0; y < field.Height; y++ {
		for x := 0; x < field.Width; x++ {
			var sum float64
			var count int
			for dx := -radius; dx <= radius; dx++ {
				xx := x + dx
				if xx < 0 || xx >= field.Width {
					continue
				}
				sum += field.Data[y*field.Width+xx]
				count++
			}
			tmp.Data[y*field.Width+x] = sum / float64(count)
		}
	}
	for y := 0; y < field.Height; y++ {
		for x := 0; x < field.Width; x++ {
			var sum float64
			var count int
			for dy := -radius; dy <= radius; dy++ {
				yy := y + dy
				if yy < 0 || yy >= field.Height {
					continue
				}
				sum += tmp.Data[yy*field.Width+x]
				count++
			}
			out.Data[y*field.Width+x] = sum / float64(count)
		}
	}
	return out
}

func heatColor(t float64) color.RGBA {
	stops := []struct {
		t float64
		c [3]float64
	}{
		{0.00, [3]float64{10, 18, 38}},
		{0.18, [3]float64{24, 74, 155}},
		{0.38, [3]float64{0, 171, 193}},
		{0.58, [3]float64{93, 214, 81}},
		{0.75, [3]float64{247, 205, 45}},
		{0.89, [3]float64{244, 99, 28}},
		{1.00, [3]float64{185, 13, 50}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].t {
			f := (t - stops[i-1].t) / (stops[i].t - stops[i-1].t)
			return color.RGBA{
				R: uint8(lerp(stops[i-1].c[0], stops[i].c[0], f)),
				G: uint8(lerp(stops[i-1].c[1], stops[i].c[1], f)),
				B: uint8(lerp(stops[i-1].c[2], stops[i].c[2], f)),
				A: 255,
			}
		}
	}
	last := stops[len(stops)-1].c
	return color.RGBA{uint8(last[0]), uint8(last[1]), uint8(last[2]), 255}
}

func percentileFromHistogram(hist []int, total int, q float64) float64 {
	if total == 0 {
		return 0
	}
	target := int(math.Ceil(float64(total) * q))
	var accum int
	for i, n := range hist {
		accum += n
		if accum >= target {
			return float64(i) / 255
		}
	}
	return 1
}

func SortedDistances(start, end float64, count int) []float64 {
	if count < 2 {
		count = 2
	}
	out := make([]float64, count)
	for i := range out {
		t := float64(i) / float64(count-1)
		out[i] = lerp(start, end, t)
	}
	sort.Float64s(out)
	return out
}
