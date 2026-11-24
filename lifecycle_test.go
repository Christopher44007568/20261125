package geomlife

import (
	"errors"
	"testing"
	"time"
)

func newTestSystem(t *testing.T, capacity int, timeout time.Duration) (*System, *ManualClock) {
	t.Helper()
	clock := NewManualClock()
	system, err := New(Config{Capacity: capacity, SubmissionTimeout: timeout, Clock: clock})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return system, clock
}

func TestCompletedAndCanceledLifecycle(t *testing.T) {
	system, _ := newTestSystem(t, 128, time.Second)

	completed, tickets, err := system.Submit(SubmitRequest{FrameID: 1, Sizes: []int{32}})
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if completed.State != StateSubmitted {
		t.Fatalf("initial state = %q, want %q", completed.State, StateSubmitted)
	}

	object, ok := system.Object(tickets[0].AllocationSeq)
	if !ok {
		t.Fatal("allocation missing")
	}
	if object.State != ObjectHeld || object.LastUsedSubmissionID != completed.ID {
		t.Fatalf("bad initial object: %+v", object)
	}

	finished, err := system.Complete(completed.ID)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if finished.State != StateCompleted {
		t.Fatalf("state = %q, want completed", finished.State)
	}
	object, _ = system.Object(tickets[0].AllocationSeq)
	if object.State != ObjectReclaimed || object.ReleaseReason != ReasonCompleted {
		t.Fatalf("completion did not release object: %+v", object)
	}
	if object.ReleasedBySubmissionID != completed.ID || object.ReleasedAt == nil {
		t.Fatalf("bad release metadata: %+v", object)
	}

	if _, err := system.Complete(completed.ID); !errors.Is(err, ErrAlreadyTerminal) {
		t.Fatalf("second Complete() error = %v, want ErrAlreadyTerminal", err)
	}

	canceled, cancelTickets, err := system.Submit(SubmitRequest{FrameID: 2, Sizes: []int{16}})
	if err != nil {
		t.Fatalf("second Submit() error = %v", err)
	}
	if _, err := system.Cancel(canceled.ID, ReasonCanceled); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	object, _ = system.Object(cancelTickets[0].AllocationSeq)
	if object.State != ObjectReclaimed || object.ReleaseReason != ReasonCanceled {
		t.Fatalf("cancel did not release object: %+v", object)
	}
	if _, err := system.Cancel(canceled.ID, ""); !errors.Is(err, ErrAlreadyTerminal) {
		t.Fatalf("second Cancel() error = %v, want ErrAlreadyTerminal", err)
	}

	if err := system.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestOldSubmissionCannotReleaseNewHeldRange(t *testing.T) {
	system, _ := newTestSystem(t, 256, time.Second)

	first, firstTickets, err := system.Submit(SubmitRequest{FrameID: 1, Sizes: []int{64, 32}})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := system.Submit(SubmitRequest{FrameID: 2, Sizes: []int{64, 32}})
	if err != nil {
		t.Fatal(err)
	}

	shared, err := system.Bind(firstTickets[0], second.ID)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if shared.LastUsedSubmissionID != second.ID {
		t.Fatalf("last used = %d, want %d", shared.LastUsedSubmissionID, second.ID)
	}

	_, activeHeldErr := system.Release(firstTickets[0], first.ID, ReasonExplicit)
	if !errors.Is(activeHeldErr, ErrRangeHeld) {
		t.Fatalf("active old Release() error = %v, want ErrRangeHeld", activeHeldErr)
	}

	if _, err := system.Complete(first.ID); err != nil {
		t.Fatal(err)
	}
	shared, _ = system.Object(firstTickets[0].AllocationSeq)
	if shared.State != ObjectHeld {
		t.Fatalf("shared range state = %q, want held by second frame", shared.State)
	}
	if len(shared.ClaimedBySubmissionIDs) != 1 || shared.ClaimedBySubmissionIDs[0] != second.ID {
		t.Fatalf("claims = %v, want only submission %d", shared.ClaimedBySubmissionIDs, second.ID)
	}

	_, err = system.Release(firstTickets[0], first.ID, ReasonCompleted)
	if !errors.Is(err, ErrRangeHeld) {
		t.Fatalf("old Release() error = %v, want ErrRangeHeld", err)
	}

	if _, err := system.Complete(second.ID); err != nil {
		t.Fatal(err)
	}
	shared, _ = system.Object(firstTickets[0].AllocationSeq)
	if shared.State != ObjectReclaimed || shared.ReleasedBySubmissionID != second.ID {
		t.Fatalf("new submission should own final release: %+v", shared)
	}

	third, thirdTickets, err := system.Submit(SubmitRequest{FrameID: 3, Sizes: []int{64}})
	if err != nil {
		t.Fatal(err)
	}
	if thirdTickets[0].Offset != firstTickets[0].Offset || thirdTickets[0].AllocationSeq == firstTickets[0].AllocationSeq {
		t.Fatalf("expected reused offset with new seq: old=%+v new=%+v", firstTickets[0], thirdTickets[0])
	}

	_, err = system.Release(firstTickets[0], first.ID, ReasonCompleted)
	if !errors.Is(err, ErrStaleTicket) || !errors.Is(err, ErrDuplicateRelease) {
		t.Fatalf("stale Release() error = %v, want stale+duplicate", err)
	}
	_, err = system.Release(firstTickets[0], first.ID, ReasonCompleted)
	if !errors.Is(err, ErrStaleTicket) || !errors.Is(err, ErrDuplicateRelease) {
		t.Fatalf("second stale Release() error = %v", err)
	}

	current, _ := system.Object(thirdTickets[0].AllocationSeq)
	if current.State != ObjectHeld || len(current.ClaimedBySubmissionIDs) != 1 {
		t.Fatalf("new submission range was disturbed: %+v", current)
	}
	if _, err := system.Complete(third.ID); err != nil {
		t.Fatal(err)
	}
	if err := system.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestTimeoutAndDuplicateRelease(t *testing.T) {
	system, clock := newTestSystem(t, 128, 50*time.Millisecond)

	timedOut, tickets, err := system.Submit(SubmitRequest{FrameID: 4, Sizes: []int{48}})
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(49 * time.Millisecond)
	if expired := system.ExpireTimeouts(); len(expired) != 0 {
		t.Fatalf("expired too early: %+v", expired)
	}

	clock.Advance(time.Millisecond)
	expired := system.ExpireTimeouts()
	if len(expired) != 1 || expired[0].ID != timedOut.ID || expired[0].State != StateCanceled {
		t.Fatalf("ExpireTimeouts() = %+v", expired)
	}
	object, _ := system.Object(tickets[0].AllocationSeq)
	if object.ReleaseReason != ReasonTimeout || object.State != ObjectReclaimed {
		t.Fatalf("timeout object = %+v", object)
	}

	_, err = system.Release(tickets[0], timedOut.ID, ReasonTimeout)
	if !errors.Is(err, ErrDuplicateRelease) {
		t.Fatalf("repeat Release() error = %v, want ErrDuplicateRelease", err)
	}
	object, _ = system.Object(tickets[0].AllocationSeq)
	if object.DuplicateReleaseAttempts != 1 {
		t.Fatalf("duplicate attempts = %d, want 1", object.DuplicateReleaseAttempts)
	}
	if _, err := system.Cancel(timedOut.ID, ""); !errors.Is(err, ErrAlreadyTerminal) {
		t.Fatalf("repeat Cancel() error = %v", err)
	}
	if err := system.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestExplicitDoubleFreePreservesFirstReason(t *testing.T) {
	system, _ := newTestSystem(t, 64, 0)
	ticket, err := system.Allocate(32)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := system.Release(ticket, 0, ReasonExplicit); err != nil {
		t.Fatalf("first Release() error = %v", err)
	}
	_, err = system.Release(ticket, 0, "")
	if !errors.Is(err, ErrDuplicateRelease) {
		t.Fatalf("second Release() error = %v, want duplicate", err)
	}
	object, _ := system.Object(ticket.AllocationSeq)
	if object.ReleaseReason != ReasonExplicit {
		t.Fatalf("reason changed to %q", object.ReleaseReason)
	}
	if object.DuplicateReleaseAttempts != 1 {
		t.Fatalf("duplicate attempts = %d", object.DuplicateReleaseAttempts)
	}
	if err := system.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}
