package main

import (
	"fmt"
	"time"
)

func nowRFC3339() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// RunInvariantChecks scans every recorded frame plus safety events and
// verifies the non-negotiable safety floor.
func RunInvariantChecks(scenarios []*ScenarioReport) []InvariantCheck {
	eyeViolations := 0
	historyOffFrames := 0
	interactionOffFrames := 0
	hasZeroEyesReject := false
	hasSilentHistoryReject := false

	for _, sr := range scenarios {
		for _, f := range sr.Frames {
			if f.State.EyeCount < 1 {
				eyeViolations++
			}
			if !f.State.History {
				historyOffFrames++
			}
			if !f.State.Interaction || !f.State.Overlay {
				interactionOffFrames++
			}
		}
		for _, ev := range sr.SafetyEvents {
			switch ev.Code {
			case "eyes_zero_rejected":
				hasZeroEyesReject = true
			case "history_silent_disable_rejected":
				hasSilentHistoryReject = true
			}
		}
	}

	return []InvariantCheck{
		{
			ID: "eye_floor", Name: "任何帧至少保留一只眼睛（禁止双眼全关）",
			Passed:   eyeViolations == 0,
			Evidence: fmt.Sprintf("扫描全部场景帧，违规 %d 帧", eyeViolations),
		},
		{
			ID: "history_retained", Name: "自动降级路径从不关闭历史",
			Passed:   historyOffFrames == 0,
			Evidence: fmt.Sprintf("历史关闭帧计数 %d（仅显式用户操作可关）", historyOffFrames),
		},
		{
			ID: "interaction_retained", Name: "交互层与状态提示始终开启",
			Passed:   interactionOffFrames == 0,
			Evidence: fmt.Sprintf("交互/遮罩关闭帧计数 %d", interactionOffFrames),
		},
		{
			ID: "reject_zero_eyes", Name: "关闭双眼的请求被拒绝并记录",
			Passed:   hasZeroEyesReject,
			Evidence: "safety event eyes_zero_rejected",
		},
		{
			ID: "reject_silent_history", Name: "静默关闭历史的请求被拒绝并记录",
			Passed:   hasSilentHistoryReject,
			Evidence: "safety event history_silent_disable_rejected",
		},
	}
}
