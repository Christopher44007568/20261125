package main

import (
	"fmt"
	"os"
	"strings"
)

func writeMarkdownReport(path string, report Report) error {
	var b strings.Builder
	b.WriteString("# HDR → SDR / PQ / HLG 模拟报告\n\n")
	fmt.Fprintf(&b, "生成时间：`%s`\n\n", report.GeneratedAt)
	b.WriteString("## 锚点\n\n")
	fmt.Fprintf(&b, "- HDR 灰卡：`%.1f nits`，即 18%% 反射率 × 100 nits 参考白。\n", report.GrayAnchor.HDRNits)
	fmt.Fprintf(&b, "- 三张软校验图都把声明灰卡映射到 sRGB proof code `%.0f`（线性 `%.4f`）。\n",
		report.GrayAnchor.ProofCode, report.GrayAnchor.ProofLuma)
	fmt.Fprintf(&b, "- 原生传递函数在该亮度的码值：SDR `%.0f`、PQ `%.0f`、HLG `%.0f`（本实现的 HLG 软校验采用 `E=Y_nits/1000` 归一）。\n\n",
		report.NativeGrayCode["sdr"], report.NativeGrayCode["pq"], report.NativeGrayCode["hlg"])
	b.WriteString("直方图为最终 proof code 的 Rec.709 亮度，64 个合并 bin；柱高使用平方根计数刻度，黄色 `GRAY=118` 线是声明位置。\n\n")

	b.WriteString("## 校验表\n\n")
	b.WriteString("| 场景 | 显示 | 灰卡 RGB | 最大误差 | 色偏 | 近白占比 | 控制区亮度 | 高光标准差 | 高光 bin | 结果 |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|:--:|\n")
	for _, scene := range report.Scenes {
		for _, m := range scene.Metrics {
			status := "PASS"
			if !m.Pass {
				status = "FAIL"
			}
			fmt.Fprintf(&b, "| `%s` | %s | `%.0f,%.0f,%.0f` | %.1f | %.1f | %.3f | %.1f | %.1f | %d | %s |\n",
				scene.ID, strings.ToUpper(m.DisplayID),
				m.GrayMeanRGB[0], m.GrayMeanRGB[1], m.GrayMeanRGB[2],
				m.GrayMaxError, m.GrayNeutralDelta, m.NearWhiteFraction,
				m.ControlMeanLuma, m.HighlightStdLuma, m.HighlightOccupancy, status)
		}
	}
	b.WriteString("\n## 输出文件\n\n")
	for _, scene := range report.Scenes {
		fmt.Fprintf(&b, "- `%s`：[%s](%s)\n", scene.ID, scene.ComparisonPNG, scene.ComparisonPNG)
	}
	b.WriteString("- 全场景总表：`hdr_three_scenes_all_displays.png`\n")
	b.WriteString("- 机器可读报告：`hdr_report.json`\n\n")
	b.WriteString("## 实现要点\n\n")
	b.WriteString("- 场景以线性 nits 生成，太阳/窗景和彩色灯可达数千 nits，阴影室内保留 `0.2–8 nits` 层级。\n")
	b.WriteString("- 高光曲线只在超过显示膝点后生效，并按每像素最大通道等比压缩，避免全局曝光改变灰卡和暗部。\n")
	b.WriteString("- SDR 使用 sRGB/BT.709；PQ 使用 ST 2084 OETF/EOTF 常量与往返校验；HLG 使用 BT.2100 OETF 分段公式。\n")
	b.WriteString("- PNG 是普通 sRGB 软校验图，因此可以直接查看；JSON 同时记录原生 PQ/HLG 灰阶码值。\n")

	return os.WriteFile(path, []byte(b.String()), 0o644)
}
