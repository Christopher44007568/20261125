package strategy

import (
	"fmt"

	"handlate/internal/camera"
	"handlate/internal/sim"
	"handlate/internal/vec3"
)

const maxDeadReckonMs = 200

// Estimate is a renderer-ready hand pose for one display frame.
type Estimate struct {
	Points     []vec3.V
	Wrist      vec3.Pose
	Grip       float64
	Hidden     []bool
	Head       vec3.Pose
	Occluders  []vec3.AABB
	Mode       string
	Note       string
	HandAgeMs  int
	WorldAgeMs int
	SampleMs   int
}

type Kind int

const (
	WorldSync Kind = iota
	LateCompose
	Freeze
)

func (k Kind) Name() string {
	switch k {
	case WorldSync:
		return "world-sync"
	case LateCompose:
		return "late-compose"
	default:
		return "freeze"
	}
}

// Build runs one strategy against one display frame.
func Build(k Kind, df sim.DisplayFrame) Estimate {
	switch k {
	case WorldSync:
		return buildWorldSync(df)
	case LateCompose:
		return buildLate(df)
	default:
		return buildFreeze(df)
	}
}

func buildWorldSync(df sim.DisplayFrame) Estimate {
	e := Estimate{Mode: "world-sync"}
	if df.World == nil || df.Hand == nil {
		e.Note = "no-frame"
		return e
	}
	// Hand and camera are consumed only from the same delivered world frame:
	// newer hand samples wait for the next world update.
	wf := df.World
	paired := df.SyncHand
	if paired == nil {
		e.Note = "waiting-for-world"
		return e
	}
	e.WorldAgeMs = df.NowMs - wf.SampleTime
	e.HandAgeMs = df.NowMs - paired.SampleTime
	e.Head = wf.Head
	e.Occluders = wf.Occluders
	e.Wrist = paired.Wrist
	e.Grip = paired.Grip
	e.Points = paired.Points
	e.SampleMs = paired.SampleTime
	e.Note = fmt.Sprintf("locked@%dms", wf.SampleTime)
	e.applyDepth()
	return e
}

func latestWorld(df sim.DisplayFrame) *sim.WorldFrame { return df.World }

func buildLate(df sim.DisplayFrame) Estimate {
	e := Estimate{Mode: "late-compose"}
	wf := latestWorld(df)
	if wf == nil || df.Hand == nil {
		e.Note = "no-frame"
		return e
	}
	e.WorldAgeMs = df.NowMs - wf.SampleTime
	e.Head = wf.Head
	e.Occluders = wf.Occluders

	h := df.Hand
	e.HandAgeMs = df.NowMs - h.SampleTime
	vel := handVelocity(df.PrevHand, h)

	if h.Tracked {
		e.Wrist = h.Wrist
		e.Grip = h.Grip
		e.Points = h.Points
		e.Note = "high-rate-late"
	} else {
		lossAge := df.NowMs - h.SampleTime
		if lossAge <= maxDeadReckonMs && vel != nil {
			e.Wrist = extrapolate(h.Wrist, *vel, lossAge)
			e.Grip = h.Grip
			e.Points = sim.WorldPoints(e.Wrist, e.Grip)
			e.Note = fmt.Sprintf("dead-reckon@%dms", lossAge)
		} else {
			e.Wrist = h.Wrist
			e.Grip = h.Grip
			e.Points = h.Points
			e.Note = "dead-reckon-timeout-freeze"
		}
	}
	e.SampleMs = h.SampleTime
	e.applyDepth()
	return e
}

func buildFreeze(df sim.DisplayFrame) Estimate {
	e := Estimate{Mode: "freeze"}
	wf := latestWorld(df)
	if wf == nil || df.Hand == nil {
		e.Note = "no-frame"
		return e
	}
	e.WorldAgeMs = df.NowMs - wf.SampleTime
	e.Head = wf.Head
	e.Occluders = wf.Occluders

	h := df.Hand
	e.HandAgeMs = df.NowMs - h.SampleTime
	if h.Tracked {
		e.Wrist = h.Wrist
		e.Grip = h.Grip
		e.Points = h.Points
		e.Note = "tracking-ok"
	} else {
		e.Wrist = h.Wrist
		e.Grip = h.Grip
		e.Points = h.Points
		e.Note = "frozen-last-pose"
	}
	e.SampleMs = h.SampleTime
	e.applyDepth()
	return e
}

func handVelocity(prev, cur *sim.HandSample) *vec3.V {
	if prev == nil || cur == nil {
		return nil
	}
	dt := float64(cur.SampleTime-prev.SampleTime) / 1000.0
	if dt <= 0 {
		return nil
	}
	v := vec3.Scale(vec3.Sub(cur.Wrist.Pos, prev.Wrist.Pos), 1/dt)
	return &v
}

func extrapolate(w vec3.Pose, vel vec3.V, ageMs int) vec3.Pose {
	out := w
	out.Pos = vec3.Add(w.Pos, vec3.Scale(vel, float64(ageMs)/1000.0))
	return out
}

func (e *Estimate) applyDepth() {
	if e.Points == nil {
		return
	}
	e.Hidden = make([]bool, len(e.Points))
	for i, p := range e.Points {
		e.Hidden[i] = camera.Occluded(e.Head, p, e.Occluders)
	}
}
