package sim

import "handlate/internal/vec3"

const (
	WorldTickMs   = 33 // ~30 Hz world frames
	WorldDelayMs  = 100
	HandTickMs    = 8 // ~120 Hz hand samples
	HandDelayMs   = 15
	DisplayTickMs = 16 // ~60 Hz display
)

// WorldFrame is a delayed, low-rate snapshot of the environment.
type WorldFrame struct {
	SampleTime int
	ArriveTime int
	Head       vec3.Pose
	Cube       vec3.AABB
	Occluders  []vec3.AABB
}

// HandSample is a delayed high-rate hand tracking sample.
type HandSample struct {
	SampleTime int
	ArriveTime int
	Wrist      vec3.Pose
	Grip       float64
	Tracked    bool
	Points     []vec3.V
}

// DisplayFrame is one renderer tick at time NowMs.
type DisplayFrame struct {
	NowMs    int
	Ground   GroundState
	World    *WorldFrame
	Hand     *HandSample
	PrevHand *HandSample
	SyncHand *HandSample
}

// Timeline holds the full replay for one scenario.
type Timeline struct {
	Scen     Scenario
	World    []WorldFrame
	Hands    []HandSample
	Displays []DisplayFrame
}

// Build replays the scenario once, queuing delayed streams.
func Build(scen Scenario) *Timeline {
	tl := &Timeline{Scen: scen}
	duration := scen.Duration()

	for t := 0; t <= duration; t += WorldTickMs {
		gs := scen.GroundAt(t)
		tl.World = append(tl.World, WorldFrame{
			SampleTime: t,
			ArriveTime: t + WorldDelayMs,
			Head:       gs.Head,
			Cube:       gs.Cube,
			Occluders:  scen.Occluders(),
		})
	}

	for t := 0; t <= duration; t += HandTickMs {
		gs := scen.GroundAt(t)
		tl.Hands = append(tl.Hands, HandSample{
			SampleTime: t,
			ArriveTime: t + HandDelayMs,
			Wrist:      gs.Wrist,
			Grip:       gs.Grip,
			Tracked:    gs.Tracked,
			Points:     gs.Points,
		})
	}

	for now := 0; now <= duration; now += DisplayTickMs {
		df := DisplayFrame{NowMs: now, Ground: scen.GroundAt(now)}
		for i := range tl.World {
			if tl.World[i].ArriveTime <= now {
				df.World = &tl.World[i]
			} else {
				break
			}
		}
		for i := range tl.Hands {
			if tl.Hands[i].ArriveTime <= now {
				df.PrevHand = df.Hand
				df.Hand = &tl.Hands[i]
			} else {
				break
			}
		}
		if df.World != nil {
			for i := range tl.Hands {
				if tl.Hands[i].SampleTime <= df.World.SampleTime {
					df.SyncHand = &tl.Hands[i]
				} else {
					break
				}
			}
		}
		tl.Displays = append(tl.Displays, df)
	}
	return tl
}
