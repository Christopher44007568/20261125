package main

import (
	"image"
	"math"
	"math/rand"
)

var exposureScales = []float64{0.125, 1.0, 6.0}
var simulatedExposureErrors = []float64{1.02, 1.045, 0.955}
var srgbDecodeLUT [256]float64
var srgbEncode8LUT [4097]uint8

func init() {
	for i := range srgbDecodeLUT {
		srgbDecodeLUT[i] = srgbDecode(float64(i) / 255)
	}
	for i := range srgbEncode8LUT {
		srgbEncode8LUT[i] = uint8(math.Round(srgbEncode(float64(i)/4096) * 255))
	}
}

type ExposureStat struct {
	Index           int     `json:"index"`
	Scale           float64 `json:"scale"`
	NominalEV       float64 `json:"nominal_ev"`
	MeasuredEV      float64 `json:"measured_ev"`
	DeviationEV     float64 `json:"deviation_ev"`
	ClippedFraction float64 `json:"clipped_fraction"`
	DarkFraction    float64 `json:"dark_fraction"`
	UsableFraction  float64 `json:"usable_fraction"`
}

type FusionResult struct {
	Camera          *LinearImage
	Confidence      []float64
	ExposureStats   []ExposureStat
	ReferenceIndex  int
	MaxAbsDeviation float64
}

func captureBackground(width, height int, k Intrinsics, rng *rand.Rand) *CaptureBundle {
	scene := renderScene(width, height, k, rng)
	bundle := &CaptureBundle{
		Intrinsics:   k,
		DepthM:       scene.DepthM,
		DepthValid:   scene.DepthValid,
		SystemLayer:  makeSystemLayer(width, height),
		WhiteBalance: RGB{R: 1.035, G: 1.0, B: 0.935},
		Sphere:       defaultSphere(),
	}
	pixCount := width * height
	noise := make([]float64, pixCount*len(exposureScales))
	for i := range noise {
		noise[i] = (rng.Float64() - 0.5) * 0.0014
	}
	for i, scale := range exposureScales {
		actual := scale * simulatedExposureErrors[i]
		frame := image.NewNRGBA(image.Rect(0, 0, width, height))
		noiseBase := i * pixCount
		parallelRows(height, func(y int) {
			for x := 0; x < width; x++ {
				p := y*width + x
				cameraRadiance := scene.Radiance.Pix[p].Mul(bundle.WhiteBalance).Scale(actual)
				n := noise[noiseBase+p]
				offset := p * 4
				frame.Pix[offset+0] = srgbEncode8(cameraRadiance.R + n)
				frame.Pix[offset+1] = srgbEncode8(cameraRadiance.G + n)
				frame.Pix[offset+2] = srgbEncode8(cameraRadiance.B + n)
				frame.Pix[offset+3] = 255
			}
		})
		bundle.Exposures = append(bundle.Exposures, ExposureFrame{
			Scale:       scale,
			ActualScale: actual,
			Image:       frame,
		})
	}
	return bundle
}

func fuseExposures(bundle *CaptureBundle) FusionResult {
	width, height := bundle.Intrinsics.pixelDimensions()
	if len(bundle.Exposures) == 0 {
		return FusionResult{
			Camera:     newLinearImage(width, height),
			Confidence: make([]float64, width*height),
		}
	}
	camera := newLinearImage(width, height)
	confidence := make([]float64, width*height)
	common := make([]bool, width*height)

	parallelRows(height, func(y int) {
		for x := 0; x < width; x++ {
			p := y*width + x
			weighted := RGB{}
			weightSum := 0.0
			best := 0
			bestPeak := math.NaN()
			bestColor := RGB{}
			pixelCommon := bundle.DepthValid[p]
			for e, frame := range bundle.Exposures {
				offset := p * 4
				c := RGB{
					R: srgbDecodeLUT[frame.Image.Pix[offset+0]],
					G: srgbDecodeLUT[frame.Image.Pix[offset+1]],
					B: srgbDecodeLUT[frame.Image.Pix[offset+2]],
				}
				peak := c.Max()
				if pixelCommon && (peak >= 0.994 || peak <= 0.0015) {
					pixelCommon = false
				}
				if math.IsNaN(bestPeak) || math.Abs(math.Log2(peak+1e-6)) < math.Abs(math.Log2(bestPeak+1e-6)) {
					best = e
					bestPeak = peak
					bestColor = c
				}
				w := exposureWeight(peak)
				if w <= 0 {
					continue
				}
				normalized := c.Scale(1 / frame.Scale)
				weighted = weighted.Add(normalized.Scale(w))
				weightSum += w
			}
			common[p] = pixelCommon
			if weightSum > 0 {
				camera.Pix[p] = weighted.Scale(1 / weightSum)
				confidence[p] = clamp01(weightSum / 1.10)
			} else {
				camera.Pix[p] = bestColor.Scale(1 / bundle.Exposures[best].Scale)
				confidence[p] = 0.05
			}
		}
	})

	stats, referenceIndex := exposureStats(bundle, common)
	result := FusionResult{
		Camera:         camera,
		Confidence:     confidence,
		ExposureStats:  stats,
		ReferenceIndex: referenceIndex,
	}
	for _, stat := range stats {
		result.MaxAbsDeviation = math.Max(result.MaxAbsDeviation, math.Abs(stat.DeviationEV))
	}
	return result
}

func (k Intrinsics) pixelDimensions() (int, int) {
	return int(math.Round(k.CX * 2)), int(math.Round(k.CY * 2))
}

func exposureWeight(peak float64) float64 {
	if peak >= 0.994 {
		return 0
	}
	if peak <= 0.0015 {
		return 0.01
	}
	distance := (peak - 0.43) / 0.42
	return 0.08 + 0.92*math.Exp(-0.5*distance*distance)
}

func exposureStats(bundle *CaptureBundle, common []bool) ([]ExposureStat, int) {
	pixCount := len(bundle.DepthValid)
	commonCount := countTrue(common)
	validCount := countTrue(bundle.DepthValid)
	if commonCount < 64 {
		copy(common, bundle.DepthValid)
		commonCount = validCount
	}
	stats := make([]ExposureStat, len(bundle.Exposures))
	measured := make([]float64, len(bundle.Exposures))
	for e, frame := range bundle.Exposures {
		var clipped, dark, usable int
		sum := RGB{}
		for p := 0; p < pixCount; p++ {
			offset := p * 4
			c := RGB{
				R: srgbDecodeLUT[frame.Image.Pix[offset+0]],
				G: srgbDecodeLUT[frame.Image.Pix[offset+1]],
				B: srgbDecodeLUT[frame.Image.Pix[offset+2]],
			}
			peak := c.Max()
			if peak >= 0.994 {
				clipped++
			}
			if peak <= 0.0015 {
				dark++
			}
			if bundle.DepthValid[p] && peak < 0.994 && peak > 0.0015 {
				usable++
			}
			if common[p] {
				sum = sum.Add(c.Scale(1 / frame.Scale))
			}
		}
		n := float64(commonCount)
		if n > 0 {
			mean := sum.Scale(1 / n)
			measured[e] = mean.Luma()
		}
		stats[e] = ExposureStat{
			Index:           e,
			Scale:           frame.Scale,
			NominalEV:       math.Log2(frame.Scale),
			ClippedFraction: float64(clipped) / float64(pixCount),
			DarkFraction:    float64(dark) / float64(pixCount),
			UsableFraction:  float64(usable) / float64(pixCount),
		}
	}
	referenceIndex := 0
	bestDistance := math.Inf(1)
	for e, frame := range bundle.Exposures {
		if measured[e] <= 0 {
			continue
		}
		distance := math.Abs(math.Log2(frame.Scale))
		if distance < bestDistance {
			referenceIndex = e
			bestDistance = distance
		}
	}
	reference := measured[referenceIndex]
	for e := range stats {
		stats[e].DeviationEV = math.Log2((measured[e] + 1e-9) / (reference + 1e-9))
		stats[e].MeasuredEV = stats[e].NominalEV + stats[e].DeviationEV
	}
	return stats, referenceIndex
}

func countTrue(values []bool) int {
	n := 0
	for _, v := range values {
		if v {
			n++
		}
	}
	return n
}

func calibrateColor(camera *LinearImage, valid []bool, hint RGB) ColorCalibration {
	if hint.R > 0 && hint.G > 0 && hint.B > 0 && hint.IsFinite() {
		g := hint.Scale(1 / hint.G)
		return ColorCalibration{Gains: g, Method: "sensor-white-balance"}
	}
	sum := RGB{}
	n := 0
	for p, c := range camera.Pix {
		if p >= len(valid) || !valid[p] {
			continue
		}
		l := c.Luma()
		if l < 0.025 || l > 0.82 {
			continue
		}
		maxc := c.Max()
		minc := math.Min(c.R, math.Min(c.G, c.B))
		if maxc > 0 && (maxc-minc)/maxc > 0.35 {
			continue
		}
		sum = sum.Add(c)
		n++
	}
	if n < 16 {
		return ColorCalibration{Gains: RGB{R: 1, G: 1, B: 1}, Method: "neutral-default"}
	}
	mean := sum.Scale(1 / float64(n))
	if mean.G <= 1e-9 {
		return ColorCalibration{Gains: RGB{R: 1, G: 1, B: 1}, Method: "neutral-default"}
	}
	return ColorCalibration{
		Gains:  RGB{R: clamp01(mean.R / mean.G), G: 1, B: clamp01(mean.B / mean.G)},
		Method: "gray-world",
	}
}

func srgbDecode(v float64) float64 {
	v = clamp01(v)
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func srgbEncode(v float64) float64 {
	v = clamp01(v)
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func srgbEncode8(v float64) uint8 {
	idx := int(clamp01(v)*4096 + 0.5)
	if idx > 4096 {
		idx = 4096
	}
	return srgbEncode8LUT[idx]
}

func toneMap8(c RGB) (uint8, uint8, uint8) {
	values := [3]float64{c.R, c.G, c.B}
	for i, v := range values {
		x := math.Max(0, v) * 1.12
		values[i] = (x * (2.51*x + 0.03)) / (x*(2.43*x+0.59) + 0.14)
	}
	return srgbEncode8(values[0]), srgbEncode8(values[1]), srgbEncode8(values[2])
}
