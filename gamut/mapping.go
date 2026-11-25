package gamut

import "math"

const (
	neutralEpsilon  = 1e-8
	gamutEpsilon    = 1e-7
	bisectionPasses = 28
)

// perceptualMap maps an unclipped linear sRGB color to the sRGB gamut.
// Oklab lightness and hue are held constant while chroma is reduced to the
// gamut boundary. In-gamut colors and neutral colors are returned untouched.
func perceptualMap(linear RGB) RGB {
	if !outOfGamut(linear) {
		return linear
	}

	// Neutral colors have no hue/chroma to compress. Returning their clipped
	// values explicitly guarantees that mapping never introduces saturation.
	if isNeutralLinear(linear) {
		return clipLinear(linear)
	}

	lab := xyzToOklab(linearSRGBToXYZ(linear))
	_, chroma, hue := oklabToOklch(lab)
	if chroma <= neutralEpsilon {
		return clipLinear(linear)
	}

	// C=0 is the neutral axis and is always inside sRGB for a display-range
	// lightness. The source chroma is outside the gamut by construction.
	low, high := 0.0, chroma
	for range bisectionPasses {
		mid := (low + high) * 0.5
		candidate := oklabToLinear(oklchToOklab(lab.l, mid, hue))
		if colorInGamut(candidate) {
			low = mid
		} else {
			high = mid
		}
	}

	mapped := oklabToLinear(oklchToOklab(lab.l, low, hue))
	// Rounding at the boundary can leave a few negative ULP-sized components.
	// Clamping those components does not introduce chroma.
	return clipLinear(mapped)
}

func isNeutralLinear(c RGB) bool {
	return math.Abs(c.R-c.G) <= neutralEpsilon &&
		math.Abs(c.G-c.B) <= neutralEpsilon
}

func colorInGamut(c RGB) bool {
	return c.R >= -gamutEpsilon && c.R <= 1+gamutEpsilon &&
		c.G >= -gamutEpsilon && c.G <= 1+gamutEpsilon &&
		c.B >= -gamutEpsilon && c.B <= 1+gamutEpsilon
}

func oklabToLinear(in lab) RGB {
	return xyzToLinearSRGBColor(oklabToXYZ(in))
}
