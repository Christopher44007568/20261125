package xrpath

import (
	"fmt"
	"strings"
)

func RenderMarkdown(report Report, storedImageMismatches []Mismatch, suite DrillSuite, hashDrills []HashBlindDrill) string {
	var b strings.Builder
	b.WriteString("# XR 固定路径录制/重放核对报告\n\n")

	status := "通过"
	if !report.OK || len(storedImageMismatches) > 0 || !suite.OK || !hashDrillsOK(hashDrills) {
		status = "失败"
	}
	fmt.Fprintf(&b, "- 总状态：**%s**\n", status)
	fmt.Fprintf(&b, "- 帧数：期望 %d / 实际 %d\n", report.FrameCountExpected, report.FrameCountActual)
	fmt.Fprintf(&b, "- 逐项检查数：%d；分叉数：%d\n", report.CheckCount, report.MismatchCount)
	fmt.Fprintf(&b, "- 落盘图片复核异常：%d\n\n", len(storedImageMismatches))

	b.WriteString("## 首个分叉\n\n")
	if report.FirstDivergence == nil {
		b.WriteString("基线与重新运行完全一致，未发现分叉。\n\n")
	} else {
		m := report.FirstDivergence
		fmt.Fprintf(&b, "- 帧：`%d`\n", m.Frame)
		fmt.Fprintf(&b, "- 子系统：`%s`\n", m.Subsystem)
		if m.Item != "" {
			fmt.Fprintf(&b, "- 条目：`%s`\n", m.Item)
		}
		if m.Field != "" {
			fmt.Fprintf(&b, "- 字段：`%s`\n", m.Field)
		}
		fmt.Fprintf(&b, "- 详情：期望 `%v`，实际 `%v`；%s\n\n", m.Expected, m.Actual, m.Detail)
	}

	b.WriteString("## 子系统检查计数\n\n")
	b.WriteString("| 子系统 | 检查项 | 分叉 |\n| --- | ---: | ---: |\n")
	for _, name := range subsystemNames() {
		fmt.Fprintf(&b, "| `%s` | %d | %d |\n", name, report.ChecksBySubsystem[name], report.MismatchesBySystem[name])
	}
	b.WriteString("\n")

	b.WriteString("## 故障注入定位演练\n\n")
	b.WriteString("| 注入帧 | 故障 | 期望首分叉子系统 | 实际首分叉 | 结果 |\n")
	b.WriteString("| ---: | --- | --- | --- | --- |\n")
	for _, drill := range suite.Drills {
		actual := "-"
		if drill.FirstDivergence != nil {
			actual = fmt.Sprintf("frame=%d, `%s`", drill.FirstDivergence.Frame, drill.FirstDivergence.Subsystem)
		}
		result := "OK"
		if !drill.Detected {
			result = "未按预期定位"
		}
		fmt.Fprintf(&b, "| %d | `%s` | `%s` | %s | %s |\n", drill.Fault.Frame, drill.Fault.Kind, drill.ExpectedSubsystem, actual, result)
	}
	b.WriteString("\n")

	b.WriteString("## 图片哈希不是几何/资源检查的替代品\n\n")
	for _, drill := range hashDrills {
		result := "OK"
		if !drill.OK {
			result = "失败"
		}
		first := "无分叉"
		if drill.FirstDivergence != nil {
			first = fmt.Sprintf("frame=%d subsystem=`%s`", drill.FirstDivergence.Frame, drill.FirstDivergence.Subsystem)
		}
		fmt.Fprintf(&b, "- `%s`: %s；首分叉 %s；姿态异常 %d，资源异常 %d\n", drill.Name, result, first, drill.PoseMismatches, drill.ResourceMismatches)
	}
	b.WriteString("\n即使左右眼 `pixel_sha256` 与 `png_sha256` 完全相同，比较器仍会先执行姿态、资源状态和可见几何检查。\n\n")

	if len(report.Mismatches) > 0 {
		b.WriteString("## 前 20 个分叉\n\n")
		b.WriteString("| 帧 | 子系统 | 条目 | 字段 | 说明 |\n| ---: | --- | --- | --- | --- |\n")
		for i, m := range report.Mismatches {
			if i >= 20 {
				break
			}
			detail := m.Detail
			if detail == "" {
				detail = fmt.Sprintf("expected=%v actual=%v", m.Expected, m.Actual)
			}
			fmt.Fprintf(&b, "| %d | `%s` | `%s` | `%s` | %s |\n", m.Frame, m.Subsystem, m.Item, m.Field, escapeTable(detail))
		}
		b.WriteString("\n")
	}

	if len(storedImageMismatches) > 0 {
		b.WriteString("## 落盘图片异常\n\n")
		for _, m := range storedImageMismatches {
			fmt.Fprintf(&b, "- frame %d `%s` `%s`: %s\n", m.Frame, m.Subsystem, m.Field, m.Detail)
		}
	}

	return b.String()
}

func subsystemNames() []string {
	return []string{
		SubsystemSequence, SubsystemLifecycle, SubsystemParameters, SubsystemTracking,
		SubsystemPose, SubsystemResource, SubsystemVisibility, SubsystemReprojection,
		SubsystemLeftImage, SubsystemRightImage, SubsystemFrameHash,
	}
}

func hashDrillsOK(drills []HashBlindDrill) bool {
	for _, drill := range drills {
		if !drill.OK {
			return false
		}
	}
	return true
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
