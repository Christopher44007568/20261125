package geomlife

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

const DefaultCapacity = 1024

type Config struct {
	Capacity          int
	SubmissionTimeout time.Duration
	Clock             Clock
}

type interval struct {
	Offset int
	Size   int
}

type objectRecord struct {
	info   Object
	claims map[uint64]struct{}
}

type submissionRecord struct {
	info Submission
}

type System struct {
	mu sync.Mutex

	clock          Clock
	capacity       int
	defaultTimeout time.Duration

	nextAllocationSeq uint64
	nextSubmissionID  uint64

	free        []interval
	live        map[uint64]*objectRecord
	objects     []*objectRecord
	submissions map[uint64]*submissionRecord

	events []Event
}

type Arena = System

func NewArena(cfg Config) (*Arena, error) {
	return New(cfg)
}

func New(cfg Config) (*System, error) {
	if cfg.Capacity < 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidCapacity, cfg.Capacity)
	}
	capacity := cfg.Capacity
	if capacity == 0 {
		capacity = DefaultCapacity
	}
	clock := cfg.Clock
	if clock == nil {
		clock = SystemClock{}
	}
	return &System{
		clock:          clock,
		capacity:       capacity,
		defaultTimeout: cfg.SubmissionTimeout,
		free:           []interval{{Offset: 0, Size: capacity}},
		live:           make(map[uint64]*objectRecord),
		submissions:    make(map[uint64]*submissionRecord),
	}, nil
}

func NewDefault() *System {
	sys, err := New(Config{})
	if err != nil {
		panic(err)
	}
	return sys
}

func (s *System) Clock() Clock {
	return s.clock
}

func (s *System) Capacity() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.capacity
}

func (s *System) emitLocked(now time.Time, event Event) {
	if event.Time.IsZero() {
		event.Time = now
	}
	s.events = append(s.events, event)
}

func (s *System) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *System) snapshotObjectLocked(record *objectRecord) Object {
	out := record.info
	out.ClaimedBySubmissionIDs = make([]uint64, 0, len(record.claims))
	for id := range record.claims {
		out.ClaimedBySubmissionIDs = append(out.ClaimedBySubmissionIDs, id)
	}
	sort.Slice(out.ClaimedBySubmissionIDs, func(i, j int) bool {
		return out.ClaimedBySubmissionIDs[i] < out.ClaimedBySubmissionIDs[j]
	})
	return out
}

func (s *System) Object(allocationSeq uint64) (Object, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.objectBySeqLocked(allocationSeq)
	if record == nil {
		return Object{}, false
	}
	return s.snapshotObjectLocked(record), true
}

func (s *System) Objects() []Object {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Object, 0, len(s.objects))
	for _, record := range s.objects {
		out = append(out, s.snapshotObjectLocked(record))
	}
	return out
}

func (s *System) snapshotSubmissionLocked(record *submissionRecord) Submission {
	out := record.info
	out.AllocationSeqs = append([]uint64(nil), record.info.AllocationSeqs...)
	return out
}

func (s *System) Submission(id uint64) (Submission, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.submissions[id]
	if !ok {
		return Submission{}, false
	}
	return s.snapshotSubmissionLocked(record), true
}

func (s *System) Submissions() []Submission {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]uint64, 0, len(s.submissions))
	for id := range s.submissions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]Submission, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.snapshotSubmissionLocked(s.submissions[id]))
	}
	return out
}

func (s *System) objectBySeqLocked(seq uint64) *objectRecord {
	if seq == 0 || int(seq-1) >= len(s.objects) {
		return nil
	}
	record := s.objects[seq-1]
	if record == nil || record.info.AllocationSeq != seq {
		return nil
	}
	return record
}

type Snapshot struct {
	Capacity          int          `json:"capacity"`
	NextAllocationSeq uint64       `json:"next_allocation_seq"`
	NextSubmissionID  uint64       `json:"next_submission_id"`
	Objects           []Object     `json:"objects"`
	Submissions       []Submission `json:"submissions"`
	Leases            []Lease      `json:"leases"`
	Events            []Event      `json:"events"`
}

func (s *System) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	objects := make([]Object, 0, len(s.objects))
	for _, record := range s.objects {
		objects = append(objects, s.snapshotObjectLocked(record))
	}
	submissionIDs := make([]uint64, 0, len(s.submissions))
	for id := range s.submissions {
		submissionIDs = append(submissionIDs, id)
	}
	sort.Slice(submissionIDs, func(i, j int) bool { return submissionIDs[i] < submissionIDs[j] })
	submissions := make([]Submission, 0, len(submissionIDs))
	for _, id := range submissionIDs {
		submissions = append(submissions, s.snapshotSubmissionLocked(s.submissions[id]))
	}
	leases := s.leasesLocked(submissionIDs)
	events := append([]Event(nil), s.events...)
	return Snapshot{
		Capacity:          s.capacity,
		NextAllocationSeq: s.nextAllocationSeq,
		NextSubmissionID:  s.nextSubmissionID,
		Objects:           objects,
		Submissions:       submissions,
		Leases:            leases,
		Events:            events,
	}
}

func (s *System) leasesLocked(submissionIDs []uint64) []Lease {
	if submissionIDs == nil {
		submissionIDs = make([]uint64, 0, len(s.submissions))
		for id := range s.submissions {
			submissionIDs = append(submissionIDs, id)
		}
		sort.Slice(submissionIDs, func(i, j int) bool { return submissionIDs[i] < submissionIDs[j] })
	}

	leases := make([]Lease, 0)
	for _, id := range submissionIDs {
		submission := s.submissions[id]
		for _, seq := range submission.info.AllocationSeqs {
			record := s.objectBySeqLocked(seq)
			ticket := Ticket{AllocationSeq: seq}
			if record != nil {
				ticket = record.info.Ticket()
			}
			leases = append(leases, Lease{
				SubmissionID:  id,
				FrameID:       submission.info.FrameID,
				AllocationSeq: seq,
				Ticket:        ticket,
				State:         submission.info.State,
				Reason:        submission.info.TerminalReason,
				CreatedAt:     submission.info.CreatedAt,
				FinishedAt:    submission.info.FinishedAt,
			})
		}
	}
	return leases
}

func (s *System) Leases() []Lease {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.leasesLocked(nil)
}
