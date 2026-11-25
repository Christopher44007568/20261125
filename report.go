package main

import (
	"fmt"
	"runtime"
)

// BuildScenarioReport converts controller records and canonical frames into
// the audited scenario document.
func BuildScenarioReport(def ScenarioDef, cfg Config, c *Controller, frames []*FrameReport) *ScenarioReport {
	episodes := []*Episode{}
	for _, rt := range c.episodes {
		e := rt.ep
		if rt.active {
			e.EndFrame = c.Frame
			e.DurationFrames = c.Frame - e.StartFrame + 1
		}

		samples := append([]Metrics{}, rt.samples...)
		e.StartMetrics = preTransitionMetrics(cfg, frames, e.StartFrame)
		if e.StartMetrics != nil && e.StartFrame >= 1 && e.StartFrame <= len(frames) {
			// The reference start snapshot already shows the new gear index
			// while scale-derived scores still reflect the previous state.
			e.StartMetrics.SampleLevel = frames[e.StartFrame-1].Metrics.SampleLevel
		}

		worst := worstMetrics(samples)
		e.WorstMetrics = worst

		if e.FalsePositive {
			// Invalidated episodes carry no end snapshot by design.
			e.EndMetrics = nil
		} else if rt.active {
			last := samples[len(samples)-1]
			e.EndMetrics = &last
		} else if len(samples) > 0 {
			lastIdx := len(samples) - 1
			patchEndSnapshot(cfg, e.Mode, &samples[lastIdx])
			last := samples[lastIdx]
			e.EndMetrics = &last
		}
		e.Samples = samples
		episodes = append(episodes, &e)
	}

	chains := []*Chain{}
	for _, rt := range c.chains {
		ch := rt.ch
		ch.Classification = classifyChain(rt)
		end := ch.StartFrame
		for _, ep := range episodes {
			if ep.ChainID == ch.ID && ep.EndFrame > end {
				end = ep.EndFrame
			}
		}
		if rt.active && c.Frame > end {
			end = c.Frame
		}
		ch.EndFrame = end
		ch.DurationFrames = ch.EndFrame - ch.StartFrame + 1
		chains = append(chains, &ch)
	}

	alerts := c.Alerts
	safety := c.SafetyEvents
	return &ScenarioReport{
		ID: def.ID, Title: def.Title, Description: def.Description,
		Frames: frames, Episodes: episodes, Chains: chains,
		Alerts: alerts, Audit: c.Audit, SafetyEvents: safety, Shots: def.Shots,
	}
}

// preTransitionMetrics reproduces the capability snapshot immediately before
// the episode start frame, paired with that frame's timing signals.
func preTransitionMetrics(cfg Config, frames []*FrameReport, startFrame int) *Metrics {
	if startFrame < 1 || startFrame > len(frames) {
		return nil
	}
	cur := frames[startFrame-1]
	prev := nominalState(cfg.SampleScales)
	if startFrame >= 2 {
		prev = frames[startFrame-2].State
	}
	m := ComputeMetrics(startFrame, cur.Signals, cfg, prev)
	return &m
}

func worstMetrics(samples []Metrics) *Metrics {
	if len(samples) == 0 {
		return nil
	}
	worst := samples[0]
	for _, s := range samples[1:] {
		if s.QualityScore < worst.QualityScore {
			worst = s
		}
	}
	return &worst
}

// patchEndSnapshot mirrors the reference recorder's recovery-frame boundary
// snapshot: the new sample level is visible while scale-derived scores still
// reflect the gear just left; reprojection closes its snapshot before restore.
func patchEndSnapshot(cfg Config, mode string, m *Metrics) {
	switch mode {
	case modeLowSampling:
		prevLevel := m.SampleLevel + 1
		if prevLevel < len(cfg.SampleScales) {
			oldScale := cfg.SampleScales[prevLevel]
			m.SamplingScale = oldScale
			m.RelativeLoad = float64(m.EyeCount) / 2.0 * oldScale * oldScale
			q := 45 + 55*oldScale
			if m.EyeCount < 2 {
				q -= 10
			}
			if !m.Reprojection {
				q -= 9
			}
			m.QualityScore = round2(q)
		}
	case modeNoReproj:
		m.Reprojection = false
		m.QualityScore = round2(m.QualityScore - 9)
	}
}

func round2(v float64) float64 {
	return float64(int64(v*100+sign(v)*0.5)) / 100
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// BuildSummary aggregates cross-scenario references and safety invariants.
func BuildSummary(scenarios []*ScenarioReport, totalSafety int) Summary {
	s := Summary{
		FalsePositiveAlerts:     []FPAlertRef{},
		FalsePositiveEpisodes:   []FPEpisodeRef{},
		ContinuousChains:        []ChainRef{},
		PartialRecoveries:       []PartialRef{},
		IsolatedRecoveredChains: []ChainRef{},
		InvariantChecks:         []InvariantCheck{},
	}
	for _, sr := range scenarios {
		for _, a := range sr.Alerts {
			if a.Verdict == "false_positive" {
				s.FalsePositiveAlerts = append(s.FalsePositiveAlerts, FPAlertRef{
					ScenarioID: sr.ID, EpisodeID: a.ID, Frame: a.CloseFrame, Detail: a.Detail,
				})
			}
		}
		for _, e := range sr.Episodes {
			if e.FalsePositive {
				s.FalsePositiveEpisodes = append(s.FalsePositiveEpisodes, FPEpisodeRef{
					ScenarioID: sr.ID, ChainID: e.ChainID, EpisodeID: e.ID,
					Frame: e.InvalidatedFrame, Detail: e.FalsePositiveNote,
				})
			}
		}
		for _, ch := range sr.Chains {
			switch ch.Classification {
			case "continuous_degradation":
				s.ContinuousChains = append(s.ContinuousChains, ChainRef{
					ScenarioID: sr.ID, ChainID: ch.ID,
					Detail: fmt.Sprintf("escalations=%d reopened=%v duration=%df",
						len(ch.Escalations), ch.Reopened, ch.DurationFrames),
				})
			case "isolated_full_recovery":
				s.IsolatedRecoveredChains = append(s.IsolatedRecoveredChains, ChainRef{
					ScenarioID: sr.ID, ChainID: ch.ID,
					Detail: fmt.Sprintf("duration=%df partial_events=%d",
						ch.DurationFrames, len(ch.PartialRecoveries)),
				})
			}
			for _, p := range ch.PartialRecoveries {
				s.PartialRecoveries = append(s.PartialRecoveries, PartialRef{
					ScenarioID: sr.ID, ChainID: ch.ID, Frame: p.Frame, Detail: p.Detail,
				})
			}
		}
		s.EpisodeCount += len(sr.Episodes)
		s.ChainCount += len(sr.Chains)
	}
	s.ScenarioCount = len(scenarios)
	s.SafetyEvents = totalSafety
	s.InvariantChecks = RunInvariantChecks(scenarios)
	return s
}

// BuildReport runs every scenario and assembles the top level document.
func BuildReport(cfg Config) *Report {
	defs := AllScenarios()
	scenarios := []*ScenarioReport{}
	safetyTotal := 0
	for _, def := range defs {
		c, frames := ExecuteScenario(def, cfg)
		scenarios = append(scenarios, BuildScenarioReport(def, cfg, c, frames))
		safetyTotal += len(c.SafetyEvents)
	}
	return &Report{
		GeneratedAt: nowRFC3339(),
		GoVersion:   runtime.Version(),
		Stack:       "Go standard library only",
		Config:      cfg,
		Summary:     BuildSummary(scenarios, safetyTotal),
		Scenarios:   scenarios,
	}
}
