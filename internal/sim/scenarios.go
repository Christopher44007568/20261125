package sim

import (
	"math"

	"handlate/internal/vec3"
)

// WaveScenario: fast lateral wave crossing an occluder, grip stays open.
type WaveScenario struct{}

func (WaveScenario) ID() string             { return "wave" }
func (WaveScenario) Title() string          { return "Fast wave" }
func (WaveScenario) Duration() int          { return 4000 }
func (WaveScenario) Occluders() []vec3.AABB { return commonPanels() }

func (WaveScenario) GroundAt(t int) GroundState {
	s := float64(t) / 1000.0
	amp := 0.55 + 0.08*math.Sin(0.7*s)
	wrist := vec3.Pose{
		Pos: vec3.V{
			X: amp * math.Sin(2*math.Pi*1.1*s),
			Y: 1.40 + 0.08*math.Sin(2*math.Pi*0.55*s),
			Z: 1.10 + 0.12*math.Cos(2*math.Pi*1.1*s),
		},
		Yaw: 0.4 * math.Cos(2*math.Pi*1.1*s),
	}
	return groundAt(t, wrist, 0, true, restCube)
}

func (WaveScenario) MeasureContact(GroundState) bool     { return false }
func (WaveScenario) ContactTarget(gs GroundState) vec3.V { return gs.Pinch }

// GrabScenario: approach the cube, pinch closed, lift and release.
type GrabScenario struct{}

func (GrabScenario) ID() string             { return "grab" }
func (GrabScenario) Title() string          { return "Grab and lift" }
func (GrabScenario) Duration() int          { return 5000 }
func (GrabScenario) Occluders() []vec3.AABB { return commonPanels() }

func (GrabScenario) GroundAt(t int) GroundState {
	s := float64(t) / 1000.0
	cubeCenter := restCube.Min
	cubeCenter = vec3.Add(cubeCenter, vec3.V{X: cubeHalf, Y: cubeHalf, Z: cubeHalf})

	var pos vec3.V
	var grip float64
	switch {
	case t < 800:
		k := smoothStep(0, 0.8, s)
		pos = vec3.Lerp(vec3.V{X: -0.35, Y: 1.05, Z: 0.95}, cubeCenter, k)
	case t < 1400:
		pos = cubeCenter
		grip = clamp01(float64(t-800) / 400.0)
	case t < 3200:
		k := smoothStep(1.4, 3.2, s)
		pos = vec3.Lerp(cubeCenter, vec3.V{X: 0.12, Y: 1.68, Z: 1.52}, k)
		grip = 1
	case t < 3800:
		pos = vec3.V{X: 0.12, Y: 1.68, Z: 1.52}
		grip = 1 - clamp01(float64(t-3200)/500.0)
	default:
		k := smoothStep(3.8, 5.0, s)
		pos = vec3.Lerp(vec3.V{X: 0.12, Y: 1.68, Z: 1.52}, vec3.V{X: 0.30, Y: 1.25, Z: 1.10}, k)
	}
	wrist := vec3.Pose{Pos: pos, Pitch: -0.15}
	return groundAt(t, wrist, grip, true, restCube)
}

func (GrabScenario) MeasureContact(gs GroundState) bool {
	return gs.Time >= 800 && gs.Time <= 1600
}

func (GrabScenario) ContactTarget(gs GroundState) vec3.V {
	return nearestCubePoint(gs.Pinch, gs.Cube)
}

// LossScenario: tracking drops for 600 ms while the hand keeps moving.
type LossScenario struct{}

func (LossScenario) ID() string             { return "loss" }
func (LossScenario) Title() string          { return "Tracking loss" }
func (LossScenario) Duration() int          { return 5000 }
func (LossScenario) Occluders() []vec3.AABB { return commonPanels() }

func (LossScenario) GroundAt(t int) GroundState {
	s := float64(t) / 1000.0
	wrist := vec3.Pose{
		Pos: vec3.V{
			X: -0.45 + 0.9*s - 0.05*math.Sin(2*math.Pi*s),
			Y: 1.30 + 0.12*math.Sin(2*math.Pi*0.5*s),
			Z: 1.05 + 0.12*s,
		},
		Yaw: 0.35 * math.Sin(2*math.Pi*0.4*s),
	}
	tracked := t < 1800 || t >= 2400
	return groundAt(t, wrist, 0.3, tracked, restCube)
}

func (LossScenario) MeasureContact(gs GroundState) bool {
	return vec3.Dist(gs.Pinch, nearestCubePoint(gs.Pinch, gs.Cube)) < 0.10
}

func (LossScenario) ContactTarget(gs GroundState) vec3.V {
	return nearestCubePoint(gs.Pinch, gs.Cube)
}

func cubeCenter(b vec3.AABB) vec3.V {
	return vec3.Scale(vec3.Add(b.Min, b.Max), 0.5)
}

func nearestCubePoint(p vec3.V, b vec3.AABB) vec3.V {
	clamp := func(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }
	return vec3.V{
		X: clamp(p.X, b.Min.X, b.Max.X),
		Y: clamp(p.Y, b.Min.Y, b.Max.Y),
		Z: clamp(p.Z, b.Min.Z, b.Max.Z),
	}
}
