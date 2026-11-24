package main

import (
	"image"
	"math"
)

type Camera struct {
	Center Vec3
	Right  Vec3
	Up     Vec3
	Depth  Vec3
	Scale  float64
	W      int
	H      int
}

func NewCamera(center Vec3, scale float64, w, h int) Camera {
	// Mixing Z into the screen-right vector makes the sagitta gap of the
	// curved shared edge visible as a horizontal opening in the zoom panels.
	right := V(0.82, 0, 0.572).Unit()
	up := V(0, 1, 0)
	return Camera{
		Center: center,
		Right:  right,
		Up:     up,
		Depth:  up.Cross(right).Unit(),
		Scale:  scale,
		W:      w,
		H:      h,
	}
}

func NewOverviewCamera(center Vec3, scale float64, w, h int) Camera {
	right := V(1, 0, 0)
	up := V(0, 0.82, 0.572).Unit()
	return Camera{
		Center: center,
		Right:  right,
		Up:     up,
		Depth:  up.Cross(right).Unit(),
		Scale:  scale,
		W:      w,
		H:      h,
	}
}

func FitCamera(points []Vec3, w, h int, margin float64) Camera {
	return fitCamera(NewCamera(V(0, 0, 0), 1, w, h), points, w, h, margin)
}

func FitOverviewCamera(points []Vec3, w, h int, margin float64) Camera {
	return fitCamera(NewOverviewCamera(V(0, 0, 0), 1, w, h), points, w, h, margin)
}

func fitCamera(cam Camera, points []Vec3, w, h int, margin float64) Camera {
	cam.W, cam.H = w, h
	if len(points) == 0 {
		return NewCamera(V(0, 0, 3), 20, w, h)
	}
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	minD, maxD := math.Inf(1), math.Inf(-1)
	for _, p := range points {
		x := p.Dot(cam.Right)
		y := p.Dot(cam.Up)
		d := p.Dot(cam.Depth)
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
		minD, maxD = math.Min(minD, d), math.Max(maxD, d)
	}
	cx := (minX + maxX) * 0.5
	cy := (minY + maxY) * 0.5
	cd := (minD + maxD) * 0.5
	cam.Center = cam.Right.Scale(cx).Add(cam.Up.Scale(cy)).Add(cam.Depth.Scale(cd))
	spanX := math.Max(maxX-minX, 1e-6)
	spanY := math.Max(maxY-minY, 1e-6)
	scaleX := (float64(w) - 2*margin) / spanX
	scaleY := (float64(h) - 2*margin) / spanY
	cam.Scale = math.Min(scaleX, scaleY)
	return cam
}

type Projected struct {
	X, Y, Z float64
}

func (c Camera) Project(p Vec3) Projected {
	d := p.Sub(c.Center)
	return Projected{
		X: float64(c.W)/2 + d.Dot(c.Right)*c.Scale,
		Y: float64(c.H)/2 - d.Dot(c.Up)*c.Scale,
		Z: d.Dot(c.Depth),
	}
}

type MeshLook struct {
	ColorA    Color
	ColorB    Color
	FillColor Color
	FillTint  float64
	Wireframe bool
	WireColor Color
	WireWidth float64
}

type rasterBuffer struct {
	w, h  int
	color []Color
	depth []float64
}

func newRasterBuffer(w, h int, bg Color) *rasterBuffer {
	r := &rasterBuffer{w: w, h: h, color: make([]Color, w*h), depth: make([]float64, w*h)}
	for i := range r.color {
		r.color[i] = bg
		r.depth[i] = math.Inf(1)
	}
	return r
}

func edgeFunction(ax, ay, bx, by, px, py float64) float64 {
	return (px-ax)*(by-ay) - (py-ay)*(bx-ax)
}

func RenderMeshPanel(m *Mesh, normals []Vec3, cam Camera, ss int, look MeshLook) *image.RGBA {
	if ss < 1 {
		ss = 1
	}
	sw, sh := cam.W*ss, cam.H*ss
	bg := RGB(238, 243, 249)
	buf := newRasterBuffer(sw, sh, bg)

	projected := make([]Projected, len(m.Pos))
	for i, p := range m.Pos {
		pr := cam.Project(p)
		pr.X *= float64(ss)
		pr.Y *= float64(ss)
		projected[i] = pr
	}

	for ti, tri := range m.Tri {
		p0 := projected[tri.A]
		p1 := projected[tri.B]
		p2 := projected[tri.C]
		area := edgeFunction(p0.X, p0.Y, p1.X, p1.Y, p2.X, p2.Y)
		if math.Abs(area) < 1e-8 {
			continue
		}

		minX := maxInt(int(math.Floor(math.Min(p0.X, math.Min(p1.X, p2.X)))), 0)
		maxX := minInt(int(math.Ceil(math.Max(p0.X, math.Max(p1.X, p2.X)))), sw-1)
		minY := maxInt(int(math.Floor(math.Min(p0.Y, math.Min(p1.Y, p2.Y)))), 0)
		maxY := minInt(int(math.Ceil(math.Max(p0.Y, math.Max(p1.Y, p2.Y)))), sh-1)
		if minX > maxX || minY > maxY {
			continue
		}

		base := look.ColorA
		if tri.Group == GroupB {
			base = look.ColorB
		}
		if tri.Filled && look.FillTint > 0 {
			base = blendColor(base, look.FillColor, look.FillTint)
		}
		n0, n1, n2 := normals[ti*3], normals[ti*3+1], normals[ti*3+2]

		for y := minY; y <= maxY; y++ {
			py := float64(y) + 0.5
			for x := minX; x <= maxX; x++ {
				px := float64(x) + 0.5
				w0 := edgeFunction(p1.X, p1.Y, p2.X, p2.Y, px, py)
				w1 := edgeFunction(p2.X, p2.Y, p0.X, p0.Y, px, py)
				w2 := edgeFunction(p0.X, p0.Y, p1.X, p1.Y, px, py)
				if area > 0 {
					if w0 < -1e-7 || w1 < -1e-7 || w2 < -1e-7 {
						continue
					}
				} else if w0 > 1e-7 || w1 > 1e-7 || w2 > 1e-7 {
					continue
				}
				b0, b1, b2 := w0/area, w1/area, w2/area
				depth := b0*p0.Z + b1*p1.Z + b2*p2.Z
				idx := y*sw + x
				if depth >= buf.depth[idx] {
					continue
				}
				n := n0.Scale(b0).Add(n1.Scale(b1)).Add(n2.Scale(b2)).Unit()
				if n.LenSqr() < 1e-12 {
					n = m.FaceNormal(ti)
				}
				buf.depth[idx] = depth
				buf.color[idx] = shadeNormal(n, base)
			}
		}
	}

	if look.Wireframe {
		width := math.Max(1, look.WireWidth*float64(ss))
		for _, tri := range m.Tri {
			ids := [3]int{tri.A, tri.B, tri.C}
			for e := 0; e < 3; e++ {
				pa := projected[ids[e]]
				pb := projected[ids[(e+1)%3]]
				drawRasterLine(buf, pa, pb, look.WireColor, width)
			}
		}
	}

	return downsampleRaster(buf, cam.W, cam.H, ss)
}

func drawRasterLine(buf *rasterBuffer, a, b Projected, c Color, width float64) {
	dx, dy := b.X-a.X, b.Y-a.Y
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps == 0 {
		blendRasterPixel(buf, int(math.Round(a.X)), int(math.Round(a.Y)), a.Z, c, width)
		return
	}
	radius := int(math.Ceil(width / 2))
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := a.X + dx*t
		y := a.Y + dy*t
		z := a.Z + (b.Z-a.Z)*t
		cx, cy := int(math.Round(x)), int(math.Round(y))
		for oy := -radius; oy <= radius; oy++ {
			for ox := -radius; ox <= radius; ox++ {
				if ox*ox+oy*oy > radius*radius+radius {
					continue
				}
				blendRasterPixel(buf, cx+ox, cy+oy, z, c, width)
			}
		}
	}
}

func blendRasterPixel(buf *rasterBuffer, x, y int, z float64, c Color, width float64) {
	if x < 0 || y < 0 || x >= buf.w || y >= buf.h {
		return
	}
	idx := y*buf.w + x
	bias := 0.015 + width*0.002
	if z > buf.depth[idx]+bias {
		return
	}
	alpha := float64(c.A) / 255
	bg := buf.color[idx]
	buf.color[idx] = blendColor(bg, c, alpha)
	if c.A == 255 {
		buf.depth[idx] = z
	}
}

func downsampleRaster(buf *rasterBuffer, w, h, ss int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, a int
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					c := buf.color[(y*ss+sy)*buf.w+x*ss+sx]
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					a += int(c.A)
				}
			}
			n := float64(ss * ss)
			o := y*img.Stride + x*4
			img.Pix[o+0] = uint8(r / int(n))
			img.Pix[o+1] = uint8(g / int(n))
			img.Pix[o+2] = uint8(b / int(n))
			img.Pix[o+3] = uint8(a / int(n))
		}
	}
	return img
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
