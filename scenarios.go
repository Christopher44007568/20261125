package main

// FrameAction is a scripted external event executed after stepping At frame.
type FrameAction struct {
	At    int
	Kind  string // "zero_eyes" | "silent_history" | "invalidate"
	Actor string
	Note  string
}

// ScenarioDef is one scripted audit scenario.
type ScenarioDef struct {
	ID          string
	Title       string
	Description string
	Signals     []Signals
	Actions     []FrameAction
	Shots       []Shot
}

func sig(ft float64, fail bool, code string, poses int, invalid bool) Signals {
	return Signals{FrameTimeMS: ft, FrustumFailed: fail, FrustumCode: code,
		PoseCount: poses, PoseInvalid: invalid}
}

// AllScenarios returns the nine reference scenarios in report order.
func AllScenarios() []ScenarioDef {
	return []ScenarioDef{
		{
			ID: "01_false_positive_spike", Title: "误报：单帧预算尖峰",
			Description: "一帧轻微超预算后立即恢复，未达到连续确认帧数；只记录误报告警，不启动降级。",
			Signals: []Signals{
				sig(9.5, false, "", 0, false), sig(12.2, false, "", 0, false),
				sig(9.2, false, "", 0, false), sig(8.8, false, "", 0, false),
				sig(8.5, false, "", 0, false), sig(8.5, false, "", 0, false),
			},
			Shots: []Shot{{Frame: 2, File: "01_false_positive_alert.png",
				Caption: "单帧超预算被标记为待确认，下一帧即判误报，双眼/历史不变"}},
		},
		{
			ID: "02_budget_sampling_recover", Title: "帧预算不足：低采样后完整恢复",
			Description: "持续超预算触发低采样并降档到 50%，余量恢复后逐档升回，链条按孤立完整恢复分类。",
			Signals: []Signals{
				sig(9.5, false, "", 0, false), sig(12, false, "", 0, false),
				sig(13, false, "", 0, false), sig(13.2, false, "", 0, false),
				sig(12.4, false, "", 0, false), sig(9.6, false, "", 0, false),
				sig(8, false, "", 0, false), sig(8, false, "", 0, false),
				sig(8, false, "", 0, false), sig(8, false, "", 0, false),
				sig(8, false, "", 0, false), sig(8, false, "", 0, false),
			},
			Shots: []Shot{{Frame: 4, File: "02_low_sampling_peak.png",
				Caption: "低采样 50%：画面像素化但双眼、历史与交互状态条保留"}},
		},
		{
			ID: "03_continuous_mono_partial", Title: "连续降级：低采样→单目，部分恢复后双眼恢复",
			Description: "严重超预算一路降级到单眼+最低采样并触底告警；恢复时先升满采样（部分能力恢复），再凭严格余量恢复双眼。",
			Signals: concatSignals(
				repeatSignal(sig(20, false, "", 0, false), 6),
				repeatSignal(sig(7, false, "", 0, false), 10)),
			Shots: []Shot{
				{Frame: 6, File: "03_monocular_floor.png",
					Caption: "连续降级到能力边界：单眼+50%采样仍超预算，拒绝关闭剩下的眼睛"},
				{Frame: 12, File: "03_partial_recovery.png",
					Caption: "部分能力恢复：采样回到 100%，双眼仍挂起，状态条明确标示"},
			},
		},
		{
			ID: "04_frustum_failure", Title: "视锥计算失败：无重投影，历史保留",
			Description: "视锥计算连续失败时立即暂停重投影并安全直绘；视锥连续合法后恢复，历史捕获始终开启。",
			Signals: []Signals{
				sig(9, false, "", 0, false),
				sig(9.4, true, "frustum_corners_nan", 0, false),
				sig(9.6, true, "frustum_planes_non_finite", 0, false),
				sig(9, false, "", 0, false), sig(8.8, false, "", 0, false),
				sig(8.8, false, "", 0, false),
			},
			Shots: []Shot{{Frame: 3, File: "04_frustum_no_reproject.png",
				Caption: "视锥失败 → 无重投影：HISTORY ON，直接渲染，状态提示恢复条件"}},
		},
		{
			ID: "05_pose_over_limit", Title: "姿态序列超限：裁剪后恢复重投影",
			Description: "姿态数超过上限触发无重投影并发出裁剪动作；裁剪后连续在限且合法，恢复重投影。",
			Signals: []Signals{
				sig(9, false, "", 14, false), sig(9.1, false, "", 18, false),
				sig(9.2, false, "", 22, false), sig(9, false, "", 12, false),
				sig(8.9, false, "", 12, false), sig(8.9, false, "", 12, false),
			},
			Shots: []Shot{{Frame: 3, File: "05_pose_overflow.png",
				Caption: "姿态序列 22>16：暂停重投影，提示裁剪到最近 16 个姿态"}},
		},
		{
			ID: "06_partial_reprojection_recovery", Title: "复合故障：重投影先恢复（部分能力恢复）",
			Description: "低采样进行中又发生视锥失败；视锥恢复后重投影先回来，采样仍降级，记为部分能力恢复。",
			Signals: []Signals{
				sig(12, false, "", 0, false), sig(12, false, "", 0, false),
				sig(12, true, "aspect_zero", 0, false), sig(8, false, "", 0, false),
				sig(8, false, "", 0, false), sig(8, false, "", 0, false),
				sig(8, false, "", 0, false),
			},
			Shots: []Shot{{Frame: 3, File: "06_compound_degraded.png",
				Caption: "低采样 + 无重投影同链：两类原因、恢复条件分别记录"}},
		},
		{
			ID: "07_rapid_reentry_continuous", Title: "快速再触发：同一降级链判连续降级",
			Description: "完整恢复后在链宽限帧内再次超预算，链条重开并分类为连续降级，所有关联片段打 continuous 标记。",
			Signals: []Signals{
				sig(12, false, "", 0, false), sig(12, false, "", 0, false),
				sig(8, false, "", 0, false), sig(8, false, "", 0, false),
				sig(8, false, "", 0, false), sig(12, false, "", 0, false),
				sig(12, false, "", 0, false), sig(8, false, "", 0, false),
				sig(8, false, "", 0, false), sig(8, false, "", 0, false),
				sig(8, false, "", 0, false),
			},
		},
		{
			ID: "08_invalidated_false_positive", Title: "核验作废：已启动片段改判误报",
			Description: "视锥故障启动无重投影后，外部核验发现是上一帧残留读数；片段与链条改判误报并立即恢复能力。",
			Signals: []Signals{
				sig(9, true, "stale_reading", 0, false),
				sig(9, true, "stale_reading", 0, false),
				sig(9, false, "", 0, false), sig(9, false, "", 0, false),
			},
			Actions: []FrameAction{{At: 2, Kind: "invalidate",
				Note: "核验为上一帧残留的重复读数，本帧视锥实际合法"}},
			Shots: []Shot{{Frame: 2, File: "08_invalidated_episode.png",
				Caption: "待核验的视锥故障片段；核验作废后整条链分类为误报"}},
		},
		{
			ID: "09_safety_interlocks", Title: "安全底线：拒绝双眼全关与静默关历史",
			Description: "外部尝试关闭全部双眼、降级流程尝试静默关闭历史，均被拒绝并记录安全事件；画面保持至少单眼、历史开启与红色安全提示。",
			Signals:     repeatSignal(sig(9, false, "", 0, false), 4),
			Actions: []FrameAction{
				{At: 1, Kind: "zero_eyes", Actor: "scenario-probe"},
				{At: 1, Kind: "silent_history", Actor: "degradation"},
			},
			Shots: []Shot{{Frame: 2, File: "09_safety_interlock.png",
				Caption: "安全拦截时刻：零眼与静默关历史请求被拒，状态条红色提示"}},
		},
	}
}

func repeatSignal(s Signals, n int) []Signals {
	out := make([]Signals, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, s)
	}
	return out
}

func concatSignals(parts ...[]Signals) []Signals {
	out := []Signals{}
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// ExecuteScenario runs the script and returns the controller plus recorded frames.
func ExecuteScenario(def ScenarioDef, cfg Config) (*Controller, []*FrameReport) {
	c := NewController(cfg)
	frames := []*FrameReport{}
	for i, s := range def.Signals {
		fr := c.Step(s)
		fr.Scenario = def.ID
		frames = append(frames, fr)
		frameNo := i + 1
		for _, a := range def.Actions {
			if a.At != frameNo {
				continue
			}
			switch a.Kind {
			case "zero_eyes":
				c.RequestZeroEyes(a.Actor)
			case "silent_history":
				c.RequestSilentHistoryDisable(a.Actor)
			case "invalidate":
				c.InvalidateEpisode(frameNo, a.Note)
			}
		}
	}
	return c, frames
}
