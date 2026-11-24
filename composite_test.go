package main

import (
	"math"
	"testing"
)

func approxEq(t *testing.T, got, want, tolerance float64, name string) {
	t.Helper()
	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %.9f, want %.9f", name, got, want)
	}
}

func TestSRGBRoundTrip(t *testing.T) {
	for _, value := range []float64{0, 0.04045, 0.25, 0.5, 1} {
		approxEq(t, linearTosRGB(sRGBToLinear(value)), value, 1e-7, "sRGB round trip")
	}
}

func TestDecodePremultiplied(t *testing.T) {
	straight := decodePremultiplied(Sample{
		Space: SRGB,
		R:     0.8,
		G:     0.2,
		B:     0.5,
		A:     0.5,
	})
	approxEq(t, straight.R, sRGBToLinear(0.8)*0.5, 1e-15, "straight red")
	approxEq(t, straight.G, sRGBToLinear(0.2)*0.5, 1e-15, "straight green")
	approxEq(t, straight.B, sRGBToLinear(0.5)*0.5, 1e-15, "straight blue")

	premultiplied := decodePremultiplied(Sample{
		Space:  SRGB,
		Premul: true,
		R:      0.4,
		G:      0.1,
		B:      0.25,
		A:      0.5,
	})
	approxEq(t, premultiplied.R, sRGBToLinear(0.8)*0.5, 1e-15, "premultiplied red")
	approxEq(t, premultiplied.G, sRGBToLinear(0.2)*0.5, 1e-15, "premultiplied green")
	approxEq(t, premultiplied.B, sRGBToLinear(0.5)*0.5, 1e-15, "premultiplied blue")
}

func TestLinearPremultipliedHDRPassthrough(t *testing.T) {
	got := decodePremultiplied(Sample{
		Space:  Linear,
		Premul: true,
		R:      1.8,
		G:      0.7,
		B:      2.4,
		A:      0.5,
	})

	approxEq(t, got.R, 1.8, 1e-15, "HDR red")
	approxEq(t, got.G, 0.7, 1e-15, "HDR green")
	approxEq(t, got.B, 2.4, 1e-15, "HDR blue")
	approxEq(t, got.A, 0.5, 1e-15, "HDR alpha")
}

func TestLinearPremultipliedOver(t *testing.T) {
	dst := PremulLinear{R: 0.2, G: 0.4, B: 0.1, A: 1}
	src := PremulLinear{R: 0.1, G: 0.2, B: 0.3, A: 0.5}
	got := over(dst, src)

	approxEq(t, got.R, 0.2, 1e-15, "red")
	approxEq(t, got.G, 0.4, 1e-15, "green")
	approxEq(t, got.B, 0.35, 1e-15, "blue")
	approxEq(t, got.A, 1, 1e-15, "alpha")
}

func TestHDRBackgroundSurvivesBeforeTonemap(t *testing.T) {
	scene := buildScene()
	_, final := scene.CompositeEye(0)
	backgroundPixel := final[sceneHeight-1][sceneWidth-1]
	if backgroundPixel.R <= 1 || backgroundPixel.G <= 1 {
		t.Fatalf("HDR linear values were clipped before tonemap: %+v", backgroundPixel)
	}

	encoded := correctEncoded(final)
	if encoded[0][0][0] < 0 || encoded[0][0][0] > 1 {
		t.Fatalf("display red is outside [0,1]: %f", encoded[0][0][0])
	}
}

func TestPerEyeAlpha(t *testing.T) {
	scene := buildScene()
	records, _ := scene.Composite()

	left := records[0][2].Decoded[10][10]
	right := records[1][2].Decoded[10][10]
	if left.A == 0 || right.A == 0 || left.A == right.A {
		t.Fatalf("expected distinct nonzero per-eye text alpha, got %.6f and %.6f", left.A, right.A)
	}

	if left.A != eyeFactor("text", 0) || right.A != eyeFactor("text", 1) {
		t.Fatalf("unexpected per-eye alpha: left %.6f right %.6f", left.A, right.A)
	}
}

func TestGammaErrorsDetectEncodedDomainCompositing(t *testing.T) {
	scene := buildScene()
	_, correct := scene.CompositeEye(1)
	wrong := scene.CompositeWrongEncodedEye(1)
	errors := gammaErrors(correct, wrong, SRGB)

	var validCount int
	var maxRMSE float64
	for _, row := range errors {
		for _, metric := range row {
			if metric.Valid {
				validCount++
			}
			maxRMSE = math.Max(maxRMSE, metric.LinearRMSE)
		}
	}

	if validCount == 0 {
		t.Fatal("expected valid gamma delta pixels")
	}
	if maxRMSE < 0.01 {
		t.Fatalf("expected measurable encoded-domain error, got max RMSE %.6f", maxRMSE)
	}
}

func TestDirectAdditionIsNotAlphaCompositing(t *testing.T) {
	scene := buildScene()
	_, correct := scene.CompositeEye(0)
	addition := scene.CompositeDirectAdditionEye(0)
	errors := gammaErrors(correct, addition, Linear)

	var maxRMSE float64
	for _, row := range errors {
		for _, metric := range row {
			maxRMSE = math.Max(maxRMSE, metric.LinearRMSE)
		}
	}

	if maxRMSE < 0.05 {
		t.Fatalf("direct addition should produce substantial error, got max RMSE %.6f", maxRMSE)
	}
}
