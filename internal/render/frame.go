package render

import (
	"image/color"

	"handlate/internal/camera"
	"handlate/internal/sim"
	"handlate/internal/strategy"
	"handlate/internal/vec3"
)

const (
	PanelW = 250
	PanelH = 200
)

var (
	colBackground = color.RGBA{R: 18, G: 20, B: 26, A: 255}
	colPanel      = color.RGBA{R: 90, G: 100, B: 120, A: 255}
	colPanelFill  = color.RGBA{R: 48, G: 54, B: 66, A: 255}
	colCube       = color.RGBA{R: 220, G: 170, B: 70, A: 255}
	colTruth      = color.RGBA{R: 90, G: 220, B: 230, A: 255}
	colThrough    = color.RGBA{R: 240, G: 70, B: 70, A: 255}
	colOverHide   = color.RGBA{R: 240, G: 170, B: 60, A: 255}
	colText       = color.RGBA{R: 225, G: 228, B: 235, A: 255}
)

func strategyColor(mode string) color.RGBA {
	switch mode {
	case "world-sync":
		return color.RGBA{R: 120, G: 160, B: 245, A: 255}
	case "late-compose":
		return color.RGBA{R: 110, G: 230, B: 150, A: 255}
	default:
		return color.RGBA{R: 235, G: 200, B: 110, A: 255}
	}
}

// DrawPanel renders one strategy view into a canvas at origin (ox, oy).
func DrawPanel(c *Canvas, ox, oy int, scen sim.Scenario, df sim.DisplayFrame, est strategy.Estimate) {
	c.FillRect(ox, oy, ox+PanelW-1, oy+PanelH-1, colBackground)
	c.Rect(ox, oy, ox+PanelW-1, oy+PanelH-1, colText)

	head := est.Head
	if est.Points == nil {
		head = df.Ground.Head
	}

	for _, b := range scen.Occluders() {
		drawBox(c, ox, oy, head, b, colPanelFill, colPanel)
	}
	drawBox(c, ox, oy, head, df.Ground.Cube, colCube, colCube)

	// Ground truth reference: hollow cyan circles.
	for _, p := range df.Ground.Points {
		if x, y, _, ok := camera.Project(df.Ground.Head, p, PanelW, PanelH); ok {
			c.Circle(ox+int(x), oy+int(y), 5, colTruth, false)
		}
	}

	if est.Points == nil {
		c.Text(ox+8, oy+PanelH/2, "NO HAND", colThrough)
	} else {
		col := strategyColor(est.Mode)
		proj := make([][2]int, 0, len(est.Points))
		for i, p := range est.Points {
			x, y, _, ok := camera.Project(est.Head, p, PanelW, PanelH)
			if !ok {
				continue
			}
			px, py := ox+int(x), oy+int(y)
			truthHidden := camera.Occluded(df.Ground.Head, df.Ground.Points[i], scen.Occluders())
			if est.Hidden[i] {
				c.Circle(px, py, 4, colPanel, true)
				if !truthHidden {
					c.Circle(px, py, 7, colOverHide, false)
				}
			} else {
				c.Circle(px, py, 4, col, true)
				if truthHidden {
					drawX(c, px, py, colThrough)
				}
			}
			proj = append(proj, [2]int{px, py})
		}
		connectHand(c, proj, col)
	}

	c.Text(ox+6, oy+11, est.Mode, colText)
	c.Text(ox+6, oy+PanelH-8, est.Note, strategyColor(est.Mode))
	c.Text(ox+PanelW-78, oy+11, ageLabel(est), colText)
}

func drawBox(c *Canvas, ox, oy int, head vec3.Pose, b vec3.AABB, fill, edge color.Color) {
	corners := []vec3.V{
		{X: b.Min.X, Y: b.Min.Y, Z: b.Min.Z},
		{X: b.Max.X, Y: b.Min.Y, Z: b.Min.Z},
		{X: b.Max.X, Y: b.Max.Y, Z: b.Min.Z},
		{X: b.Min.X, Y: b.Max.Y, Z: b.Min.Z},
		{X: b.Min.X, Y: b.Min.Y, Z: b.Max.Z},
		{X: b.Max.X, Y: b.Min.Y, Z: b.Max.Z},
		{X: b.Max.X, Y: b.Max.Y, Z: b.Max.Z},
		{X: b.Min.X, Y: b.Max.Y, Z: b.Max.Z},
	}
	pts := make([][2]int, 0, 8)
	for _, p := range corners {
		if x, y, _, ok := camera.Project(head, p, PanelW, PanelH); ok {
			pts = append(pts, [2]int{ox + int(x), oy + int(y)})
		}
	}
	if len(pts) < 8 {
		return
	}
	edges := [][2]int{{0, 1}, {1, 2}, {2, 3}, {3, 0}, {4, 5}, {5, 6}, {6, 7}, {7, 4}, {0, 4}, {1, 5}, {2, 6}, {3, 7}}
	for _, e := range edges {
		c.Line(pts[e[0]][0], pts[e[0]][1], pts[e[1]][0], pts[e[1]][1], edge)
	}
}

func connectHand(c *Canvas, p [][2]int, col color.Color) {
	links := [][2]int{{0, 1}, {1, 2}, {2, 3}, {1, 4}, {1, 5}, {1, 6}, {6, 7}}
	for _, l := range links {
		if l[0] < len(p) && l[1] < len(p) {
			c.Line(p[l[0]][0], p[l[0]][1], p[l[1]][0], p[l[1]][1], col)
		}
	}
}

func drawX(c *Canvas, x, y int, col color.Color) {
	c.Line(x-5, y-5, x+5, y+5, col)
	c.Line(x-5, y+5, x+5, y-5, col)
}

func ageLabel(e strategy.Estimate) string {
	if e.Points == nil {
		return "--"
	}
	return itoa(e.HandAgeMs) + "/" + itoa(e.WorldAgeMs) + "MS"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
