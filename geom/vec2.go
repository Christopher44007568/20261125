package geom

import "math"

// Vec2 is a 2D double-precision vector used inside a cutting plane.
type Vec2 struct {
	X, Y float64
}

func V2(x, y float64) Vec2 { return Vec2{x, y} }

func (a Vec2) Add(b Vec2) Vec2      { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2      { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Scale(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Dot(b Vec2) float64   { return a.X*b.X + a.Y*b.Y }
func (a Vec2) Cross(b Vec2) float64 { return a.X*b.Y - a.Y*b.X }
func (a Vec2) Len() float64         { return math.Hypot(a.X, a.Y) }
func (a Vec2) Len2() float64        { return a.X*a.X + a.Y*a.Y }
func (a Vec2) Neg() Vec2            { return Vec2{-a.X, -a.Y} }

func (a Vec2) Normalized() Vec2 {
	l := a.Len()
	if l < 1e-15 {
		panic("geom: normalize of near-zero 2D vector")
	}
	return a.Scale(1 / l)
}

// Distance2 returns the Euclidean distance between two 2D points.
func Distance2(a, b Vec2) float64 { return a.Sub(b).Len() }

// PolygonArea returns the signed area of a simple polygon loop.
// Positive means counter-clockwise in a right-handed 2D frame.
func PolygonArea(loop []Vec2) float64 {
	var s float64
	for i, p := range loop {
		q := loop[(i+1)%len(loop)]
		s += p.Cross(q)
	}
	return s * 0.5
}

// PointInLoop reports winding classification: +1 inside (CCW outer style),
// -1 inside a clockwise loop, 0 on the boundary or outside.
func PointInLoop(p Vec2, loop []Vec2) int {
	inside := false
	n := len(loop)
	for i := 0; i < n; i++ {
		a := loop[i]
		b := loop[(i+1)%n]
		if pointOnSegment2(p, a, b) {
			if PolygonArea(loop) >= 0 {
				return 1
			}
			return -1
		}
		if (a.Y > p.Y) != (b.Y > p.Y) {
			x := a.X + (p.Y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
			if x > p.X {
				inside = !inside
			}
		}
	}
	if !inside {
		return 0
	}
	return 1
}

func pointOnSegment2(p, a, b Vec2) bool {
	if math.Abs((b.Sub(a)).Cross(p.Sub(a))) > 1e-10 {
		return false
	}
	return math.Min(a.X, b.X)-1e-10 <= p.X && p.X <= math.Max(a.X, b.X)+1e-10 &&
		math.Min(a.Y, b.Y)-1e-10 <= p.Y && p.Y <= math.Max(a.Y, b.Y)+1e-10
}
