package stereorollback

import (
	"encoding/json"
	"runtime"
	"time"
)

// Summary aggregates the headline counters across all named scenarios.
type Summary struct {
	ScenarioCount          int  `json:"scenario_count"`
	AllInvariantsHold      bool `json:"all_invariants_hold"`
	TotalStereoFrames      int  `json:"total_stereo_frames"`
	TotalSuccessfulFrames  int  `json:"total_successful_frames"`
	TotalPlaceholderFrames int  `json:"total_placeholder_eye_frames"`
	TotalSharedFallback    int  `json:"total_shared_fallback_eye_frames"`
	LeftFailures           int  `json:"left_eye_failures"`
	RightFailures          int  `json:"right_eye_failures"`
	SharedCorruptions      int  `json:"shared_texture_corruptions"`
}

// Report is the top-level artifact written by the demo command.
type Report struct {
	GeneratedAt string            `json:"generated_at"`
	GoVersion   string            `json:"go_version"`
	Module      string            `json:"module"`
	Scenarios   []*ScenarioReport `json:"scenarios"`
	Summary     Summary           `json:"summary"`
}

// RunAll executes the left-eye, right-eye and shared-texture scenarios.
func RunAll() (*Report, error) {
	left, err := RunLeftEyeFailure()
	if err != nil {
		return nil, err
	}
	right, err := RunRightEyeFailure()
	if err != nil {
		return nil, err
	}
	shared, err := RunSharedTextureCorruption()
	if err != nil {
		return nil, err
	}

	report := &Report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		GoVersion:   runtime.Version(),
		Module:      "stereorollback (Go standard library only)",
		Scenarios:   []*ScenarioReport{left, right, shared},
	}
	s := Summary{ScenarioCount: len(report.Scenarios), AllInvariantsHold: true}
	for _, sc := range report.Scenarios {
		s.TotalStereoFrames += sc.Counters.TotalStereoFrames
		s.TotalSuccessfulFrames += sc.Counters.SuccessfulFrames
		s.TotalPlaceholderFrames += sc.Counters.PlaceholderEyeFrames
		s.TotalSharedFallback += sc.Counters.SharedFallbackEyeFrames
		s.LeftFailures += sc.Counters.LeftFailures
		s.RightFailures += sc.Counters.RightFailures
		s.SharedCorruptions += sc.Counters.SharedCorruptions
		if !sc.Clean {
			s.AllInvariantsHold = false
		}
	}
	report.Summary = s
	return report, nil
}

// MarshalJSON returns indented JSON for the report.
func (r *Report) MarshalJSONBytes() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
