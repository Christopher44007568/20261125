package mesh

import "math"

type ProfilePoint struct {
	Z, Y float64
	NZ   float64
	NY   float64
}

func (p ProfilePoint) Vec(x float64) Vec3 { return V(x, p.Y, p.Z) }
func (p ProfilePoint) Normal() Vec3       { return V(0, p.NY, p.NZ).Norm() }

type ProfilePiece struct {
	Points []ProfilePoint
	Hint   Feature
}

func LinePiece(z0, y0, z1, y1 float64, hint Feature) ProfilePiece {
	dz, dy := z1-z0, y1-y0
	nz, ny := -dy, dz
	l := math.Hypot(nz, ny)
	if l > 0 {
		nz, ny = nz/l, ny/l
	}
	steps := 1
	if math.Hypot(dz, dy) > 24 {
		steps = int(math.Ceil(math.Hypot(dz, dy) / 18))
	}
	pts := make([]ProfilePoint, 0, steps+1)
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		pts = append(pts, ProfilePoint{Z: z0 + dz*t, Y: y0 + dy*t, NZ: nz, NY: ny})
	}
	return ProfilePiece{Points: pts, Hint: hint}
}

// ArcPiece follows the contour in increasing or decreasing angle order. The
// outward normal is the contour tangent rotated 90 degrees counter-clockwise.
func ArcPiece(cz, cy, r, a0, a1 float64, samples int, hint Feature) ProfilePiece {
	if samples < 2 {
		samples = 2
	}
	pts := make([]ProfilePoint, 0, samples)
	ccw := a1 > a0
	for i := 0; i < samples; i++ {
		t := float64(i) / float64(samples-1)
		a := a0 + (a1-a0)*t
		tz, ty := -math.Sin(a), math.Cos(a)
		if !ccw {
			tz, ty = -tz, -ty
		}
		nz, ny := -ty, tz
		pts = append(pts, ProfilePoint{
			Z: cz + r*math.Cos(a), Y: cy + r*math.Sin(a), NZ: nz, NY: ny,
		})
	}
	return ProfilePiece{Points: pts, Hint: hint}
}

func AppendPieces(pieces ...ProfilePiece) []ProfilePiece {
	out := make([]ProfilePiece, 0, len(pieces))
	for _, p := range pieces {
		if len(p.Points) == 0 {
			continue
		}
		out = append(out, p)
	}
	return out
}

func ExtrudeProfile(m *Mesh, pieces []ProfilePiece, x0, x1 float64) {
	for _, piece := range pieces {
		for i := 0; i < len(piece.Points)-1; i++ {
			a, b := piece.Points[i], piece.Points[i+1]
			na, nb := a.Normal(), b.Normal()
			want := na.Add(nb).Norm()
			m.AddQuad(
				a.Vec(x0), b.Vec(x0), b.Vec(x1), a.Vec(x1),
				na, nb, nb, na, want, piece.Hint,
			)
		}
	}

	var contour []ProfilePoint
	for _, piece := range pieces {
		for _, p := range piece.Points {
			if len(contour) > 0 {
				last := contour[len(contour)-1]
				if math.Hypot(last.Z-p.Z, last.Y-p.Y) < 1e-6 {
					continue
				}
			}
			contour = append(contour, p)
		}
	}
	if len(contour) > 2 {
		a, b := contour[0], contour[len(contour)-1]
		if math.Hypot(a.Z-b.Z, a.Y-b.Y) < 1e-6 {
			contour = contour[:len(contour)-1]
		}
	}
	tris := Triangulate2D(contour)
	for _, tri := range tris {
		for _, x := range []float64{x0, x1} {
			w := -1.0
			if x == x1 {
				w = 1.0
			}
			want := V(w, 0, 0)
			p := [3]Vec3{contour[tri[0]].Vec(x), contour[tri[1]].Vec(x), contour[tri[2]].Vec(x)}
			n := [3]Vec3{want, want, want}
			m.AddTri(p, n, want, Plane)
		}
	}
}

func Triangulate2D(points []ProfilePoint) [][3]int {
	n := len(points)
	if n < 3 {
		return nil
	}
	idx := make([]int, n)
	area := 0.0
	for i := 0; i < n; i++ {
		idx[i] = i
		j := (i + 1) % n
		area += points[i].Z*points[j].Y - points[j].Z*points[i].Y
	}
	if math.Abs(area) < 1e-9 {
		return nil
	}
	ccw := area > 0
	var out [][3]int
	for len(idx) > 3 {
		found := false
		for k := 0; k < len(idx); k++ {
			prev := idx[(k+len(idx)-1)%len(idx)]
			cur := idx[k]
			next := idx[(k+1)%len(idx)]
			a, b, c := points[prev], points[cur], points[next]
			cross := (b.Z-a.Z)*(c.Y-a.Y) - (b.Y-a.Y)*(c.Z-a.Z)
			if (ccw && cross <= 1e-9) || (!ccw && cross >= -1e-9) {
				continue
			}
			inside := false
			for _, j := range idx {
				if j == prev || j == cur || j == next {
					continue
				}
				if pointInTri(points[j], a, b, c) {
					inside = true
					break
				}
			}
			if inside {
				continue
			}
			out = append(out, [3]int{prev, cur, next})
			idx = append(idx[:k], idx[k+1:]...)
			found = true
			break
		}
		if !found {
			break
		}
	}
	if len(idx) == 3 {
		out = append(out, [3]int{idx[0], idx[1], idx[2]})
	}
	return out
}

func pointInTri(p, a, b, c ProfilePoint) bool {
	d1 := sign2(p, a, b)
	d2 := sign2(p, b, c)
	d3 := sign2(p, c, a)
	neg := d1 < 0 || d2 < 0 || d3 < 0
	pos := d1 > 0 || d2 > 0 || d3 > 0
	return !(neg && pos)
}

func sign2(p, a, b ProfilePoint) float64 {
	return (p.Z-b.Z)*(a.Y-b.Y) - (a.Z-b.Z)*(p.Y-b.Y)
}
