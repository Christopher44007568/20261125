package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeMarkdownReport(outDir string, report *Report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# 立体 UI 双眼校验报告\n\n")
	fmt.Fprintf(&b, "- 生成时间：`%s`\n", report.GeneratedAt)
	fmt.Fprintf(&b, "- 单眼分辨率：`%dx%d`；IPD：`%.3f m`\n", report.Options.Width, report.Options.Height, report.Options.IPD)
	fmt.Fprintf(&b, "- 场景/UI 组/每眼画面：`%d / %d / %d`\n", report.Summary.ScenarioCount, report.Summary.LayerCount, report.Summary.EyeFrameCount)
	fmt.Fprintf(&b, "- 元素状态：PASS `%d`，WARNING/EDGE_RISK `%d`，FAIL `%d`\n", report.Summary.PassCount, report.Summary.WarningCount, report.Summary.FailCount)
	fmt.Fprintf(&b, "- 坐标保护：`%s`\n", report.Summary.CopyGuard)
	fmt.Fprintf(&b, "- 系统层边缘清晰度：`%t`\n", report.Summary.SystemSharpnessPass)
	fmt.Fprintf(&b, "- 总结论：`%s`\n\n", map[bool]string{true: "PASS", false: "FAIL"}[report.Summary.OverallPass])

	b.WriteString("## 结果汇总\n\n")
	b.WriteString("| 场景 | UI 组 | 状态 | 原始水平坐标差 px | 水平角视差 arcmin | 朴素复制误差 px | 最大垂直角差 arcmin | 系统锐度 |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|---:|---|\n")
	for _, scenario := range report.Scenarios {
		for _, layer := range scenario.Layers {
			p := summarizeParallax(layer.Glyphs)
			sharpness := "-"
			if layer.DepthClass == "system" {
				sharpness = sharpnessSummary(layer)
			}
			fmt.Fprintf(&b, "| %s | %s | `%s` | %.2f | %.1f | %.2f | %.2f | %s |\n",
				scenario.TitleZH, layer.NameZH, layer.Status,
				p.MeanHorizontalPx, p.MeanAngularArcMin, p.MeanCopyErrorPx,
				p.MaxAbsVerticalArcMin, sharpness)
		}
	}

	b.WriteString("\n## 关键判定说明\n\n")
	b.WriteString("- `horizontalPxLeftMinusRight > 0` 表示近平面常见的交叉视差；近平面数值最大，中距离次之，系统层最小。\n")
	b.WriteString("- 非对称 FOV 会移动每眼主点，因此原始像素坐标差包含标定偏移；深度判断应同时查看 `angular_horizontal_disparity_arcmin`。\n")
	b.WriteString("- `naiveSameScreenCoordinateCopyErrorPx` 是把左眼屏幕坐标原样复制到右眼时，相对正确独立投影的像素误差。\n")
	b.WriteString("- 边界状态包括 `visible`、`edge_risk`、`clipped`、`culled`；完整逐元素数据见 `boundary.csv`。\n")
	b.WriteString("- 可读性包含角字高、笔画设备像素、WCAG 对比度，以及快速转头/单眼降级后的运动残差，见 `readability.csv`。\n")
	b.WriteString("- 系统层锐度使用实际 PNG 扫描线的 10%–90% 边缘过渡宽度，见 `sharpness.csv`。\n\n")

	b.WriteString("## 每眼画面\n\n")
	for _, scenario := range report.Scenarios {
		fmt.Fprintf(&b, "### %s（`%s`）\n\n", scenario.TitleZH, scenario.ID)
		for _, layer := range scenario.Layers {
			left := layer.FrameFiles[LeftEye]
			right := layer.FrameFiles[RightEye]
			fmt.Fprintf(&b, "#### %s · %s\n\n", layer.NameZH, layer.LayerID)
			fmt.Fprintf(&b, "<img src=\"%s\" alt=\"%s left\" width=\"360\"> <img src=\"%s\" alt=\"%s right\" width=\"360\">\n\n",
				left, layer.LayerID, right, layer.LayerID)
			fmt.Fprintf(&b, "- 左眼：`%s`\n", left)
			fmt.Fprintf(&b, "- 右眼：`%s`\n\n", right)
		}
	}

	path := filepath.Join(outDir, "report.md")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func sharpnessSummary(layer LayerReport) string {
	var medians []string
	pass := true
	for _, element := range layer.Elements {
		for _, eye := range []EyeID{LeftEye, RightEye} {
			s, ok := element.Sharpness[eye]
			if !ok {
				continue
			}
			pass = pass && s.Pass
			medians = append(medians, fmt.Sprintf("%s %.2fpx", eye, s.MedianTransitionPx))
		}
	}
	if len(medians) == 0 {
		return "未检测"
	}
	status := "PASS"
	if !pass {
		status = "FAIL"
	}
	return fmt.Sprintf("`%s` %s", status, strings.Join(medians, " / "))
}
