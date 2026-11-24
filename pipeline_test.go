package notify

import (
	"testing"
	"time"
)

const testNotificationID = "system-notification"

func baseNotification() State {
	return State{
		ID:        testNotificationID,
		Revision:  1,
		Closed:    false,
		Position:  Vec2{X: 120, Y: 24},
		Opacity:   0.8,
		UpdatedAt: 0,
	}
}

func updateFrom(state State, revision int64, at time.Duration, mutate func(*State)) Update {
	return updateState(state, revision, at, mutate)
}

func mustCommit(t *testing.T, p *Pipeline) *Frame {
	t.Helper()
	frame, err := p.Commit()
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	return frame
}

func TestBeforeDeadlinePositionUpdateIsCommitted(t *testing.T) {
	p := NewPipeline(DefaultConfig())
	if err := p.SetInitial(baseNotification()); err != nil {
		t.Fatal(err)
	}

	schedule := Schedule{Number: 1, Deadline: 8 * time.Millisecond, CommitAt: 10 * time.Millisecond, ScanoutAt: 16 * time.Millisecond}
	if err := p.BeginFrame(schedule); err != nil {
		t.Fatal(err)
	}

	input := updateFrom(baseNotification(), 2, 4*time.Millisecond, func(s *State) {
		s.Position.X = 180
	})
	record, err := p.Inject(input)
	if err != nil {
		t.Fatal(err)
	}

	frame := mustCommit(t, p)
	if record.Timing != TimingBeforeDeadline || record.TargetFrame != 1 {
		t.Fatalf("record timing = %q target %d", record.Timing, record.TargetFrame)
	}
	if record.Outcome != OutcomeCommitted {
		t.Fatalf("outcome = %q, want committed", record.Outcome)
	}

	state, ok := frame.State(testNotificationID)
	if !ok || state.Revision != 2 {
		t.Fatalf("committed state = %+v ok=%v, want revision 2", state, ok)
	}
	diff := frame.Diffs[testNotificationID]
	if !diff.HasPositionChange() || diff.HasOpacityChange() || diff.HasCloseChange() {
		t.Fatalf("diff kinds = %v, want position only", diff.Kinds())
	}
	if diff.Position.Delta.X != 60 || diff.Position.Delta.Y != 0 {
		t.Fatalf("position delta = %+v, want {60 0}", diff.Position.Delta)
	}
}

func TestLateOpacityUpdateOnlyEntersNextFrame(t *testing.T) {
	p := NewPipeline(DefaultConfig())
	if err := p.SetInitial(baseNotification()); err != nil {
		t.Fatal(err)
	}

	if err := p.BeginFrame(Schedule{Number: 1, Deadline: 8 * time.Millisecond, CommitAt: 10 * time.Millisecond, ScanoutAt: 16 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	onTime := updateFrom(baseNotification(), 2, 4*time.Millisecond, func(s *State) {
		s.Position.X = 180
	})
	if _, err := p.Inject(onTime); err != nil {
		t.Fatal(err)
	}
	first := mustCommit(t, p)

	late := updateFrom(first.States[testNotificationID], 3, 9*time.Millisecond, func(s *State) {
		s.Opacity = 0.35
	})
	record, err := p.Inject(late)
	if err != nil {
		t.Fatal(err)
	}
	if record.Timing != TimingLate || record.TargetFrame != 2 {
		t.Fatalf("late record timing=%q target=%d", record.Timing, record.TargetFrame)
	}

	scanoutOne, err := p.Scanout(1)
	if err != nil {
		t.Fatal(err)
	}
	if got := scanoutOne.States[testNotificationID]; got.Revision != 2 || got.Opacity != 0.8 {
		t.Fatalf("scanout frame 1 = %+v, want revision 2 opacity 0.8", got)
	}

	if err := p.BeginFrame(Schedule{Number: 2, Deadline: 24 * time.Millisecond, CommitAt: 26 * time.Millisecond, ScanoutAt: 32 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	second := mustCommit(t, p)
	if record.Outcome != OutcomeCommitted {
		t.Fatalf("late outcome after second commit = %q", record.Outcome)
	}

	state := second.States[testNotificationID]
	if state.Revision != 3 || !nearlyEqual(state.Opacity, 0.35) {
		t.Fatalf("second frame state = %+v, want revision 3 opacity .35", state)
	}
	diff := second.Diffs[testNotificationID]
	if diff.HasPositionChange() || !diff.HasOpacityChange() {
		t.Fatalf("second frame diff kinds=%v, want opacity only", diff.Kinds())
	}

	scanoutOne.States[testNotificationID] = State{Revision: 999}
	again, err := p.Scanout(1)
	if err != nil {
		t.Fatal(err)
	}
	if again.States[testNotificationID].Revision != 2 {
		t.Fatalf("scanout snapshot mutated to revision %d", again.States[testNotificationID].Revision)
	}
}
