// Package gamut maps wide-gamut Rec.2020 colors into the sRGB display gamut.
package gamut

import (
	"math"
)

// RGB is an encoded, display-referred component triplet in [0, 1].
type RGB struct {
	R float64
	G float64
	B float64
}

// Method selects how colors outside the sRGB gamut are handled.
type Method uint8

const (
	// Clip clamps the linear sRGB components to [0, 1].
	Clip Method = iota
	// Perceptual holds Oklab lightness and hue while reducing chroma.
	Perceptual
)

// Result is the sRGB encoding of a Rec.2020 input plus its gamut status.
type Result struct {
	SRGB       RGB
	OutOfGamut bool
}

// MapRec2020ToSRGB converts an encoded Rec.2020 color to encoded sRGB.
//
// Out-of-gamut detection is performed once on the unclipped linear sRGB
// components, so both mapping methods report the same mask.
func MapRec2020ToSRGB(in RGB, method Method) Result {
	linear := rec2020ToLinearSRGB(in)
	out := clipLinear(linear)
	if method == Perceptual {
		out = perceptualMap(linear)
	}
	return Result{
		SRGB:       linearSRGBToEncoded(out),
		OutOfGamut: outOfGamut(linear),
	}
}

// InSRGBGamut reports whether a Rec.2020 color is inside the sRGB gamut.
func InSRGBGamut(in RGB) bool {
	return !outOfGamut(rec2020ToLinearSRGB(in))
}

// SRGBToRec2020 converts an encoded sRGB color to an encoded Rec.2020 color.
// It is useful for describing test colors that originate in sRGB while still
// exercising the Rec.2020 to sRGB mapping pipeline.
func SRGBToRec2020(in RGB) RGB {
	linear := RGB{
		R: srgbEOTF(in.R),
		G: srgbEOTF(in.G),
		B: srgbEOTF(in.B),
	}
	xyzColor := linearSRGBToXYZ(linear)
	return rec2020LinearToEncoded(xyzToRec2020Linear(xyzColor))
}

func rec2020ToLinearSRGB(in RGB) RGB {
	return xyzToLinearSRGBColor(rec2020LinearToXYZ(rec2020RGBToLinear(in)))
}

func outOfGamut(linear RGB) bool {
	const epsilon = 1e-7
	return linear.R < -epsilon || linear.R > 1+epsilon ||
		linear.G < -epsilon || linear.G > 1+epsilon ||
		linear.B < -epsilon || linear.B > 1+epsilon
}

func clip01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clipLinear(c RGB) RGB {
	return RGB{clip01(c.R), clip01(c.G), clip01(c.B)}
}

func rec2020RGBToLinear(in RGB) RGB {
	return RGB{
		R: rec2020EOTF(in.R),
		G: rec2020EOTF(in.G),
		B: rec2020EOTF(in.B),
	}
}

func rec2020EOTF(v float64) float64 {
	v = clip01(v)
	const (
		alpha = 1.09929682680944
		beta  = 0.018053968510807
	)
	if v < 4.5*beta {
		return v / 4.5
	}
	return math.Pow((v+(alpha-1))/alpha, 1/0.45)
}

func linearSRGBToEncoded(in RGB) RGB {
	return RGB{
		R: srgbOETF(in.R),
		G: srgbOETF(in.G),
		B: srgbOETF(in.B),
	}
}

func rec2020LinearToEncoded(in RGB) RGB {
	return RGB{
		R: rec2020OETF(in.R),
		G: rec2020OETF(in.G),
		B: rec2020OETF(in.B),
	}
}

func srgbEOTF(v float64) float64 {
	v = clip01(v)
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func rec2020OETF(v float64) float64 {
	const (
		alpha = 1.09929682680944
		beta  = 0.018053968510807
	)
	v = clip01(v)
	if v < beta {
		return 4.5 * v
	}
	return alpha*math.Pow(v, 0.45) - (alpha - 1)
}

func srgbOETF(v float64) float64 {
	v = clip01(v)
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}
