package main

import "fmt"

const (
	okHeadline           = "全部能力正常：双眼 / 全采样 / 重投影 / 历史 / 交互均在线"
	budgetHeadline       = "帧预算不足：已降低采样，等待帧时间恢复余量"
	compoundHeadline     = "低采样 + 无重投影：历史继续捕获，画面保持交互"
	monoSamplingHeadline = "连续降级中：单目 + 低采样；另一眼挂起并标示，交互保留"
	monoOnlyHeadline     = "单目模式：仅渲染左眼/主眼，另一眼挂起（非静默关闭）"
	floorHeadline        = "已达降级下限：单眼 + 最低采样仍超预算；保留单眼画面与全部交互"
	reprojHeadline       = "重投影已暂停（视锥/姿态异常）：直接渲染中，历史未关闭"
	safetyHeadline       = "安全底线拦截：禁止双眼全关或静默关闭历史，当前画面能力已保留"

	budgetLabelFmt = "帧时间 <= %.2fms 后采样升一档"
	stereoLabelFmt = "满采样下帧时间 <= %.2fms 后恢复双眼"
	frustumLabel   = "视锥连续合法帧"
	poseLabel      = "姿态序列在限且合法的连续帧"
)

func (c *Controller) buildStatus(f int, sig Signals) StatusDetail {
	st := c.State
	lines := []string{}
	eyePart := "2/2 在渲染（双眼）"
	if st.EyeCount == 1 {
		eyePart = "1/2 在渲染（单目）"
	}
	lines = append(lines, fmt.Sprintf("眼睛：%s；采样：%d%%（档位 %d）",
		eyePart, scalePercent(st.SamplingScale), st.SampleLevel))
	reprojOn := "开"
	if !st.Reprojection {
		reprojOn = "关"
	}
	lines = append(lines, fmt.Sprintf("重投影：%s；历史缓冲：开；交互/状态层：始终开启", reprojOn))

	active := c.activeEpisodes()
	modes := []string{}
	epIDs := []string{}
	for _, ep := range active {
		modes = append(modes, ep.mode)
		epIDs = append(epIDs, ep.ep.ID)
		lines = append(lines, fmt.Sprintf("%s %s：%s（已持续 %d 帧）",
			ep.ep.ID, ep.mode, ep.ep.ReasonCode, f-ep.ep.StartFrame+1))
	}

	rp := []RecoveryProgress{}
	if low := c.activeEpisode(modeLowSampling); low != nil {
		thr := c.Cfg.RecoveryThresholdMS()
		label := fmt.Sprintf(budgetLabelFmt, thr)
		rp = append(rp, RecoveryProgress{
			Kind: "budget_headroom_hold", Label: label,
			Streak: c.budgetGood, Need: c.Cfg.RecoverHoldFrames, ThresholdMS: &thr,
		})
		lines = append(lines, fmt.Sprintf("恢复条件[budget_headroom_hold]：%s 进度 %d/%d",
			label, c.budgetGood, c.Cfg.RecoverHoldFrames))
	}
	if mono := c.activeEpisode(modeMonocular); mono != nil {
		thr := c.Cfg.StereoThresholdMS()
		label := fmt.Sprintf(stereoLabelFmt, thr)
		rp = append(rp, RecoveryProgress{
			Kind: "stereo_headroom_hold", Label: label,
			Streak: c.stereoGood, Need: c.Cfg.StereoRecoverHoldFrames, ThresholdMS: &thr,
		})
		lines = append(lines, fmt.Sprintf("恢复条件[stereo_headroom_hold]：%s 进度 %d/%d",
			label, c.stereoGood, c.Cfg.StereoRecoverHoldFrames))
	}
	if rep := c.activeEpisode(modeNoReproj); rep != nil {
		if rep.causeFrustum {
			rp = append(rp, RecoveryProgress{
				Kind: "frustum_valid_hold", Label: frustumLabel,
				Streak: c.frustumGood, Need: c.Cfg.FrustumRecoverHold,
			})
			lines = append(lines, fmt.Sprintf("恢复条件[frustum_valid_hold]：%s 进度 %d/%d",
				frustumLabel, c.frustumGood, c.Cfg.FrustumRecoverHold))
		} else {
			rp = append(rp, RecoveryProgress{
				Kind: "pose_within_limit_hold", Label: poseLabel,
				Streak: c.poseGood, Need: c.Cfg.PoseRecoverHold,
			})
			lines = append(lines, fmt.Sprintf("恢复条件[pose_within_limit_hold]：%s 进度 %d/%d",
				poseLabel, c.poseGood, c.Cfg.PoseRecoverHold))
		}
	}

	level, headline := c.deriveHeadline()

	chainID := ""
	if len(active) > 0 {
		chainID = active[0].ep.ChainID
	} else if c.lastChain != nil && c.lastChain.ch.FullRecovery &&
		f-c.lastChain.ch.FullRecoveryFrame <= c.Cfg.ChainGapFrames {
		chainID = c.lastChain.ch.ID + "（恢复观察期，再触发将计为连续降级）"
	}

	if c.safetyStatusLine != "" {
		level = "critical"
		headline = safetyHeadline
		lines = append(lines, "安全拦截："+c.safetyStatusLine)
	}

	return StatusDetail{
		Level: level, Headline: headline, Lines: lines,
		ActiveModes: modesOrNil(modes), ChainID: chainID,
		EpisodeIDs: idsOrNil(epIDs), RecoveryProgress: rpOrNil(rp),
		InteractionHint: interactionHint, FloorWarning: c.floorWarnStreak >= 3,
	}
}

func modesOrNil(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}

func idsOrNil(s []string) []string { return modesOrNil(s) }

func rpOrNil(s []RecoveryProgress) []RecoveryProgress {
	if len(s) == 0 {
		return nil
	}
	return s
}

func (c *Controller) deriveHeadline() (string, string) {
	low := c.activeEpisode(modeLowSampling)
	mono := c.activeEpisode(modeMonocular)
	rep := c.activeEpisode(modeNoReproj)
	if c.floorWarnStreak >= 3 {
		return "critical", floorHeadline
	}
	if low != nil && rep != nil && mono == nil {
		return "warn", compoundHeadline
	}
	if mono != nil {
		if c.State.SampleLevel > 0 {
			return "warn", monoSamplingHeadline
		}
		return "warn", monoOnlyHeadline
	}
	if low != nil {
		return "warn", budgetHeadline
	}
	if rep != nil {
		return "warn", reprojHeadline
	}
	return "ok", okHeadline
}
