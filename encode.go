package main

import (
	"image"
	"image/png"
	"os"
)

func WriteBeautyPNG(path string, pixels []Vec3, width, height int) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i, pixel := range pixels {
		exposed := pixel.Mul(1.12)
		mapped := acesFilm(exposed)
		srgb := linearToSRGB(mapped)
		img.Pix[4*i+0] = uint8(clamp01(srgb.X)*255 + 0.5)
		img.Pix[4*i+1] = uint8(clamp01(srgb.Y)*255 + 0.5)
		img.Pix[4*i+2] = uint8(clamp01(srgb.Z)*255 + 0.5)
		img.Pix[4*i+3] = 255
	}
	return writePNG(path, img)
}

func WriteLinearVecPNG(path string, pixels []Vec3, width, height int, exposure float64) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i, pixel := range pixels {
		value := linearToSRGB(pixel.Mul(exposure))
		img.Pix[4*i+0] = uint8(clamp01(value.X)*255 + 0.5)
		img.Pix[4*i+1] = uint8(clamp01(value.Y)*255 + 0.5)
		img.Pix[4*i+2] = uint8(clamp01(value.Z)*255 + 0.5)
		img.Pix[4*i+3] = 255
	}
	return writePNG(path, img)
}

func WriteDeepTransmissionPNG(path string, dm *DeepOpacityMap) error {
	size := dm.Size
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	backlight := srgbToLinear(Vec(1.0, 0.84, 0.60))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			transmission := dm.TotalTransmittance(x, y).MulVec(backlight)
			value := linearToSRGB(transmission.Mul(1.15))
			i := y*size + x
			img.Pix[4*i+0] = uint8(clamp01(value.X)*255 + 0.5)
			img.Pix[4*i+1] = uint8(clamp01(value.Y)*255 + 0.5)
			img.Pix[4*i+2] = uint8(clamp01(value.Z)*255 + 0.5)
			img.Pix[4*i+3] = 255
		}
	}
	return writePNG(path, img)
}

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := encoder.Encode(file, img); err != nil {
		return err
	}
	return nil
}

func acesFilm(x Vec3) Vec3 {
	ace := func(v float64) float64 {
		v = maxFloat(v, 0)
		return clamp01((v * (2.51*v + 0.03)) / (v*(2.43*v+0.59) + 0.14))
	}
	return Vec(ace(x.X), ace(x.Y), ace(x.Z))
}
