package geomlife

import (
	"errors"
	"time"
)

type CanonicalReport struct {
	OK                  bool      `json:"ok"`
	SubmittedCheckpoint *Snapshot `json:"submitted_checkpoint,omitempty"`
	Snapshot            Snapshot  `json:"snapshot"`
	Checks              []Check   `json:"checks"`
}

func RunCanonicalScenario() CanonicalReport {
	clock := NewManualClockAt(time.Date(2026, time.November, 25, 8, 0, 0, 0, time.UTC))
	system, err := New(Config{Capacity: 256, SubmissionTimeout: 45 * time.Millisecond, Clock: clock})
	report := CanonicalReport{Checks: []Check{}}
	if err != nil {
		report.Checks = append(report.Checks, Check{Name: "create_arena", OK: false, Detail: err.Error()})
		return report
	}

	addCheck := func(name string, ok bool, detail string) {
		report.Checks = append(report.Checks, Check{Name: name, OK: ok, Detail: detail})
	}
	errText := func(err error) string {
		if err == nil {
			return ""
		}
		return err.Error()
	}
	expectOK := func(name string, err error) {
		addCheck(name, err == nil, errText(err))
	}
	expectIs := func(name string, err error, targets ...error) {
		ok := err != nil
		for _, target := range targets {
			ok = ok && errors.Is(err, target)
		}
		addCheck(name, ok, errText(err))
	}
	fatal := func(name string, err error) CanonicalReport {
		addCheck(name, false, err.Error())
		report.Snapshot = system.Snapshot()
		return report
	}

	first, firstTickets, err := system.Submit(SubmitRequest{FrameID: 1, Sizes: []int{64, 32}})
	if err != nil {
		return fatal("submit_frame_1", err)
	}
	clock.Advance(10 * time.Millisecond)

	second, _, err := system.Submit(SubmitRequest{FrameID: 2, Sizes: []int{64, 32}})
	if err != nil {
		return fatal("submit_frame_2", err)
	}
	if _, err := system.Bind(firstTickets[0], second.ID); err != nil {
		return fatal("bind_shared_range_to_frame_2", err)
	}

	submissions := system.Submissions()
	doubleInFlight := len(submissions) == 2
	for _, submission := range submissions {
		doubleInFlight = doubleInFlight && submission.State == StateSubmitted
	}
	addCheck("double_frame_submitted", doubleInFlight, "frames 1 and 2 are both in submitted state")
	checkpoint := system.Snapshot()
	report.SubmittedCheckpoint = &checkpoint

	clock.Advance(5 * time.Millisecond)
	if _, err := system.Complete(first.ID); err != nil {
		return fatal("complete_frame_1", err)
	}
	shared, _ := system.Object(firstTickets[0].AllocationSeq)
	addCheck("shared_range_survives_old_completion",
		shared.State == ObjectHeld &&
			len(shared.ClaimedBySubmissionIDs) == 1 &&
			shared.ClaimedBySubmissionIDs[0] == second.ID,
		"old submission only removes its own claim")

	_, heldErr := system.Release(firstTickets[0], first.ID, ReasonCompleted)
	expectIs("old_submission_cannot_release_new_held_range", heldErr, ErrRangeHeld)

	if _, err := system.Complete(second.ID); err != nil {
		return fatal("complete_frame_2", err)
	}
	shared, _ = system.Object(firstTickets[0].AllocationSeq)
	addCheck("new_submission_releases_shared_range",
		shared.State == ObjectReclaimed && shared.ReleasedBySubmissionID == second.ID,
		shared.ReleaseReason.String())

	third, thirdTickets, err := system.Submit(SubmitRequest{FrameID: 3, Sizes: []int{64}})
	if err != nil {
		return fatal("submit_frame_3_reuses_interval", err)
	}
	addCheck("interval_reused_with_new_allocation_seq",
		thirdTickets[0].Offset == firstTickets[0].Offset &&
			thirdTickets[0].AllocationSeq > firstTickets[0].AllocationSeq,
		"same offset is safe because tickets carry allocation sequence")

	_, staleErr := system.Release(firstTickets[0], first.ID, ReasonCompleted)
	expectIs("old_ticket_blocked_from_new_interval", staleErr, ErrStaleTicket, ErrDuplicateRelease)
	_, staleErr = system.Release(firstTickets[0], first.ID, ReasonCompleted)
	expectIs("repeated_old_ticket_still_blocked", staleErr, ErrStaleTicket, ErrDuplicateRelease)

	newObject, _ := system.Object(thirdTickets[0].AllocationSeq)
	addCheck("new_submission_object_still_held", newObject.State == ObjectHeld, "")
	_, thirdCompleteErr := system.Complete(third.ID)
	expectOK("complete_frame_3", thirdCompleteErr)

	clock.Advance(20 * time.Millisecond)
	timedOut, timeoutTickets, err := system.Submit(SubmitRequest{
		FrameID: 4,
		Sizes:   []int{48},
		Timeout: 30 * time.Millisecond,
	})
	if err != nil {
		return fatal("submit_timeout_frame", err)
	}
	clock.Advance(29 * time.Millisecond)
	addCheck("timeout_not_reclaimed_early", len(system.ExpireTimeouts()) == 0, "")
	clock.Advance(2 * time.Millisecond)
	expired := system.ExpireTimeouts()
	addCheck("timeout_cancels_and_reclaims",
		len(expired) == 1 && expired[0].ID == timedOut.ID && expired[0].State == StateCanceled,
		"timed-out submission enters canceled state")
	timeoutObject, _ := system.Object(timeoutTickets[0].AllocationSeq)
	addCheck("timeout_release_reason_recorded", timeoutObject.ReleaseReason == ReasonTimeout, "")
	_, timeoutDoubleErr := system.Release(timeoutTickets[0], timedOut.ID, ReasonTimeout)
	expectIs("timeout_double_release_blocked", timeoutDoubleErr, ErrDuplicateRelease)
	_, timeoutCancelErr := system.Cancel(timedOut.ID, ReasonCanceled)
	expectIs("timeout_double_cancel_blocked", timeoutCancelErr, ErrAlreadyTerminal)

	canceled, cancelTickets, err := system.Submit(SubmitRequest{FrameID: 5, Sizes: []int{16}})
	if err != nil {
		return fatal("submit_canceled_frame", err)
	}
	_, cancelErr := system.Cancel(canceled.ID, ReasonCanceled)
	expectOK("cancel_frame_5", cancelErr)
	canceledObject, _ := system.Object(cancelTickets[0].AllocationSeq)
	addCheck("cancel_reason_recorded", canceledObject.ReleaseReason == ReasonCanceled, "")
	_, cancelDoubleErr := system.Release(cancelTickets[0], canceled.ID, ReasonCanceled)
	expectIs("cancel_double_release_blocked", cancelDoubleErr, ErrDuplicateRelease)

	metadataOK := true
	for _, object := range system.Objects() {
		if object.AllocationSeq == 0 ||
			object.LastUsedSubmissionID == 0 ||
			object.LifecycleState != StateCompleted && object.LifecycleState != StateCanceled ||
			object.ReleaseReason == "" || object.ReleaseReason == ReasonPending {
			metadataOK = false
		}
	}
	addCheck("every_object_has_seq_last_submission_and_release_reason", metadataOK, "")

	stateCounts := map[SubmissionState]int{}
	for _, lease := range system.Leases() {
		stateCounts[lease.State]++
	}
	addCheck("lease_lifecycle_contains_completed_and_canceled",
		stateCounts[StateCompleted] > 0 && stateCounts[StateCanceled] > 0,
		"submitted state is recorded by submitted events and the double-frame checkpoint")

	invariantErr := system.CheckInvariants()
	addCheck("arena_invariants", invariantErr == nil, errText(invariantErr))

	report.Snapshot = system.Snapshot()
	report.OK = true
	for _, check := range report.Checks {
		if !check.OK {
			report.OK = false
		}
	}
	return report
}

func (r ReleaseReason) String() string {
	return string(r)
}
