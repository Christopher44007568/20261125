package binocular

import (
	"errors"
	"fmt"
)

// Eye identifies one side of a binocular pipeline.
type Eye string

const (
	EyeLeft  Eye = "left"
	EyeRight Eye = "right"
)

func (e Eye) Valid() bool { return e == EyeLeft || e == EyeRight }

// DisconnectScope describes the event that invalidated the current session.
// Any eye failure tears down the whole binocular session because every task
// needs both eyes to produce a valid frame.
type DisconnectScope string

const (
	DisconnectLeftEye  DisconnectScope = "left_eye"
	DisconnectRightEye DisconnectScope = "right_eye"
	DisconnectDevice   DisconnectScope = "device"
)

func (s DisconnectScope) Valid() bool {
	return s == DisconnectLeftEye || s == DisconnectRightEye || s == DisconnectDevice
}

var (
	ErrSessionClosed  = errors.New("binocular session is closed")
	ErrSessionClosing = errors.New("binocular session is closing")
	ErrSessionActive  = errors.New("a session is already active")
	ErrNoSession      = errors.New("no active binocular session")
	ErrStaleLease     = errors.New("write lease belongs to another session or generation")
	ErrLeaseRevoked   = errors.New("write lease has been revoked")
	ErrUnknownBuffer  = errors.New("unknown buffer")
	ErrInvalidScope   = errors.New("invalid disconnect scope")
)

type ResourceKind string

const (
	ResourceTexture ResourceKind = "texture"
	ResourceDepth   ResourceKind = "depth"
	ResourceHistory ResourceKind = "history"
	ResourceCache   ResourceKind = "cache"
)

type TaskState string

const (
	TaskQueued   TaskState = "queued"
	TaskRunning  TaskState = "running"
	TaskStopped  TaskState = "stopped"
	TaskCanceled TaskState = "canceled"
	TaskFailed   TaskState = "failed"
)

func (s TaskState) terminal() bool {
	return s == TaskStopped || s == TaskCanceled || s == TaskFailed
}

type ObjectRef struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Session uint64 `json:"session_id"`
	Detail  string `json:"detail,omitempty"`
}

func (r ObjectRef) String() string {
	if r.Detail == "" {
		return fmt.Sprintf("%s:%s@%d", r.Kind, r.ID, r.Session)
	}
	return fmt.Sprintf("%s:%s@%d(%s)", r.Kind, r.ID, r.Session, r.Detail)
}

type ReleaseCategory struct {
	Name       string   `json:"name"`
	Discovered int      `json:"discovered"`
	Released   int      `json:"released"`
	Failed     int      `json:"failed"`
	OK         bool     `json:"ok"`
	Objects    []string `json:"objects,omitempty"`
	Keys       []string `json:"keys,omitempty"`
}

type TaskRelease struct {
	Submitted    int `json:"submitted"`
	Running      int `json:"running_at_disconnect"`
	Queued       int `json:"queued_at_disconnect"`
	Stopped      int `json:"running_stopped"`
	QueuedCancel int `json:"queued_canceled"`
	Failed       int `json:"failed"`
	StillRunning int `json:"still_running"`
}

type ReleaseReport struct {
	Scope         DisconnectScope   `json:"scope"`
	SessionID     uint64            `json:"session_id"`
	Generation    uint64            `json:"generation"`
	Categories    []ReleaseCategory `json:"categories"`
	Tasks         TaskRelease       `json:"tasks"`
	LeasesRevoked int               `json:"leases_revoked"`
	Unreleased    []ObjectRef       `json:"unreleased_objects"`
	Complete      bool              `json:"complete"`
	TeardownNS    int64             `json:"teardown_ns"`
}

func (r ReleaseReport) Category(name string) (ReleaseCategory, bool) {
	for _, c := range r.Categories {
		if c.Name == name {
			return c, true
		}
	}
	return ReleaseCategory{}, false
}

type Frame struct {
	Left  []byte
	Right []byte
}

type FrameSideDiff struct {
	Eye          Eye    `json:"eye"`
	BaselineHash string `json:"baseline_hash"`
	ObservedHash string `json:"observed_hash"`
	DiffBytes    int    `json:"diff_bytes"`
	DiffPixels   int    `json:"diff_pixels"`
	Equal        bool   `json:"equal"`
}

type FirstFrameDiff struct {
	Equal     bool            `json:"equal"`
	DiffBytes int             `json:"diff_bytes"`
	Sides     []FrameSideDiff `json:"sides"`
}

type ReconnectMetrics struct {
	Attempt    int     `json:"attempt"`
	SessionID  uint64  `json:"session_id"`
	Generation uint64  `json:"generation"`
	TeardownNS int64   `json:"teardown_ns"`
	SetupNS    int64   `json:"setup_ns"`
	TotalNS    int64   `json:"total_ns"`
	TotalMS    float64 `json:"total_ms"`
}

type PrivateStateCheck struct {
	Clean              bool     `json:"clean"`
	Inherited          []string `json:"inherited"`
	OldToken           string   `json:"old_token"`
	NewToken           string   `json:"new_token"`
	NewEpoch           uint64   `json:"new_epoch"`
	NewParentSessionID uint64   `json:"new_parent_session_id"`
	NewHistoryFrames   int      `json:"new_history_frames"`
	NewCacheEntries    int      `json:"new_cache_entries"`
}

type StaleWriteCheck struct {
	Path     string `json:"path"`
	Accepted bool   `json:"accepted"`
	Error    string `json:"error,omitempty"`
}

type CycleReport struct {
	Cycle        int               `json:"cycle"`
	Scope        DisconnectScope   `json:"scope"`
	Release      ReleaseReport     `json:"release"`
	Reconnect    ReconnectMetrics  `json:"reconnect"`
	Unreleased   []ObjectRef       `json:"unreleased_objects"`
	PrivateState PrivateStateCheck `json:"private_state"`
	StaleWrites  []StaleWriteCheck `json:"stale_writes"`
	FirstFrame   FirstFrameDiff    `json:"first_frame"`
	Clean        bool              `json:"clean"`
}

type ScenarioReport struct {
	Scope  DisconnectScope `json:"scope"`
	Cycles []CycleReport   `json:"cycles"`
	Clean  bool            `json:"clean"`
}

type Summary struct {
	AllClean           bool    `json:"all_clean"`
	TotalUnreleased    int     `json:"total_unreleased_objects"`
	MaxFirstFrameDiff  int     `json:"max_first_frame_diff_bytes"`
	ReconnectLatencies []int64 `json:"reconnect_latency_ns"`
}

type Report struct {
	GeneratedAt string           `json:"generated_at"`
	GoVersion   string           `json:"go_version"`
	Scenarios   []ScenarioReport `json:"scenarios"`
	Summary     Summary          `json:"summary"`
}
