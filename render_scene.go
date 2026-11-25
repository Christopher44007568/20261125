package main

import (
	"image"
	"image/color"
	"strconv"
)

var (
	colBackground = color.RGBA{R: 248, G: 250, B: 252, A: 255}
	colGrid       = color.RGBA{R: 226, G: 232, B: 240, A: 255}
	colPatch      = color.RGBA{R: 220, G: 252, B: 231, A: 255}
	colOuter      = color.RGBA{R: 21, G: 128, B: 61, A: 255}
	colHoleFill   = color.RGBA{R: 30, G: 58, B: 138, A: 82}
	colHoleHatch  = color.RGBA{R: 191, G: 219, B: 254, A: 150}
	colHoleEdge   = color.RGBA{R: 234, G: 88, B: 12, A: 255}
	colTriEdge    = color.RGBA{R: 15, G: 23, B: 42, A: 175}
	colBadFill    = color.RGBA{R: 239, G: 68, B: 68, A: 118}
	colBadEdge    = color.RGBA{R: 185, G: 28, B: 28, A: 255}
	colCrossFill  = color.RGBA{R: 250, G: 204, B: 21, A: 230}
	colInk        = color.RGBA{R: 15, G: 23, B: 42, A: 255}
	colMuted      = color.RGBA{R: 71, G: 85, B: 105, A: 255}
	colOBJEdge    = color.RGBA{R: 124, G: 58, B: 237, A: 255}
)

func diagnosticPlot() *Plot {
	return NewPlot(1080, 1080, 2, image.Rect(80, 150, 1000, 1070), v2(-4, -4), v2(104, 104))
}

func drawGrid(p *Plot) {
	for x := 0.0; x <= 100; x += 10 {
		p.LineWorld(v2(x, 0), v2(x, 100), 1, colGrid)
	}
	for y := 0.0; y <= 100; y += 10 {
		p.LineWorld(v2(0, y), v2(100, y), 1, colGrid)
	}
	p.StrokeWorld([]Vec2{v2(0, 0), v2(100, 0), v2(100, 100), v2(0, 100)}, true, 1.4, colGrid)
}

func drawLegend(img *image.RGBA, mode string) {
	type item struct {
		label string
		kind  string
		c     color.RGBA
	}
	items := []item{
		{"OUTER", "line", colOuter},
		{"HOLE MASK", "fill", colHoleFill},
		{"TRI EDGES", "line", colTriEdge},
		{"BAD TRI", "fill", colBadFill},
		{"CROSSING", "dot", colCrossFill},
	}
	if mode == "obj" {
		items = []item{
			{"OBJ TRI EDGES", "line", colTriEdge},
			{"OBJ BOUNDARY", "line", colOBJEdge},
			{"RELOADED VERT", "dot", colOBJEdge},
		}
	}
	if mode == "obj-overlay" {
		items = []item{
			{"OBJ EDGES", "line", colTriEdge},
			{"OBJ BOUNDARY", "line", colOBJEdge},
			{"HOLE MASK", "fill", colHoleFill},
			{"BAD TRI", "fill", colBadFill},
		}
	}
	x := 40
	y := 108
	for _, it := range items {
		switch it.kind {
		case "line":
			fillScreenRect(img, x, y+5, 28, 4, it.c)
		case "fill":
			fillScreenRect(img, x, y, 28, 16, it.c)
		case "dot":
			if it.label == "CROSSING" {
				drawScreenDisc(img, x+14, y+8, 8, colBadEdge)
				drawScreenDisc(img, x+14, y+8, 5, it.c)
			} else {
				drawScreenDisc(img, x+14, y+8, 7, it.c)
			}
		}
		DrawText(img, it.label, x+36, y+2, 2, colInk)
		x += 36 + TextWidth(it.label, 2) + 28
	}
}

func fillScreenRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			blendPixel(img, xx, yy, c)
		}
	}
}

func drawScreenDisc(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func finishDiagnostic(p *Plot, title, subtitle, legendMode string) *image.RGBA {
	img := p.Finish()
	DrawText(img, title, 40, 34, 3, colInk)
	DrawText(img, subtitle, 42, 78, 2, colMuted)
	drawLegend(img, legendMode)
	return img
}

func drawTriangleEdges(p *Plot, mesh *Mesh2, c color.RGBA, width float64) {
	for _, tri := range mesh.Triangles {
		a, b, cpt := mesh.Vertices[tri[0]], mesh.Vertices[tri[1]], mesh.Vertices[tri[2]]
		p.LineWorld(a, b, width, c)
		p.LineWorld(b, cpt, width, c)
		p.LineWorld(cpt, a, width, c)
	}
}

func drawBadFills(p *Plot, mesh *Mesh2, bad []InvalidTriangle) {
	for _, item := range bad {
		tri := item.Triangle
		poly := []Vec2{mesh.Vertices[tri[0]], mesh.Vertices[tri[1]], mesh.Vertices[tri[2]]}
		p.FillPolygonWorld(poly, colBadFill, nil)
	}
}

func drawBadEdges(p *Plot, mesh *Mesh2, bad []InvalidTriangle) {
	for _, item := range bad {
		tri := item.Triangle
		a, b, c := mesh.Vertices[tri[0]], mesh.Vertices[tri[1]], mesh.Vertices[tri[2]]
		p.LineWorld(a, b, 2.6, colBadEdge)
		p.LineWorld(b, c, 2.6, colBadEdge)
		p.LineWorld(c, a, 2.6, colBadEdge)
	}
}

func drawCrossings(p *Plot, domain TrimDomain) {
	for _, loop := range domain.RawLoops {
		for _, cross := range loop.Crossings {
			p.DiscWorld(cross.Point, 8, colBadEdge)
			p.DiscWorld(cross.Point, 5, colCrossFill)
		}
	}
}

func RenderOverlayPNG(title, subtitle string, domain TrimDomain, mesh *Mesh2, bad []InvalidTriangle) *image.RGBA {
	p := diagnosticPlot()
	p.Clear(colBackground)
	drawGrid(p)
	p.FillPolygonWorld(domain.Outer, colPatch, nil)
	drawBadFills(p, mesh, bad)
	for _, hole := range domain.Holes {
		p.FillPolygonWorld(hole, colHoleFill, &colHoleHatch)
	}
	drawTriangleEdges(p, mesh, colTriEdge, 1.15)
	drawBadEdges(p, mesh, bad)
	p.StrokeWorld(domain.Outer, true, 3.4, colOuter)
	for _, hole := range domain.Holes {
		p.StrokeWorld(hole, true, 2.5, colHoleEdge)
	}
	drawCrossings(p, domain)
	badSubtitle := subtitle + "  BAD=" + strconv.Itoa(len(bad))
	return finishDiagnostic(p, title, badSubtitle, "trim")
}

func RenderOBJReloadPNG(obj *OBJMesh, title, subtitle string) *image.RGBA {
	p := diagnosticPlot()
	p.Clear(colBackground)
	drawGrid(p)

	mesh := objAsMesh2(obj)
	drawTriangleEdges(p, mesh, color.RGBA{R: 100, G: 116, B: 139, A: 150}, 0.9)
	for _, edge := range OBJBoundaryEdges(obj) {
		p.LineWorld(edge[0], edge[1], 3.2, colOBJEdge)
	}
	for _, v := range obj.Vertices {
		p.DiscWorld(v2(v.X, v.Y), 2.2, colOBJEdge)
	}

	fullSubtitle := subtitle + "  V=" + strconv.Itoa(len(obj.Vertices)) +
		" F=" + strconv.Itoa(len(obj.Triangles)) +
		" BOUNDARY_EDGES=" + strconv.Itoa(len(OBJBoundaryEdges(obj)))
	return finishDiagnostic(p, title, fullSubtitle, "obj")
}

func RenderOBJOverlayPNG(obj *OBJMesh, domain TrimDomain, bad []InvalidTriangle, title, subtitle string) *image.RGBA {
	p := diagnosticPlot()
	p.Clear(colBackground)
	drawGrid(p)
	p.FillPolygonWorld(domain.Outer, colPatch, nil)
	mesh := objAsMesh2(obj)
	drawBadFills(p, mesh, bad)
	for _, hole := range domain.Holes {
		p.FillPolygonWorld(hole, colHoleFill, &colHoleHatch)
	}
	drawTriangleEdges(p, mesh, color.RGBA{R: 51, G: 65, B: 85, A: 190}, 1.0)
	for _, edge := range OBJBoundaryEdges(obj) {
		p.LineWorld(edge[0], edge[1], 3.0, colOBJEdge)
	}
	drawBadEdges(p, mesh, bad)
	drawCrossings(p, domain)
	fullSubtitle := subtitle + "  BAD=" + strconv.Itoa(len(bad)) +
		"  BOUNDARY_EDGES=" + strconv.Itoa(len(OBJBoundaryEdges(obj)))
	return finishDiagnostic(p, title, fullSubtitle, "obj-overlay")
}

func objAsMesh2(obj *OBJMesh) *Mesh2 {
	mesh := &Mesh2{Vertices: make([]Vec2, len(obj.Vertices))}
	for i, v := range obj.Vertices {
		mesh.Vertices[i] = v2(v.X, v.Y)
	}
	mesh.Triangles = make([]Triangle, len(obj.Triangles))
	for i, t := range obj.Triangles {
		mesh.Triangles[i] = Triangle{t[0], t[1], t[2]}
	}
	return mesh
}
