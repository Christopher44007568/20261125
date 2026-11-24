package main

import (
	"fmt"
	"image"
	"math"
)

var worldUp = Vec3{Y: 1}

func renderReference(scene *Scene, eye EyeState) (*image.RGBA, error) {
	if err := validateScene(scene); err != nil {
		return nil, err
	}
	if err := eye.Validate(); err != nil {
		return nil, err
	}
	img := newRGBA(scene.Layout.Width, scene.Layout.Height)
	renderBackground(img, eye.Camera, eye.Constants, eye.Jitter)
	for _, panel := range scene.Layout.Panels {
		material, ok := findMaterial(eye, panel.Instance)
		if !ok {
			return nil, fmt.Errorf("%s: missing material instance %q", eye.Eye, panel.Instance)
		}
		prepared, err := prepareDraw(scene, eye, material)
		if err != nil {
			return nil, err
		}
		if err := renderPanel(img, scene, panel, prepared); err != nil {
			return nil, err
		}
	}
	return img, nil
}

func findMaterial(eye EyeState, instance string) (MaterialState, bool) {
	for _, material := range eye.Materials {
		if material.Instance == instance {
			return material, true
		}
	}
	return MaterialState{}, false
}

func cameraBasis(camera Camera) (Vec3, Vec3, Vec3) {
	cosPitch := math.Cos(camera.PitchRadians)
	forward := Vec3{
		X: math.Sin(camera.YawRadians) * cosPitch,
		Y: math.Sin(camera.PitchRadians),
		Z: -math.Cos(camera.YawRadians) * cosPitch,
	}.Normalize()
	right := forward.Cross(worldUp).Normalize()
	up := right.Cross(forward).Normalize()
	return forward, right, up
}

func primaryRay(camera Camera, jitter Vec3, x, y, width, height int) (Vec3, Vec3) {
	forward, right, up := cameraBasis(camera)
	aspect := float64(width) / float64(height)
	halfHeight := math.Tan(camera.VerticalFOVDegree * math.Pi / 360)
	halfWidth := halfHeight * aspect
	ndcX := ((float64(x)+0.5)/float64(width))*2 - 1
	ndcY := 1 - ((float64(y)+0.5)/float64(height))*2
	ndcX += jitter.X / float64(width)
	ndcY += jitter.Y / float64(height)
	direction := forward.
		Add(right.Scale(ndcX * halfWidth)).
		Add(up.Scale(ndcY * halfHeight)).
		Normalize()
	return camera.Position, direction
}

func renderBackground(img *image.RGBA, camera Camera, constants map[string]float64, jitter Vec3) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	gain := constants["background_gain"]
	horizon := constants["horizon"]
	vignetteAmount := constants["vignette"]
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ndcX := ((float64(x)+0.5)/float64(width))*2 - 1
			ndcY := 1 - ((float64(y)+0.5)/float64(height))*2
			vertical := 0.5 + 0.5*ndcY + camera.PitchRadians*0.5
			horizontal := ndcX + camera.YawRadians*0.8 + jitter.X*0.004
			vignette := 1 - vignetteAmount*(ndcX*ndcX+ndcY*ndcY)
			exposure := camera.Exposure * gain
			r := (0.025 + 0.20*vertical + horizon*0.18 + 0.025*horizontal) * exposure * vignette
			g := (0.035 + 0.16*vertical + horizon*0.12 - 0.018*horizontal) * exposure * vignette
			b := (0.070 + 0.24*vertical + horizon*0.08) * exposure * vignette
			setColor(img, x, y, r, g, b, 1)
		}
	}
}

func renderPanel(img *image.RGBA, scene *Scene, panel Panel, prepared *PreparedUniforms) error {
	values, err := decodeUniforms(scene, prepared)
	if err != nil {
		return err
	}
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	light := Vec3{X: -0.55, Y: 0.72, Z: 0.60}.Normalize()
	fogAmount := values.EyeConstants["fog"]
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			origin, direction := primaryRay(values.Camera, values.Jitter, x, y, width, height)
			if math.Abs(direction.Z) < 1e-9 {
				continue
			}
			t := (scene.Layout.PlaneZ - origin.Z) / direction.Z
			if t <= values.Camera.NearPlane {
				continue
			}
			hit := origin.Add(direction.Scale(t))
			dx := hit.X - panel.Center.X
			dy := hit.Y - panel.Center.Y
			distance := math.Hypot(dx, dy)
			if distance > panel.Radius {
				continue
			}

			normal := Vec3{Z: 1}
			toView := values.Camera.Position.Sub(hit).Normalize()
			halfVector := light.Add(toView).Normalize()
			diffuse := math.Max(normal.Dot(light), 0)
			shininess := 2 + math.Pow(1-values.Roughness, 2)*120
			specular := math.Pow(math.Max(normal.Dot(halfVector), 0), shininess) * values.SpecGain
			rim := math.Exp(-math.Pow((distance-(panel.Radius-0.026))/0.020, 2))
			pattern := 0.94 + 0.06*math.Sin((hit.X-hit.Y)*23+values.Jitter.Z*2)
			lighting := (0.20 + 0.80*diffuse) * values.Gain * pattern
			r := values.Tint.X*lighting + specular*1.6 + rim*values.Ring*1.1
			g := values.Tint.Y*lighting + specular*1.6 + rim*values.Ring*0.92
			b := values.Tint.Z*lighting + specular*1.6 + rim*values.Ring*1.08
			fog := math.Max(0, math.Min(1, fogAmount*(t-2.5)))
			r = (r*(1-fog) + 0.06*fog) * values.Camera.Exposure
			g = (g*(1-fog) + 0.07*fog) * values.Camera.Exposure
			b = (b*(1-fog) + 0.09*fog) * values.Camera.Exposure
			blendPixel(img, x, y, r, g, b, values.Alpha)
		}
	}
	return nil
}

func blendPixel(img *image.RGBA, x, y int, r, g, b, alpha float64) {
	r = clamp01(r)
	g = clamp01(g)
	b = clamp01(b)
	alpha = clamp01(alpha)
	offset := img.PixOffset(x, y)
	dstR := float64(img.Pix[offset+0]) / 255
	dstG := float64(img.Pix[offset+1]) / 255
	dstB := float64(img.Pix[offset+2]) / 255
	img.Pix[offset+0] = toByte(r*alpha + dstR*(1-alpha))
	img.Pix[offset+1] = toByte(g*alpha + dstG*(1-alpha))
	img.Pix[offset+2] = toByte(b*alpha + dstB*(1-alpha))
	img.Pix[offset+3] = 255
}

func setColor(img *image.RGBA, x, y int, r, g, b, alpha float64) {
	offset := img.PixOffset(x, y)
	img.Pix[offset+0] = toByte(clamp01(r))
	img.Pix[offset+1] = toByte(clamp01(g))
	img.Pix[offset+2] = toByte(clamp01(b))
	img.Pix[offset+3] = toByte(clamp01(alpha))
}

func toByte(value float64) uint8 {
	return uint8(math.Round(clamp01(value) * 255))
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	if math.IsNaN(value) {
		return 0
	}
	return value
}
