package sim

// Run executes one scenario deterministically and builds its report.
func Run(spec ScenarioSpec) *ScenarioReport {
	e := newEngine(spec)

	for i := 0; i < spec.WarmupFrames; i++ {
		e.present("warmup", e.steadyWork(), nil)
	}
	for i := 0; i < spec.StableFrames; i++ {
		e.present("stable", e.steadyWork(), nil)
	}

	var attemptErr error
	attempt := 1
	for {
		switch spec.Strategy {
		case StrategyInplace:
			attemptErr = e.runInplace(attempt)
		case StrategyDoubleBuf:
			attemptErr = e.runDoubleBuf(attempt)
		case StrategyStaged:
			attemptErr = e.runStaged(attempt)
		}
		if attemptErr == nil {
			break
		}
		e.injectedErr = attemptErr.Error()
		// Every failed attempt produces a visible rollback frame on the old
		// configuration before either retrying or terminating, so no
		// half-migrated state is ever presented.
		ev := "one-eye-failure-rolled-back"
		e.present("rollback", e.steadyWork(), &ev)
		if !spec.RetryAllowed {
			e.outcome = OutcomeRolledBack
			break
		}
		e.retriesUsed++
		attempt++
		e.kindOverride = "upgrade-retry"
	}

	for i := 0; i < spec.PostFrames; i++ {
		e.present("post", e.steadyWork(), nil)
	}
	return e.buildReport()
}

func (e *engine) steadyWork() int {
	work := 0
	for _, eye := range e.eyes {
		spp, samples := e.budgetFor(eye.size)
		_ = spp
		work += samples
	}
	return work
}

// upgradeWork models sampling at the target size plus 4-tap bilinear remap
// work for every destination texel of every eye.
func (e *engine) upgradeWork() int {
	work := 0
	for range e.eyes {
		_, samples := e.budgetFor(e.spec.Target)
		work += samples
		if !e.spec.Discard {
			// The discard control group skips the 4-tap history remap.
			work += 4 * e.spec.Target.pixels()
		}
	}
	return work
}

// bandWork models one staged band: sampling for the band's target pixels plus
// 4-tap remap taps, for both eyes.
func (e *engine) bandWork(bandRows int) int {
	bandPixels := e.spec.Target.Width * bandRows
	work := 0
	for range e.eyes {
		// Every staged frame still samples the full target at the target SPP;
		// only the 4-tap remap work is limited to the band migrated this frame.
		_, samples := e.budgetFor(e.spec.Target)
		work += samples + 4*bandPixels
	}
	return work
}
