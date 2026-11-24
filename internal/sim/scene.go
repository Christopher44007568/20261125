package sim

import (
	"math"

	"handlate/internal/vec3"
)

// GroundState is the true state of the world at one millisecond tick.
type GroundState struct {
	Time    int
	Head    vec3.Pose
	Wrist   vec3.Pose
	Grip    float64
	Tracked bool
	Cube    vec3.AABB
	Points  []vec3.V
	Pinch   vec3.V
}

// Scenario is one reproducible motion test.
type Scenario interface {
	ID() string
	Title() string
	Duration() int
	Occluders() []vec3.AABB
	GroundAt(t int) GroundState
	MeasureContact(gs GroundState) bool
	ContactTarget(gs GroundState) vec3.V
}

const cubeHalf = 0.06

var (
	panelA   = vec3.NewAABB(vec3.V{X: 0.25, Y: 1.45, Z: 1.20}, vec3.V{X: 0.22, Y: 0.30, Z: 0.03})
	panelB   = vec3.NewAABB(vec3.V{X: -0.55, Y: 1.45, Z: 1.10}, vec3.V{X: 0.15, Y: 0.30, Z: 0.03})
	restCube = vec3.NewAABB(vec3.V{X: 0.0, Y: 1.30, Z: 1.50}, vec3.V{X: cubeHalf, Y: cubeHalf, Z: cubeHalf})
)

func commonPanels() []vec3.AABB { return []vec3.AABB{panelA, panelB} }

// headAt keeps the camera off the world origin with mild natural sway.
func headAt(t int) vec3.Pose {
	s := float64(t) / 1000.0
	return vec3.Pose{
		Pos: vec3.V{
			X: 0.04 * math.Sin(0.8*s),
			Y: 1.55 + 0.01*math.Sin(0.5*s),
			Z: 0.05 * math.Cos(0.7*s),
		},
		Yaw:   0.05 * math.Sin(0.6*s),
		Pitch: 0.02 * math.Cos(0.9*s),
	}
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(1, x)) }

func smoothStep(a, b, s float64) float64 {
	t := clamp01((s - a) / (b - a))
	return t * t * (3 - 2*t)
}

func groundAt(t int, wrist vec3.Pose, grip float64, tracked bool, cube vec3.AABB) GroundState {
	pts := WorldPoints(wrist, grip)
	return GroundState{
		Time:    t,
		Head:    headAt(t),
		Wrist:   wrist,
		Grip:    grip,
		Tracked: tracked,
		Cube:    cube,
		Points:  pts,
		Pinch:   PinchPoint(pts),
	}
}
