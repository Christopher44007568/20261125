package main

import (
	"image"
	"math"
)

type PanelMode int

const (
	ModeSilhouette PanelMode = iota
	ModeNormalSign
	ModeZebra
	ModeCurvature
)

func edge2D(ax, ay, bx, by, px, py float64) float64 {
	return (px-ax)*(by-ay) - (py-ay)*(bx-ax)
}

func renderSurface(mesh Mesh, cam Camera, mode PanelMode, w, h, ss int, maxK float64) (*image.RGBA, []float32) {
	sw, sh := w*ss, h*ss
	zBuffer := make([]float64, sw*sh)
	for i := range zBuffer {
		zBuffer[i] = math.Inf(1)
	}
	colorBuffer := make([]uint8, sw*sh*4)
	projected := make([]Projected, len(mesh.Vertices))
	for i, vertex := range mesh.Vertices {
		projected[i] = cam.Project(vertex.P)
		projected[i].X *= float64(ss)
		projected[i].Y *= float64(ss)
	}

	toEye := cam.Forward.Scale(-1)
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		i0 := mesh.Indices[t]
		i1 := mesh.Indices[t+1]
		i2 := mesh.Indices[t+2]
		p0, p1, p2 := projected[i0], projected[i1], projected[i2]
		area := edge2D(p0.X, p0.Y, p1.X, p1.Y, p2.X, p2.Y)
		if math.Abs(area) < 1e-9 {
			continue
		}

		minX := int(math.Floor(math.Min(p0.X, math.Min(p1.X, p2.X))))
		maxX := int(math.Ceil(math.Max(p0.X, math.Max(p1.X, p2.X))))
		minY := int(math.Floor(math.Min(p0.Y, math.Min(p1.Y, p2.Y))))
		maxY := int(math.Ceil(math.Max(p0.Y, math.Max(p1.Y, p2.Y))))
		minX, maxX = maxInt(minX, 0), minInt(maxX, sw-1)
		minY, maxY = maxInt(minY, 0), minInt(maxY, sh-1)
		if minX > maxX || minY > maxY {
			continue
		}

		v0, v1, v2 := mesh.Vertices[i0], mesh.Vertices[i1], mesh.Vertices[i2]
		for py := minY; py <= maxY; py++ {
			yp := float64(py) + 0.5
			for px := minX; px <= maxX; px++ {
				xp := float64(px) + 0.5
				w0 := edge2D(p1.X, p1.Y, p2.X, p2.Y, xp, yp)
				w1 := edge2D(p2.X, p2.Y, p0.X, p0.Y, xp, yp)
				w2 := edge2D(p0.X, p0.Y, p1.X, p1.Y, xp, yp)
				if area > 0 {
					if w0 < -1e-7 || w1 < -1e-7 || w2 < -1e-7 {
						continue
					}
				} else if w0 > 1e-7 || w1 > 1e-7 || w2 > 1e-7 {
					continue
				}

				b0, b1, b2 := w0/area, w1/area, w2/area
				depth := b0*p0.Depth + b1*p1.Depth + b2*p2.Depth
				idx := py*sw + px
				if depth >= zBuffer[idx]-1e-12 {
					continue
				}

				n := v0.N.Scale(b0).Add(v1.N.Scale(b1)).Add(v2.N.Scale(b2)).Unit()
				k := b0*v0.K + b1*v1.K + b2*v2.K
				var c RGB
				switch mode {
				case ModeSilhouette:
					c = shadeClay(n, toEye)
				case ModeNormalSign:
					c = normalSignColor(n)
				case ModeZebra:
					c = zebraColor(n, cam.Forward)
				case ModeCurvature:
					c = curvatureColor(k, maxK)
					c = c.Scale(0.88 + 0.12*math.Max(n.Dot(toEye), 0))
				}
				r, g, b := c.Bytes()
				o := idx * 4
				colorBuffer[o+0] = r
				colorBuffer[o+1] = g
				colorBuffer[o+2] = b
				colorBuffer[o+3] = 255
				zBuffer[idx] = depth
			}
		}
	}
	return downsampleRGBA(colorBuffer, sw, sh, ss)
}

func downsampleRGBA(src []uint8, sw, sh, ss int) (*image.RGBA, []float32) {
	w, h := sw/ss, sh/ss
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	coverage := make([]float32, w*h)
	area := float64(ss * ss)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, a float64
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					o := ((y*ss+sy)*sw + x*ss + sx) * 4
					r += float64(src[o+0])
					g += float64(src[o+1])
					b += float64(src[o+2])
					a += float64(src[o+3])
				}
			}
			o := y*img.Stride + x*4
			img.Pix[o+0] = uint8(r/area + 0.5)
			img.Pix[o+1] = uint8(g/area + 0.5)
			img.Pix[o+2] = uint8(b/area + 0.5)
			img.Pix[o+3] = uint8(a/area + 0.5)
			coverage[y*w+x] = float32(a / area / 255)
		}
	}
	return img, coverage
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
