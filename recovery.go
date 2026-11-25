package main

import "fmt"

func capabilityTag(mode string) string {
	switch mode {
	case modeLowSampling:
		return tagFullSampling
	case modeMonocular:
		return tagStereo
	case modeNoReproj:
		return tagReprojection
	}
	return mode
}

func bracketTags(tags []string) string {
	out := ""
	for i, t := range tags {
		if i > 0 {
			out += ", "
		}
		out += "[" + t + "]"
	}
	return out
}

// classifyChain computes the final/audit classification of a chain.
func classifyChain(ch *chainRuntime) string {
	if ch.ch.FalsePositive {
		return "false_positive"
	}
	if ch.ch.Reopened || len(ch.ch.Escalations) > 0 {
		return "continuous_degradation"
	}
	if !ch.ch.FullRecovery {
		return "ongoing"
	}
	return "isolated_full_recovery"
}

// recoverModeEpisode closes a normally recovered episode and emits the
// matching recovery / partial / full-chain audit events.
func (c *Controller) recoverModeEpisode(f int, ep *epRuntime) {
	ep.active = false
	remaining := c.activeEpisodes()
	outcome := "recovered_full"
	if len(remaining) > 0 {
		outcome = "recovered_partial"
	}
	ep.ep.Outcome = outcome
	ep.ep.EndFrame = f
	ep.ep.DurationFrames = f - ep.ep.StartFrame + 1
	met := f
	ep.ep.RecoveryMetFrame = &met

	c.audit(f, "notice", "degradation_recover",
		fmt.Sprintf("第 %d 帧 %s 达到恢复条件并退出（结局=%s，持续 %d 帧）",
			f, ep.mode, outcome, ep.ep.DurationFrames),
		ep.ep.ID, ep.ep.ChainID)

	ch := c.findChain(ep.ep.ChainID)
	if len(remaining) > 0 {
		recovered := []string{capabilityTag(ep.mode)}
		still := []string{}
		for _, other := range remaining {
			still = append(still, capabilityTag(other.mode))
		}
		detail := fmt.Sprintf("恢复能力 %s；仍降级 %s", bracketTags(recovered), bracketTags(still))
		if ch != nil {
			ch.ch.HadPartialRecovery = true
			ch.ch.PartialRecoveries = append(ch.ch.PartialRecoveries, PartialRecovery{
				Frame: f, Recovered: recovered, StillDegraded: still, Detail: detail,
			})
		}
		c.audit(f, "notice", "partial_recovery",
			fmt.Sprintf("部分能力恢复：%s 已恢复，%s 仍降级", bracketTags(recovered), bracketTags(still)),
			"", ep.ep.ChainID)
		return
	}

	if ch != nil {
		ch.active = false
		ch.ch.FullRecovery = true
		ch.ch.FullRecoveryFrame = f
		ch.endFrame = f
		class := classifyChain(ch)
		c.audit(f, "info", "full_recovery",
			fmt.Sprintf("全部能力恢复，降级链结束（分类=%s，持续 %d 帧）",
				class, f-ch.ch.StartFrame+1),
			"", ch.ch.ID)
	}
}

func (c *Controller) applySamplingRecovery(f int, sig Signals) bool {
	geared := false
	thr := c.Cfg.RecoveryThresholdMS()
	if sig.FrameTimeMS <= thr {
		c.budgetGood++
	} else {
		c.budgetGood = 0
	}
	low := c.activeEpisode(modeLowSampling)
	if low == nil || c.budgetGood < c.Cfg.RecoverHoldFrames {
		return false
	}
	c.budgetGood = 0
	// A sampling gear change restarts the stricter stereo hold window.
	c.stereoGood = 0
	if c.State.SampleLevel > 0 {
		oldLevel := c.State.SampleLevel
		newLevel := oldLevel - 1
		oldScale := c.Cfg.SampleScales[oldLevel]
		newScale := c.Cfg.SampleScales[newLevel]
		c.State.SampleLevel = newLevel
		c.State.SamplingScale = newScale
		lc := LevelChange{Frame: f, FromLevel: oldLevel, ToLevel: newLevel,
			FromScale: oldScale, ToScale: newScale, Reason: "headroom_recovery"}
		low.levelChanges = append(low.levelChanges, lc)
		low.ep.LevelChanges = append(low.ep.LevelChanges, lc)
		c.audit(f, "notice", "sampling_level_up",
			fmt.Sprintf("第 %d 帧采样恢复一档 %d%% -> %d%%", f, scalePercent(oldScale), scalePercent(newScale)),
			low.ep.ID, low.ep.ChainID)
		geared = true
	}
	if c.State.SampleLevel == 0 {
		c.recoverModeEpisode(f, low)
	}
	return geared
}

func (c *Controller) applyStereoRecovery(f int, sig Signals, gearedThisFrame bool) {
	mono := c.activeEpisode(modeMonocular)
	if gearedThisFrame {
		// The gear change just reset the hold window; do not count this frame twice.
		return
	}
	if sig.FrameTimeMS <= c.Cfg.StereoThresholdMS() {
		c.stereoGood++
	} else {
		c.stereoGood = 0
	}
	if mono != nil && c.State.SampleLevel == 0 && c.stereoGood >= c.Cfg.StereoRecoverHoldFrames {
		c.stereoGood = 0
		c.State.EyeCount = 2
		c.State.Stereo = true
		c.recoverModeEpisode(f, mono)
	}
}

func (c *Controller) applyReprojection(f int, sig Signals) {
	rp := c.activeEpisode(modeNoReproj)
	frustumBad := sig.FrustumFailed
	poseBad := sig.PoseCount > c.Cfg.PoseLimit || sig.PoseInvalid

	if rp == nil && (frustumBad || poseBad) {
		c.State.Reprojection = false
		var trigger, code, detail string
		var rc RecoveryCondition
		causeFrustum := frustumBad
		if frustumBad {
			trigger = "frustum_compute_failed"
			code = sig.FrustumCode
			if code == "" {
				code = "frustum_compute_failed"
			}
			detail = frustumReason
			rc = RecoveryCondition{
				Kind: "frustum_valid_hold",
				Description: fmt.Sprintf("视锥计算连续 %d 帧成功，且 fov/aspect/near/far 全部有限合法",
					c.Cfg.FrustumRecoverHold),
				HoldFrames: c.Cfg.FrustumRecoverHold,
			}
		} else {
			trigger = "pose_sequence_over_limit"
			if sig.PoseCount > c.Cfg.PoseLimit {
				code = "pose_count_exceeds_limit"
				detail = fmt.Sprintf("姿态序列长度 %d 超过安全上限 %d，重投影引用窗口不可信",
					sig.PoseCount, c.Cfg.PoseLimit)
			} else {
				code = "pose_entry_invalid"
				detail = "姿态序列包含非法（NaN/Inf）或不连续条目，重投影引用窗口不可信"
			}
			rc = RecoveryCondition{
				Kind: "pose_within_limit_hold",
				Description: fmt.Sprintf("裁剪/重置后姿态数 <= %d 且全部合法，连续 %d 帧",
					c.Cfg.PoseLimit, c.Cfg.PoseRecoverHold),
				Metric: "pose_count", Threshold: float64(c.Cfg.PoseLimit),
				HoldFrames: c.Cfg.PoseRecoverHold,
			}
		}
		rp = c.beginEpisode(f, modeNoReproj, trigger, code, detail, rc, causeFrustum)
	}

	if !frustumBad {
		c.frustumGood++
	} else {
		c.frustumGood = 0
	}
	if !poseBad {
		c.poseGood++
	} else {
		c.poseGood = 0
	}
	if rp == nil || !rp.active {
		return
	}
	if rp.causeFrustum && c.frustumGood >= c.Cfg.FrustumRecoverHold {
		c.State.Reprojection = true
		c.recoverModeEpisode(f, rp)
	} else if !rp.causeFrustum && c.poseGood >= c.Cfg.PoseRecoverHold {
		c.State.Reprojection = true
		c.recoverModeEpisode(f, rp)
	}
}

func (c *Controller) updateFloorWarning(f int, sig Signals) {
	floorLevel := len(c.Cfg.SampleScales) - 1
	atMonoFloor := c.State.EyeCount == 1 && c.State.SampleLevel == floorLevel &&
		sig.FrameTimeMS > c.Cfg.BudgetMS
	if atMonoFloor {
		c.floorWarnStreak++
	} else {
		c.floorWarnStreak = 0
	}
	if c.floorWarnStreak == 3 {
		epID := ""
		if mono := c.activeEpisode(modeMonocular); mono != nil {
			epID = mono.ep.ID
		}
		c.audit(f, "critical", "degradation_floor_still_over_budget",
			"已处于单眼 + 最低采样（降级能力边界）仍超预算：保留单眼渲染、交互与状态提示，不关闭画面",
			epID, "")
	}
}
