package geomlife

import (
	"fmt"
	"sort"
	"time"
)

type SubmitRequest struct {
	FrameID uint64
	Sizes   []int
	Timeout time.Duration
}

func (s *System) Submit(request SubmitRequest) (Submission, []Ticket, error) {
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, size := range request.Sizes {
		if size <= 0 {
			return Submission{}, nil, fmt.Errorf("%w: %d", ErrInvalidSize, size)
		}
	}

	s.nextSubmissionID++
	id := s.nextSubmissionID
	timeout := s.defaultTimeout
	if request.Timeout != 0 {
		timeout = request.Timeout
	}
	var timeoutAt time.Time
	if timeout != 0 {
		timeoutAt = now.Add(timeout)
	}

	record := &submissionRecord{info: Submission{
		ID:             id,
		FrameID:        request.FrameID,
		State:          StateSubmitted,
		AllocationSeqs: make([]uint64, 0, len(request.Sizes)),
		CreatedAt:      now,
		TimeoutAt:      timeoutAt,
	}}
	s.submissions[id] = record

	tickets := make([]Ticket, 0, len(request.Sizes))
	for _, size := range request.Sizes {
		ticket, objectRecord, err := s.allocateLocked(now, size)
		if err != nil {
			s.finishLocked(now, record, StateCanceled, ReasonRollback)
			return Submission{}, nil, err
		}
		if err := s.bindRecordLocked(now, ticket, id); err != nil {
			s.finishLocked(now, record, StateCanceled, ReasonRollback)
			return Submission{}, nil, err
		}
		tickets = append(tickets, ticket)
		_ = objectRecord
	}

	s.emitLocked(now, Event{
		Kind:         "submitted",
		OK:           true,
		Time:         now,
		SubmissionID: id,
		FrameID:      request.FrameID,
		State:        StateSubmitted,
		Message:      fmt.Sprintf("submission %d for frame %d is in flight", id, request.FrameID),
	})
	return s.snapshotSubmissionLocked(record), tickets, nil
}

func (s *System) Bind(ticket Ticket, submissionID uint64) (Object, error) {
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	submission, ok := s.submissions[submissionID]
	if !ok {
		return Object{}, fmt.Errorf("%w: submission %d", ErrNotFound, submissionID)
	}
	record, err := s.resolveTicketLocked(ticket)
	if err != nil {
		if record != nil {
			return s.snapshotObjectLocked(record), err
		}
		return Object{}, err
	}
	if err := s.bindRecordLocked(now, ticket, submissionID); err != nil {
		return s.snapshotObjectLocked(record), err
	}
	_ = submission
	return s.snapshotObjectLocked(record), nil
}

func (s *System) bindRecordLocked(now time.Time, ticket Ticket, submissionID uint64) error {
	submission, ok := s.submissions[submissionID]
	if !ok {
		return fmt.Errorf("%w: submission %d", ErrNotFound, submissionID)
	}
	if submission.info.State != StateSubmitted {
		return fmt.Errorf("%w: submission %d is %s", ErrAlreadyTerminal, submissionID, submission.info.State)
	}

	record, err := s.resolveTicketLocked(ticket)
	if err != nil {
		return err
	}
	if record.info.State != ObjectHeld {
		return fmt.Errorf("%w: allocation %d is %s", ErrInvalidState, ticket.AllocationSeq, record.info.State)
	}

	if _, exists := record.claims[submissionID]; !exists {
		record.claims[submissionID] = struct{}{}
		submission.info.AllocationSeqs = append(submission.info.AllocationSeqs, ticket.AllocationSeq)
	}
	record.info.LastUsedSubmissionID = submissionID
	record.info.LastUsedFrameID = submission.info.FrameID
	record.info.LifecycleState = StateSubmitted

	s.emitLocked(now, Event{
		Kind:          "bound",
		OK:            true,
		SubmissionID:  submissionID,
		FrameID:       submission.info.FrameID,
		AllocationSeq: ticket.AllocationSeq,
		Offset:        ticket.Offset,
		Size:          ticket.Size,
		Message:       fmt.Sprintf("allocation %d last used by submission %d", ticket.AllocationSeq, submissionID),
	})
	return nil
}

func (s *System) Complete(submissionID uint64) (Submission, error) {
	return s.finish(submissionID, StateCompleted, ReasonCompleted)
}

func (s *System) Cancel(submissionID uint64, reason ReleaseReason) (Submission, error) {
	if reason == "" {
		reason = ReasonCanceled
	}
	return s.finish(submissionID, StateCanceled, reason)
}

func (s *System) finish(submissionID uint64, state SubmissionState, reason ReleaseReason) (Submission, error) {
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.submissions[submissionID]
	if !ok {
		return Submission{}, fmt.Errorf("%w: submission %d", ErrNotFound, submissionID)
	}
	if record.info.State != StateSubmitted {
		s.emitLocked(now, Event{
			Kind:         "duplicate_terminal_blocked",
			OK:           false,
			SubmissionID: submissionID,
			FrameID:      record.info.FrameID,
			State:        record.info.State,
			Reason:       reason,
			Message:      fmt.Sprintf("ignored second transition to %s", state),
		})
		return s.snapshotSubmissionLocked(record), fmt.Errorf(
			"%w: submission %d is already %s",
			ErrAlreadyTerminal, submissionID, record.info.State,
		)
	}

	finished := s.finishLocked(now, record, state, reason)
	return finished, nil
}

func (s *System) finishLocked(now time.Time, record *submissionRecord, state SubmissionState, reason ReleaseReason) Submission {
	if reason == "" {
		if state == StateCompleted {
			reason = ReasonCompleted
		} else {
			reason = ReasonCanceled
		}
	}

	record.info.State = state
	record.info.TerminalReason = reason
	finishedAt := now
	record.info.FinishedAt = &finishedAt

	s.emitLocked(now, Event{
		Kind:         string(state),
		OK:           true,
		SubmissionID: record.info.ID,
		FrameID:      record.info.FrameID,
		State:        state,
		Reason:       reason,
	})

	for _, seq := range append([]uint64(nil), record.info.AllocationSeqs...) {
		objectRecord := s.objectBySeqLocked(seq)
		if objectRecord == nil || objectRecord.info.State != ObjectHeld {
			continue
		}

		delete(objectRecord.claims, record.info.ID)
		if len(objectRecord.claims) == 0 {
			objectRecord.info.LifecycleState = state
			s.reclaimLocked(now, objectRecord, reason, record.info.ID)
			continue
		}

		holder := s.maxHolderLocked(objectRecord, record.info.ID)
		objectRecord.info.LifecycleState = StateSubmitted
		s.emitLocked(now, Event{
			Kind:              "release_deferred",
			OK:                true,
			SubmissionID:      record.info.ID,
			OtherSubmissionID: holder,
			FrameID:           record.info.FrameID,
			AllocationSeq:     seq,
			Offset:            objectRecord.info.Offset,
			Size:              objectRecord.info.Size,
			Reason:            ReasonPending,
			Message: fmt.Sprintf(
				"%s release did not reclaim seq=%d because submission %d still holds it",
				state, seq, holder,
			),
		})
	}

	return s.snapshotSubmissionLocked(record)
}

func (s *System) ExpireTimeouts() []Submission {
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]uint64, 0)
	for id, record := range s.submissions {
		info := record.info
		if info.State == StateSubmitted && !info.TimeoutAt.IsZero() && !now.Before(info.TimeoutAt) {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	expired := make([]Submission, 0, len(ids))
	for _, id := range ids {
		record := s.submissions[id]
		if record.info.State == StateSubmitted {
			expired = append(expired, s.finishLocked(now, record, StateCanceled, ReasonTimeout))
		}
	}
	return expired
}
