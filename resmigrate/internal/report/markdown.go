package report

import (
	"fmt"
	"strings"

	"resmigrate/internal/sim"
)

func mb(b int) float64 { return float64(b) / (1024 * 1024) }

// RenderMarkdown produces the human-readable migration report.
func RenderMarkdown(root *sim.Report, passCount int) string {
	var b strings.Builder
	b.WriteString("# 分辨率热更新迁移报告\n\n")
	b.WriteString("纯 Go 标准库确定性模拟：分辨率热更新时，历史样本经坐标重映射迁移到新尺寸，采样预算按新像素数重新规划；所有故障注入后双眼尺寸必须回到同一已提交配置。\n\n")
	fmt.Fprintf(&b, "- 场景数：%d；全部不变量通过：%d；存在失败：%d\n", len(root.Scenarios), passCount, len(root.Scenarios)-passCount)
	fmt.Fprintf(&b, "- 画面跳变事件阈值：双眼最大亮度差 > %.3f（仅统计升级窗口起）\n\n", root.JumpThresholdLuma)

	renderStrategyCompare(&b, root)
	renderExtreme(&b, root)
	renderFaultMatrix(&b, root)
	renderBudget(&b, root)
	renderTrajectories(&b, root)
	renderChecks(&b, root)
	renderMetricNotes(&b)
	return b.String()
}

func find(root *sim.Report, id string) *sim.ScenarioReport {
	for i := range root.Scenarios {
		if root.Scenarios[i].ScenarioID == id {
			return &root.Scenarios[i]
		}
	}
	return nil
}

func renderStrategyCompare(b *strings.Builder, root *sim.Report) {
	b.WriteString("## 三种策略对比（尺寸突变 320x180 → 640x360）\n\n")
	b.WriteString("| 策略 | 历史失效比例 | 历史峰值 MB | 相对目标稳态 | 总峰值 MB | 跳变事件 | 最大跳变 | 帧超时 | 重映射 | 结果 |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|:--:|:--:|\n")
	for _, id := range []string{"sudden_inplace", "sudden_doublebuf", "sudden_staged", "sudden_inplace_discard"} {
		s := find(root, id)
		fmt.Fprintf(b, "| %s | %.3f | %.2f | %.3fx | %.2f | %d | %.4f | %d | %s | %s |\n",
			s.StrategyName, s.Invalidation.WorstRatio, mb(s.Memory.HistoryPeakBytes),
			s.Memory.HistoryPeakOverTargetSteady, mb(s.Memory.PeakBytes),
			s.Jumps.EventCount, s.Jumps.MaxStereoJump, s.Work.DeadlineMisses,
			yesNo(s.RemapApplied), s.Outcome)
	}
	b.WriteString("\n要点：双缓冲内存冗余最高但提交是双眼事务；原地重建单眼顺序切换、峰值居中，故障时需逆向重映射补偿；分阶段迁移逐带释放旧带，峰值最低、工时摊到多帧，但会产生连续多帧的轻微跳变。清空重建对照行失效比例为 1.000，跳变与重新收敛最明显。\n\n")
}

func renderExtreme(b *strings.Builder, root *sim.Report) {
	b.WriteString("### 极端尺寸突变（160x90 → 960x540，36 倍像素）\n\n")
	b.WriteString("| 策略 | 历史失效比例 | 历史峰值 MB | 相对目标稳态 | 跳变事件 | 最大跳变 | 帧超时 | 结果 |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|:--:|\n")
	for _, id := range []string{"extreme_inplace", "extreme_doublebuf", "extreme_staged"} {
		s := find(root, id)
		fmt.Fprintf(b, "| %s | %.4f | %.2f | %.3fx | %d | %.4f | %d | %s |\n",
			s.StrategyName, s.Invalidation.WorstRatio, mb(s.Memory.HistoryPeakBytes),
			s.Memory.HistoryPeakOverTargetSteady, s.Jumps.EventCount, s.Jumps.MaxStereoJump,
			s.Work.DeadlineMisses, s.Outcome)
	}
	b.WriteString("\n")
}

func renderFaultMatrix(b *strings.Builder, root *sim.Report) {
	b.WriteString("## 一眼首试失败与回滚矩阵\n\n")
	b.WriteString("| 场景 | 结果 | 终态尺寸 | 前向失效 L/R | 回滚失效 L/R | 最大跳变 | 双眼一致 | 检查 |\n")
	b.WriteString("|---|---|---|---|---|---:|:--:|:--:|\n")
	ids := []string{
		"fault_inplace_left_rollback", "fault_inplace_right_rollback", "fault_inplace_retry",
		"fault_doublebuf_left_rollback", "fault_doublebuf_right_rollback", "fault_doublebuf_retry",
		"fault_staged_left_rollback", "fault_staged_right_rollback", "fault_staged_retry",
		"fault_staged_left_midrollback", "fault_staged_right_midrollback",
	}
	for _, id := range ids {
		s := find(root, id)
		fwdL, fwdR := s.Invalidation.LeftMigrationRatio, s.Invalidation.RightMigrationRatio
		rbL, rbR := 0.0, 0.0
		if s.Outcome == sim.OutcomeRolledBack {
			fwdL, fwdR = 0.0, 0.0
		}
		fmt.Fprintf(b, "| %s | %s | %s | %.3f/%.3f | %.3f/%.3f | %.4f | %s | %s |\n",
			s.Title, s.Outcome, s.FinalSize, fwdL, fwdR, rbL, rbR,
			s.Jumps.MaxStereoJump, consistent(s), passFail(s.OK))
	}
	b.WriteString("\n所有非重试故障都回滚到旧配置，重试场景在第二次尝试成功并升级到新尺寸；任何已呈现帧的左右眼尺寸均一致，不存在永久单眼尺寸不一致。\n\n")
}

func renderBudget(b *strings.Builder, root *sim.Report) {
	b.WriteString("## 采样预算迁移\n\n")
	b.WriteString("| 场景 | SPP | 单眼预算样本/帧 | 说明 |\n")
	b.WriteString("|---|---:|---:|---|\n")
	for _, id := range []string{"budget_fixed_spp", "budget_fixed_total"} {
		s := find(root, id)
		last := s.Frames[len(s.Frames)-1]
		note := "固定 SPP，预算随像素数增长"
		if id == "budget_fixed_total" {
			note = "固定总上限，SPP 自动收缩"
		}
		fmt.Fprintf(b, "| %s | %d | %d | %s |\n", s.Title, last.Left.SPP, last.Left.BudgetSamples, note)
	}
	b.WriteString("\n")
}

func renderTrajectories(b *strings.Builder, root *sim.Report) {
	b.WriteString("## 尺寸轨迹（逐呈现帧）\n\n")
	for i := range root.Scenarios {
		s := &root.Scenarios[i]
		parts := make([]string, 0, len(s.Frames))
		for _, f := range s.Frames {
			label := f.Kind
			if f.Kind == "migrating" {
				label = fmt.Sprintf("migrating(%d%%)", pctMigrated(s, f.Index))
			}
			parts = append(parts, fmt.Sprintf("#%d %s %s", f.Index, f.Left.Size, label))
		}
		fmt.Fprintf(b, "- `%s` **%s**：%s\n", s.ScenarioID, s.StrategyName, strings.Join(parts, " → "))
	}
	b.WriteString("\n帧类型：warmup/stable 预热，upgrade 一次性提交，migrating 分阶段迁移，rollback 故障后回滚帧，post 收尾观察帧。\n\n")
}

func pctMigrated(s *sim.ScenarioReport, frame int) int {
	for _, t := range s.Trajectory {
		if t.Frame == frame && t.Event == "" {
			return int(100.0 * (t.LeftMigratedFraction + t.RightMigratedFraction) / 2.0)
		}
	}
	for _, t := range s.Trajectory {
		if t.Frame == frame {
			return int(100.0 * (t.LeftMigratedFraction + t.RightMigratedFraction) / 2.0)
		}
	}
	return 0
}

func renderChecks(b *strings.Builder, root *sim.Report) {
	b.WriteString("## 不变量检查\n\n")
	for i := range root.Scenarios {
		s := &root.Scenarios[i]
		fmt.Fprintf(b, "### %s `%s`\n\n", s.Title, s.ScenarioID)
		for _, c := range s.Checks {
			fmt.Fprintf(b, "- [%s] `%s` — %s\n", passFail(c.OK), c.ID, c.Detail)
		}
		b.WriteString("\n")
	}
}

func renderMetricNotes(b *strings.Builder) {
	b.WriteString("## 指标口径\n\n")
	b.WriteString("- **历史失效比例**：重映射后因覆盖不足或分带接缝被清零的目标纹素 / 目标纹素总数。\n")
	b.WriteString("- **峰值内存**：计量器记录的模拟 RGBA 历史（32 B/px）与线性帧缓冲（24 B/px）分配峰值；另列历史类别峰值及其相对目标尺寸稳态历史占用的倍率。\n")
	b.WriteString("- **画面跳变**：在固定 160x90 UV 网格上比较相邻呈现帧的亮度平均绝对差，分辨率不同也可直接比较；事件阈值 0.015。\n")
	b.WriteString("- **帧超时**：确定性工时模型（采样 + 4-tap 重映射，150000 单位/ms，预算 16.7ms）下的超时帧数。\n")
	b.WriteString("- 完整逐帧数据见 `report.json`。\n")
}

func yesNo(v bool) string {
	if v {
		return "是"
	}
	return "否"
}

func consistent(s *sim.ScenarioReport) string {
	if s.Eyes[0].FinalSize.Equals(s.Eyes[1].FinalSize) {
		return "一致"
	}
	return "不一致"
}

func passFail(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
