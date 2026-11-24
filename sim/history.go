package sim

import "math"

const (
	historyDecay  = 0.88
	historyWeight = 0.30
)

func recordTrack(track HighlightTrack, eyeID int) trackRecord {
	return trackRecord{
		center:     track.Center,
		stretch:    track.Stretch,
		thickness:  track.Thickness,
		direction:  track.Direction,
		peak:       track.Peak,
		energy:     track.Energy,
		reflection: track.Reflection,
		eye:        eyeID,
	}
}

// resampleHistory evaluates only the prior highlight kernel at the current
// center. There is no scene transform, depth buffer, or full-frame motion
// vector involved.
func resampleHistory(current HighlightTrack, prior trackRecord, found bool, eyeID int) HistorySample {
	if !found {
		return HistorySample{
			Valid:           false,
			Center:          current.Center,
			Stretch:         current.Stretch,
			Thickness:       current.Thickness,
			Direction:       current.Direction,
			SampledPeak:     0,
			SampledEnergy:   0,
			CompositePeak:   current.Peak,
			CompositeEnergy: current.Energy,
			ResampleWeight:  0,
			SourceEye:       eyeID,
			SameEye:         true,
		}
	}

	// The eye is part of the map key. Re-checking the stored eye makes an
	// accidental stereo leak fail closed instead of reusing the wrong lobe.
	if prior.eye != eyeID {
		return HistorySample{
			Valid: false, Center: current.Center, Stretch: current.Stretch,
			Thickness: current.Thickness, Direction: current.Direction,
			CompositePeak: current.Peak, CompositeEnergy: current.Energy,
			SourceEye: prior.eye, SameEye: false,
		}
	}

	sigma := math.Max(12, math.Max(current.Stretch, prior.stretch)*0.52)
	distance := math.Hypot(
		current.Center.X-prior.center.X,
		current.Center.Y-prior.center.Y,
	)
	weight := math.Exp(-0.5 * distance * distance / (sigma * sigma))
	sampledPeak := prior.peak * weight * historyDecay
	sampledEnergy := prior.energy * weight * historyDecay

	// A convex blend is used instead of additive accumulation. The explicit
	// min ensures history can never lift a pixel above either input and the
	// outer clamp prevents repeated samples from drifting through white.
	compositePeak := clamp(
		current.Peak*(1-historyWeight)+sampledPeak*historyWeight,
		0, math.Max(current.Peak, sampledPeak),
	)
	compositeEnergy := clamp(
		current.Energy*(1-historyWeight)+sampledEnergy*historyWeight,
		0, math.Max(current.Energy, sampledEnergy),
	)

	return HistorySample{
		Valid:           true,
		Center:          prior.center,
		Stretch:         prior.stretch,
		Thickness:       prior.thickness,
		Direction:       prior.direction,
		SampledPeak:     sampledPeak,
		SampledEnergy:   sampledEnergy,
		CompositePeak:   compositePeak,
		CompositeEnergy: compositeEnergy,
		ResampleWeight:  weight,
		SourceEye:       prior.eye,
		SameEye:         true,
		EnergyCapped:    current.Energy+sampledEnergy > 1,
	}
}

func auditTimeline(eyes []EyeTimeline) Audit {
	audit := Audit{}
	weightSum := 0.0
	weightCount := 0

	for _, eye := range eyes {
		for _, frame := range eye.Frames {
			audit.Frames++
			for _, track := range frame.Tracks {
				audit.Tracks++
				audit.FullFrameTimewarps += 0
				if !track.History.SameEye || track.History.SourceEye != eye.ID {
					audit.CrossEyeSamples++
				}
				audit.MaxPeak = math.Max(audit.MaxPeak, track.Peak)
				if track.History.Valid {
					weightSum += track.History.ResampleWeight
					weightCount++
					limit := math.Max(track.Energy, track.History.SampledEnergy)
					if track.History.CompositeEnergy > limit+1e-9 ||
						track.History.CompositeEnergy > 1+1e-9 {
						audit.EnergyCapViolations++
					}
					audit.MaxCompositeEnergy = math.Max(
						audit.MaxCompositeEnergy,
						track.History.CompositeEnergy,
					)
				}
			}
		}
	}

	if weightCount > 0 {
		audit.MeanResampleWeight = weightSum / float64(weightCount)
	}
	audit.EyeDirectionDeltaMax = maxEyeDirectionDelta(eyes)
	return audit
}

func maxEyeDirectionDelta(eyes []EyeTimeline) float64 {
	if len(eyes) < 2 {
		return 0
	}
	maxDelta := 0.0
	frames := min(len(eyes[0].Frames), len(eyes[1].Frames))
	for frameIndex := 0; frameIndex < frames; frameIndex++ {
		left := eyes[0].Frames[frameIndex].Tracks
		right := eyes[1].Frames[frameIndex].Tracks
		for trackIndex := 0; trackIndex < min(len(left), len(right)); trackIndex++ {
			if left[trackIndex].ObjectID != right[trackIndex].ObjectID {
				continue
			}
			delta := angleBetween(left[trackIndex].Reflection, right[trackIndex].Reflection)
			maxDelta = math.Max(maxDelta, delta)
		}
	}
	return maxDelta
}

func angleBetween(a, b Vec2) float64 {
	dot := clamp(a.X*b.X+a.Y*b.Y, -1, 1)
	return math.Acos(dot)
}
