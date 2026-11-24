package eyestage

import "time"

const (
	EyeLeft  = "left"
	EyeRight = "right"

	StageCaptureScene = "capture_scene"
	StageReprojection = "reprojection"
	StageDistortion   = "distortion"
	StageComposition  = "composition"
)

var (
	Eyes   = []string{EyeLeft, EyeRight}
	Stages = []string{StageCaptureScene, StageReprojection, StageDistortion, StageComposition}
)

type Timestamp struct {
	NS     int64  `json:"ns"`
	Source string `json:"source"`
	Raw    string `json:"raw,omitempty"`
}

type StageTiming struct {
	Stage          string     `json:"stage"`
	Start          *Timestamp `json:"start,omitempty"`
	End            *Timestamp `json:"end,omitempty"`
	DurationNS     *int64     `json:"duration_ns,omitempty"`
	DurationSource string     `json:"duration_source,omitempty"`
}

type EyeTiming struct {
	Eye    string                  `json:"eye"`
	Stages map[string]*StageTiming `json:"stages"`
}

type Frame struct {
	ID   string                `json:"id"`
	Eyes map[string]*EyeTiming `json:"eyes"`
}

type Diagnostic struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	Frame          string `json:"frame,omitempty"`
	Eye            string `json:"eye,omitempty"`
	Stage          string `json:"stage,omitempty"`
	Field          string `json:"field,omitempty"`
	Raw            string `json:"raw,omitempty"`
	FallbackSource string `json:"fallback_source,omitempty"`
}

type Dataset struct {
	Frames      []*Frame     `json:"frames"`
	Unit        string       `json:"unit,omitempty"`
	UnitSource  string       `json:"unit_source,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type Config struct {
	TailPercentile   float64       `json:"tail_percentile"`
	HistogramBuckets int           `json:"histogram_buckets"`
	MinIdleWait      time.Duration `json:"min_idle_wait_ns"`
}

func DefaultConfig() Config {
	return Config{
		TailPercentile:   95,
		HistogramBuckets: 12,
		MinIdleWait:      0,
	}
}

type HistogramBucket struct {
	LowerNS int64 `json:"lower_ns"`
	UpperNS int64 `json:"upper_ns"`
	Count   int   `json:"count"`
}

type StageStat struct {
	Eye        string            `json:"eye"`
	Stage      string            `json:"stage"`
	Count      int               `json:"count"`
	MinNS      int64             `json:"min_ns"`
	P50NS      int64             `json:"p50_ns"`
	P95NS      int64             `json:"p95_ns"`
	P99NS      int64             `json:"p99_ns"`
	MaxNS      int64             `json:"max_ns"`
	SumNS      int64             `json:"sum_ns"`
	Histogram  []HistogramBucket `json:"histogram,omitempty"`
	TailCutoff int64             `json:"tail_cutoff_ns,omitempty"`
}

type LongTail struct {
	FrameID     string  `json:"frame_id"`
	Eye         string  `json:"eye"`
	Stage       string  `json:"stage"`
	DurationNS  int64   `json:"duration_ns"`
	ThresholdNS int64   `json:"threshold_ns"`
	Percentile  float64 `json:"percentile"`
}

type LongTailFrame struct {
	FrameID    string     `json:"frame_id"`
	StageCount int        `json:"stage_count"`
	LongestNS  int64      `json:"longest_ns"`
	Stages     []LongTail `json:"stages"`
}

type Interval struct {
	StartNS int64 `json:"start_ns"`
	EndNS   int64 `json:"end_ns"`
}

type EyeFrameSummary struct {
	Eye             string        `json:"eye"`
	Present         bool          `json:"present"`
	CompleteStages  int           `json:"complete_stages"`
	MissingStages   []string      `json:"missing_stages,omitempty"`
	Stages          []StageDetail `json:"stages,omitempty"`
	StageDurationNS int64         `json:"stage_duration_ns"`
	ActiveSpanNS    int64         `json:"active_span_ns"`
	IdleWaitNS      *int64        `json:"idle_wait_ns,omitempty"`
	IdleWindows     []Interval    `json:"idle_windows,omitempty"`
}

type StageDetail struct {
	Stage          string     `json:"stage"`
	Start          *Timestamp `json:"start,omitempty"`
	End            *Timestamp `json:"end,omitempty"`
	DurationNS     *int64     `json:"duration_ns,omitempty"`
	DurationSource string     `json:"duration_source,omitempty"`
}

type FrameSummary struct {
	FrameID     string            `json:"frame_id"`
	FrameSpanNS int64             `json:"frame_span_ns"`
	Eyes        []EyeFrameSummary `json:"eyes"`
}

type StageOverlap struct {
	Stage          string  `json:"stage"`
	FramesCompared int     `json:"frames_compared"`
	FramesOverlap  int     `json:"frames_overlap"`
	LeftNS         int64   `json:"left_ns"`
	RightNS        int64   `json:"right_ns"`
	OverlapNS      int64   `json:"overlap_ns"`
	LeftOnlyNS     int64   `json:"left_only_ns"`
	RightOnlyNS    int64   `json:"right_only_ns"`
	OverlapRatio   float64 `json:"overlap_ratio"`
}

type CrossStageOverlap struct {
	LeftStage  string `json:"left_stage"`
	RightStage string `json:"right_stage"`
	OverlapNS  int64  `json:"overlap_ns"`
	FrameCount int    `json:"frame_count"`
}

type IdleSummary struct {
	Eye            string `json:"eye"`
	FramesWithWait int    `json:"frames_with_wait"`
	TotalWaitNS    int64  `json:"total_wait_ns"`
	MedianWaitNS   int64  `json:"median_wait_ns"`
	P95WaitNS      int64  `json:"p95_wait_ns"`
	MaxWaitNS      int64  `json:"max_wait_ns"`
}

type Summary struct {
	Frames               int `json:"frames"`
	PairedFrames         int `json:"paired_frames"`
	UnpairedFrames       int `json:"unpaired_frames"`
	MissingStages        int `json:"missing_stages"`
	TimestampRegressions int `json:"timestamp_regressions"`
	UnsupportedTimes     int `json:"unsupported_timestamps"`
	FallbackTimes        int `json:"fallback_timestamps"`
	LongTailStages       int `json:"long_tail_stages"`
	LongTailFrames       int `json:"long_tail_frames"`
}

type Report struct {
	Config             Config              `json:"config"`
	Summary            Summary             `json:"summary"`
	StageStats         []StageStat         `json:"stage_stats"`
	LongTail           []LongTail          `json:"long_tail"`
	LongTailFrames     []LongTailFrame     `json:"long_tail_frames"`
	StageOverlaps      []StageOverlap      `json:"stage_overlaps"`
	CrossStageOverlaps []CrossStageOverlap `json:"cross_stage_overlaps,omitempty"`
	IdleWaits          []IdleSummary       `json:"idle_waits"`
	Frames             []FrameSummary      `json:"frames"`
	Diagnostics        []Diagnostic        `json:"diagnostics"`
}
