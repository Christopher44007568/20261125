package binocular

import (
	"fmt"
	"runtime"
	"time"
)

type ScenarioOptions struct {
	Cycles        int
	SettleTimeout time.Duration
}

func (o ScenarioOptions) normalized() ScenarioOptions {
	if o.Cycles <= 0 {
		o.Cycles = 2
	}
	if o.SettleTimeout <= 0 {
		o.SettleTimeout = 2 * time.Second
	}
	return o
}

// RunScenario performs repeated failure/reconnect cycles for one disconnect
// scope. Every cycle starts a live binocular workload, tears down the session,
// reconnects, tries old leases against the new session, and verifies the first
// frame against the canonical empty-session frame.
func RunScenario(scope DisconnectScope, options ScenarioOptions) (ScenarioReport, error) {
	if !scope.Valid() {
		return ScenarioReport{}, ErrInvalidScope
	}
	options = options.normalized()
	device := NewDevice()
	report := ScenarioReport{Scope: scope}

	for cycle := 1; cycle <= options.Cycles; cycle++ {
		old := device.CurrentSession()
		if old == nil {
			return report, ErrNoSession
		}
		workload, err := old.PrepareWorkload()
		if err != nil {
			return report, fmt.Errorf("prepare workload: %w", err)
		}
		if !old.WaitUntilReady(8, 2, 4, options.SettleTimeout) {
			running, queued, total := old.TaskCounts()
			return report, fmt.Errorf("workload did not settle: running=%d queued=%d total=%d", running, queued, total)
		}

		start := time.Now()
		release, err := device.Disconnect(scope)
		if err != nil {
			return report, fmt.Errorf("disconnect: %w", err)
		}
		next, setup, err := device.Reconnect()
		if err != nil {
			return report, fmt.Errorf("reconnect: %w", err)
		}
		finished := time.Now()

		private := CheckPrivateState(old, next)
		stale := CheckStaleWrites(old, next, workload)
		frame, frameErr := CheckFirstFrame(device.Baseline(), next)
		unreleased := device.UnreleasedObjects()
		if unreleased == nil {
			unreleased = []ObjectRef{}
		}
		clean := releaseOK(release) &&
			len(unreleased) == 0 &&
			private.Clean &&
			!anyStaleAccepted(stale) &&
			frameErr == nil &&
			frame.Equal

		total := finished.Sub(start)
		cycleReport := CycleReport{
			Cycle:   cycle,
			Scope:   scope,
			Release: release,
			Reconnect: ReconnectMetrics{
				Attempt:    cycle,
				SessionID:  next.ID(),
				Generation: next.Generation(),
				TeardownNS: release.TeardownNS,
				SetupNS:    setup.Nanoseconds(),
				TotalNS:    total.Nanoseconds(),
				TotalMS:    float64(total.Nanoseconds()) / 1e6,
			},
			Unreleased:   unreleased,
			PrivateState: private,
			StaleWrites:  stale,
			FirstFrame:   frame,
			Clean:        clean,
		}
		if frameErr != nil {
			cycleReport.Unreleased = append(cycleReport.Unreleased, ObjectRef{
				Kind: "frame", ID: "first-frame", Session: next.ID(), Detail: frameErr.Error(),
			})
		}
		if cycleReport.Reconnect.TotalNS < 0 {
			cycleReport.Reconnect.TotalNS = 0
		}
		report.Cycles = append(report.Cycles, cycleReport)
	}

	report.Clean = len(report.Cycles) > 0
	for _, cycle := range report.Cycles {
		if !cycle.Clean {
			report.Clean = false
			break
		}
	}
	return report, nil
}

func BuildReport(scenarios []ScenarioReport) Report {
	report := Report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		GoVersion:   runtime.Version(),
		Scenarios:   scenarios,
		Summary:     Summary{AllClean: true},
	}
	for _, scenario := range scenarios {
		if !scenario.Clean {
			report.Summary.AllClean = false
		}
		for _, cycle := range scenario.Cycles {
			report.Summary.TotalUnreleased += len(cycle.Unreleased)
			if cycle.FirstFrame.DiffBytes > report.Summary.MaxFirstFrameDiff {
				report.Summary.MaxFirstFrameDiff = cycle.FirstFrame.DiffBytes
			}
			report.Summary.ReconnectLatencies = append(report.Summary.ReconnectLatencies, cycle.Reconnect.TotalNS)
		}
	}
	if len(scenarios) == 0 {
		report.Summary.AllClean = false
	}
	return report
}
