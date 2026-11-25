package hair

import "math"

type Selection struct {
	LevelA          int     `json:"levelA"`
	LevelB          int     `json:"levelB"`
	Blend           float64 `json:"blend"`
	ContinuousIndex float64 `json:"continuousIndex"`
	EffectiveCount  float64 `json:"effectiveCount"`
}

func (a *Asset) Select(distance float64) Selection {
	if len(a.Levels) == 1 {
		return Selection{LevelA: 0, LevelB: 0}
	}
	if distance < a.Camera.FocusDistance {
		distance = a.Camera.FocusDistance
	}
	last := len(a.Levels) - 1
	q := math.Log2(distance / a.Camera.FocusDistance)
	q = clamp(q, 0, float64(last))
	aLevel := int(math.Floor(q))
	bLevel := aLevel
	blend := 0.0
	if aLevel < last {
		bLevel = aLevel + 1
		blend = q - float64(aLevel)
	}
	ca := float64(len(a.Levels[aLevel].Entries))
	cb := float64(len(a.Levels[bLevel].Entries))
	return Selection{
		LevelA:          aLevel,
		LevelB:          bLevel,
		Blend:           blend,
		ContinuousIndex: q,
		EffectiveCount:  lerp(ca, cb, blend),
	}
}

func (a *Asset) LevelDistance(index int) float64 {
	return a.Camera.FocusDistance * math.Pow(2, float64(index))
}

func (a *Asset) DefaultErrorDistance() float64 {
	if len(a.Levels) < 2 {
		return a.Camera.FocusDistance
	}
	return a.LevelDistance(len(a.Levels) - 1)
}
