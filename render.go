package main

import "math"

type HairFragment struct {
	Segment int
	T       float64
	Depth   float64
	Alpha   float64
	Side    float64
}

type RenderResult struct {
	Width        int
	Height       int
	Beauty       []Vec3
	Shadow       []Vec3
	Transmission []Vec3
}

type Renderer struct {
	Scene  *Scene
	Deep   *DeepOpacityMap
	Camera Camera

	skinPoint  []Vec3
	skinNormal []Vec3
	skinDepth  []float64
	fragments  [][]HairFragment
}

func NewRenderer(scene *Scene, deep *DeepOpacityMap, width, height int) *Renderer {
	pixelCount := width * height
	return &Renderer{
		Scene:      scene,
		Deep:       deep,
		Camera:     NewCamera(width, height),
		skinPoint:  make([]Vec3, pixelCount),
		skinNormal: make([]Vec3, pixelCount),
		skinDepth:  make([]float64, pixelCount),
		fragments:  make([][]HairFragment, pixelCount),
	}
}

func (r *Renderer) Render() RenderResult {
	r.traceSkin()
	r.rasterizeHair()

	beauty := make([]Vec3, r.Camera.Width*r.Camera.Height)
	shadow := make([]Vec3, len(beauty))
	transmission := make([]Vec3, len(beauty))

	for y := 0; y < r.Camera.Height; y++ {
		for x := 0; x < r.Camera.Width; x++ {
			idx := y*r.Camera.Width + x
			beauty[idx], shadow[idx], transmission[idx] = r.shadePixel(x, y, idx)
		}
	}

	return RenderResult{
		Width:        r.Camera.Width,
		Height:       r.Camera.Height,
		Beauty:       beauty,
		Shadow:       shadow,
		Transmission: transmission,
	}
}

func (r *Renderer) traceSkin() {
	for y := 0; y < r.Camera.Height; y++ {
		for x := 0; x < r.Camera.Width; x++ {
			idx := y*r.Camera.Width + x
			r.skinDepth[idx] = math.Inf(1)
			ray := r.Camera.Ray(x, y)
			if t, normal, ok := r.Scene.Skin.Intersect(r.Camera.Eye, ray); ok {
				point := r.Camera.Eye.Add(ray.Mul(t))
				r.skinPoint[idx] = point
				r.skinNormal[idx] = normal
				r.skinDepth[idx] = point.Sub(r.Camera.Eye).Dot(r.Camera.Forward)
			}
		}
	}
}

func (r *Renderer) rasterizeHair() {
	for segmentIndex, segment := range r.Scene.Segments {
		x0, y0, z0, ok0 := r.Camera.Project(segment.A)
		x1, y1, z1, ok1 := r.Camera.Project(segment.B)
		if !ok0 || !ok1 {
			continue
		}

		dx := x1 - x0
		dy := y1 - y0
		screenLenSqr := dx*dx + dy*dy
		if screenLenSqr < 1e-9 {
			continue
		}
		screenLen := math.Sqrt(screenLenSqr)
		tx := dx / screenLen
		ty := dy / screenLen
		nx := -ty
		ny := tx

		r0 := r.Camera.Focal * segment.RA / z0
		r1 := r.Camera.Focal * segment.RB / z1
		boundRadius := math.Max(r0, r1) + 1.0

		minX := maxInt(0, int(math.Floor(math.Min(x0, x1)-boundRadius-1)))
		maxX := minInt(r.Camera.Width-1, int(math.Ceil(math.Max(x0, x1)+boundRadius+1)))
		minY := maxInt(0, int(math.Floor(math.Min(y0, y1)-boundRadius-1)))
		maxY := minInt(r.Camera.Height-1, int(math.Ceil(math.Max(y0, y1)+boundRadius+1)))

		for y := minY; y <= maxY; y++ {
			py := float64(y) + 0.5
			for x := minX; x <= maxX; x++ {
				px := float64(x) + 0.5
				t := clamp01(((px-x0)*dx + (py-y0)*dy) / screenLenSqr)
				cx := x0 + dx*t
				cy := y0 + dy*t
				offsetX := px - cx
				offsetY := py - cy
				dist := math.Hypot(offsetX, offsetY)
				radius := lerp(r0, r1, t)
				coverage := 1 - smoothstep(radius*0.54, radius+0.75, dist)
				if coverage < 0.035 {
					continue
				}

				depth := 1 / lerp(1/z0, 1/z1, t)
				idx := y*r.Camera.Width + x
				if depth > r.skinDepth[idx]+0.012 {
					continue
				}

				side := clamp((offsetX*nx+offsetY*ny)/math.Max(radius, 1e-6), -1, 1)
				r.fragments[idx] = append(r.fragments[idx], HairFragment{
					Segment: segmentIndex,
					T:       t,
					Depth:   depth,
					Alpha:   coverage,
					Side:    side,
				})
			}
		}
	}
}
