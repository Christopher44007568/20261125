package sim

// FrameBudget is the per-eye sampling budget recorded on a presented frame.
type FrameBudget struct {
	Size          Size `json:"size"`
	SPP           int  `json:"spp"`
	BudgetSamples int  `json:"budget_samples"`
	SpentSamples  int  `json:"spent_samples"`
}

type FrameJump struct {
	LeftMeanLuma  float64 `json:"left_mean_luma"`
	RightMeanLuma float64 `json:"right_mean_luma"`
	StereoWorst   float64 `json:"stereo_worst"`
}

// FrameRecord is one presented frame. Only presented frames are visible to
// the user; transient half-migrated states are never presented.
type FrameRecord struct {
	Index           int         `json:"index"`
	Kind            string      `json:"kind"`
	Left            FrameBudget `json:"left"`
	Right           FrameBudget `json:"right"`
	Jump            FrameJump   `json:"jump"`
	WorkUnits       int         `json:"work_units"`
	ModeledFrameMS  float64     `json:"modeled_frame_ms"`
	DeadlineMiss    bool        `json:"deadline_miss"`
	PeakMemoryBytes int         `json:"peak_memory_bytes"`
	Event           *string     `json:"event"`
}

type TrajectoryPoint struct {
	Frame                 int     `json:"frame"`
	Event                 string  `json:"event,omitempty"`
	Left                  Size    `json:"left"`
	Right                 Size    `json:"right"`
	Committed             Size    `json:"committed"`
	LeftMigratedFraction  float64 `json:"left_migrated_fraction"`
	RightMigratedFraction float64 `json:"right_migrated_fraction"`
	Consistent            bool    `json:"consistent"`
}

type EyeReport struct {
	Eye                 string      `json:"eye"`
	FinalSize           Size        `json:"final_size"`
	Remap               *RemapStats `json:"remap,omitempty"`
	ConfidenceLossRatio float64     `json:"confidence_loss_ratio"`
	JumpScores          []float64   `json:"jump_scores"`
	JumpEventFrames     []int       `json:"jump_event_frames"`
	MaxJump             float64     `json:"max_jump"`
}

type MemoryReport struct {
	BaselineBytes               int     `json:"baseline_bytes"`
	PeakBytes                   int     `json:"peak_bytes"`
	PeakOverBaseline            float64 `json:"peak_over_baseline"`
	HistoryPeakBytes            int     `json:"history_peak_bytes"`
	ImagePeakBytes              int     `json:"image_peak_bytes"`
	HistoryPeakOverTargetSteady float64 `json:"history_peak_over_target_steady"`
	FinalLiveBytes              int     `json:"final_live_bytes"`
	AllocCount                  int     `json:"alloc_count"`
	FreeCount                   int     `json:"free_count"`
}

type InvalidationReport struct {
	LeftMigrationRatio    float64 `json:"left_migration_ratio"`
	RightMigrationRatio   float64 `json:"right_migration_ratio"`
	WorstRatio            float64 `json:"worst_ratio"`
	DiscardReferenceRatio float64 `json:"discard_reference_ratio"`
}

type JumpsReport struct {
	EventThresholdLuma    float64   `json:"event_threshold_luma"`
	BaselineMedianLuma    float64   `json:"baseline_median_luma"`
	UpgradeStartFrame     int       `json:"upgrade_start_frame"`
	StereoScores          []float64 `json:"stereo_scores"`
	EventFrames           []int     `json:"event_frames"`
	EventCount            int       `json:"event_count"`
	MaxStereoJump         float64   `json:"max_stereo_jump"`
	MeanUpgradeWindowJump float64   `json:"mean_upgrade_window_jump"`
}

type WorkReport struct {
	TotalUnits     int     `json:"total_units"`
	UpgradeFrameMS float64 `json:"upgrade_frame_ms"`
	MaxFrameMS     float64 `json:"max_frame_ms"`
	DeadlineMisses int     `json:"deadline_misses"`
	FrameBudgetMS  float64 `json:"frame_budget_ms"`
}

type Check struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type ScenarioReport struct {
	Schema        string             `json:"schema"`
	ScenarioID    string             `json:"scenario_id"`
	Title         string             `json:"title"`
	Strategy      Strategy           `json:"strategy"`
	StrategyName  string             `json:"strategy_name"`
	RemapApplied  bool               `json:"remap_applied"`
	InitialSize   Size               `json:"initial_size"`
	TargetSize    Size               `json:"target_size"`
	FinalSize     Size               `json:"final_size"`
	Outcome       Outcome            `json:"outcome"`
	Fault         *FaultSpec         `json:"fault,omitempty"`
	RetryAllowed  bool               `json:"retry_allowed"`
	RetriesUsed   int                `json:"retries_used"`
	InjectedError string             `json:"injected_error,omitempty"`
	Stages        *int               `json:"stages"`
	Frames        []FrameRecord      `json:"frames"`
	Trajectory    []TrajectoryPoint  `json:"trajectory"`
	Eyes          []EyeReport        `json:"eyes"`
	Memory        MemoryReport       `json:"memory"`
	Invalidation  InvalidationReport `json:"invalidation"`
	Jumps         JumpsReport        `json:"jumps"`
	Work          WorkReport         `json:"work"`
	Checks        []Check            `json:"checks"`
	OK            bool               `json:"ok"`
}

type ComparisonRow struct {
	ScenarioID        string   `json:"scenario_id"`
	Title             string   `json:"title"`
	Strategy          Strategy `json:"strategy"`
	StrategyName      string   `json:"strategy_name"`
	Outcome           Outcome  `json:"outcome"`
	FinalSize         Size     `json:"final_size"`
	PeakOverBaseline  float64  `json:"peak_over_baseline"`
	PeakMemoryBytes   int      `json:"peak_memory_bytes"`
	HistoryPeakBytes  int      `json:"history_peak_bytes"`
	InvalidationWorst float64  `json:"invalidation_worst"`
	JumpEvents        int      `json:"jump_events"`
	MaxJump           float64  `json:"max_jump"`
	DeadlineMisses    int      `json:"deadline_misses"`
	RemapApplied      bool     `json:"remap_applied"`
	StereoConsistent  bool     `json:"stereo_consistent"`
	OK                bool     `json:"ok"`
}

type Report struct {
	Schema            string           `json:"schema"`
	JumpThresholdLuma float64          `json:"jump_threshold_luma"`
	Scenarios         []ScenarioReport `json:"scenarios"`
	Comparison        []ComparisonRow  `json:"comparison"`
}
