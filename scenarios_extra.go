package notify

import (
	"fmt"
	"time"
)

func injectRevision(p *Pipeline, current State, revision int64, at time.Duration, mutate func(*State)) State {
	update := updateState(current, revision, at, mutate)
	record := mustInject(p, update)
	return record.Update.State
}

func continuousCase() *CaseReport {
	old := DefaultOldState()
	p := NewPipeline(DefaultConfig())
	mustSetInitial(p, old)
	mustBegin(p, frameSchedule(1))

	current := old
	current = injectRevision(p, current, 2, 2*time.Millisecond, func(state *State) {
		state.Position.X = 140
	})
	current = injectRevision(p, current, 3, 4*time.Millisecond, func(state *State) {
		state.Opacity = 0.6
	})
	current = injectRevision(p, current, 4, 6*time.Millisecond, func(state *State) {
		state.Position.Y = 40
		state.Opacity = 0.4
	})
	current = injectRevision(p, current, 5, 9*time.Millisecond, func(state *State) {
		state.Position.X = 200
	})
	current = injectRevision(p, current, 6, 10*time.Millisecond, func(state *State) {
		state.Opacity = 0.2
	})

	mustCommitPipeline(p)
	mustBegin(p, frameSchedule(2))
	mustCommitPipeline(p)

	events := p.Events()
	report := newCaseReport(
		"continuous-coalesced-updates",
		"Continuous updates",
		"Rapid revisions are coalesced; intermediate revisions remain recorded as superseded.",
		old, p, 1,
	)

	report.check("all_five_revisions_recorded", len(events) == 5,
		fmt.Sprintf("events=%d", len(events)))
	report.check("ontime_intermediate_superseded",
		events[0].Outcome == OutcomeSuperseded && events[1].Outcome == OutcomeSuperseded,
		fmt.Sprintf("rev2=%s rev3=%s", events[0].Outcome, events[1].Outcome))
	report.check("late_intermediate_superseded", events[3].Outcome == OutcomeSuperseded,
		fmt.Sprintf("rev5=%s", events[3].Outcome))
	report.check("ontime_latest_committed", report.Committed.Revision == 4,
		fmt.Sprintf("committed revision=%d", report.Committed.Revision))
	report.check("late_latest_reaches_next_frame", report.Final.Revision == 6,
		fmt.Sprintf("final revision=%d", report.Final.Revision))
	report.check("position_event_distinguished",
		events[0].Change.HasPositionChange() && !events[0].Change.HasOpacityChange(),
		fmt.Sprintf("rev2 kinds=%v", events[0].Change.Kinds()))
	report.check("opacity_event_distinguished",
		events[1].Change.HasOpacityChange() && !events[1].Change.HasPositionChange(),
		fmt.Sprintf("rev3 kinds=%v", events[1].Change.Kinds()))
	report.check("cumulative_diff_has_both",
		report.CommittedDiff.HasPositionChange() && report.CommittedDiff.HasOpacityChange(),
		fmt.Sprintf("committed kinds=%v", report.CommittedDiff.Kinds()))
	return report
}

func closeCase() *CaseReport {
	old := DefaultOldState()
	p := NewPipeline(DefaultConfig())
	mustSetInitial(p, old)
	mustBegin(p, frameSchedule(1))

	current := injectRevision(p, old, 2, 4*time.Millisecond, func(state *State) {
		state.Position.X = 180
	})
	first := mustCommitPipeline(p)

	closeState := current
	closeState.Revision = 3
	closeState.Closed = true
	closeState.UpdatedAt = 9 * time.Millisecond
	mustInject(p, Update{At: 9 * time.Millisecond, State: closeState})

	mustSubmitWorld(p, WorldFrame{
		Number:        103,
		SubmittedAt:   11 * time.Millisecond,
		Notifications: []State{first.States[old.ID]},
	})
	scanoutOne, err := p.Scanout(1)
	if err != nil {
		panic(err)
	}

	mustBegin(p, frameSchedule(2))
	mustSubmitWorld(p, WorldFrame{
		Number:        104,
		SubmittedAt:   18 * time.Millisecond,
		Notifications: []State{old},
	})
	second := mustCommitPipeline(p)

	mustBegin(p, frameSchedule(3))
	third := mustCommitPipeline(p)

	report := newCaseReport(
		"close-tombstone-persists",
		"Close must not reappear",
		"A late close becomes a monotonic tombstone; stale open world snapshots are rejected in later frames.",
		old, p, 1,
	)
	receipts := p.WorldReceipts()
	allBlocked := len(receipts) == 2
	for _, receipt := range receipts {
		for _, verdict := range receipt.Verdicts {
			if !verdict.RegressionBlocked {
				allBlocked = false
			}
		}
	}

	report.check("committed_frame_still_visible_before_late_close",
		scanoutOne.States[old.ID].Visible(),
		fmt.Sprintf("closed=%v", scanoutOne.States[old.ID].Closed))
	report.check("second_frame_closed",
		second.States[old.ID].Closed && !second.States[old.ID].Visible(),
		fmt.Sprintf("state=%+v", second.States[old.ID]))
	report.check("third_frame_remains_closed",
		third.States[old.ID].Closed && !third.States[old.ID].Visible(),
		fmt.Sprintf("state=%+v", third.States[old.ID]))
	report.check("no_visible_notifications",
		len(second.VisibleIDs()) == 0 && len(third.VisibleIDs()) == 0,
		fmt.Sprintf("frame2=%v frame3=%v", second.VisibleIDs(), third.VisibleIDs()))
	report.check("old_open_world_snapshots_blocked", allBlocked,
		fmt.Sprintf("receipts=%d", len(receipts)))
	report.check("close_diff_recorded", report.CommittedToFinalDiff.HasCloseChange(),
		fmt.Sprintf("delta kinds=%v", report.CommittedToFinalDiff.Kinds()))
	return report
}
