package main

import (
	"image"
	"math"
)

type DisplaySpec struct {
	ID             string
	Label          string
	Standard       string
	PeakNits       float64
	KneeU          float64
	EndU           float64
	OutEndU        float64
	Beta           float64
	ProofWhite     float64
	ShadowMapGamma float64
	Saturation     float64
	Accent         RGB
	Transfer       func(nits float64) float64
}

func DisplaySpecs() []DisplaySpec {
	return []DisplaySpec{
		{
			ID: "sdr", Label: "SDR DISPLAY", Standard: "SRGB / BT.709  100 NIT",
			PeakNits: sdrWhiteNits, KneeU: 0.34, EndU: 92, OutEndU: 0.985, Beta: 0.22, ProofWhite: 0.985, ShadowMapGamma: 1.0, Saturation: 1.12,
			Accent:   RGB{80, 200, 255},
			Transfer: func(nits float64) float64 { return sRGBEncodeLinear(nits / sdrWhiteNits) },
		},
		{
			ID: "pq", Label: "PQ DISPLAY", Standard: "ST 2084  1000 NIT",
			PeakNits: pqPeakNits, KneeU: 0.62, EndU: 92, OutEndU: 9.92, Beta: 0.16, ProofWhite: 0.992, ShadowMapGamma: 1.42, Saturation: 2.15,
			Accent:   RGB{255, 196, 74},
			Transfer: pqEncodeNits,
		},
		{
			ID: "hlg", Label: "HLG DISPLAY", Standard: "BT.2100 HLG  1000 NIT",
			PeakNits: hlgPeakNits, KneeU: 0.50, EndU: 92, OutEndU: 9.76, Beta: 0.30, ProofWhite: 0.976, ShadowMapGamma: 1.04, Saturation: 1.72,
			Accent: RGB{126, 231, 135},
			Transfer: func(nits float64) float64 {
				return hlgEncodeScene(nits / hlgPeakNits)
			},
		},
	}
}

type DisplayFrame struct {
	Spec      DisplaySpec
	Image     *image.NRGBA
	Histogram [256]int
	BinMax    int
}

// projectDisplayU maps HDR nits/100 into display-emitted nits/100. The gray
// anchor is below the knee and therefore remains exactly unchanged.
func projectDisplayU(x float64, spec DisplaySpec) float64 {
	if x <= spec.KneeU {
		return x
	}
	denom := math.Log(1 + spec.Beta*(spec.EndU-spec.KneeU))
	if denom == 0 {
		return spec.KneeU
	}
	t := math.Log(1+spec.Beta*(x-spec.KneeU)) / denom
	t = clamp01(t)
	return spec.KneeU + (spec.OutEndU-spec.KneeU)*t
}

// calibrateProofCode rescales a native transfer code into the ordinary PNG
// soft-proof coordinate. Black and the declared gray anchor are fixed; codes
// above gray are fitted to the display proof white.
func calibrateProofCode(code, codeGray, codeWhite, proofWhite, shadowGamma float64) float64 {
	gray := grayProofCode / 255.0
	if code <= codeGray {
		if codeGray == 0 {
			return 0
		}
		t := clamp01(code / codeGray)
		if shadowGamma > 0 {
			t = math.Pow(t, shadowGamma)
		}
		return gray * t
	}
	if codeWhite == codeGray {
		return gray
	}
	t := (code - codeGray) / (codeWhite - codeGray)
	return gray + (proofWhite-gray)*clamp01(t)
}

func RenderDisplay(scene *Scene, spec DisplaySpec) *DisplayFrame {
	img := image.NewNRGBA(image.Rect(0, 0, scene.Width, scene.Height))
	frame := &DisplayFrame{Spec: spec, Image: img}
	codeGray := spec.Transfer(grayNits)
	codeWhite := spec.Transfer(spec.OutEndU * sdrWhiteNits)

	for i, hdr := range scene.Pixels {
		u := hdr.mul(1.0 / sdrWhiteNits)
		maxU := u.maxChannel()
		factor := 1.0
		if maxU > 1e-12 {
			factor = projectDisplayU(maxU, spec) / maxU
		}
		displayNits := u.mul(factor * sdrWhiteNits)

		var proof RGB
		for channel := 0; channel < 3; channel++ {
			nativeCode := spec.Transfer(displayNits[channel])
			proof[channel] = calibrateProofCode(nativeCode, codeGray, codeWhite, spec.ProofWhite, spec.ShadowMapGamma)
		}
		proof = saturateProof(proof, spec.Saturation)

		offset := i * 4
		img.Pix[offset+0] = clamp255(proof[0])
		img.Pix[offset+1] = clamp255(proof[1])
		img.Pix[offset+2] = clamp255(proof[2])
		img.Pix[offset+3] = 255

		lumaCode := luma709(proof)
		bin := clamp255(lumaCode)
		frame.Histogram[bin]++
		if frame.Histogram[bin] > frame.BinMax {
			frame.BinMax = frame.Histogram[bin]
		}
	}
	return frame
}

func saturateProof(c RGB, amount float64) RGB {
	luma := luma709(c)
	return RGB{
		clamp01(luma + (c[0]-luma)*amount),
		clamp01(luma + (c[1]-luma)*amount),
		clamp01(luma + (c[2]-luma)*amount),
	}
}

func NativeGrayCodes() map[string]float64 {
	return map[string]float64{
		"sdr": roundTo(sRGBEncodeLinear(grayNits/sdrWhiteNits)*255, 0.01),
		"pq":  roundTo(pqEncodeNits(grayNits)*255, 0.01),
		"hlg": roundTo(hlgEncodeScene(grayNits/hlgPeakNits)*255, 0.01),
	}
}

func roundTo(v, unit float64) float64 {
	return math.Round(v/unit) * unit
}
