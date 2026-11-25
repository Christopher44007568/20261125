package main

import (
	"image"
	"math"
)

const (
	MatBackground uint8 = 0
	MatSkin       uint8 = 1
	MatHair       uint8 = 2
	MatCloth      uint8 = 3
	MatEye        uint8 = 4
)

type Vertex struct {
	Pos   Vec3
	Color RGB
}

type Tri struct {
	V   [3]Vertex
	Mat uint8
}

type FrameBuffer struct {
	W     int
	H     int
	Image *image.RGBA
	Depth []float64
	Mat   []uint8
}

func NewFrameBuffer(w, h int) *FrameBuffer {
	return &FrameBuffer{
		W:     w,
		H:     h,
		Image: image.NewRGBA(image.Rect(0, 0, w, h)),
		Depth: make([]float64, w*h),
		Mat:   make([]uint8, w*h),
	}
}

func (fb *FrameBuffer) Clear() {
	top := RGB{R: 0.84, G: 0.89, B: 0.93}
	bottom := RGB{R: 0.66, G: 0.73, B: 0.80}
	for y := 0; y < fb.H; y++ {
		t := float64(y) / float64(fb.H-1)
		c := RGB{
			R: lerpFloat(top.R, bottom.R, t),
			G: lerpFloat(top.G, bottom.G, t),
			B: lerpFloat(top.B, bottom.B, t),
		}
		r := uint8(clampFloat(c.R*255, 0, 255))
		g := uint8(clampFloat(c.G*255, 0, 255))
		b := uint8(clampFloat(c.B*255, 0, 255))
		for x := 0; x < fb.W; x++ {
			i := y*fb.W + x
			o := y*fb.Image.Stride + x*4
			fb.Image.Pix[o+0] = r
			fb.Image.Pix[o+1] = g
			fb.Image.Pix[o+2] = b
			fb.Image.Pix[o+3] = 255
			fb.Depth[i] = math.Inf(1)
			fb.Mat[i] = MatBackground
		}
	}
}

type projectedVertex struct {
	x, y, invZ float64
	color      RGB
}

func RenderTriangles(fb *FrameBuffer, cam Camera, width, height int, tris []Tri) {
	for _, tri := range tris {
		var pv [3]projectedVertex
		visible := true
		for i, v := range tri.V {
			sx, sy, _, invZ, ok := cam.Project(v.Pos, width, height)
			if !ok {
				visible = false
				break
			}
			pv[i] = projectedVertex{x: sx, y: sy, invZ: invZ, color: v.Color}
		}
		if !visible {
			continue
		}
		drawProjectedTri(fb, tri.Mat, pv)
	}
}

func edgeValue(ax, ay, bx, by, px, py float64) float64 {
	return (bx-ax)*(py-ay) - (by-ay)*(px-ax)
}

func drawProjectedTri(fb *FrameBuffer, mat uint8, pv [3]projectedVertex) {
	area := edgeValue(pv[0].x, pv[0].y, pv[1].x, pv[1].y, pv[2].x, pv[2].y)
	if math.Abs(area) < 1e-8 {
		return
	}

	minX := int(math.Floor(math.Min(pv[0].x, math.Min(pv[1].x, pv[2].x))))
	maxX := int(math.Ceil(math.Max(pv[0].x, math.Max(pv[1].x, pv[2].x))))
	minY := int(math.Floor(math.Min(pv[0].y, math.Min(pv[1].y, pv[2].y))))
	maxY := int(math.Ceil(math.Max(pv[0].y, math.Max(pv[1].y, pv[2].y))))
	minX = maxInt(minX, 0)
	minY = maxInt(minY, 0)
	maxX = minInt(maxX, fb.W-1)
	maxY = minInt(maxY, fb.H-1)
	if minX > maxX || minY > maxY {
		return
	}

	invArea := 1 / area
	for y := minY; y <= maxY; y++ {
		py := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			l0 := edgeValue(pv[1].x, pv[1].y, pv[2].x, pv[2].y, px, py) * invArea
			l1 := edgeValue(pv[2].x, pv[2].y, pv[0].x, pv[0].y, px, py) * invArea
			l2 := edgeValue(pv[0].x, pv[0].y, pv[1].x, pv[1].y, px, py) * invArea
			if l0 < 0 || l1 < 0 || l2 < 0 {
				continue
			}

			invZSum := l0*pv[0].invZ + l1*pv[1].invZ + l2*pv[2].invZ
			if invZSum <= 0 {
				continue
			}
			z := 1 / invZSum
			idx := y*fb.W + x
			if z >= fb.Depth[idx] {
				continue
			}

			r := (l0*pv[0].color.R*pv[0].invZ +
				l1*pv[1].color.R*pv[1].invZ +
				l2*pv[2].color.R*pv[2].invZ) / invZSum
			g := (l0*pv[0].color.G*pv[0].invZ +
				l1*pv[1].color.G*pv[1].invZ +
				l2*pv[2].color.G*pv[2].invZ) / invZSum
			b := (l0*pv[0].color.B*pv[0].invZ +
				l1*pv[1].color.B*pv[1].invZ +
				l2*pv[2].color.B*pv[2].invZ) / invZSum

			fb.Depth[idx] = z
			fb.Mat[idx] = mat
			o := y*fb.Image.Stride + x*4
			fb.Image.Pix[o+0] = colorByte(r)
			fb.Image.Pix[o+1] = colorByte(g)
			fb.Image.Pix[o+2] = colorByte(b)
			fb.Image.Pix[o+3] = 255
		}
	}
}

func colorByte(v float64) uint8 {
	return uint8(clampFloat(math.Round(v*255), 0, 255))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
