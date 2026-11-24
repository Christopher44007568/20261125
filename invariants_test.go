package notify

import (
	"testing"
	"time"
)

func reviseLatest(t *testing.T, p *Pipeline, revision int64, at time.Duration, mutate func(*State)) Update {
	t.Helper()
	current, ok := p.Latest(testNotificationID)
	if !ok {
		t.Fatal("latest notification missing")
	}
	return updateFrom(current, revision, at, mutate)
}

func TestContinuousUpdatesCoalesceToLatestRevision(t *testing.T) {
	p := NewPipeline(DefaultConfig())
	if err := p.SetInitial(baseNotification()); err != nil {
		t.Fatal(err)
	}
	if err := p.BeginFrame(Schedule{Number: 1, Deadline: 8 * time.Millisecond, CommitAt: 12 * time.Millisecond, ScanoutAt: 16 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}

	first := updateFrom(baseNotification(), 2, 2*time.Millisecond, func(s *State) {
		s.Position.X = 140
	})
	if _, err := p.Inject(first); err != nil {
		t.Fatal(err)
	}
	second := reviseLatest(t, p, 3, 4*time.Millisecond, func(s *State) {
		s.Opacity = 0.6
	})
	if _, err := p.Inject(second); err != nil {
		t.Fatal(err)
	}
	third := reviseLatest(t, p, 4, 6*time.Millisecond, func(s *State) {
		s.Position.Y = 40
		s.Opacity = 0.4
	})
	if _, err := p.Inject(third); err != nil {
		t.Fatal(err)
	}

	lateFourth := reviseLatest(t, p, 5, 9*time.Millisecond, func(s *State) {
		s.Position.X = 200
	})
	if _, err := p.Inject(lateFourth); err != nil {
		t.Fatal(err)
	}
	lateFifth := reviseLatest(t, p, 6, 10*time.Millisecond, func(s *State) {
		s.Opacity = 0.2
	})
	if _, err := p.Inject(lateFifth); err != nil {
		t.Fatal(err)
	}

	frameOne := mustCommit(t, p)
	if got := frameOne.States[testNotificationID]; got.Revision != 4 || got.Position.Y != 40 || got.Opacity != 0.4 {
		t.Fatalf("frame one = %+v, want coalesced revision 4", got)
	}
	if len(frameOne.Inputs) != 1 || frameOne.Inputs[0].Update.State.Revision != 4 {
		t.Fatalf("frame one inputs = %+v, want only revision 4", frameOne.Inputs)
	}

	events := p.Events()
	if events[0].Outcome != OutcomeSuperseded || events[1].Outcome != OutcomeSuperseded {
		t.Fatalf("early outcomes = %q, %q; want superseded", events[0].Outcome, events[1].Outcome)
	}
	if !events[0].Change.HasPositionChange() || events[0].Change.HasOpacityChange() {
		t.Fatalf("revision 2 kinds = %v, want position only", events[0].Change.Kinds())
	}
	if events[1].Change.HasPositionChange() || !events[1].Change.HasOpacityChange() {
		t.Fatalf("revision 3 kinds = %v, want opacity only", events[1].Change.Kinds())
	}
	if events[3].Outcome != OutcomeSuperseded {
		t.Fatalf("late revision 5 outcome = %q, want superseded", events[3].Outcome)
	}

	if err := p.BeginFrame(Schedule{Number: 2, Deadline: 24 * time.Millisecond, CommitAt: 26 * time.Millisecond, ScanoutAt: 32 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	frameTwo := mustCommit(t, p)
	if got := frameTwo.States[testNotificationID]; got.Revision != 6 || got.Position.X != 200 || got.Opacity != 0.2 {
		t.Fatalf("frame two = %+v, want late coalesced revision 6", got)
	}
	if len(frameTwo.Inputs) != 1 || frameTwo.Inputs[0].Timing != TimingLate {
		t.Fatalf("frame two inputs = %+v, want one late input", frameTwo.Inputs)
	}
}

func TestWorldFrameWithOldRevisionCannotRestoreOldVersion(t *testing.T) {
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
	frameOne := mustCommit(t, p)

	late := updateFrom(frameOne.States[testNotificationID], 3, 9*time.Millisecond, func(s *State) {
		s.Opacity = 0.35
	})
	if _, err := p.Inject(late); err != nil {
		t.Fatal(err)
	}

	oldWorld := WorldFrame{
		Number:      91,
		SubmittedAt: 11 * time.Millisecond,
		Notifications: []State{
			frameOne.States[testNotificationID],
			baseNotification(),
		},
	}
	receipt, err := p.SubmitWorldFrame(oldWorld)
	if err != nil {
		t.Fatal(err)
	}
	for _, verdict := range receipt.Verdicts {
		if !verdict.RegressionBlocked || verdict.Applied {
			t.Fatalf("verdict %+v should block regression", verdict)
		}
	}

	scanout, err := p.Scanout(1)
	if err != nil {
		t.Fatal(err)
	}
	if scanout.States[testNotificationID].Revision != 2 {
		t.Fatalf("scanout revision = %d, want committed 2", scanout.States[testNotificationID].Revision)
	}

	if err := p.BeginFrame(Schedule{Number: 2, Deadline: 24 * time.Millisecond, CommitAt: 26 * time.Millisecond, ScanoutAt: 32 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	frameTwo := mustCommit(t, p)
	if got := frameTwo.States[testNotificationID]; got.Revision != 3 || got.Opacity != 0.35 {
		t.Fatalf("frame two = %+v, want newer late revision 3", got)
	}
}

func TestClosedStateDoesNotReappearInNextFrame(t *testing.T) {
	p := NewPipeline(DefaultConfig())
	if err := p.SetInitial(baseNotification()); err != nil {
		t.Fatal(err)
	}
	if err := p.BeginFrame(Schedule{Number: 1, Deadline: 8 * time.Millisecond, CommitAt: 10 * time.Millisecond, ScanoutAt: 16 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	positionUpdate := updateFrom(baseNotification(), 2, 4*time.Millisecond, func(s *State) {
		s.Position.X = 180
	})
	if _, err := p.Inject(positionUpdate); err != nil {
		t.Fatal(err)
	}
	frameOne := mustCommit(t, p)

	closeUpdate := updateFrom(frameOne.States[testNotificationID], 3, 9*time.Millisecond, func(s *State) {
		s.Closed = true
	})
	if _, err := p.Inject(closeUpdate); err != nil {
		t.Fatal(err)
	}
	staleOpenWorld := WorldFrame{
		Number:        92,
		SubmittedAt:   11 * time.Millisecond,
		Notifications: []State{frameOne.States[testNotificationID]},
	}
	if _, err := p.SubmitWorldFrame(staleOpenWorld); err != nil {
		t.Fatal(err)
	}

	scanoutOne, err := p.Scanout(1)
	if err != nil {
		t.Fatal(err)
	}
	if !scanoutOne.States[testNotificationID].Visible() {
		t.Fatal("late close must not rewrite already committed frame one")
	}

	if err := p.BeginFrame(Schedule{Number: 2, Deadline: 24 * time.Millisecond, CommitAt: 26 * time.Millisecond, ScanoutAt: 32 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	olderOpen := WorldFrame{
		Number:        93,
		SubmittedAt:   18 * time.Millisecond,
		Notifications: []State{baseNotification()},
	}
	receipt, err := p.SubmitWorldFrame(olderOpen)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Verdicts[0].RegressionBlocked {
		t.Fatalf("old open verdict = %+v, want blocked", receipt.Verdicts[0])
	}

	frameTwo := mustCommit(t, p)
	closed := frameTwo.States[testNotificationID]
	if !closed.Closed || closed.Visible() {
		t.Fatalf("frame two state = %+v, want closed/invisible", closed)
	}
	if ids := frameTwo.VisibleIDs(); len(ids) != 0 {
		t.Fatalf("visible ids = %v, want none", ids)
	}
	if !frameTwo.Diffs[testNotificationID].HasCloseChange() {
		t.Fatalf("frame two diff = %+v, want closed change", frameTwo.Diffs[testNotificationID])
	}

	if err := p.BeginFrame(Schedule{Number: 3, Deadline: 40 * time.Millisecond, CommitAt: 42 * time.Millisecond, ScanoutAt: 48 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	frameThree := mustCommit(t, p)
	if got := frameThree.States[testNotificationID]; !got.Closed || got.Visible() {
		t.Fatalf("frame three = %+v, closed tombstone was lost", got)
	}
	if ids := frameThree.VisibleIDs(); len(ids) != 0 {
		t.Fatalf("frame three visible ids = %v, want none", ids)
	}
}
