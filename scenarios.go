package notify

import (
	"fmt"
	"time"
)

// DefaultNotificationID is used by the bundled deterministic scenarios.
const DefaultNotificationID = "system-notification"

// DefaultOldState is the known old version before injections.
func DefaultOldState() State {
	return State{
		ID:        DefaultNotificationID,
		Revision:  1,
		Position:  Vec2{X: 120, Y: 24},
		Opacity:   0.8,
		UpdatedAt: 0,
	}
}

func frameSchedule(number int) Schedule {
	start := Millis(int64(number-1) * 16)
	return Schedule{
		Number:    number,
		Deadline:  start + 8*time.Millisecond,
		CommitAt:  start + 10*time.Millisecond,
		ScanoutAt: start + 16*time.Millisecond,
	}
}

func mustSetInitial(p *Pipeline, states ...State) {
	if err := p.SetInitial(states...); err != nil {
		panic(err)
	}
}

func mustBegin(p *Pipeline, schedule Schedule) {
	if err := p.BeginFrame(schedule); err != nil {
		panic(err)
	}
}

func mustInject(p *Pipeline, update Update) *InputRecord {
	record, err := p.Inject(update)
	if err != nil {
		panic(err)
	}
	return record
}

func mustCommitPipeline(p *Pipeline) *Frame {
	frame, err := p.Commit()
	if err != nil {
		panic(err)
	}
	return frame
}

func mustSubmitWorld(p *Pipeline, frame WorldFrame) *WorldReceipt {
	receipt, err := p.SubmitWorldFrame(frame)
	if err != nil {
		panic(err)
	}
	return receipt
}

func beforeDeadlineCase() *CaseReport {
	old := DefaultOldState()
	p := NewPipeline(DefaultConfig())
	mustSetInitial(p, old)
	mustBegin(p, frameSchedule(1))

	update := updateState(old, 2, 4*time.Millisecond, func(state *State) {
		state.Position.X = 180
	})
	mustInject(p, update)
	mustCommitPipeline(p)

	receipt := mustSubmitWorld(p, WorldFrame{
		Number:        101,
		SubmittedAt:   11 * time.Millisecond,
		Notifications: []State{old},
	})
	scanout, err := p.Scanout(1)
	if err != nil {
		panic(err)
	}

	report := newCaseReport(
		"before-deadline-position",
		"Before deadline: position",
		"A position revision arrives before the input deadline and is committed in frame 1.",
		old, p, 1,
	)
	report.check("committed_revision", report.Committed.Revision == 2,
		fmt.Sprintf("committed revision=%d", report.Committed.Revision))
	report.check("position_change_recorded", report.CommittedDiff.HasPositionChange(),
		fmt.Sprintf("kinds=%v", report.CommittedDiff.Kinds()))
	report.check("opacity_unchanged", !report.CommittedDiff.HasOpacityChange(),
		fmt.Sprintf("opacity=%.2f", report.Committed.Opacity))
	report.check("stale_world_rejected", receipt.Verdicts[0].RegressionBlocked,
		receipt.Verdicts[0].Reason)
	report.check("scanout_matches_commit", scanout.States[old.ID].Revision == report.Committed.Revision,
		fmt.Sprintf("scanout revision=%d", scanout.States[old.ID].Revision))
	return report
}

func lateCase() *CaseReport {
	old := DefaultOldState()
	p := NewPipeline(DefaultConfig())
	mustSetInitial(p, old)
	mustBegin(p, frameSchedule(1))

	position := updateState(old, 2, 4*time.Millisecond, func(state *State) {
		state.Position.X = 180
	})
	mustInject(p, position)
	first := mustCommitPipeline(p)

	opacity := updateState(first.States[old.ID], 3, 9*time.Millisecond, func(state *State) {
		state.Opacity = 0.35
	})
	lateRecord := mustInject(p, opacity)

	receipt := mustSubmitWorld(p, WorldFrame{
		Number:        102,
		SubmittedAt:   11 * time.Millisecond,
		Notifications: []State{first.States[old.ID]},
	})
	scanoutOne, err := p.Scanout(1)
	if err != nil {
		panic(err)
	}

	mustBegin(p, frameSchedule(2))
	mustCommitPipeline(p)
	scanoutTwo, err := p.Scanout(2)
	if err != nil {
		panic(err)
	}

	report := newCaseReport(
		"late-opacity-next-frame",
		"Late arrival: opacity",
		"Revision 2 is committed; revision 3 arrives after the deadline and only enters frame 2.",
		old, p, 1,
	)
	report.check("frame_one_keeps_committed_revision", scanoutOne.States[old.ID].Revision == 2,
		fmt.Sprintf("frame 1 revision=%d", scanoutOne.States[old.ID].Revision))
	report.check("late_input_targets_frame_two", lateRecord.Timing == TimingLate && lateRecord.TargetFrame == 2,
		fmt.Sprintf("timing=%s target=%d", lateRecord.Timing, lateRecord.TargetFrame))
	report.check("frame_two_has_opacity", scanoutTwo.States[old.ID].Revision == 3 && nearlyEqual(scanoutTwo.States[old.ID].Opacity, 0.35),
		fmt.Sprintf("frame 2=%+v", scanoutTwo.States[old.ID]))
	report.check("committed_diff_is_position", report.CommittedDiff.HasPositionChange() && !report.CommittedDiff.HasOpacityChange(),
		fmt.Sprintf("committed kinds=%v", report.CommittedDiff.Kinds()))
	report.check("committed_to_final_is_opacity", report.CommittedToFinalDiff.HasOpacityChange() && !report.CommittedToFinalDiff.HasPositionChange(),
		fmt.Sprintf("delta kinds=%v", report.CommittedToFinalDiff.Kinds()))
	report.check("world_cannot_restore_revision_two", receipt.Verdicts[0].RegressionBlocked,
		fmt.Sprintf("world=%d latest=%d", receipt.Verdicts[0].WorldRevision, receipt.Verdicts[0].LatestRevision))
	return report
}

func updateState(state State, revision int64, at time.Duration, mutate func(*State)) Update {
	state.Revision = revision
	state.UpdatedAt = at
	if mutate != nil {
		mutate(&state)
	}
	return Update{At: at, State: state}
}
