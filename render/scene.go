package render

import (
	"math"
	"sort"

	"ruledintersect/geom"
	"ruledintersect/intersect"
)

var (
	Background     = RGB{246, 244, 238}
	Untrimmed      = RGB{130, 181, 214}
	UntrimmedAlt   = RGB{151, 177, 207}
	SurfaceWire    = RGB{82, 112, 137}
	ValidColor     = RGB{20, 154, 78}
	DuplicateColor = RGB{226, 92, 22}
	TextColor      = RGB{38, 45, 52}
	MutedText      = RGB{91, 99, 108}
	PanelBorder    = RGB{201, 203, 196}
)

const (
	viewW    = 458
	viewH    = 478
	panelGap = 16
	headerH  = 78
	footerH  = 64
	margin   = 16
)

type viewDef struct {
	name  string
	xAxis int
	yAxis int
	dAxis int
}

var views = []viewDef{
	{name: "FRONT  X-Z", xAxis: 0, yAxis: 2, dAxis: 1},
	{name: "TOP  X-Y", xAxis: 0, yAxis: 1, dAxis: 2},
	{name: "SIDE  Y-Z", xAxis: 1, yAxis: 2, dAxis: 0},
}

// PanelRect returns the pixel rectangle of a zero-based view panel in the
// composite image.
func PanelRect(index int) (x, y, w, h int) {
	return margin + index*(viewW+panelGap), headerH, viewW, viewH
}

type viewProject struct {
	def              viewDef
	panelX, panelY   int
	centerX, centerY float64
	scale            float64
}

type projectedFace struct {
	p     [3]PixelPoint
	depth float64
	shade float64
	alt   bool
}

// ThreeViews renders filled untrimmed surface regions, duplicate intersection
// support, and the stitched valid intersection in three orthographic views.
func ThreeViews(title string, s1, s2 *geom.RuledSurface,
	result intersect.Result) *Image {
	totalW := 2*margin + 3*viewW + 2*panelGap
	totalH := headerH + viewH + footerH
	im := NewImage(totalW, totalH, Background)
	drawHeader(im, title, "VALID GREEN / DUPLICATE ORANGE / UNTRIMMED BLUE")
	drawFooter(im, result)

	m1, m2 := geom.BuildMesh(s1), geom.BuildMesh(s2)
	lo, hi := meshBounds(m1, m2)
	span := math.Max(hi.X-lo.X, math.Max(hi.Y-lo.Y, hi.Z-lo.Z))
	if span < 1e-9 {
		span = 1
	}
	scale := math.Min(float64(viewW)*0.84/span, float64(viewH)*0.82/span)
	center := lo.Add(hi).Mul(0.5)
	for i, def := range views {
		px := margin + i*(viewW+panelGap)
		py := headerH
		vp := viewProject{
			def: def, panelX: px, panelY: py, scale: scale,
			centerX: center.At(def.xAxis), centerY: center.At(def.yAxis),
		}
		im.Rect(px, py, px+viewW-1, py+viewH-1, RGB{252, 251, 247})
		drawFaces(im, vp, m1, m2)
		drawWireGrid(im, vp, m1, false)
		drawWireGrid(im, vp, m2, true)
		drawIntersection(im, vp, result)
		im.Rect(px, py, px+viewW-1, py+2, PanelBorder)
		im.Rect(px, py+viewH-3, px+viewW-1, py+viewH-1, PanelBorder)
		im.Rect(px, py, px+2, py+viewH-1, PanelBorder)
		im.Rect(px+viewW-3, py, px+viewW-1, py+viewH-1, PanelBorder)
		im.Text(px+12, py+10, def.name, TextColor, 1)
	}
	return im
}

func meshBounds(m1, m2 *geom.Mesh) (geom.Vec3, geom.Vec3) {
	lo1, hi1 := m1.Bounds()
	lo2, hi2 := m2.Bounds()
	return geom.V(math.Min(lo1.X, lo2.X), math.Min(lo1.Y, lo2.Y), math.Min(lo1.Z, lo2.Z)),
		geom.V(math.Max(hi1.X, hi2.X), math.Max(hi1.Y, hi2.Y), math.Max(hi1.Z, hi2.Z))
}

func (vp viewProject) point(p geom.Vec3) PixelPoint {
	x := p.At(vp.def.xAxis)
	y := p.At(vp.def.yAxis)
	return PixelPoint{
		X: float64(vp.panelX) + float64(viewW)/2 + (x-vp.centerX)*vp.scale,
		Y: float64(vp.panelY) + float64(viewH)/2 - (y-vp.centerY)*vp.scale,
	}
}

func projectFaces(vp viewProject, mesh *geom.Mesh, alt bool) []projectedFace {
	faces := make([]projectedFace, 0, len(mesh.Triangles))
	light := geom.V(-0.35, 0.45, 0.82).Unit()
	for _, t := range mesh.Triangles {
		n := t.Normal()
		dot := math.Abs(n.Dot(light))
		faces = append(faces, projectedFace{
			p: [3]PixelPoint{
				vp.point(t.P[0]), vp.point(t.P[1]), vp.point(t.P[2]),
			},
			depth: (t.P[0].At(vp.def.dAxis) + t.P[1].At(vp.def.dAxis) +
				t.P[2].At(vp.def.dAxis)) / 3,
			shade: 0.72 + 0.28*dot,
			alt:   alt,
		})
	}
	return faces
}

func drawFaces(im *Image, vp viewProject, m1, m2 *geom.Mesh) {
	faces := append(projectFaces(vp, m1, false), projectFaces(vp, m2, true)...)
	sort.Slice(faces, func(i, j int) bool { return faces[i].depth > faces[j].depth })
	for _, f := range faces {
		base := Untrimmed
		if f.alt {
			base = UntrimmedAlt
		}
		col := shadeRGB(base, f.shade)
		im.FillTriangle(f.p[0], f.p[1], f.p[2], col, 0.62)
	}
}

func shadeRGB(c RGB, f float64) RGB {
	clamp := func(v float64) uint8 {
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint8(v)
	}
	return RGB{clamp(float64(c.R) * f), clamp(float64(c.G) * f), clamp(float64(c.B) * f)}
}
