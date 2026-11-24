package sorter

import (
	"fmt"
	"strings"
	"time"
)

// Queue identifies one of the four independent render queues.
type Queue uint8

const (
	QueueUnknown Queue = iota
	QueueWorld
	QueueUI
	QueueHand
	QueuePassthrough
)

var defaultQueueOrder = []Queue{
	QueueWorld,
	QueueUI,
	QueueHand,
	QueuePassthrough,
}

func (q Queue) String() string {
	switch q {
	case QueueWorld:
		return "world"
	case QueueUI:
		return "ui"
	case QueueHand:
		return "hand"
	case QueuePassthrough:
		return "passthrough"
	case QueueUnknown:
		return "unknown"
	default:
		return fmt.Sprintf("queue(%d)", uint8(q))
	}
}

// ParseQueue parses the canonical queue name. The alias "pass" is accepted for
// convenience in hand-written input.
func ParseQueue(value string) (Queue, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "world":
		return QueueWorld, nil
	case "ui":
		return QueueUI, nil
	case "hand":
		return QueueHand, nil
	case "passthrough", "pass":
		return QueuePassthrough, nil
	default:
		return QueueUnknown, fmt.Errorf("unknown queue %q", value)
	}
}

func (q Queue) MarshalText() ([]byte, error) {
	return []byte(q.String()), nil
}

func (q *Queue) UnmarshalText(text []byte) error {
	parsed, err := ParseQueue(string(text))
	if err != nil {
		*q = QueueUnknown
		return nil
	}
	*q = parsed
	return nil
}

// Eye identifies the eye for which an entry was produced.
type Eye uint8

const (
	EyeUnknown Eye = iota
	EyeLeft
	EyeRight
	EyeBoth
)

var defaultEyeOrder = []Eye{EyeLeft, EyeRight, EyeBoth}

func (e Eye) String() string {
	switch e {
	case EyeLeft:
		return "left"
	case EyeRight:
		return "right"
	case EyeBoth:
		return "both"
	case EyeUnknown:
		return "unknown"
	default:
		return fmt.Sprintf("eye(%d)", uint8(e))
	}
}

func ParseEye(value string) (Eye, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "left":
		return EyeLeft, nil
	case "right":
		return EyeRight, nil
	case "both":
		return EyeBoth, nil
	default:
		return EyeUnknown, fmt.Errorf("unknown eye %q", value)
	}
}

func (e Eye) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *Eye) UnmarshalText(text []byte) error {
	parsed, err := ParseEye(string(text))
	if err != nil {
		*e = EyeUnknown
		return nil
	}
	*e = parsed
	return nil
}

// Submission is one completion from a render queue. Frame is the logical frame
// number, not an arrival rank. SubmittedAt and CompletedAt describe the two
// independently ordered timelines and are never used to decide which frame is
// newest.
type Submission struct {
	ID          string    `json:"id,omitempty"`
	Frame       uint64    `json:"frame"`
	Queue       Queue     `json:"queue"`
	Eye         Eye       `json:"eye"`
	SubmittedAt time.Time `json:"submitted_at,omitzero"`
	CompletedAt time.Time `json:"completed_at,omitzero"`
	Deadline    time.Time `json:"deadline,omitzero"`
}

// Config controls which late entries are still eligible and how entries for
// the same frame are ordered in the final output.
type Config struct {
	// ReorderWindow is the number of frame numbers behind the authoritative
	// latest frame that may still be adopted. A frame exactly
	// ReorderWindow behind is eligible; one further behind is a historical
	// backfill. Zero therefore accepts only the latest frame.
	ReorderWindow uint64 `json:"reorder_window"`

	// Timeout rejects an entry whose completion took longer than this duration.
	// It is evaluated only when SubmittedAt and CompletedAt are both set.
	Timeout time.Duration `json:"-"`

	// QueueOrder is the deterministic order used inside one frame. Missing
	// queues are appended in the canonical world, UI, hand, passthrough order.
	QueueOrder []Queue `json:"queue_order,omitempty"`

	// EyeOrder is the deterministic order used inside one queue and frame.
	// Missing eyes are appended in the canonical left, right, both order.
	EyeOrder []Eye `json:"eye_order,omitempty"`
}

type RejectReason string

const (
	ReasonInvalidQueue       RejectReason = "invalid_queue"
	ReasonInvalidEye         RejectReason = "invalid_eye"
	ReasonDuplicate          RejectReason = "duplicate"
	ReasonTimeout            RejectReason = "timeout"
	ReasonHistoricalBackfill RejectReason = "historical_backfill"
	ReasonFutureFrame        RejectReason = "future_frame"
)

// Counts summarizes decisions without requiring callers to rescan the report.
type Counts struct {
	Submitted           int `json:"submitted"`
	Adopted             int `json:"adopted"`
	OutOfOrderAdopted   int `json:"out_of_order_adopted"`
	Duplicates          int `json:"duplicates"`
	Timeouts            int `json:"timeouts"`
	HistoricalBackfills int `json:"historical_backfills"`
	FutureFrames        int `json:"future_frames"`
	Invalid             int `json:"invalid"`
}

// Adopted is an entry in the final output. Order is one-based and follows
// frame, configured queue order, configured eye order, then ID.
type Adopted struct {
	Submission
	Order           int  `json:"adopted_order"`
	SubmissionOrder int  `json:"submission_order"`
	CompletionOrder int  `json:"completion_order"`
	OutOfOrder      bool `json:"out_of_order"`
}

// Rejected records why an entry did not enter the final output. Eye is part of
// Submission and is therefore present for both adopted and rejected entries.
type Rejected struct {
	Submission
	Reason          RejectReason `json:"reason"`
	DuplicateOf     string       `json:"duplicate_of,omitempty"`
	SubmissionOrder int          `json:"submission_order"`
	CompletionOrder int          `json:"completion_order"`
}

// Report is a deterministic snapshot of an ordering run.
type Report struct {
	LatestFrame uint64     `json:"latest_frame"`
	Adopted     []Adopted  `json:"adopted"`
	Rejected    []Rejected `json:"rejected"`
	Counts      Counts     `json:"counts"`
}

// entryKey is the identity of a logical queue result. A repeated key is a
// duplicate even if its payload ID differs.
type entryKey struct {
	Frame uint64
	Queue Queue
	Eye   Eye
}
