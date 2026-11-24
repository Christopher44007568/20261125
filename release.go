package geomlife

import (
	"fmt"
)

func (s *System) Release(ticket Ticket, submissionID uint64, reason ReleaseReason) (Object, error) {
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	record, resolveErr := s.resolveTicketLocked(ticket)
	if resolveErr != nil {
		if record != nil {
			return s.snapshotObjectLocked(record), resolveErr
		}
		return Object{}, resolveErr
	}

	seq := record.info.AllocationSeq
	if record.info.State == ObjectReclaimed {
		record.info.DuplicateReleaseAttempts++
		current := s.overlapOwnerLocked(ticket)
		if current != nil && current.info.AllocationSeq != seq {
			record.info.StaleReleaseAttempts++
			err := fmt.Errorf(
				"%w: seq=%d is retired; interval is held by seq=%d: %w",
				ErrStaleTicket, seq, current.info.AllocationSeq, ErrDuplicateRelease,
			)
			s.emitLocked(now, Event{
				Kind:              "stale_release_blocked",
				OK:                false,
				SubmissionID:      submissionID,
				OtherSubmissionID: current.info.LastUsedSubmissionID,
				AllocationSeq:     seq,
				Offset:            ticket.Offset,
				Size:              ticket.Size,
				Reason:            record.info.ReleaseReason,
				Message:           err.Error(),
			})
			return s.snapshotObjectLocked(record), err
		}

		err := fmt.Errorf("%w: seq=%d already released as %q", ErrDuplicateRelease, seq, record.info.ReleaseReason)
		s.emitLocked(now, Event{
			Kind:          "duplicate_release_blocked",
			OK:            false,
			SubmissionID:  submissionID,
			AllocationSeq: seq,
			Offset:        ticket.Offset,
			Size:          ticket.Size,
			Reason:        record.info.ReleaseReason,
			Message:       err.Error(),
		})
		return s.snapshotObjectLocked(record), err
	}

	var submission *submissionRecord
	if submissionID != 0 {
		var ok bool
		submission, ok = s.submissions[submissionID]
		if !ok {
			return s.snapshotObjectLocked(record), fmt.Errorf("%w: submission %d", ErrNotFound, submissionID)
		}
	}

	if holder := s.maxHolderLocked(record, submissionID); holder != 0 {
		record.info.BlockedHeldReleaseAttempts++
		err := fmt.Errorf("%w: submission %d cannot release seq=%d held by submission %d",
			ErrRangeHeld, submissionID, seq, holder)
		s.emitLocked(now, Event{
			Kind:              "release_blocked_held",
			OK:                false,
			SubmissionID:      submissionID,
			OtherSubmissionID: holder,
			AllocationSeq:     seq,
			Offset:            ticket.Offset,
			Size:              ticket.Size,
			Message:           err.Error(),
		})
		return s.snapshotObjectLocked(record), err
	}
	if submission != nil && submission.info.State == StateSubmitted {
		err := fmt.Errorf("%w: submission %d is still submitted", ErrInvalidState, submissionID)
		s.emitLocked(now, Event{
			Kind:          "release_blocked_active_submission",
			OK:            false,
			SubmissionID:  submissionID,
			FrameID:       submission.info.FrameID,
			State:         StateSubmitted,
			AllocationSeq: seq,
			Offset:        ticket.Offset,
			Size:          ticket.Size,
			Message:       err.Error(),
		})
		return s.snapshotObjectLocked(record), err
	}

	lifecycleState := StateCompleted
	if submission != nil {
		lifecycleState = submission.info.State
	}

	delete(record.claims, submissionID)
	record.info.LifecycleState = lifecycleState
	if reason == "" {
		reason = ReasonExplicit
	}
	s.reclaimLocked(now, record, reason, submissionID)
	return s.snapshotObjectLocked(record), nil
}
