package gamut

import (
	"math"
	"testing"
)

func TestNeutralAxisIsNeverSaturated(t *testing.T) {
	grays := []float64{0, 0.001, 0.01, 0.1, 0.25, 0.5, 0.75, 0.99, 1}
	for _, v := range grays {
		in := RGB{R: v, G: v, B: v}
		clip := MapRec2020ToSRGB(in, Clip)
		perceptual := MapRec2020ToSRGB(in, Perceptual)

		for _, result := range []Result{clip, perceptual} {
			if result.OutOfGamut {
				t.Fatalf("neutral %v reported out of gamut", v)
			}
			if spread(result.SRGB) > 1e-12 {
				t.Fatalf("neutral %v became chromatic: %+v", v, result.SRGB)
			}
		}
		if rgbDistance(clip.SRGB, perceptual.SRGB) > 1e-12 {
			t.Fatalf("mapping methods disagree on neutral %v: %+v vs %+v",
				v, clip.SRGB, perceptual.SRGB)
		}
	}
}

func TestRec2020RedIsOutOfSRGBGamut(t *testing.T) {
	in := RGB{R: 1, G: 0, B: 0}
	if InSRGBGamut(in) {
		t.Fatal("Rec.2020 red must be outside sRGB")
	}

	clip := MapRec2020ToSRGB(in, Clip)
	if !clip.OutOfGamut {
		t.Fatal("clip result did not retain the source gamut mask")
	}
	if rgbDistance(clip.SRGB, RGB{R: 1, G: 0, B: 0}) > 1e-6 {
		t.Fatalf("clip should collapse Rec.2020 red to sRGB red: %+v", clip.SRGB)
	}

	perceptual := MapRec2020ToSRGB(in, Perceptual)
	if perceptual.OutOfGamut != clip.OutOfGamut {
		t.Fatal("methods must use the same source-gamut mask")
	}
	if !isUnitRGB(perceptual.SRGB) {
		t.Fatalf("perceptual output is not displayable: %+v", perceptual.SRGB)
	}

	sourceLab := xyzToOklab(linearSRGBToXYZ(rec2020ToLinearSRGB(in)))
	mappedLab := xyzToOklab(linearSRGBToXYZ(RGB{
		R: srgbEOTF(perceptual.SRGB.R),
		G: srgbEOTF(perceptual.SRGB.G),
		B: srgbEOTF(perceptual.SRGB.B),
	}))
	sourceL, sourceC, sourceH := oklabToOklch(sourceLab)
	mappedL, mappedC, mappedH := oklabToOklch(mappedLab)
	if math.Abs(sourceL-mappedL) > 2e-4 {
		t.Fatalf("lightness changed too much: %.8f -> %.8f", sourceL, mappedL)
	}
	if math.Abs(sourceH-mappedH) > 2e-4 {
		t.Fatalf("hue changed too much: %.8f -> %.8f", sourceH, mappedH)
	}
	if mappedC >= sourceC {
		t.Fatalf("chroma was not reduced: %.8f -> %.8f", sourceC, mappedC)
	}
}

func TestMaskAgreesAcrossMethodsAndLattice(t *testing.T) {
	for r := 0; r <= 8; r++ {
		for g := 0; g <= 8; g++ {
			for b := 0; b <= 8; b++ {
				in := RGB{R: float64(r) / 8, G: float64(g) / 8, B: float64(b) / 8}
				clip := MapRec2020ToSRGB(in, Clip)
				perceptual := MapRec2020ToSRGB(in, Perceptual)
				if clip.OutOfGamut != perceptual.OutOfGamut {
					t.Fatalf("mask mismatch for %+v", in)
				}
				if clip.OutOfGamut != !InSRGBGamut(in) {
					t.Fatalf("InSRGBGamut mismatch for %+v", in)
				}
				if !isUnitRGB(clip.SRGB) || !isUnitRGB(perceptual.SRGB) {
					t.Fatalf("non-displayable output for %+v: %+v / %+v",
						in, clip.SRGB, perceptual.SRGB)
				}
			}
		}
	}
}

func TestInGamutColorsPassThroughUnchanged(t *testing.T) {
	colors := []RGB{
		{R: 0.2, G: 0.4, B: 0.7},
		{R: 0.9, G: 0.7, B: 0.2},
		{R: 0.1, G: 0.8, B: 0.3},
	}
	for _, in := range colors {
		rec2020 := SRGBToRec2020(in)
		clip := MapRec2020ToSRGB(rec2020, Clip)
		perceptual := MapRec2020ToSRGB(rec2020, Perceptual)
		if clip.OutOfGamut || perceptual.OutOfGamut {
			t.Fatalf("sRGB source %+v unexpectedly exported out of gamut", in)
		}
		if rgbDistance(clip.SRGB, in) > 2e-6 || rgbDistance(perceptual.SRGB, in) > 2e-6 {
			t.Fatalf("round trip failed for %+v: clip=%+v perceptual=%+v",
				in, clip.SRGB, perceptual.SRGB)
		}
	}
}

func TestOklabRoundTrip(t *testing.T) {
	for _, c := range []RGB{{0, 0, 0}, {1, 1, 1}, {1, 0, 0}, {0.2, 0.4, 0.7}} {
		in := xyzToOklab(linearSRGBToXYZ(c))
		out := oklabToLinear(in)
		if rgbDistance(out, c) > 1e-6 {
			t.Fatalf("Oklab round trip failed: %+v -> %+v", c, out)
		}
	}
}

func spread(c RGB) float64 {
	mx := math.Max(c.R, math.Max(c.G, c.B))
	mn := math.Min(c.R, math.Min(c.G, c.B))
	return mx - mn
}

func rgbDistance(a, b RGB) float64 {
	return math.Abs(a.R-b.R) + math.Abs(a.G-b.G) + math.Abs(a.B-b.B)
}

func isUnitRGB(c RGB) bool {
	const e = 1e-9
	return c.R >= -e && c.R <= 1+e &&
		c.G >= -e && c.G <= 1+e &&
		c.B >= -e && c.B <= 1+e
}
