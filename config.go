package main

// Config holds every threshold / confirmation window used by the
// degradation controller. All values are part of the machine-readable
// report so a run is fully auditable.
type Config struct {
	BudgetMS                float64   `json:"budget_ms"`
	BudgetConfirmFrames     int       `json:"budget_confirm_frames"`
	SevereRatio             float64   `json:"severe_ratio"`
	SampleScales            []float64 `json:"sample_scales"`
	LevelDownFrames         int       `json:"level_down_frames"`
	EscalateMonoFrames      int       `json:"escalate_mono_frames"`
	RecoveryRatio           float64   `json:"recovery_ratio"`
	RecoverHoldFrames       int       `json:"recover_hold_frames"`
	StereoHeadroomRatio     float64   `json:"stereo_headroom_ratio"`
	StereoRecoverHoldFrames int       `json:"stereo_recover_hold_frames"`
	FrustumRecoverHold      int       `json:"frustum_recover_hold"`
	PoseLimit               int       `json:"pose_limit"`
	PoseRecoverHold         int       `json:"pose_recover_hold"`
	ChainGapFrames          int       `json:"chain_gap_frames"`
}

// DefaultConfig returns the tuned values from the reference audit:
// 11.11 ms budget (~90 Hz), two-frame confirmation, pose cap 16.
func DefaultConfig() Config {
	return Config{
		BudgetMS:                11.11,
		BudgetConfirmFrames:     2,
		SevereRatio:             1.45,
		SampleScales:            []float64{1, 0.75, 0.5},
		LevelDownFrames:         2,
		EscalateMonoFrames:      2,
		RecoveryRatio:           0.8,
		RecoverHoldFrames:       3,
		StereoHeadroomRatio:     0.7,
		StereoRecoverHoldFrames: 4,
		FrustumRecoverHold:      2,
		PoseLimit:               16,
		PoseRecoverHold:         2,
		ChainGapFrames:          6,
	}
}

// RecoveryThresholdMS is the budget headroom bar for raising sampling.
// The reference report records it rounded to three decimals (8.888).
func (c Config) RecoveryThresholdMS() float64 {
	return round3(c.BudgetMS * c.RecoveryRatio)
}

// StereoThresholdMS is the stricter headroom bar for restoring stereo.
// It intentionally keeps the raw float product in the report.
func (c Config) StereoThresholdMS() float64 {
	return c.BudgetMS * c.StereoHeadroomRatio
}

// SevereThresholdMS marks a frame severe enough to skip confirmation.
func (c Config) SevereThresholdMS() float64 {
	return c.BudgetMS * c.SevereRatio
}

func round3(v float64) float64 {
	return float64(int64(v*1000+0.5)) / 1000
}
