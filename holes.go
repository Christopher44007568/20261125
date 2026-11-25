package main

import (
	"math"
	"sort"
)

type HoleCell struct {
	IX, IY int
	X, Y   float64
	W, H   float64
}

type HoleComponent struct {
	Cells      []HoleCell
	MinX, MaxX float64
	MinY, MaxY float64
	CenterX    float64
	CenterY    float64
}

type HoleField struct {
	NX, NY       int
	MinX, MaxX   float64
	MinY, MaxY   float64
	CellW, CellH float64
	Hole         []bool
	Components   []HoleComponent
}

func (h *HoleField) at(ix, iy int) bool {
	if ix < 0 || iy < 0 || ix >= h.NX || iy >= h.NY {
		return false
	}
	return h.Hole[iy*h.NX+ix]
}

// OcclusionMask returns the scanner's three occlusion masks. These masks are
// display metadata only; the surface fit never receives them.
func OcclusionMask(points []ScanPoint) HoleField {
	nx, ny := 52, 34
	h := HoleField{
		NX: nx, NY: ny,
		MinX: FitXMin, MaxX: FitXMax, MinY: FitYMin, MaxY: FitYMax,
		CellW: (FitXMax - FitXMin) / float64(nx),
		CellH: (FitYMax - FitYMin) / float64(ny),
		Hole:  make([]bool, nx*ny),
	}
	for iy := 0; iy < ny; iy++ {
		for ix := 0; ix < nx; ix++ {
			cx := h.MinX + (float64(ix)+0.5)*h.CellW
			cy := h.MinY + (float64(iy)+0.5)*h.CellH
			if inScanHole(cx/PatchA, cy/PatchB) {
				h.Hole[iy*nx+ix] = true
			}
		}
	}
	h.Components = connectedHoleComponents(&h)
	for i := range points {
		for _, c := range h.Components {
			if math.Abs(points[i].X-c.CenterX) < (c.MaxX-c.MinX)*0.5+7 &&
				math.Abs(points[i].Y-c.CenterY) < (c.MaxY-c.MinY)*0.5+7 {
				points[i].HoleAdjacent = true
				break
			}
		}
	}
	return h
}

func connectedHoleComponents(h *HoleField) []HoleComponent {
	seen := make([]bool, len(h.Hole))
	var out []HoleComponent
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for iy := 0; iy < h.NY; iy++ {
		for ix := 0; ix < h.NX; ix++ {
			start := iy*h.NX + ix
			if !h.Hole[start] || seen[start] {
				continue
			}
			queue := []int{start}
			seen[start] = true
			comp := HoleComponent{MinX: math.MaxFloat64, MinY: math.MaxFloat64}
			for len(queue) > 0 {
				id := queue[0]
				queue = queue[1:]
				cx, cy := id%h.NX, id/h.NX
				cell := HoleCell{
					IX: cx, IY: cy,
					X: h.MinX + (float64(cx)+0.5)*h.CellW,
					Y: h.MinY + (float64(cy)+0.5)*h.CellH,
					W: h.CellW * 0.94, H: h.CellH * 0.94,
				}
				comp.Cells = append(comp.Cells, cell)
				comp.MinX = math.Min(comp.MinX, cell.X)
				comp.MaxX = math.Max(comp.MaxX, cell.X)
				comp.MinY = math.Min(comp.MinY, cell.Y)
				comp.MaxY = math.Max(comp.MaxY, cell.Y)
				for _, d := range dirs {
					nx, ny := cx+d[0], cy+d[1]
					nid := ny*h.NX + nx
					if h.at(nx, ny) && !seen[nid] {
						seen[nid] = true
						queue = append(queue, nid)
					}
				}
			}
			comp.CenterX = (comp.MinX + comp.MaxX) * 0.5
			comp.CenterY = (comp.MinY + comp.MaxY) * 0.5
			out = append(out, comp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return len(out[i].Cells) > len(out[j].Cells)
	})
	return out
}

func (h *HoleField) IsHole(x, y float64) bool {
	ix := int((x - h.MinX) / h.CellW)
	iy := int((y - h.MinY) / h.CellH)
	return h.at(ix, iy)
}

func EstimateTrim(points []ScanPoint) (a, b float64) {
	a, b = PatchA, PatchB
	maxX, maxY := 0.0, 0.0
	for _, p := range points {
		if p.Kind != PointClean || p.Rejected {
			continue
		}
		maxX = math.Max(maxX, math.Abs(p.X))
		maxY = math.Max(maxY, math.Abs(p.Y))
	}
	if maxX > 0 {
		a = maxX / 0.992
	}
	if maxY > 0 {
		b = maxY / 0.992
	}
	return
}
