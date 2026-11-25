package main

import (
	"fmt"
	"strings"
)

func renderMDScenarios(b *strings.Builder, r *Report) {
	b.WriteString("## 4. 逐场景记录\n\n")
	for _, sc := range r.Scenarios {
		fmt.Fprintf(b, "### %s %s\n\n%s\n\n", sc.ID, sc.Title, sc.Description)
		for _, shot := range sc.Shots {
			fmt.Fprintf(b, "![%s](%s)\n\n*%s（第 %d 帧）*\n\n",
				shot.Caption, shot.File, shot.Caption, shot.Frame)
		}
		if len(sc.Alerts) > 0 {
			b.WriteString("告警：\n")
			for _, a := range sc.Alerts {
				fmt.Fprintf(b, "- `%s` %s verdict=`%s` frame=%d close=%d value=%.2f\n",
					a.ID, a.ReasonCode, a.Verdict, a.Frame, a.CloseFrame, a.SignalValue)
			}
			b.WriteString("\n")
		}
		if len(sc.Episodes) > 0 {
			b.WriteString("| 片段 | 链 | 方案 | 触发 | 原因码 | 起 | 止 | 持续帧 | 结局 | 连续 | 误报 | 画质(起/最差/终) |\n")
			b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
			for _, e := range sc.Episodes {
				endQ := "-"
				if e.EndMetrics != nil {
					endQ = fmt.Sprintf("%.0f", e.EndMetrics.QualityScore)
				}
				fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %d | %d | %d | %s | %t | %t | %.0f / %.0f / %s |\n",
					e.ID, e.ChainID, e.Mode, e.Trigger, e.ReasonCode,
					e.StartFrame, e.EndFrame, e.DurationFrames, e.Outcome,
					e.Continuous, e.FalsePositive,
					e.StartMetrics.QualityScore, e.WorstMetrics.QualityScore, endQ)
			}
			b.WriteString("\n")
			for _, e := range sc.Episodes {
				fmt.Fprintf(b, "- %s 恢复条件：%s（需 %d 帧；指标阈值 %.2f）\n",
					e.ID, e.RecoveryCondition.Description,
					e.RecoveryCondition.HoldFrames, e.RecoveryCondition.Threshold)
			}
			b.WriteString("\n")
		}
		for _, ch := range sc.Chains {
			fmt.Fprintf(b, "- 链 `%s`：class=`%s` 持续=%df 重开=%t 升级=%d 部分恢复=%d 全恢复=%t 误报=%t\n\n",
				ch.ID, ch.Classification, ch.DurationFrames, ch.Reopened,
				len(ch.Escalations), len(ch.PartialRecoveries),
				ch.FullRecovery, ch.FalsePositive)
		}
	}
}

func renderMDRunbook(b *strings.Builder) {
	b.WriteString("## 5. 运行方式\n\n")
	b.WriteString("```sh\n")
	b.WriteString("go test ./...          # 状态机、分类、安全不变量、Web API 测试\n")
	b.WriteString("go run .               # 重新生成 artifacts/ 下报告、trace 与 PNG\n")
	b.WriteString("go run . -serve :8080  # 纯标准库交互状态画面（注入故障/步进/裁剪/重置）\n")
	b.WriteString("```\n")
}
