package render

import (
	"fmt"

	"ruledintersect/geom"
	"ruledintersect/intersect"
)

func drawHeader(im *Image, title, subtitle string) {
	im.Rect(0, 0, im.W-1, 4, TextColor)
	im.Text(margin, 15, title, TextColor, 2)
	im.Text(margin, 48, subtitle, MutedText, 1)
	right := "THREE ORTHOGRAPHIC VIEWS"
	x := im.W - margin - im.TextWidth(right, 1)
	im.Text(x, 20, right, MutedText, 1)
}

func drawFooter(im *Image, r intersect.Result) {
	y := headerH + viewH + 5
	im.Rect(margin, y, im.W-margin-1, y+1, PanelBorder)
	y += 15
	x := margin
	swatch := func(c RGB, label string) {
		im.Rect(x, y, x+16, y+16, c)
		im.Rect(x, y, x+16, y+16, TextColor)
		im.Rect(x+1, y+1, x+15, y+15, c)
		im.Text(x+24, y+5, label, TextColor, 1)
		x += 30 + im.TextWidth(label, 1) + 24
	}
	swatch(ValidColor, "VALID")
	swatch(DuplicateColor, "DUPLICATE")
	swatch(Untrimmed, "UNTRIMMED SURFACE")
	stats := fmt.Sprintf("VALID %d  DUP %d  CURVES %d", len(r.Valid), len(r.Duplicates), len(r.Curves))
	im.Text(x+4, y+5, stats, TextColor, 1)
	note := "SURFACE GAP CHECKED ON BOTH SIDES"
	nx := im.W - margin - im.TextWidth(note, 1)
	im.Text(nx, y+5, note, MutedText, 1)
}

func drawWireGrid(im *Image, vp viewProject, mesh *geom.Mesh, alt bool) {
	col := SurfaceWire
	if alt {
		col = RGB{104, 103, 143}
	}
	stepU := mesh.Nu / 8
	if stepU < 1 {
		stepU = 1
	}
	stepV := mesh.Nv / 5
	if stepV < 1 {
		stepV = 1
	}
	drawMeshLine := func(a, b int) {
		im.Line(vp.point(mesh.Vertices[a]), vp.point(mesh.Vertices[b]), col, 1)
	}
	for i := 0; i <= mesh.Nu; i += stepU {
		for j := 1; j <= mesh.Nv; j++ {
			drawMeshLine(mesh.Vertex(i, j-1), mesh.Vertex(i, j))
		}
	}
	for j := 0; j <= mesh.Nv; j += stepV {
		for i := 1; i <= mesh.Nu; i++ {
			drawMeshLine(mesh.Vertex(i-1, j), mesh.Vertex(i, j))
		}
	}
}

func drawIntersection(im *Image, vp viewProject, r intersect.Result) {
	for _, s := range r.Duplicates {
		im.Line(vp.point(s.A), vp.point(s.B), DuplicateColor, 7)
	}
	drawCurve := func(poly intersect.Polyline) {
		if len(poly.Points) < 2 {
			return
		}
		for i := 1; i < len(poly.Points); i++ {
			im.Line(vp.point(poly.Points[i-1]), vp.point(poly.Points[i]), ValidColor, 3)
		}
	}
	for _, c := range r.Curves {
		drawCurve(c)
	}
	if len(r.Curves) == 0 {
		for _, s := range r.Valid {
			im.Line(vp.point(s.A), vp.point(s.B), ValidColor, 3)
		}
	}
}
