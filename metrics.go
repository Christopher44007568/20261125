package stereorollback

// Counters aggregates frame and eye-surface outcomes for a scenario run.
//
// SuccessfulFrames counts genuinely rendered eye-frames only. A failed-eye
// placeholder (StatusPlaceholder) and a shared fallback image never increment
// it, regardless of their Success flag.
type Counters struct {
	TotalStereoFrames       int            `json:"total_stereo_frames"`
	TotalEyeSurfaces        int            `json:"total_eye_surfaces"`
	SuccessfulFrames        int            `json:"successful_frames"`
	FullSuccessFrames       int            `json:"full_success_stereo_frames"`
	PartialFrames           int            `json:"partial_stereo_frames"`
	PlaceholderEyeFrames    int            `json:"placeholder_eye_frames"`
	SharedFallbackEyeFrames int            `json:"shared_fallback_eye_frames"`
	SharedIncidentFrames    int            `json:"shared_incident_frames"`
	LeftFailures            int            `json:"left_eye_failures"`
	RightFailures           int            `json:"right_eye_failures"`
	SharedCorruptions       int            `json:"shared_texture_corruptions"`
	EyeSuccessFrames        map[string]int `json:"eye_success_frames"`
	EyePlaceholderFrames    map[string]int `json:"eye_placeholder_frames"`
	FailuresByStage         map[string]int `json:"failures_by_stage"`
	FailuresByFault         map[string]int `json:"failures_by_fault"`
}

// NewCounters returns an initialised counter set.
func NewCounters() Counters {
	return Counters{
		EyeSuccessFrames:     map[string]int{},
		EyePlaceholderFrames: map[string]int{},
		FailuresByStage:      map[string]int{},
		FailuresByFault:      map[string]int{},
	}
}

// RecordSurface updates the success/placeholder counters from one eye surface.
// This is the single place that decides what counts as a success.
func (c *Counters) RecordSurface(s EyeSurface) {
	c.TotalEyeSurfaces++
	switch s.Status {
	case StatusRendered:
		// Defensive double condition: only a real render is a success.
		if s.Success && !s.Placeholder {
			c.SuccessfulFrames++
			c.EyeSuccessFrames[string(s.Eye)]++
		}
	case StatusPlaceholder:
		c.PlaceholderEyeFrames++
		c.EyePlaceholderFrames[string(s.Eye)]++
	case StatusSharedFallback:
		c.SharedFallbackEyeFrames++
	}
}

// RecordFrame folds a complete stereo frame into the counters.
func (c *Counters) RecordFrame(fr FrameRecord) {
	c.TotalStereoFrames++
	c.RecordSurface(fr.Left)
	c.RecordSurface(fr.Right)

	leftOK := fr.Left.Status == StatusRendered
	rightOK := fr.Right.Status == StatusRendered
	switch {
	case leftOK && rightOK:
		c.FullSuccessFrames++
	case leftOK != rightOK:
		c.PartialFrames++
	}

	failures := fr.Failures
	if len(failures) == 0 && fr.Failure != nil {
		failures = []FailureRecord{*fr.Failure}
	}
	for i := range failures {
		fail := failures[i]
		c.FailuresByStage[string(fail.FailureStage)]++
		c.FailuresByFault[string(fail.Fault)]++
		switch fail.Scope {
		case ScopeLeftEye:
			c.LeftFailures++
		case ScopeRightEye:
			c.RightFailures++
		}
	}
	if fr.SharedCorruption != nil {
		c.SharedCorruptions++
		c.SharedIncidentFrames++
	}
}

// SuccessFrames is an alias accessor for callers that prefer the verb form.
func (c Counters) SuccessFrames() int { return c.SuccessfulFrames }
