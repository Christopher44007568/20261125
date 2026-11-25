package main

import (
	"fmt"
	"math"
	"strings"
)

// RenderMarkdown renders the human readable audit report.
func RenderMarkdown(r *Report) string {
	var b strings.Builder
	hz := int(math.Round(1000 / r.Config.BudgetMS))
	fmt.Fprintf(&b, "# 降级控制器审计报告\n\n")
	fmt.Fprintf(&b, "- 生成时间（UTC）：%s\n", r.GeneratedAt)
	fmt.Fprintf(&b, "- 运行时：%s；实现栈：%s\n", r.GoVersion, r.Stack)
	fmt.Fprintf(&b, "- 帧预算：%.2f ms（约 %d Hz）；确认帧=%d；姿态上限=%d\n\n",
		r.Config.BudgetMS, hz, r.Config.BudgetConfirmFrames, r.Config.PoseLimit)

	renderMDCapabilities(&b)
	renderMDClassification(&b, r)
	renderMDInvariants(&b, r)
	renderMDScenarios(&b, r)
	renderMDRunbook(&b)
	return b.String()
}

func renderMDCapabilities(b *strings.Builder) {
	b.WriteString("## 1. 三类降级方案与能力边界\n\n")
	b.WriteString("| 方案 | 触发 | 能力边界（不可越过） | 恢复条件 |\n| --- | --- | --- | --- |\n")
	b.WriteString("| 低采样 `low_sampling` | 帧预算不足（连续 2 帧超 11.11ms，或严重超标 1.45x） | 采样档到底为 50%，不会到 0 采样 | 帧时间 ≤ 8.89ms 连续 3 帧，逐档恢复 |\n")
	b.WriteString("| 单目 `monocular` | 最低采样仍连续 2 帧超预算 | 至少保留 1 只眼；请求双眼全关会被拒绝 | 采样先回满，且帧时间 ≤ 7.78ms 连续 4 帧才恢复双眼 |\n")
	b.WriteString("| 无重投影 `no_reprojection` | 视锥计算失败 / 姿态序列超 16 或非法 | 只关重投影，**历史捕获不关**，画面不黑 | 视锥连续 2 帧合法；或姿态裁剪到限内并连续 2 帧合法 |\n\n")
	b.WriteString("所有降级片段都记录：触发原因（含机器码与实测值）、起止帧与持续帧数、起始/最差/结束及逐帧画质指标、结构化恢复条件与恢复进度。\n\n")
}

func renderMDClassification(b *strings.Builder, r *Report) {
	s := r.Summary
	b.WriteString("## 2. 连续降级 / 部分恢复 / 误报 的区分\n\n")
	fmt.Fprintf(b, "- 场景数 `%d`，降级片段 `%d`，降级链 `%d`。\n",
		s.ScenarioCount, s.EpisodeCount, s.ChainCount)

	fmt.Fprintf(b, "- **连续降级链**（含升级或恢复宽限内重开）：%d 条\n", len(s.ContinuousChains))
	for _, c := range s.ContinuousChains {
		fmt.Fprintf(b, "  - `%s` %s：%s\n", c.ScenarioID, c.ChainID, c.Detail)
	}
	fmt.Fprintf(b, "- **部分能力恢复事件**：%d 次\n", len(s.PartialRecoveries))
	for _, p := range s.PartialRecoveries {
		fmt.Fprintf(b, "  - `%s` %s 第 %d 帧：%s\n", p.ScenarioID, p.ChainID, p.Frame, p.Detail)
	}

	fpAlerts := 0
	for _, sr := range r.Scenarios {
		for _, a := range sr.Alerts {
			if a.Verdict == "false_positive" {
				fpAlerts++
				fmt.Fprintf(b, "- **误报（未确认的瞬时告警）**：%d 条\n", len(s.FalsePositiveAlerts))
				fmt.Fprintf(b, "  - `%s` 告警 %s（关闭于第 %d 帧）：%s\n",
					sr.ID, a.ID, a.CloseFrame, a.Detail)
			}
		}
	}
	_ = fpAlerts
	fmt.Fprintf(b, "- **误报（启动后经核验作废的片段/链）**：%d 条\n", len(s.FalsePositiveEpisodes))
	for _, e := range s.FalsePositiveEpisodes {
		fmt.Fprintf(b, "  - `%s` %s %s（作废于第 %d 帧）：%s\n",
			e.ScenarioID, e.ChainID, e.EpisodeID, e.Frame, e.Detail)
	}
	fmt.Fprintf(b, "- 孤立完整恢复链：%d 条。\n\n", len(s.IsolatedRecoveredChains))
}

func renderMDInvariants(b *strings.Builder, r *Report) {
	b.WriteString("## 3. 安全不变量（不可静默关闭双眼或历史）\n\n")
	b.WriteString("| 检查 | 结果 | 证据 |\n| --- | --- | --- |\n")
	for _, ic := range r.Summary.InvariantChecks {
		res := "PASS"
		if !ic.Passed {
			res = "FAIL"
		}
		fmt.Fprintf(b, "| %s | %s | %s |\n", ic.Name, res, ic.Evidence)
	}
	b.WriteString("\n")
}
