package notify

import (
	"fmt"
	"sort"
	"time"
)

// Config controls deterministic frame scheduling defaults.
type Config struct {
	FrameInterval time.Duration `json:"frame_interval_ns"`
}

// DefaultConfig returns a common 60 Hz schedule.
func DefaultConfig() Config {
	return Config{FrameInterval: 16 * time.Millisecond}
}

func (c Config) withDefaults() Config {
	if c.FrameInterval <= 0 {
		c.FrameInterval = 16 * time.Millisecond
	}
	return c
}

// Schedule is the ordering contract for one frame.
type Schedule struct {
	Number    int           `json:"number"`
	Deadline  time.Duration `json:"deadline_ns"`
	CommitAt  time.Duration `json:"commit_at_ns"`
	ScanoutAt time.Duration `json:"scanout_at_ns"`
}

func (s Schedule) validate() error {
	if s.Number <= 0 {
		return fmt.Errorf("frame number must be positive, got %d", s.Number)
	}
	if s.Deadline < 0 || s.CommitAt < s.Deadline || s.ScanoutAt < s.CommitAt {
		return fmt.Errorf("frame %d has invalid schedule %+v", s.Number, s)
	}
	return nil
}

// Source identifies where an injected state came from.
type Source string

const (
	SourceNotification Source = "notification"
	SourceWorldFrame   Source = "world_frame"
)

// Timing classifies arrival relative to the frame input deadline.
type Timing string

const (
	TimingBeforeDeadline Timing = "before_deadline"
	TimingLate           Timing = "late"
)

// Outcome is the final pipeline disposition of one input.
type Outcome string

const (
	OutcomeAccepted   Outcome = "accepted"
	OutcomeCommitted  Outcome = "committed"
	OutcomeSuperseded Outcome = "superseded"
	OutcomeStale      Outcome = "ignored_stale"
)

// Update is one injected notification revision.
type Update struct {
	At     time.Duration `json:"at_ns"`
	Source Source        `json:"source"`
	State  State         `json:"state"`
}

// InputRecord preserves every injection and its disposition.
type InputRecord struct {
	Seq         int     `json:"seq"`
	Update      Update  `json:"update"`
	TargetFrame int     `json:"target_frame"`
	Timing      Timing  `json:"timing"`
	Outcome     Outcome `json:"outcome"`
	Reason      string  `json:"reason,omitempty"`
	Change      Diff    `json:"change"`
}

// Frame is an immutable committed world/notification snapshot.
type Frame struct {
	Number        int              `json:"number"`
	Deadline      time.Duration    `json:"deadline_ns"`
	CommitAt      time.Duration    `json:"commit_at_ns"`
	ScanoutAt     time.Duration    `json:"scanout_at_ns"`
	States        map[string]State `json:"states"`
	Diffs         map[string]Diff  `json:"diffs"`
	Inputs        []*InputRecord   `json:"inputs"`
	Arrivals      []*InputRecord   `json:"arrivals"`
	WorldReceipts []*WorldReceipt  `json:"world_receipts,omitempty"`
}

// State returns a cloned state lookup result.
func (f *Frame) State(id string) (State, bool) {
	if f == nil || f.States == nil {
		return State{}, false
	}
	state, ok := f.States[id]
	return state, ok
}

// VisibleIDs returns sorted notification IDs visible in this frame.
func (f *Frame) VisibleIDs() []string {
	ids := make([]string, 0)
	for id, state := range f.States {
		if state.Visible() {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// Pipeline owns versioned notification state and deterministic frame windows.
type Pipeline struct {
	cfg           Config
	expectedFrame int
	schedule      Schedule
	open          bool

	latest map[string]State
	base   map[string]State

	staged    map[string]State
	stagedRec map[string]*InputRecord
	queued    map[string]State
	queuedRec map[string]*InputRecord

	arrivals      []*InputRecord
	frameReceipts []*WorldReceipt

	events   []*InputRecord
	frames   []*Frame
	receipts []*WorldReceipt
	last     *Frame
	seq      int
}

// NewPipeline creates an empty deterministic pipeline.
func NewPipeline(cfg Config) *Pipeline {
	return &Pipeline{
		cfg:           cfg.withDefaults(),
		expectedFrame: 1,
		latest:        make(map[string]State),
		base:          make(map[string]State),
		staged:        make(map[string]State),
		stagedRec:     make(map[string]*InputRecord),
		queued:        make(map[string]State),
		queuedRec:     make(map[string]*InputRecord),
	}
}

// SetInitial seeds the old version. It is intended for test/scenario setup.
func (p *Pipeline) SetInitial(states ...State) error {
	for _, state := range states {
		if err := state.validate(); err != nil {
			return err
		}
		p.latest[state.ID] = state
		p.base[state.ID] = state
	}
	return nil
}

// BeginFrame opens the input window for schedule.Number.
func (p *Pipeline) BeginFrame(schedule Schedule) error {
	if p.open {
		return fmt.Errorf("frame %d is already open", p.schedule.Number)
	}
	if schedule.Number != p.expectedFrame {
		return fmt.Errorf("expected frame %d, got %d", p.expectedFrame, schedule.Number)
	}
	if err := schedule.validate(); err != nil {
		return err
	}

	p.schedule = schedule
	p.open = true
	p.staged = p.queued
	p.stagedRec = p.queuedRec
	p.queued = make(map[string]State)
	p.queuedRec = make(map[string]*InputRecord)
	p.arrivals = nil
	p.frameReceipts = nil
	return nil
}
