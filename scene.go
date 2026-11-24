package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
)

type sceneSamples struct {
	Radiance   *LinearImage
	DepthM     []float64
	DepthValid []bool
}

type surfaceHit struct {
	T      float64
	Normal Vec3
	Albedo RGB
	Kind   int
}

const (
	surfaceWall = iota
	surfaceFloor
	surfacePost
	surfaceGrayCard
)

var sceneLight = Vec3{X: -0.38, Y: 0.72, Z: -0.58}.Normalize()

func defaultSphere() Sphere {
	return Sphere{
		Center:  Vec3{X: 0.08, Y: 0.08, Z: 1.92},
		Radius:  0.47,
		AlbedoA: RGB{R: 0.035, G: 0.22, B: 0.42},
		AlbedoB: RGB{R: 0.025, G: 0.055, B: 0.075},
	}
}

func renderScene(width, height int, k Intrinsics, rng *rand.Rand) sceneSamples {
	out := sceneSamples{
		Radiance:   newLinearImage(width, height),
		DepthM:     make([]float64, width*height),
		DepthValid: make([]bool, width*height),
	}
	depthNoise := make([]float64, width*height)
	for i := range depthNoise {
		depthNoise[i] = (rng.Float64() - 0.5) * 0.006
	}
	parallelRows(height, func(y int) {
		for x := 0; x < width; x++ {
			colorSum := RGB{}
			minDepth := math.Inf(1)
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					ray := k.rayForSample(x, y, 0.25+0.5*float64(sx), 0.25+0.5*float64(sy))
					hit := traceBackground(ray)
					colorSum = colorSum.Add(shadeBackground(hit, ray))
					minDepth = math.Min(minDepth, hit.T)
				}
			}
			i := y*width + x
			out.Radiance.Pix[i] = colorSum.Scale(0.25)

			u := (float64(x) + 0.5) / float64(width)
			v := (float64(y) + 0.5) / float64(height)
			valid := !invalidDepthRegion(u, v)
			if valid {
				minDepth += depthNoise[i]
			} else {
				minDepth = 0
			}
			out.DepthM[i] = minDepth
			out.DepthValid[i] = valid
		}
	})
	return out
}

func (k Intrinsics) rayForSample(x, y int, sx, sy float64) Vec3 {
	return Vec3{
		X: (float64(x) + sx - k.CX) / k.FX,
		Y: -(float64(y) + sy - k.CY) / k.FY,
		Z: 1,
	}
}

func traceBackground(ray Vec3) surfaceHit {
	best := surfaceHit{T: math.Inf(1)}
	if ray.Z > 0 {
		t := 3.35 / ray.Z
		best = surfaceHit{T: t, Normal: Vec3{Z: -1}, Albedo: wallAlbedo(ray.Mul(t)), Kind: surfaceWall}
		if p := ray.Mul(t); isGrayCard(p) {
			best.Kind = surfaceGrayCard
			best.Albedo = RGB{R: 0.50, G: 0.50, B: 0.50}
		}
	}
	if ray.Y < -0.02 {
		t := -0.88 / ray.Y
		if t > 0 && t < best.T {
			best = surfaceHit{T: t, Normal: Vec3{Y: 1}, Albedo: floorAlbedo(ray.Mul(t)), Kind: surfaceFloor}
		}
	}
	if t, n, ok := intersectPost(ray); ok && t < best.T {
		best = surfaceHit{T: t, Normal: n, Albedo: postAlbedo(ray.Mul(t)), Kind: surfacePost}
	}
	return best
}

func intersectPost(ray Vec3) (float64, Vec3, bool) {
	const (
		postX = -0.31
		postZ = 1.22
		postR = 0.245
	)
	a := ray.X*ray.X + ray.Z*ray.Z
	b := -2 * (ray.X*postX + ray.Z*postZ)
	c := postX*postX + postZ*postZ - postR*postR
	disc := b*b - 4*a*c
	if disc < 0 || math.Abs(a) < 1e-12 {
		return 0, Vec3{}, false
	}
	t := (-b - math.Sqrt(disc)) / (2 * a)
	if t <= 0 {
		t = (-b + math.Sqrt(disc)) / (2 * a)
	}
	if t <= 0 {
		return 0, Vec3{}, false
	}
	p := ray.Mul(t)
	if p.Y < -0.88 || p.Y > 0.92 {
		return 0, Vec3{}, false
	}
	n := Vec3{X: p.X - postX, Z: p.Z - postZ}.Normalize()
	return t, n, true
}

func wallAlbedo(p Vec3) RGB {
	base := RGB{R: 0.23, G: 0.31, B: 0.30}
	checker := (int(math.Floor(p.X*1.8)) + int(math.Floor(p.Y*1.8))) & 1
	if checker == 0 {
		base = base.Scale(0.91)
	} else {
		base = base.Scale(1.05)
	}
	gx := math.Abs(p.X*2.7 - math.Round(p.X*2.7))
	gy := math.Abs(p.Y*2.7 - math.Round(p.Y*2.7))
	if gx < 0.018 || gy < 0.018 {
		base = base.Scale(0.72)
	}
	return base
}

func floorAlbedo(p Vec3) RGB {
	cx := int(math.Floor(p.X * 3.2))
	cz := int(math.Floor(p.Z * 3.2))
	if (cx+cz)&1 == 0 {
		return RGB{R: 0.36, G: 0.33, B: 0.27}
	}
	return RGB{R: 0.23, G: 0.22, B: 0.20}
}

func postAlbedo(p Vec3) RGB {
	f := p.X*10.5 + p.Z*2.2
	f = math.Abs(f-math.Floor(f)-0.5) * 2
	stripe := smoothstep(0.18, 0.42, f)
	return RGB{
		R: 0.48 + 0.14*stripe,
		G: 0.13 + 0.05*stripe,
		B: 0.075 + 0.025*stripe,
	}
}

func isGrayCard(p Vec3) bool {
	return p.X > 0.82 && p.X < 1.34 && p.Y > -0.32 && p.Y < 0.26
}

func shadeBackground(hit surfaceHit, ray Vec3) RGB {
	if !isFinite(hit.T) {
		return RGB{}
	}
	n := hit.Normal
	view := ray.Mul(-1).Normalize()
	diffuse := math.Max(0, n.Dot(sceneLight))
	half := sceneLight.Add(view).Normalize()
	specular := powInt(math.Max(0, n.Dot(half)), 28) * 0.10
	ambient := 0.24
	c := hit.Albedo.Scale(ambient + 0.92*diffuse)
	c = c.Add(RGB{R: specular, G: specular * 0.97, B: specular * 0.90})
	if hit.Kind == surfacePost {
		c = c.Scale(0.92)
	}
	return c
}

func powInt(x float64, n int) float64 {
	result := 1.0
	for n > 0 {
		if n&1 == 1 {
			result *= x
		}
		x *= x
		n >>= 1
	}
	return result
}

func invalidDepthRegion(u, v float64) bool {
	ellipse := math.Pow((u-0.655)/0.052, 2) + math.Pow((v-0.565)/0.145, 2)
	if ellipse < 1 {
		return true
	}
	leftGloss := math.Pow((u-0.105)/0.040, 2) + math.Pow((v-0.285)/0.100, 2)
	if leftGloss < 1 {
		return true
	}
	if u > 0.875 && u < 0.925 && v > 0.08 && v < 0.34 {
		return true
	}
	if v > 0.835 && v < 0.858 && u > 0.16 && u < 0.72 {
		return true
	}
	return false
}

func makeSystemLayer(width, height int) *image.NRGBA {
	layer := image.NewNRGBA(image.Rect(0, 0, width, height))
	if !systemOverlayEnabled {
		return layer
	}
	panel := color.NRGBA{R: 111, G: 205, B: 226, A: 54}
	fillRoundedRect(layer, int(0.035*float64(width)), int(0.045*float64(height)),
		int(0.315*float64(width)), int(0.145*float64(height)), int(0.025*float64(height)), panel)

	rail := color.NRGBA{R: 236, G: 245, B: 246, A: 39}
	x0 := int(0.905 * float64(width))
	x1 := int(0.962 * float64(width))
	fillRoundedRect(layer, x0, int(0.18*float64(height)), x1, int(0.82*float64(height)),
		int(0.025*float64(height)), rail)
	for i := 0; i < 4; i++ {
		y0 := int((0.245 + float64(i)*0.13) * float64(height))
		y1 := y0 + int(0.070*float64(height))
		fillRoundedRect(layer, x0+int(0.012*float64(width)), y0, x1-int(0.012*float64(width)), y1,
			int(0.018*float64(height)), color.NRGBA{R: 132, G: 220, B: 239, A: 72})
	}
	fillRoundedRect(layer, int(0.055*float64(width)), int(0.885*float64(height)),
		int(0.280*float64(width)), int(0.925*float64(height)), int(0.012*float64(height)),
		color.NRGBA{R: 255, G: 255, B: 255, A: 27})
	return layer
}

func fillRoundedRect(img *image.NRGBA, x0, y0, x1, y1, radius int, c color.NRGBA) {
	if radius < 0 {
		radius = 0
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if !image.Pt(x, y).In(img.Rect) {
				continue
			}
			dx := 0
			if x < x0+radius {
				dx = x0 + radius - x
			} else if x >= x1-radius {
				dx = x - (x1 - radius - 1)
			}
			dy := 0
			if y < y0+radius {
				dy = y0 + radius - y
			} else if y >= y1-radius {
				dy = y - (y1 - radius - 1)
			}
			if dx*dx+dy*dy > radius*radius {
				continue
			}
			img.SetNRGBA(x, y, c)
		}
	}
}
