package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

var sheetPalette = []string{"#65a30d", "#d97706", "#0d9488", "#7c3aed", "#db2777"}

func sheetColor(sheet string, order []string) string {
	for i, s := range order {
		if s == sheet {
			return sheetPalette[i%len(sheetPalette)]
		}
	}
	return "#475569"
}

func sheetOrderOf(m *Mesh) []string {
	var order []string
	for _, f := range m.Faces {
		if !contains(order, f.Provenance.Sheet) {
			order = append(order, f.Provenance.Sheet)
		}
	}
	return order
}

type renderOptions struct {
	Title       string
	Subtitle    string
	Mode        string // before | after | coverage
	Topo        *Topology
	Splits      []SplitRecord
	OrigVerts   int
	StatusLines []string
	StatusColor string
}

// explodeOffsets 为拆分后的复制顶点计算屏幕错位（沿薄片法线方向），
// 使重合的拓扑边在修复后视图中彼此分离。
func explodeOffsets(m *Mesh, splits []SplitRecord) map[int]p2 {
	offsets := map[int]p2{}
	for _, rec := range splits {
		for _, g := range rec.Groups {
			if g.ReusedSpine {
				continue
			}
			var n Vec3
			for _, fi := range g.FaceIndices {
				n = n.Add(m.FaceNormal(m.Faces[fi]))
			}
			dir := projectDir(n.Normalized())
			l := math.Hypot(dir.x, dir.y)
			if l < 1e-9 {
				dir = p2{1, 0}
				l = 1
			}
			mag := 15.0 * float64(g.Index)
			off := p2{dir.x / l * mag, dir.y / l * mag}
			for _, nv := range g.NewVertexMap {
				offsets[nv] = off
			}
		}
	}
	return offsets
}

// RenderScene 渲染拓扑边 + 面法线主视图，右侧附拓扑摘要面板。
func RenderScene(m *Mesh, opts renderOptions) string {
	t := opts.Topo
	var offsets map[int]p2
	if opts.Mode == "after" {
		offsets = explodeOffsets(m, opts.Splits)
	}
	pr := projectMesh(m, offsets)
	order := sheetOrderOf(m)
	s := newSVG(opts.Title)
	s.text(38, 49, `font-family="sans-serif" font-size="25" font-weight="700" fill="#0f172a"`, opts.Title)
	s.text(38, 76, `font-family="sans-serif" font-size="13" fill="#475569"`, opts.Subtitle)
	idx := make([]int, len(m.Faces))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool {
		return faceDepth(pr, m.Faces[idx[a]]) < faceDepth(pr, m.Faces[idx[b]])
	})
	coverageOf := map[string]string{}
	for _, fc := range t.FaceCoverage {
		coverageOf[fc.FaceID] = fc.Coverage
	}
	for _, fi := range idx {
		f := m.Faces[fi]
		color := sheetColor(f.Provenance.Sheet, order)
		var pts []string
		for _, vi := range f.Vertices {
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", pr.pts[vi].x, pr.pts[vi].y))
		}
		joined := strings.Join(pts, " ")
		s.raw(`<polygon points="%s" fill="%s" fill-opacity=".20" stroke="%s" stroke-opacity=".50" stroke-width="1.1"/>`,
			joined, color, color)
		if opts.Mode == "coverage" {
			switch coverageOf[f.ID] {
			case "direct":
				s.raw(`<polygon points="%s" fill="url(#defectHatch) #fecaca" stroke="#dc2626" stroke-width="1.4" stroke-dasharray="5 3"/>`,
					joined)
			case "same-sheet":
				s.raw(`<polygon points="%s" fill="#f59e0b" fill-opacity=".16"/>`, joined)
			}
		}
	}
	for _, fi := range idx {
		f := m.Faces[fi]
		c := centroid2(pr, f)
		label := f.ID + " · " + f.Provenance.Sheet
		w := 18.2 + 6.2*float64(len([]rune(label)))
		s.raw(`<rect x="%.1f" y="%.1f" width="%.1f" height="18" rx="5" fill="#ffffff" fill-opacity=".82"/>`,
			c.x-w/2, c.y-9, w)
		s.text(c.x-w/2+6, c.y+4, `font-family="monospace" font-size="10" font-weight="700" fill="#334155"`, label)
	}
	for _, e := range t.Edges {
		a, b := pr.pts[e.U], pr.pts[e.V]
		mid := p2{(a.x + b.x) / 2, (a.y + b.y) / 2}
		switch e.Status {
		case StatusNonManifold:
			s.line(a.x, a.y, b.x, b.y, `stroke="#fecaca" stroke-width="10" stroke-linecap="round" opacity=".8"`)
			s.line(a.x, a.y, b.x, b.y, `stroke="#dc2626" stroke-width="3.4" stroke-linecap="round"`)
			s.pill(mid.x, mid.y, e.ID, "#dc2626", 9)
		case StatusBoundary:
			s.line(a.x, a.y, b.x, b.y, `stroke="#2563eb" stroke-width="1.8" stroke-linecap="round"`)
			s.pill(mid.x, mid.y, e.ID, "#2563eb", 9)
		default:
			s.line(a.x, a.y, b.x, b.y, `stroke="#334155" stroke-width="1.5" stroke-linecap="round"`)
			s.pill(mid.x, mid.y, e.ID, "#334155", 9)
		}
	}
	for _, e := range t.NonManifoldEdges {
		a, b := pr.pts[e.U], pr.pts[e.V]
		mid := p2{(a.x + b.x) / 2, (a.y + b.y) / 2}
		sheets := map[string]bool{}
		for _, fi := range e.FaceIndices {
			sheets[m.Faces[fi].Provenance.Sheet] = true
		}
		label := fmt.Sprintf("NON-MANIFOLD · %d faces / %d sheets", e.Count, len(sheets))
		bw := 16.0 + 7.0*float64(len(label))
		bx, by := mid.x+28, mid.y-13
		if bx+bw > 836 {
			bx = mid.x - 28 - bw
		}
		s.line(mid.x, mid.y, bx+bw/2, by+13, `stroke="#dc2626" stroke-width="1"`)
		s.circle(mid.x, mid.y, 4, `fill="#dc2626"`)
		s.raw(`<rect x="%.1f" y="%.1f" width="%.1f" height="26" rx="7" fill="#dc2626" stroke="#991b1b"/>`, bx, by, bw)
		s.text(bx+bw/2, by+17, `text-anchor="middle" font-family="sans-serif" font-size="11" font-weight="700" fill="#ffffff"`, label)
	}
	for fi, f := range m.Faces {
		c := centroid2(pr, f)
		dir := projectDir(m.FaceNormal(f))
		l := math.Hypot(dir.x, dir.y)
		if l < 1e-9 {
			continue
		}
		const arrowLen = 74.0
		ex, ey := c.x+dir.x/l*arrowLen, c.y+dir.y/l*arrowLen
		s.line(c.x, c.y, ex, ey, `stroke="#0f766e" stroke-width="1.5" marker-end="url(#normalArrow)"`)
		s.text(ex+4, ey-3, `font-family="monospace" font-size="9" fill="#0f766e"`, fmt.Sprintf("N%02d", fi))
	}
	for i, pt := range pr.pts {
		attrs := `fill="#0f172a" stroke="#ffffff" stroke-width="1.2"`
		if opts.Mode == "after" && i >= opts.OrigVerts {
			attrs = `fill="#7c3aed" stroke="#ffffff" stroke-width="1.2"`
		}
		s.circle(pt.x, pt.y, 3, attrs)
	}
	renderPanel(s, m, opts)
	s.text(38, 755, `font-family="sans-serif" font-size="11" fill="#64748b"`,
		"纯 Go 标准库生成 · 面编号与来源在修复前后保持稳定 · 虚线/错位描边仅用于展示重合的拓扑边")
	return s.String()
}

func faceDepth(pr *projection, f Face) float64 {
	var d float64
	for _, vi := range f.Vertices {
		d += pr.depth[vi]
	}
	return d / float64(len(f.Vertices))
}

func centroid2(pr *projection, f Face) p2 {
	var c p2
	for _, vi := range f.Vertices {
		c.x += pr.pts[vi].x
		c.y += pr.pts[vi].y
	}
	n := float64(len(f.Vertices))
	return p2{c.x / n, c.y / n}
}
