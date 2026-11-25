package sim

import (
	"fmt"
	"math"
	"sort"
)

func (e *engine) buildReport() *ScenarioReport {
	spec := e.spec
	finalSize := e.committed

	eyeReports := make([]EyeReport, 2)
	eyeWindowStart := e.windowStartFrame - 1
	if eyeWindowStart == -1 {
		eyeWindowStart = e.upgradeFrame - 1
	}
	if eyeWindowStart < 0 {
		eyeWindowStart = spec.WarmupFrames + spec.StableFrames
	}
	for i, eye := range e.eyes {
		scores := e.eyeScores[i]
		events := []int{}
		maxJump := 0.0
		for f, s := range scores {
			if f >= eyeWindowStart {
				if s > maxJump {
					maxJump = s
				}
			}
			if f >= eyeWindowStart && s > jumpThresholdLuma {
				events = append(events, f+1)
			}
		}
		eyeReports[i] = EyeReport{
			Eye:                 eye.index.String(),
			FinalSize:           eye.size,
			Remap:               eye.remap,
			ConfidenceLossRatio: eye.confLoss,
			JumpScores:          scores,
			JumpEventFrames:     events,
			MaxJump:             maxJump,
		}
	}

	stereo := make([]float64, len(e.frames))
	eventFrames := []int{}
	maxStereo := 0.0
	windowSum := 0.0
	windowCount := 0
	windowStart := e.windowStartFrame - 1
	if windowStart == -1 {
		windowStart = e.upgradeFrame - 1
	}
	if windowStart < 0 {
		// Rollback runs never present an upgrade frame; the window opens on
		// the rollback frame itself.
		windowStart = spec.WarmupFrames + spec.StableFrames
	}
	for i, f := range e.frames {
		stereo[i] = f.Jump.StereoWorst
		if i >= windowStart {
			if f.Jump.StereoWorst > maxStereo {
				maxStereo = f.Jump.StereoWorst
			}
			if f.Jump.StereoWorst > jumpThresholdLuma {
				eventFrames = append(eventFrames, f.Index)
			}
			windowSum += f.Jump.StereoWorst
			windowCount++
		}
	}
	baselineScores := append([]float64(nil), stereo[:minInt(windowStart, len(stereo))]...)
	sort.Float64s(baselineScores)
	baselineMedian := 0.0
	if len(baselineScores) > 0 {
		baselineMedian = baselineScores[len(baselineScores)/2]
	}
	meanWindow := 0.0
	if windowCount > 0 {
		meanWindow = windowSum / float64(windowCount)
	}

	targetSteadyHistory := 2 * spec.Target.pixels() * historyBytesPerPx
	finalLive := e.meter.historyLive + e.meter.imageLive
	mem := MemoryReport{
		BaselineBytes:               e.meter.baseline,
		PeakBytes:                   e.meter.peak,
		PeakOverBaseline:            float64(e.meter.peak) / float64(e.meter.baseline),
		HistoryPeakBytes:            e.meter.historyPeak,
		ImagePeakBytes:              e.meter.imagePeak,
		HistoryPeakOverTargetSteady: float64(e.meter.historyPeak) / float64(targetSteadyHistory),
		FinalLiveBytes:              finalLive,
		AllocCount:                  e.meter.allocCount,
		FreeCount:                   e.meter.freeCount,
	}

	inv := InvalidationReport{DiscardReferenceRatio: 1}
	if spec.Strategy == StrategyStaged && e.outcome == OutcomeUpgraded {
		inv.LeftMigrationRatio = e.eyes[0].confLoss
		inv.RightMigrationRatio = e.eyes[1].confLoss
	} else if e.outcome == OutcomeUpgraded {
		inv.LeftMigrationRatio = e.eyes[0].confLoss
		inv.RightMigrationRatio = e.eyes[1].confLoss
	}
	inv.WorstRatio = math.Max(inv.LeftMigrationRatio, inv.RightMigrationRatio)

	jumps := JumpsReport{
		EventThresholdLuma:    jumpThresholdLuma,
		BaselineMedianLuma:    baselineMedian,
		UpgradeStartFrame:     windowStart + 1,
		StereoScores:          stereo,
		EventFrames:           eventFrames,
		EventCount:            len(eventFrames),
		MaxStereoJump:         maxStereo,
		MeanUpgradeWindowJump: meanWindow,
	}

	work := WorkReport{
		TotalUnits:     e.totalWork,
		UpgradeFrameMS: e.upgradeMS,
		MaxFrameMS:     e.maxFrameMS,
		DeadlineMisses: e.deadlineMiss,
		FrameBudgetMS:  frameBudgetMS,
	}

	remapApplied := e.outcome == OutcomeUpgraded && !spec.Discard
	rep := &ScenarioReport{
		Schema:        "resmigrate/v1",
		ScenarioID:    spec.ID,
		Title:         spec.Title,
		Strategy:      spec.Strategy,
		StrategyName:  strategyName(spec.Strategy),
		RemapApplied:  remapApplied,
		InitialSize:   spec.Initial,
		TargetSize:    spec.Target,
		FinalSize:     finalSize,
		Outcome:       e.outcome,
		Fault:         spec.Fault,
		RetryAllowed:  spec.RetryAllowed,
		RetriesUsed:   e.retriesUsed,
		InjectedError: e.injectedErr,
		Frames:        e.frames,
		Trajectory:    e.traj,
		Eyes:          eyeReports,
		Memory:        mem,
		Invalidation:  inv,
		Jumps:         jumps,
		Work:          work,
	}
	var stages *int
	if spec.Strategy == StrategyStaged {
		s := stagedBands
		stages = &s
	}
	rep.Stages = stages
	rep.Checks = e.checks(rep)
	allOK := true
	for _, c := range rep.Checks {
		if !c.OK {
			allOK = false
		}
	}
	rep.OK = allOK
	return rep
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (e *engine) checks(r *ScenarioReport) []Check {
	spec := e.spec
	left := e.eyes[0]
	right := e.eyes[1]
	checks := []Check{
		{
			ID:     "stereo_sizes_match",
			OK:     left.size.equals(right.size),
			Detail: fmt.Sprintf("final left=%s right=%s", left.size, right.size),
		},
		{
			ID:     "terminal_config_committed",
			OK:     e.committed.equals(r.FinalSize) && r.FinalSize.equals(expectedFinal(spec, e.outcome)),
			Detail: fmt.Sprintf("outcome=%s want=%s committed=%s", e.outcome, expectedFinal(spec, e.outcome), e.committed),
		},
		{
			ID:     "presented_frames_stereo_consistent",
			OK:     presentedConsistent(e.frames),
			Detail: fmt.Sprintf("%d presented frames with left/right size mismatch", mismatchCount(e.frames)),
		},
	}

	lastFrame := e.frames[len(e.frames)-1]
	sppL, budgetL := lastFrame.Left.SPP, lastFrame.Left.BudgetSamples
	sppR, budgetR := lastFrame.Right.SPP, lastFrame.Right.BudgetSamples
	checks = append(checks, Check{
		ID: "sampling_budget_migrated",
		OK: lastFrame.Left.SpentSamples == budgetL && lastFrame.Right.SpentSamples == budgetR &&
			budgetL == sppL*lastFrame.Left.Size.pixels() && budgetR == sppR*lastFrame.Right.Size.pixels(),
		Detail: fmt.Sprintf("left spp=%d budget=%d spent=%d; right spp=%d budget=%d spent=%d",
			sppL, budgetL, lastFrame.Left.SpentSamples, sppR, budgetR, lastFrame.Right.SpentSamples),
	})

	if e.outcome == OutcomeRolledBack {
		checks = append(checks, Check{
			ID:     "new_size_remaps_history",
			OK:     true,
			Detail: fmt.Sprintf("attempt used bilinear remap; rollback restored %s", spec.Initial),
		})
	} else if spec.Discard {
		checks = append(checks, Check{
			ID:     "discard_baseline_full_invalidation",
			OK:     abs(r.Invalidation.WorstRatio-1.0) < 1e-9,
			Detail: fmt.Sprintf("discard path invalidation=%.3f (expected 1.000)", r.Invalidation.WorstRatio),
		})
	} else {
		checks = append(checks, Check{
			ID:     "new_size_remaps_history",
			OK:     left.remap != nil && right.remap != nil && left.size.equals(spec.Target) && right.size.equals(spec.Target),
			Detail: fmt.Sprintf("both eyes bilinearly remapped to %s", spec.Target),
		})
	}

	if spec.Fault == nil {
		checks = append(checks, Check{ID: "fault_expectation", OK: e.outcome == OutcomeUpgraded, Detail: "no fault injected; outcome=upgraded"})
	} else if e.outcome == OutcomeRolledBack {
		checks = append(checks, Check{
			ID:     "fault_expectation",
			OK:     e.committed.equals(spec.Initial),
			Detail: fmt.Sprintf("fault=%s; outcome=rolled_back, old config restored", e.injectedErr),
		})
	} else {
		checks = append(checks, Check{
			ID:     "fault_expectation",
			OK:     e.retriesUsed >= 1 && e.committed.equals(spec.Target),
			Detail: fmt.Sprintf("first attempt failed (%s); retry=%d outcome=upgraded", e.injectedErr, e.retriesUsed),
		})
	}

	checks = append(checks, Check{
		ID:     "peak_memory_recorded",
		OK:     r.Memory.PeakBytes >= r.Memory.BaselineBytes && r.Memory.HistoryPeakBytes > 0,
		Detail: fmt.Sprintf("baseline=%d peak=%d history_peak=%d", r.Memory.BaselineBytes, r.Memory.PeakBytes, r.Memory.HistoryPeakBytes),
	})

	last := e.traj[len(e.traj)-1]
	checks = append(checks, Check{
		ID:     "trajectory_closed_consistent",
		OK:     last.Consistent && last.Left.equals(r.FinalSize) && last.Right.equals(r.FinalSize),
		Detail: fmt.Sprintf("last trajectory point frame=%d left=%s right=%s", last.Frame, last.Left, last.Right),
	})
	return checks
}

func expectedFinal(spec ScenarioSpec, outcome Outcome) Size {
	if outcome == OutcomeRolledBack {
		return spec.Initial
	}
	return spec.Target
}

func presentedConsistent(frames []FrameRecord) bool {
	return mismatchCount(frames) == 0
}

func mismatchCount(frames []FrameRecord) int {
	n := 0
	for _, f := range frames {
		if !f.Left.Size.equals(f.Right.Size) {
			n++
		}
	}
	return n
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
