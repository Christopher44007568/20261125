package main

import "fmt"

const (
	modeLowSampling = "low_sampling"
	modeMonocular   = "monocular"
	modeNoReproj    = "no_reprojection"
	tagFullSampling = "full_sampling"
	tagStereo       = "stereo"
	tagReprojection = "reprojection"
)

const (
	alertOpenDetail = "首次观测到帧时间超过预算，等待连续帧确认"
	alertFPDetail   = "超预算未连续达到确认帧数即恢复：判定为瞬时抖动/误报，未启动任何降级"
	frustumReason   = "视锥角点/平面计算失败，重投影缺少有效投影矩阵；本帧起用安全视锥直接绘制，历史缓冲继续捕获"
)

func scalePercent(s float64) int { return int(s*100 + 0.5) }

func (c *Controller) newChain(f int) *chainRuntime {
	id := c.newID("C", &c.chainSeq)
	ch := &chainRuntime{active: true, endFrame: f}
	ch.ch.ID = id
	ch.ch.StartFrame = f
	c.chains = append(c.chains, ch)
	c.lastChain = ch
	return ch
}

// beginEpisode resolves (and possibly reopens/creates) the chain and stores
// a new active episode runtime.
func (c *Controller) beginEpisode(f int, mode, trigger, code, detail string, rc RecoveryCondition, causeFrustum bool) *epRuntime {
	id := c.newID("E", &c.epSeq)

	var chain *chainRuntime
	if c.lastChain != nil && !c.lastChain.ch.FullRecovery {
		chain = c.lastChain
	} else if c.lastChain != nil && c.lastChain.ch.FullRecovery &&
		f-c.lastChain.ch.FullRecoveryFrame <= c.Cfg.ChainGapFrames {
		chain = c.lastChain
		chain.ch.Reopened = true
		chain.ch.FullRecovery = false
		chain.active = true
		c.audit(f, "warn", "chain_reopened",
			fmt.Sprintf("恢复观察期内再次降级，归入连续降级链 %s", chain.ch.ID), "", chain.ch.ID)
		for _, old := range c.episodes {
			if old.ep.ChainID == chain.ch.ID {
				old.ep.Continuous = true
			}
		}
	} else {
		chain = c.newChain(f)
	}

	// A second mode joining a live chain is a compound/continuous event.
	compound := len(c.activeEpisodes()) > 0

	ep := &epRuntime{
		mode: mode, active: true, causeFrustum: causeFrustum,
		recoveryCondition: rc,
	}
	ep.ep = Episode{
		ID: id, ChainID: chain.ch.ID, Mode: mode, Trigger: trigger,
		ReasonCode: code, ReasonDetail: detail,
		StartFrame: f, EndFrame: f, DurationFrames: 1,
		RecoveryCondition: rc, Outcome: "active",
		Continuous: compound || chain.ch.Reopened,
		Samples:    []Metrics{},
	}
	chain.ch.EpisodeIDs = append(chain.ch.EpisodeIDs, id)
	c.episodes = append(c.episodes, ep)
	if f > chain.endFrame {
		chain.endFrame = f
	}

	c.audit(f, "warn", "degradation_start",
		fmt.Sprintf("第 %d 帧启动降级 %s（触发=%s，原因=%s）", f, mode, trigger, code),
		id, chain.ch.ID)
	return ep
}

func (c *Controller) applyBudget(f int, sig Signals, pre Metrics) {
	over := sig.FrameTimeMS > c.Cfg.BudgetMS
	severe := over && sig.FrameTimeMS >= c.Cfg.SevereThresholdMS()

	var openAlert *Alert
	for _, a := range c.Alerts {
		if a.Verdict == "" {
			openAlert = a
		}
	}

	if !over {
		c.overStreak = 0
		c.floorOverStreak = 0
		if openAlert != nil {
			openAlert.CloseFrame = f
			openAlert.Verdict = "false_positive"
			openAlert.Detail = alertFPDetail
			c.audit(f, "notice", "false_positive_alert",
				fmt.Sprintf("第 %d 帧的预算告警 %s 被判定为误报（单帧抖动），画面能力不变", f, openAlert.ID),
				"", "")
		}
		return
	}

	c.overStreak++
	if c.overStreak == 1 {
		c.alertSeq++
		id := fmt.Sprintf("A%04d", c.alertSeq)
		a := &Alert{
			ID: id, Trigger: "frame_budget_insufficient",
			ReasonCode: "over_budget_unconfirmed", Detail: alertOpenDetail,
			Frame: f, SignalValue: sig.FrameTimeMS,
		}
		c.Alerts = append(c.Alerts, a)
		openAlert = a
	}

	low := c.activeEpisode(modeLowSampling)
	if low == nil {
		confirmed := c.overStreak >= c.Cfg.BudgetConfirmFrames
		count := c.overStreak
		if severe && c.overStreak == 1 {
			confirmed = true
		}
		if !confirmed {
			return
		}
		openAlert.CloseFrame = f
		openAlert.Verdict = "confirmed"
		c.State.SampleLevel = 1
		c.State.SamplingScale = c.Cfg.SampleScales[1]
		detail := fmt.Sprintf("frame_time=%.2fms > budget=%.2fms，连续超预算 %d 帧",
			sig.FrameTimeMS, c.Cfg.BudgetMS, count)
		rc := RecoveryCondition{
			Kind: "budget_headroom_hold",
			Description: fmt.Sprintf("帧时间连续 %d 帧 <= %.2fms（预算的 %d%%）后逐档恢复采样",
				c.Cfg.RecoverHoldFrames, c.Cfg.RecoveryThresholdMS(), int(c.Cfg.RecoveryRatio*100+0.5)),
			Metric: "frame_time_ms", Threshold: c.Cfg.RecoveryThresholdMS(),
			HoldFrames: c.Cfg.RecoverHoldFrames,
		}
		ep := c.beginEpisode(f, modeLowSampling, "frame_budget_insufficient",
			"frame_time_over_budget", detail, rc, false)
		ep.startPreMetrics = &pre
		return
	}

	// Low sampling already active: each continued over-budget frame gears down.
	floorLevel := len(c.Cfg.SampleScales) - 1
	wasAtFloor := c.State.SampleLevel >= floorLevel
	if c.State.SampleLevel < floorLevel {
		oldLevel := c.State.SampleLevel
		newLevel := oldLevel + 1
		oldScale := c.Cfg.SampleScales[oldLevel]
		newScale := c.Cfg.SampleScales[newLevel]
		c.State.SampleLevel = newLevel
		c.State.SamplingScale = newScale
		lc := LevelChange{Frame: f, FromLevel: oldLevel, ToLevel: newLevel,
			FromScale: oldScale, ToScale: newScale, Reason: "continued_over_budget"}
		low.levelChanges = append(low.levelChanges, lc)
		low.ep.LevelChanges = append(low.ep.LevelChanges, lc)
		c.audit(f, "warn", "sampling_level_down",
			fmt.Sprintf("第 %d 帧采样降档 %d%% -> %d%%", f, scalePercent(oldScale), scalePercent(newScale)),
			low.ep.ID, low.ep.ChainID)
	}

	// Only over-budget frames that START at the floor count towards mono
	// escalation; the frame that gears down onto the floor is not counted.
	if wasAtFloor {
		c.floorOverStreak++
	} else {
		c.floorOverStreak = 0
	}
	mono := c.activeEpisode(modeMonocular)
	if c.State.SampleLevel == floorLevel && mono == nil &&
		c.floorOverStreak >= c.Cfg.EscalateMonoFrames {
		c.State.EyeCount = 1
		c.State.Stereo = false
		if low != nil {
			low.ep.Continuous = true
		}
		detail := fmt.Sprintf("采样已到底（%d%%）且连续 %d 帧超预算，升级为单目",
			scalePercent(c.Cfg.SampleScales[len(c.Cfg.SampleScales)-1]), c.Cfg.EscalateMonoFrames)
		rc := RecoveryCondition{
			Kind: "stereo_headroom_hold",
			Description: fmt.Sprintf("采样先恢复到 100%%，且帧时间连续 %d 帧 <= %.2fms（预算的 %d%%）后恢复双眼",
				c.Cfg.StereoRecoverHoldFrames, c.Cfg.BudgetMS*c.Cfg.StereoHeadroomRatio,
				int(c.Cfg.StereoHeadroomRatio*100+0.5)),
			Metric: "frame_time_ms", Threshold: c.Cfg.StereoThresholdMS(),
			HoldFrames: c.Cfg.StereoRecoverHoldFrames,
		}
		mep := c.beginEpisode(f, modeMonocular, "frame_budget_insufficient",
			"sampling_floor_still_over_budget", detail, rc, false)
		mep.startPreMetrics = &pre
		if ch := c.findChain(mep.ep.ChainID); ch != nil {
			ch.ch.Escalations = append(ch.ch.Escalations, Escalation{
				Frame: f, From: "low_sampling@floor", To: "monocular",
				Reason: "continued_over_budget_at_sampling_floor",
			})
		}
		c.audit(f, "critical", "escalate_monocular",
			fmt.Sprintf("第 %d 帧连续降级：低采样已到能力边界，升级为单目（另一眼挂起并在画面明确标示）", f),
			mep.ep.ID, mep.ep.ChainID)
	}
}

func (c *Controller) findChain(id string) *chainRuntime {
	for _, ch := range c.chains {
		if ch.ch.ID == id {
			return ch
		}
	}
	return nil
}
