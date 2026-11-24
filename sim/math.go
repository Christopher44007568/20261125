package sim

import "math"

const (
	sceneWidth     = 640.0
	sceneHeight    = 360.0
	pixelsPerMeter = 76.0
)

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func rotate(v Vec2, angle float64) Vec2 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Vec2{X: v.X*c - v.Y*s, Y: v.X*s + v.Y*c}
}

func add(a, b Vec2) Vec2 {
	return Vec2{X: a.X + b.X, Y: a.Y + b.Y}
}

func scale(v Vec2, factor float64) Vec2 {
	return Vec2{X: v.X * factor, Y: v.Y * factor}
}

func unit2(v Vec2) Vec2 {
	length := math.Hypot(v.X, v.Y)
	if length < 1e-9 {
		return Vec2{X: 1}
	}
	return Vec2{X: v.X / length, Y: v.Y / length}
}

func screenPoint(x, y, eyeOffset float64) Vec2 {
	return Vec2{
		X: sceneWidth*0.5 + (x-eyeOffset)*pixelsPerMeter,
		Y: sceneHeight*0.5 - y*pixelsPerMeter,
	}
}

type vec3 struct {
	x float64
	y float64
	z float64
}

func (v vec3) add(other vec3) vec3 {
	return vec3{v.x + other.x, v.y + other.y, v.z + other.z}
}

func (v vec3) scale(f float64) vec3 {
	return vec3{v.x * f, v.y * f, v.z * f}
}

func (v vec3) unit() vec3 {
	length := math.Sqrt(v.x*v.x + v.y*v.y + v.z*v.z)
	if length < 1e-9 {
		return vec3{z: 1}
	}
	return vec3{v.x / length, v.y / length, v.z / length}
}

func specularNormal(light, eye, center vec3) vec3 {
	view := eye.add(center.scale(-1)).unit()
	return light.add(view).unit()
}

func gaussianEnergy(peak, stretch, thickness float64) float64 {
	area := math.Sqrt((stretch * thickness) / (72.0 * 24.0))
	return clamp(peak*area, 0, 1)
}

func reflection2(center, highlight Vec2) Vec2 {
	return unit2(Vec2{X: highlight.X - center.X, Y: highlight.Y - center.Y})
}
