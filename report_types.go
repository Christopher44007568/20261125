package main

// RecoveryCondition describes how a degradation episode can end.
type RecoveryCondition struct {
	Kind        string  `json:"kind"`
	Description string  `json:"description"`
	Metric      string  `json:"metric,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
	HoldFrames  int     `json:"hold_frames"`
}

// LevelChange is one sampling gear transition.
type LevelChange struct {
	Frame     int     `json:"frame"`
	FromLevel int     `json:"from_level"`
	ToLevel   int     `json:"to_level"`
	FromScale float64 `json:"from_scale"`
	ToScale   float64 `json:"to_scale"`
	Reason    string  `json:"reason"`
}

// Episode is one continuous degradation segment for a single mode.
type Episode struct {
	ID                string            `json:"id"`
	ChainID           string            `json:"chain_id"`
	Mode              string            `json:"mode"`
	Trigger           string            `json:"trigger"`
	ReasonCode        string            `json:"reason_code"`
	ReasonDetail      string            `json:"reason_detail"`
	StartFrame        int               `json:"start_frame"`
	EndFrame          int               `json:"end_frame"`
	DurationFrames    int               `json:"duration_frames"`
	RecoveryCondition RecoveryCondition `json:"recovery_condition"`
	RecoveryMetFrame  *int              `json:"recovery_met_frame,omitempty"`
	Outcome           string            `json:"outcome"`
	Continuous        bool              `json:"continuous"`
	FalsePositive     bool              `json:"false_positive"`
	FalsePositiveNote string            `json:"false_positive_note,omitempty"`
	InvalidatedFrame  int               `json:"invalidated_frame,omitempty"`
	StartMetrics      *Metrics          `json:"start_metrics"`
	WorstMetrics      *Metrics          `json:"worst_metrics"`
	EndMetrics        *Metrics          `json:"end_metrics,omitempty"`
	Samples           []Metrics         `json:"samples"`
	LevelChanges      []LevelChange     `json:"level_changes,omitempty"`
}

// Escalation records a degradation getting worse (sampling floor -> monocular).
type Escalation struct {
	Frame  int    `json:"frame"`
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
}

// PartialRecovery marks one capability returning while others stay degraded.
type PartialRecovery struct {
	Frame         int      `json:"frame"`
	Recovered     []string `json:"recovered"`
	StillDegraded []string `json:"still_degraded"`
	Detail        string   `json:"detail"`
}

// Chain groups episodes that belong to one continuous degradation story.
type Chain struct {
	ID                 string            `json:"id"`
	StartFrame         int               `json:"start_frame"`
	EndFrame           int               `json:"end_frame"`
	DurationFrames     int               `json:"duration_frames"`
	EpisodeIDs         []string          `json:"episode_ids"`
	Escalations        []Escalation      `json:"escalations,omitempty"`
	PartialRecoveries  []PartialRecovery `json:"partial_recoveries,omitempty"`
	Reopened           bool              `json:"reopened"`
	FullRecovery       bool              `json:"full_recovery"`
	FullRecoveryFrame  int               `json:"full_recovery_frame,omitempty"`
	HadPartialRecovery bool              `json:"had_partial_recovery"`
	FalsePositive      bool              `json:"false_positive"`
	Classification     string            `json:"classification"`
}

// ScenarioReport is the full record for one scripted scenario.
type ScenarioReport struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Frames       []*FrameReport `json:"frames"`
	Episodes     []*Episode     `json:"episodes"`
	Chains       []*Chain       `json:"chains"`
	Alerts       []*Alert       `json:"alerts"`
	Audit        []*AuditEntry  `json:"audit"`
	SafetyEvents []*SafetyEvent `json:"safety_events"`
	Shots        []Shot         `json:"shots"`
}

// Summary reference item shapes.

type FPAlertRef struct {
	ScenarioID string `json:"scenario_id"`
	EpisodeID  string `json:"episode_id"`
	Frame      int    `json:"frame"`
	Detail     string `json:"detail"`
}

type FPEpisodeRef struct {
	ScenarioID string `json:"scenario_id"`
	ChainID    string `json:"chain_id"`
	EpisodeID  string `json:"episode_id"`
	Frame      int    `json:"frame"`
	Detail     string `json:"detail"`
}

type ChainRef struct {
	ScenarioID string `json:"scenario_id"`
	ChainID    string `json:"chain_id"`
	Detail     string `json:"detail"`
}

type PartialRef struct {
	ScenarioID string `json:"scenario_id"`
	ChainID    string `json:"chain_id"`
	Frame      int    `json:"frame"`
	Detail     string `json:"detail"`
}

type InvariantCheck struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Evidence string `json:"evidence"`
}

type Summary struct {
	ScenarioCount           int              `json:"scenario_count"`
	EpisodeCount            int              `json:"episode_count"`
	ChainCount              int              `json:"chain_count"`
	FalsePositiveAlerts     []FPAlertRef     `json:"false_positive_alerts"`
	FalsePositiveEpisodes   []FPEpisodeRef   `json:"false_positive_episodes"`
	ContinuousChains        []ChainRef       `json:"continuous_chains"`
	PartialRecoveries       []PartialRef     `json:"partial_recoveries"`
	IsolatedRecoveredChains []ChainRef       `json:"isolated_recovered_chains"`
	SafetyEvents            int              `json:"safety_events"`
	InvariantChecks         []InvariantCheck `json:"invariant_checks"`
}

// Report is the top level degrade_report.json document.
type Report struct {
	GeneratedAt string            `json:"generated_at"`
	GoVersion   string            `json:"go_version"`
	Stack       string            `json:"stack"`
	Config      Config            `json:"config"`
	Summary     Summary           `json:"summary"`
	Scenarios   []*ScenarioReport `json:"scenarios"`
}
