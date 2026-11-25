package main

// Signals are the per-frame measurements fed into the controller.
type Signals struct {
	FrameTimeMS   float64 `json:"frame_time_ms"`
	FrustumFailed bool    `json:"frustum_failed"`
	FrustumCode   string  `json:"frustum_code,omitempty"`
	PoseCount     int     `json:"pose_count"`
	PoseInvalid   bool    `json:"pose_invalid"`
}

// State is the effective capability state after processing a frame.
type State struct {
	EyeCount      int     `json:"eye_count"`
	Stereo        bool    `json:"stereo"`
	SampleLevel   int     `json:"sample_level"`
	SamplingScale float64 `json:"sampling_scale"`
	Reprojection  bool    `json:"reprojection"`
	History       bool    `json:"history"`
	Interaction   bool    `json:"interaction"`
	Overlay       bool    `json:"overlay"`
}

func nominalState(scales []float64) State {
	return State{
		EyeCount: 2, Stereo: true, SampleLevel: 0, SamplingScale: scales[0],
		Reprojection: true, History: true, Interaction: true, Overlay: true,
	}
}

// Metrics is the scored / measured view of a frame.
type Metrics struct {
	Frame            int     `json:"frame"`
	FrameTimeMS      float64 `json:"frame_time_ms"`
	BudgetMS         float64 `json:"budget_ms"`
	FPS              float64 `json:"fps"`
	BudgetHeadroomMS float64 `json:"budget_headroom_ms"`
	EyeCount         int     `json:"eye_count"`
	Stereo           bool    `json:"stereo"`
	SampleLevel      int     `json:"sample_level"`
	SamplingScale    float64 `json:"sampling_scale"`
	Reprojection     bool    `json:"reprojection"`
	HistoryCapture   bool    `json:"history_capture"`
	Interaction      bool    `json:"interaction"`
	RelativeLoad     float64 `json:"relative_load"`
	QualityScore     float64 `json:"quality_score"`
}

// RecoveryProgress is the structured hold counter shown in the status panel.
type RecoveryProgress struct {
	Kind        string   `json:"kind"`
	Label       string   `json:"label"`
	Streak      int      `json:"streak"`
	Need        int      `json:"need"`
	ThresholdMS *float64 `json:"threshold_ms,omitempty"`
}

// StatusDetail carries the human readable status block.
type StatusDetail struct {
	Level            string             `json:"level"`
	Headline         string             `json:"headline"`
	Lines            []string           `json:"lines"`
	ActiveModes      []string           `json:"active_modes"`
	ChainID          string             `json:"chain_id,omitempty"`
	EpisodeIDs       []string           `json:"episode_ids"`
	RecoveryProgress []RecoveryProgress `json:"recovery_progress,omitempty"`
	InteractionHint  string             `json:"interaction_hint"`
	FloorWarning     bool               `json:"floor_warning"`
}

// FrameReport is one fully recorded frame.
type FrameReport struct {
	Scenario     string              `json:"scenario"`
	Frame        int                 `json:"frame"`
	Signals      Signals             `json:"signals"`
	State        State               `json:"state"`
	Metrics      Metrics             `json:"metrics"`
	Actions      []FrameActionRecord `json:"actions,omitempty"`
	StatusLevel  string              `json:"status_level"`
	Headline     string              `json:"headline"`
	StatusDetail StatusDetail        `json:"status_detail"`
	ChainID      string              `json:"chain_id,omitempty"`
}

// FrameActionRecord is a remediation action attached to a frame.
type FrameActionRecord struct {
	Kind    string `json:"kind"`
	Keep    int    `json:"keep,omitempty"`
	Message string `json:"message"`
}

// Alert is an unconfirmed budget warning; it can end as confirmed or false positive.
type Alert struct {
	ID          string  `json:"id"`
	Trigger     string  `json:"trigger"`
	ReasonCode  string  `json:"reason_code"`
	Detail      string  `json:"detail"`
	Frame       int     `json:"frame"`
	CloseFrame  int     `json:"close_frame"`
	Verdict     string  `json:"verdict"`
	SignalValue float64 `json:"signal_value"`
}

// AuditEntry is an append-only timeline event.
type AuditEntry struct {
	Frame     int    `json:"frame"`
	Level     string `json:"level"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	EpisodeID string `json:"episode_id,omitempty"`
	ChainID   string `json:"chain_id,omitempty"`
}

// SafetyEvent records a rejected unsafe request.
type SafetyEvent struct {
	Frame   int    `json:"frame"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Actor   string `json:"actor"`
}

// Shot marks which frame the rendered PNG was taken from.
type Shot struct {
	Frame   int    `json:"frame"`
	File    string `json:"file"`
	Caption string `json:"caption"`
}
