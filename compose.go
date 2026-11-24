package main

import (
	"image"
	"math"
)

type VisibilityStats struct {
	FootprintSamples    int
	VisibleSamples      int
	OccludedSamples     int
	InvalidDepthSamples int
	FootprintPixels     int
	VisiblePixels       int
	OccludedPixels      int
	InvalidDepthPixels  int
}

func renderVirtualLayer(bundle *CaptureBundle, calibration ColorCalibration) (*VirtualLayer, VisibilityStats) {
	width, height := bundle.Intrinsics.pixelDimensions()
	layer := newVirtualLayer(width, height)
	samples := [4][2]float64{{0.25, 0.25}, {0.75, 0.25}, {0.25, 0.75}, {0.75, 0.75}}
	rowStats := make([]VisibilityStats, height)

	parallelRows(height, func(y int) {
		stats := &rowStats[y]
		for x := 0; x < width; x++ {
			p := y*width + x
			colorSum := RGB{}
			visible := 0
			hitCount := 0
			occluded := 0
			invalid := 0
			for _, s := range samples {
				ray := bundle.Intrinsics.rayForSample(x, y, s[0], s[1])
				z, normal, ok := intersectSphere(ray, bundle.Sphere)
				if !ok {
					continue
				}
				hitCount++
				if !bundle.DepthValid[p] {
					stats.InvalidDepthSamples++
					invalid++
					continue
				}
				if z >= bundle.DepthM[p]-depthOcclusionBiasM {
					stats.OccludedSamples++
					occluded++
					continue
				}
				visible++
				colorSum = colorSum.Add(shadeSphere(ray, normal, bundle.Sphere))
			}
			if hitCount == 0 {
				continue
			}
			stats.FootprintSamples += hitCount
			stats.FootprintPixels++
			layer.Footprint[p] = true
			if visible == 0 {
				if occluded > 0 {
					stats.OccludedPixels++
				} else if invalid > 0 {
					stats.InvalidDepthPixels++
				}
				continue
			}
			stats.VisibleSamples += visible
			stats.VisiblePixels++
			layer.Alpha[p] = float64(visible) / float64(len(samples))
			c := colorSum.Scale(1 / float64(visible)).Mul(calibration.Gains)
			layer.Pix[p] = c
		}
	})
	stats := VisibilityStats{}
	for _, row := range rowStats {
		stats.FootprintSamples += row.FootprintSamples
		stats.VisibleSamples += row.VisibleSamples
		stats.OccludedSamples += row.OccludedSamples
		stats.InvalidDepthSamples += row.InvalidDepthSamples
		stats.FootprintPixels += row.FootprintPixels
		stats.VisiblePixels += row.VisiblePixels
		stats.OccludedPixels += row.OccludedPixels
		stats.InvalidDepthPixels += row.InvalidDepthPixels
	}
	return layer, stats
}

func intersectSphere(ray Vec3, sphere Sphere) (float64, Vec3, bool) {
	a := ray.Dot(ray)
	halfB := ray.Dot(sphere.Center)
	c := sphere.Center.Dot(sphere.Center) - sphere.Radius*sphere.Radius
	disc := halfB*halfB - a*c
	if disc < 0 {
		return 0, Vec3{}, false
	}
	root := math.Sqrt(disc)
	t := (halfB - root) / a
	if t <= 0 {
		t = (halfB + root) / a
	}
	if t <= 0 {
		return 0, Vec3{}, false
	}
	p := ray.Mul(t)
	return p.Z, p.Sub(sphere.Center).Normalize(), true
}

func shadeSphere(ray Vec3, normal Vec3, sphere Sphere) RGB {
	view := ray.Mul(-1).Normalize()
	diffuse := math.Max(0, normal.Dot(sceneLight))
	half := sceneLight.Add(view).Normalize()
	specular := powInt(math.Max(0, normal.Dot(half)), 72) * 0.62
	fresnel := math.Pow(1-math.Max(0, normal.Dot(view)), 3) * 0.10

	u := math.Atan2(normal.X, normal.Z)/(2*math.Pi) + 0.5
	v := math.Acos(math.Max(-1, math.Min(1, normal.Y))) / math.Pi
	pattern := math.Sin((u*7.0 + v*5.0) * 2 * math.Pi)
	blend := smoothstep(-0.30, 0.30, pattern)
	albedo := RGB{
		R: lerp(sphere.AlbedoA.R, sphere.AlbedoB.R, blend),
		G: lerp(sphere.AlbedoA.G, sphere.AlbedoB.G, blend),
		B: lerp(sphere.AlbedoA.B, sphere.AlbedoB.B, blend),
	}
	c := albedo.Scale(0.20 + 1.05*diffuse)
	c = c.Add(RGB{R: specular + fresnel, G: specular*0.97 + fresnel, B: specular*0.92 + fresnel})
	return c
}

func imageFromLinear(camera *LinearImage) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, camera.Width, camera.Height))
	parallelRows(camera.Height, func(y int) {
		for x := 0; x < camera.Width; x++ {
			p := y*camera.Width + x
			r, g, b := toneMap8(camera.Pix[p])
			offset := p * 4
			out.Pix[offset+0] = r
			out.Pix[offset+1] = g
			out.Pix[offset+2] = b
			out.Pix[offset+3] = 255
		}
	})
	return out
}

func imageFromVirtual(virtual *VirtualLayer) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, virtual.Width, virtual.Height))
	parallelRows(virtual.Height, func(y int) {
		for x := 0; x < virtual.Width; x++ {
			p := y*virtual.Width + x
			a := virtual.Alpha[p]
			if a <= 0 {
				continue
			}
			r, g, b := toneMap8(virtual.Pix[p])
			offset := p * 4
			out.Pix[offset+0] = r
			out.Pix[offset+1] = g
			out.Pix[offset+2] = b
			out.Pix[offset+3] = uint8(math.Round(clamp01(a) * 255))
		}
	})
	return out
}

func composite(camera *LinearImage, passthrough *image.NRGBA, virtual *VirtualLayer, system *image.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, camera.Width, camera.Height))
	copy(out.Pix, passthrough.Pix)
	parallelRows(camera.Height, func(y int) {
		for x := 0; x < camera.Width; x++ {
			p := y*camera.Width + x
			a := virtual.Alpha[p]
			if a <= 0 {
				continue
			}
			blended := camera.Pix[p].Scale(1 - a).Add(virtual.Pix[p].Scale(a))
			r, g, b := toneMap8(blended)
			offset := p * 4
			out.Pix[offset+0] = r
			out.Pix[offset+1] = g
			out.Pix[offset+2] = b
		}
	})
	if system == nil {
		return out
	}
	parallelRows(camera.Height, func(y int) {
		for x := 0; x < camera.Width; x++ {
			overlay := system.NRGBAAt(x, y)
			if overlay.A == 0 {
				continue
			}
			a := float64(overlay.A) / 255
			offset := (y*camera.Width + x) * 4
			out.Pix[offset+0] = uint8(math.Round((1-a)*float64(out.Pix[offset+0]) + a*float64(overlay.R)))
			out.Pix[offset+1] = uint8(math.Round((1-a)*float64(out.Pix[offset+1]) + a*float64(overlay.G)))
			out.Pix[offset+2] = uint8(math.Round((1-a)*float64(out.Pix[offset+2]) + a*float64(overlay.B)))
		}
	})
	return out
}

func buildConfidence(bundle *CaptureBundle, fused FusionResult) []float64 {
	width, height := bundle.Intrinsics.pixelDimensions()
	conf := make([]float64, width*height)
	parallelRows(height, func(y int) {
		for x := 0; x < width; x++ {
			p := y*width + x
			if !bundle.DepthValid[p] {
				continue
			}
			maxJump := 0.0
			depth := bundle.DepthM[p]
			for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= width || ny >= height {
					continue
				}
				np := ny*width + nx
				if !bundle.DepthValid[np] {
					maxJump = math.Max(maxJump, 0.25)
					continue
				}
				maxJump = math.Max(maxJump, math.Abs(depth-bundle.DepthM[np]))
			}
			edgeConfidence := 1 - 0.48*smoothstep(0.035, 0.24, maxJump)
			conf[p] = math.Sqrt(edgeConfidence * fused.Confidence[p])
		}
	})
	return conf
}

func confidenceImage(conf []float64, width, height int) *image.Gray {
	out := image.NewGray(image.Rect(0, 0, width, height))
	for p, c := range conf {
		out.Pix[p] = uint8(math.Round(clamp01(c) * 255))
	}
	return out
}
