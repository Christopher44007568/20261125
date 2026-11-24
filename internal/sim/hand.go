package sim

import (
	"math"

	"handlate/internal/vec3"
)

// HandPoint is one tracked point of the hand in wrist-local space.
type HandPoint struct {
	Name   string
	Open   vec3.V
	Closed vec3.V
}

// HandModel is a simplified 8-point hand. +Z is the pointing direction.
var HandModel = []HandPoint{
	{"wrist", vec3.V{X: 0, Y: 0, Z: 0}, vec3.V{X: 0, Y: 0, Z: 0}},
	{"palm", vec3.V{X: 0, Y: 0, Z: 0.05}, vec3.V{X: 0, Y: 0, Z: 0.05}},
	{"index_knuckle", vec3.V{X: 0.03, Y: 0, Z: 0.09}, vec3.V{X: 0.025, Y: 0, Z: 0.085}},
	{"index_tip", vec3.V{X: 0.03, Y: 0, Z: 0.16}, vec3.V{X: 0.02, Y: 0, Z: 0.085}},
	{"middle_tip", vec3.V{X: 0.01, Y: 0, Z: 0.16}, vec3.V{X: 0.005, Y: 0, Z: 0.082}},
	{"ring_tip", vec3.V{X: -0.01, Y: 0, Z: 0.15}, vec3.V{X: -0.01, Y: 0, Z: 0.08}},
	{"thumb_knuckle", vec3.V{X: 0.05, Y: -0.02, Z: 0.04}, vec3.V{X: 0.035, Y: -0.01, Z: 0.06}},
	{"thumb_tip", vec3.V{X: 0.06, Y: 0, Z: 0.10}, vec3.V{X: 0.03, Y: 0, Z: 0.09}},
}

const (
	indexTipIdx = 3
	thumbTipIdx = 7
)

// LocalPoints applies grip in [0,1]: 0 open, 1 fully closed.
func LocalPoints(grip float64) []vec3.V {
	g := math.Max(0, math.Min(1, grip))
	out := make([]vec3.V, len(HandModel))
	for i, hp := range HandModel {
		out[i] = vec3.Lerp(hp.Open, hp.Closed, g)
	}
	return out
}

// WorldPoints poses the hand in world space.
func WorldPoints(wrist vec3.Pose, grip float64) []vec3.V {
	local := LocalPoints(grip)
	out := make([]vec3.V, len(local))
	for i, p := range local {
		out[i] = wrist.Transform(p)
	}
	return out
}

// PinchPoint is the midpoint between thumb and index tips, in world space.
func PinchPoint(points []vec3.V) vec3.V {
	return vec3.Lerp(points[indexTipIdx], points[thumbTipIdx], 0.5)
}
