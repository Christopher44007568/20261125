package main

import (
	"fmt"
	"math"
	"strings"
)

// svg 是一个极简 SVG 画布构建器，仅依赖标准库。
type svg struct {
	b strings.Builder
}

func newSVG(title string) *svg {
	s := &svg{}
	s.b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	s.b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="1120" height="780" viewBox="0 0 1120 780">` + "\n")
	fmt.Fprintf(&s.b, "<title>%s</title>\n", xmlEsc(title))
	s.b.WriteString(`<defs>
<pattern id="defectHatch" width="8" height="8" patternUnits="userSpaceOnUse" patternTransform="rotate(35)">
  <rect width="8" height="8" fill="#fecaca" fill-opacity=".52"/>
  <line x1="0" y1="0" x2="0" y2="8" stroke="#dc2626" stroke-width="2"/>
</pattern>
<pattern id="grid" width="24" height="24" patternUnits="userSpaceOnUse">
  <circle cx="1" cy="1" r=".8" fill="#cbd5e1"/>
</pattern>
<marker id="normalArrow" markerWidth="7" markerHeight="7" refX="6" refY="3.5" orient="auto">
  <path d="M0,0 L7,3.5 L0,7 Z" fill="#0f766e"/>
</marker>
</defs>
`)
	s.b.WriteString(`<rect width="1120" height="780" fill="#f8fafc"/>`)
	s.b.WriteString(`<rect x="18" y="18" width="1084" height="744" rx="18" fill="url(#grid) #eef2f7" opacity=".65"/>`)
	return s
}

func (s *svg) raw(format string, args ...any) {
	fmt.Fprintf(&s.b, format, args...)
	s.b.WriteByte('\n')
}

func (s *svg) line(x1, y1, x2, y2 float64, attrs string) {
	s.raw(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" %s/>`, x1, y1, x2, y2, attrs)
}

func (s *svg) circle(cx, cy, r float64, attrs string) {
	s.raw(`<circle cx="%.1f" cy="%.1f" r="%.1f" %s/>`, cx, cy, r, attrs)
}

func (s *svg) text(x, y float64, attrs, content string) {
	s.raw(`<text x="%.1f" y="%.1f" %s>%s</text>`, x, y, attrs, xmlEsc(content))
}

func (s *svg) pill(cx, cy float64, label, textColor string, fontSize float64) {
	w := 7.7 + 5.6*float64(len([]rune(label)))
	s.raw(`<rect x="%.1f" y="%.1f" width="%.1f" height="15" rx="4" fill="#ffffff" fill-opacity=".88"/>`,
		cx-w/2, cy-7.5, w)
	s.raw(`<text x="%.1f" y="%.1f" text-anchor="middle" font-family="monospace" font-size="%.0f" font-weight="700" fill="%s">%s</text>`,
		cx, cy+3.5, fontSize, textColor, xmlEsc(label))
}

func (s *svg) String() string { return s.b.String() + "</svg>\n" }

func xmlEsc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

type p2 struct{ x, y float64 }

// projection 保存正交投影结果：屏幕坐标与深度（用于画家算法）。
type projection struct {
	pts   []p2
	depth []float64
}

const (
	viewAz = -0.62 // 绕 Z 轴方位角（弧度）
	viewEl = 0.46  // 俯仰角（弧度）
)

// projectMesh 把网格顶点正交投影到绘图区 (40,96)-(824,724)，
// offsets 为额外的屏幕像素偏移（用于拆分后错位展示重合边）。
func projectMesh(m *Mesh, offsets map[int]p2) *projection {
	n := len(m.Vertices)
	sx := make([]float64, n)
	sy := make([]float64, n)
	dep := make([]float64, n)
	ca, sa := math.Cos(viewAz), math.Sin(viewAz)
	ce, se := math.Cos(viewEl), math.Sin(viewEl)
	for i, v := range m.Vertices {
		x1 := v.X*ca - v.Y*sa
		y1 := v.X*sa + v.Y*ca
		sx[i] = x1
		sy[i] = -(v.Z*ce - y1*se)
		dep[i] = v.Z*se + y1*ce
	}
	minX, maxX := sx[0], sx[0]
	minY, maxY := sy[0], sy[0]
	for i := range sx {
		minX, maxX = math.Min(minX, sx[i]), math.Max(maxX, sx[i])
		minY, maxY = math.Min(minY, sy[i]), math.Max(maxY, sy[i])
	}
	const (
		boxX0, boxY0 = 40.0, 96.0
		boxX1, boxY1 = 824.0, 724.0
	)
	dx, dy := maxX-minX, maxY-minY
	if dx < 1e-9 {
		dx = 1
	}
	if dy < 1e-9 {
		dy = 1
	}
	scale := math.Min((boxX1-boxX0)/dx, (boxY1-boxY0)/dy)
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	ox := (boxX0+boxX1)/2 - cx*scale
	oy := (boxY0+boxY1)/2 - cy*scale
	pr := &projection{pts: make([]p2, n), depth: dep}
	for i := range sx {
		pt := p2{sx[i]*scale + ox, sy[i]*scale + oy}
		if off, ok := offsets[i]; ok {
			pt.x += off.x
			pt.y += off.y
		}
		pr.pts[i] = pt
	}
	return pr
}

// projectDir 把三维方向向量投影为屏幕方向（用于法线箭头与错位方向）。
func projectDir(d Vec3) p2 {
	ca, sa := math.Cos(viewAz), math.Sin(viewAz)
	ce, se := math.Cos(viewEl), math.Sin(viewEl)
	x1 := d.X*ca - d.Y*sa
	y1 := d.X*sa + d.Y*ca
	return p2{x1, -(d.Z*ce - y1*se)}
}
