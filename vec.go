package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

type Vec2 struct {
	U, V float64
}

func add(a, b Vec3) Vec3 {
	return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z}
}

func sub(a, b Vec3) Vec3 {
	return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z}
}

func scale(a Vec3, s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

func dot(a, b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func cross(a, b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func length(a Vec3) float64 {
	return math.Sqrt(dot(a, a))
}

func normalize(a Vec3) Vec3 {
	n := length(a)
	if n < 1e-15 {
		return Vec3{}
	}
	return scale(a, 1/n)
}

func distance(a, b Vec3) float64 {
	return length(sub(a, b))
}

func closestOnSegment(p, a, b Vec3) (Vec3, float64) {
	ab := sub(b, a)
	den := dot(ab, ab)
	t := 0.0
	if den > 1e-18 {
		t = dot(sub(p, a), ab) / den
	}
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	q := add(a, scale(ab, t))
	return q, distance(p, q)
}

func pointTriangleDistance(p, a, b, c Vec3) (float64, Vec3) {
	ab := sub(b, a)
	ac := sub(c, a)
	ap := sub(p, a)
	normal := cross(ab, ac)
	normalLengthSquared := dot(normal, normal)
	if normalLengthSquared < 1e-18 {
		q1, d1 := closestOnSegment(p, a, b)
		q2, d2 := closestOnSegment(p, b, c)
		q3, d3 := closestOnSegment(p, c, a)
		if d2 < d1 {
			d1, q1 = d2, q2
		}
		if d3 < d1 {
			d1, q1 = d3, q3
		}
		return d1, q1
	}

	d00 := dot(ab, ab)
	d01 := dot(ab, ac)
	d11 := dot(ac, ac)
	d20 := dot(ap, ab)
	d21 := dot(ap, ac)
	denom := d00*d11 - d01*d01
	v := (d11*d20 - d01*d21) / denom
	w := (d00*d21 - d01*d20) / denom
	u := 1 - v - w
	if u >= -1e-10 && v >= -1e-10 && w >= -1e-10 {
		q := add(a, add(scale(ab, v), scale(ac, w)))
		return distance(p, q), q
	}

	q1, d1 := closestOnSegment(p, a, b)
	q2, d2 := closestOnSegment(p, b, c)
	q3, d3 := closestOnSegment(p, c, a)
	if d2 < d1 {
		d1, q1 = d2, q2
	}
	if d3 < d1 {
		d1, q1 = d3, q3
	}
	return d1, q1
}
