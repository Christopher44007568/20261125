package render

import (
	"fmt"
	"math"
	"sort"

	"adaptivebezier/internal/geom"
	"adaptivebezier/internal/mesh"
)

var (
	paper      = RGB{246, 243, 235}
	panelEdge  = RGB{218, 211, 197}
	ink        = RGB{28, 42, 47}
	meshInk    = RGB{31, 74, 106}
	trimAccent = RGB{188, 70, 40}
	solidDark  = RGB{18, 77, 82}
	solidLight = RGB{144, 219, 187}
	wireDark   = RGB{169, 194, 203}
	wireLight  = RGB{235, 240, 239}
)

type PanelSpec struct {
	Title string
	Wire  bool
}

func Panel(m *mesh.Mesh, spec PanelSpec) *Canvas {
	c := NewCanvas(500, 360, paper)
	c.FillRect(0, 0, 500, 1, panelEdge)
	c.FillRect(0, 359, 500, 360, panelEdge)
	c.FillRect(0, 0, 1, 360, panelEdge)
	c.FillRect(499, 0, 500, 360, panelEdge)
	colors := vertexColors(m, spec.Wire)
	for _, t := range m.Triangles {
		a := screenOf(m.Vertices[t.A])
		b := screenOf(m.Vertices[t.B])
		d := screenOf(m.Vertices[t.C])
		c.FillTriangle(a, b, d, colors[t.A], colors[t.B], colors[t.C])
	}
	if spec.Wire {
		drawWire(c, m)
	} else {
		drawSilhouette(c, m)
	}
	caption(c, m, spec)
	return c
}

func screenOf(v mesh.Vertex) Vec2 {
	return Vec2{X: v.Screen.X, Y: v.Screen.Y}
}

func vertexColors(m *mesh.Mesh, wire bool) []RGB {
	light := geom.Vec3{X: -0.35, Y: -0.55, Z: 0.78}.Normalize()
	half := light.Add(geom.Vec3{X: 0, Y: 0, Z: 1}).Normalize()
	out := make([]RGB, len(m.Vertices))
	for i, v := range m.Vertices {
		n := v.Normal.Normalize()
		diff := math.Max(n.Dot(light), 0)
		spec := math.Pow(math.Max(n.Dot(half), 0), 22) * 0.26
		t := geom.Clamp(0.16+0.78*diff+spec, 0, 1)
		if wire {
			out[i] = mix(wireDark, wireLight, 0.18+0.82*t)
		} else {
			out[i] = mix(solidDark, solidLight, t)
		}
	}
	return out
}

func mix(a, b RGB, t float64) RGB {
	t = geom.Clamp(t, 0, 1)
	return RGB{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
	}
}

func drawWire(c *Canvas, m *mesh.Mesh) {
	seen := make(map[mesh.Edge]bool, len(m.Triangles)*3)
	for _, t := range m.Triangles {
		edges := [3][2]int{{t.A, t.B}, {t.B, t.C}, {t.C, t.A}}
		for _, e := range edges {
			a, b := e[0], e[1]
			if a > b {
				a, b = b, a
			}
			seen[mesh.Edge{A: a, B: b}] = true
		}
	}
	for _, e := range sortedEdges(seen) {
		if m.TrimEdges[e] {
			continue
		}
		lineEdge(c, m, e, meshInk, 1, 0.48)
	}
	for _, e := range sortedEdges(m.TrimEdges) {
		lineEdge(c, m, e, trimAccent, 2, 0.94)
	}
}

func drawSilhouette(c *Canvas, m *mesh.Mesh) {
	for _, e := range sortedEdges(m.TrimEdges) {
		lineEdge(c, m, e, ink, 2, 0.88)
	}
}

func sortedEdges(set map[mesh.Edge]bool) []mesh.Edge {
	out := make([]mesh.Edge, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].A != out[j].A {
			return out[i].A < out[j].A
		}
		return out[i].B < out[j].B
	})
	return out
}

func lineEdge(c *Canvas, m *mesh.Mesh, e mesh.Edge, col RGB, width int, alpha float64) {
	a, b := m.Vertices[e.A].Screen, m.Vertices[e.B].Screen
	c.Line(int(math.Round(a.X)), int(math.Round(a.Y)), int(math.Round(b.X)), int(math.Round(b.Y)), col, width, alpha)
}

func caption(c *Canvas, m *mesh.Mesh, spec PanelSpec) {
	kind := "WIRE"
	if !spec.Wire {
		kind = "SOLID"
	}
	line1 := fmt.Sprintf("%s TOL %s PX  TRIS %d", kind, spec.Title, len(m.Triangles))
	line2 := fmt.Sprintf("CHORD %s PX  CONTOUR %s PX", f3(m.MaxChord), f3(m.Contour))
	line3 := fmt.Sprintf("CRACK %s PX  CLIP %dT %dQ %dN", f3(m.Crack),
		m.Stats.ClippedTriangles, m.Stats.ClippedQuads, m.Stats.ClippedPolygons)
	DrawText(c, 13, 316, line1, 1, ink)
	DrawText(c, 13, 327, line2, 1, ink)
	DrawText(c, 13, 338, line3, 1, ink)
}

func f3(v float64) string {
	return fmt.Sprintf("%.3f", v)
}
