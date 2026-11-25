package main

import (
	"math"
)

const (
	geomEps   = 1e-9
	attachEps = 2e-6
	areaEps   = 1e-8
	pointEps  = 1e-6
	bridgeEps = 1e-6
)

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func v2(x, y float64) Vec2 { return Vec2{X: x, Y: y} }

func add2(a, b Vec2) Vec2 { return Vec2{X: a.X + b.X, Y: a.Y + b.Y} }

func sub2(a, b Vec2) Vec2 { return Vec2{X: a.X - b.X, Y: a.Y - b.Y} }

func scale2(a Vec2, s float64) Vec2 { return Vec2{X: a.X * s, Y: a.Y * s} }

func dot2(a, b Vec2) float64 { return a.X*b.X + a.Y*b.Y }

func cross2(a, b Vec2) float64 { return a.X*b.Y - a.Y*b.X }

func orient2(a, b, c Vec2) float64 {
	return cross2(sub2(b, a), sub2(c, a))
}

func dist2(a, b Vec2) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}

func dist(a, b Vec2) float64 { return math.Sqrt(dist2(a, b)) }

func approx2(a, b Vec2, eps float64) bool {
	return math.Abs(a.X-b.X) <= eps && math.Abs(a.Y-b.Y) <= eps
}

func lerp2(a, b Vec2, t float64) Vec2 {
	return Vec2{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t}
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func signedArea(poly []Vec2) float64 {
	var sum float64
	for i := range poly {
		j := (i + 1) % len(poly)
		sum += cross2(poly[i], poly[j])
	}
	return sum * 0.5
}

func ensureOrientation(poly []Vec2, wantCCW bool) []Vec2 {
	if len(poly) < 3 {
		return append([]Vec2(nil), poly...)
	}
	area := signedArea(poly)
	if (area > 0) != wantCCW {
		out := append([]Vec2(nil), poly...)
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
		return out
	}
	return append([]Vec2(nil), poly...)
}

func dedupeAdjacent(poly []Vec2, eps float64) []Vec2 {
	if len(poly) == 0 {
		return nil
	}
	out := make([]Vec2, 0, len(poly))
	for _, p := range poly {
		if len(out) == 0 || !approx2(out[len(out)-1], p, eps) {
			out = append(out, p)
		}
	}
	for len(out) >= 2 && approx2(out[0], out[len(out)-1], eps) {
		out = out[:len(out)-1]
	}
	return out
}

func removeCollinear(poly []Vec2, distanceTol float64) []Vec2 {
	poly = dedupeAdjacent(poly, attachEps)
	if len(poly) < 4 {
		return poly
	}
	changed := true
	for changed {
		changed = false
		n := len(poly)
		for i := 0; i < n; i++ {
			a := poly[(i-1+n)%n]
			b := poly[i]
			c := poly[(i+1)%n]
			diag := sub2(c, a)
			ldiag := math.Hypot(diag.X, diag.Y)
			if ldiag < geomEps {
				poly = append(poly[:i], poly[i+1:]...)
				changed = true
				break
			}
			d := math.Abs(cross2(diag, sub2(b, a))) / ldiag
			if d <= distanceTol {
				poly = append(poly[:i], poly[i+1:]...)
				changed = true
				break
			}
		}
	}
	return poly
}
