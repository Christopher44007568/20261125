package geomlife

import (
	"errors"
	"time"
)

type SubmissionState string

const (
	StateSubmitted SubmissionState = "submitted"
	StateCompleted SubmissionState = "completed"
	StateCanceled  SubmissionState = "canceled"
)

type ObjectState string

const (
	ObjectHeld      ObjectState = "held"
	ObjectReclaimed ObjectState = "reclaimed"
)

type ReleaseReason string

const (
	ReasonPending   ReleaseReason = "pending"
	ReasonCompleted ReleaseReason = "submission_completed"
	ReasonCanceled  ReleaseReason = "submission_canceled"
	ReasonTimeout   ReleaseReason = "timeout_reclaimed"
	ReasonExplicit  ReleaseReason = "explicit_release"
	ReasonReset     ReleaseReason = "arena_reset"
	ReasonRollback  ReleaseReason = "submit_rollback"
)

var (
	ErrInvalidCapacity    = errors.New("arena capacity must be positive")
	ErrInvalidSize        = errors.New("allocation size must be positive")
	ErrOutOfMemory        = errors.New("arena does not have a suitable free interval")
	ErrNotFound           = errors.New("resource not found")
	ErrAlreadyTerminal    = errors.New("submission is already completed or canceled")
	ErrInvalidState       = errors.New("operation is not valid for the current resource state")
	ErrRangeMismatch      = errors.New("ticket interval does not match allocation sequence")
	ErrStaleTicket        = errors.New("allocation ticket is stale")
	ErrDuplicateRelease   = errors.New("duplicate release blocked")
	ErrRangeHeld          = errors.New("range is still held by another submission")
	ErrTimeoutNotReached  = errors.New("submission timeout has not been reached")
	ErrInvariantViolation = errors.New("arena invariant violation")
)

type Ticket struct {
	AllocationSeq uint64 `json:"allocation_seq"`
	Offset        int    `json:"offset"`
	Size          int    `json:"size"`
}

type Object struct {
	AllocationSeq              uint64          `json:"allocation_seq"`
	Offset                     int             `json:"offset"`
	Size                       int             `json:"size"`
	State                      ObjectState     `json:"state"`
	LifecycleState             SubmissionState `json:"lifecycle_state"`
	LastUsedSubmissionID       uint64          `json:"last_used_submission_id"`
	LastUsedFrameID            uint64          `json:"last_used_frame_id"`
	ClaimedBySubmissionIDs     []uint64        `json:"claimed_by_submission_ids"`
	ReleaseReason              ReleaseReason   `json:"release_reason"`
	ReleasedAt                 *time.Time      `json:"released_at,omitempty"`
	ReleasedBySubmissionID     uint64          `json:"released_by_submission_id,omitempty"`
	DuplicateReleaseAttempts   int             `json:"duplicate_release_attempts"`
	StaleReleaseAttempts       int             `json:"stale_release_attempts"`
	BlockedHeldReleaseAttempts int             `json:"blocked_held_release_attempts"`
}

func (o Object) Ticket() Ticket {
	return Ticket{AllocationSeq: o.AllocationSeq, Offset: o.Offset, Size: o.Size}
}

type Lease struct {
	SubmissionID  uint64          `json:"submission_id"`
	FrameID       uint64          `json:"frame_id"`
	AllocationSeq uint64          `json:"allocation_seq"`
	Ticket        Ticket          `json:"ticket"`
	State         SubmissionState `json:"state"`
	Reason        ReleaseReason   `json:"reason,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
}

type Submission struct {
	ID             uint64          `json:"id"`
	FrameID        uint64          `json:"frame_id"`
	State          SubmissionState `json:"state"`
	AllocationSeqs []uint64        `json:"allocation_seqs"`
	CreatedAt      time.Time       `json:"created_at"`
	TimeoutAt      time.Time       `json:"timeout_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	TerminalReason ReleaseReason   `json:"terminal_reason,omitempty"`
}

type Event struct {
	Time              time.Time       `json:"time"`
	Kind              string          `json:"kind"`
	OK                bool            `json:"ok"`
	SubmissionID      uint64          `json:"submission_id,omitempty"`
	OtherSubmissionID uint64          `json:"other_submission_id,omitempty"`
	FrameID           uint64          `json:"frame_id,omitempty"`
	AllocationSeq     uint64          `json:"allocation_seq,omitempty"`
	Offset            int             `json:"offset,omitempty"`
	Size              int             `json:"size,omitempty"`
	State             SubmissionState `json:"state,omitempty"`
	Reason            ReleaseReason   `json:"reason,omitempty"`
	Message           string          `json:"message,omitempty"`
}

type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}
