package main

import "fmt"

// RequestZeroEyes simulates an external request to disable every eye.
// Safety floor: at least one eye must always render. The request is rejected
// and recorded; capability state is untouched.
func (c *Controller) RequestZeroEyes(actor string) *SafetyEvent {
	msg := "请求关闭全部双眼已被安全底线拒绝：始终保留至少一只眼睛渲染"
	ev := &SafetyEvent{Frame: c.Frame, Code: "eyes_zero_rejected", Message: msg, Actor: actor}
	c.SafetyEvents = append(c.SafetyEvents, ev)
	c.audit(c.Frame, "safety", ev.Code, msg, "", "")
	return ev
}

// RequestSilentHistoryDisable models the degradation pipeline attempting to
// switch history capture off without an explicit user action. It is rejected.
func (c *Controller) RequestSilentHistoryDisable(actor string) *SafetyEvent {
	msg := "降级流程试图静默关闭历史已被拒绝：无重投影期间继续捕获历史"
	ev := &SafetyEvent{Frame: c.Frame, Code: "history_silent_disable_rejected", Message: msg, Actor: actor}
	c.SafetyEvents = append(c.SafetyEvents, ev)
	c.audit(c.Frame, "safety", ev.Code, msg, "", "")
	c.safetyStatusLine = msg
	return ev
}

// SetHistoryEnabled allows an explicit user action to toggle history capture.
// Non-explicit disable attempts are rejected as silent disables.
func (c *Controller) SetHistoryEnabled(enabled, explicit bool, actor string) (bool, *SafetyEvent) {
	if !enabled && !explicit {
		ev := c.RequestSilentHistoryDisable(actor)
		return c.State.History, ev
	}
	c.State.History = enabled
	c.historyUserDisabled = !enabled
	word := "开启"
	if !enabled {
		word = "关闭"
	}
	c.audit(c.Frame, "info", "history_user_toggle",
		fmt.Sprintf("用户显式%s历史捕获（explicit=%v）", word, explicit), "", "")
	return c.State.History, nil
}

// TrimPoseCount clips an over-long pose sequence to the most recent entries.
func (c *Controller) TrimPoseCount(count int) int {
	trimmed := count
	if trimmed > c.Cfg.PoseLimit {
		trimmed = c.Cfg.PoseLimit
		c.audit(c.Frame, "notice", "poses_trimmed",
			fmt.Sprintf("姿态序列裁剪到最近 %d 个（上限 %d）", trimmed, c.Cfg.PoseLimit), "", "")
	}
	return trimmed
}

// InvalidateEpisode marks the latest no-reprojection episode as a false
// positive after external verification and restores reprojection immediately.
func (c *Controller) InvalidateEpisode(frame int, note string) *epRuntime {
	var target *epRuntime
	for i := len(c.episodes) - 1; i >= 0; i-- {
		if c.episodes[i].mode == modeNoReproj {
			target = c.episodes[i]
			break
		}
	}
	if target == nil {
		return nil
	}
	f := frame
	if f == 0 {
		f = c.Frame
	}
	target.active = false
	target.ep.Outcome = "false_positive"
	target.ep.FalsePositive = true
	target.ep.FalsePositiveNote = note
	target.ep.InvalidatedFrame = f
	target.ep.EndFrame = f
	target.ep.DurationFrames = f - target.ep.StartFrame + 1

	c.State.Reprojection = true
	c.audit(f, "notice", "episode_invalidated_false_positive",
		fmt.Sprintf("第 %d 帧经核验触发信号不实，%s 标记为误报并恢复能力：%s",
			f, target.mode, note), target.ep.ID, target.ep.ChainID)

	if ch := c.findChain(target.ep.ChainID); ch != nil {
		ch.active = false
		ch.ch.FalsePositive = true
		ch.ch.FullRecovery = true
		ch.ch.FullRecoveryFrame = f
		ch.endFrame = f
		c.audit(f, "info", "full_recovery",
			fmt.Sprintf("全部能力恢复，降级链结束（分类=false_positive，持续 %d 帧）",
				f-ch.ch.StartFrame+1), "", ch.ch.ID)
	}
	return target
}
