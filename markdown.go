package eyebw

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteMarkdown(path string, report *Report) error {
	var b strings.Builder
	fmt.Fprintln(&b, "# 眼缓冲带宽台账")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "口径：逻辑字节包含缓存命中的访问请求；实际传输只统计未命中读取、写入和不可缓存的显示读取。缓存命中的 `transfer_bytes` 为 0，不会二次计传输。")
	fmt.Fprintln(&b)

	for _, scenario := range report.Scenarios {
		fmt.Fprintf(&b, "## %s\n\n", scenario.Name)
		fmt.Fprintf(&b, "%s\n\n", scenario.Description)
		fmt.Fprintf(&b, "- 单眼分辨率：`%dx%d`；MSAA：`%dx`；帧数：`%d`；动态注视点：`%t`\n",
			scenario.Config.Width, scenario.Config.Height, scenario.Config.MSAASamples, scenario.Frames, scenario.DynamicGaze)
		if scenario.Config.Budget.FrameBytes > 0 {
			fmt.Fprintf(&b, "- 每帧预算：`%s`\n", FormatBytes(scenario.Config.Budget.FrameBytes))
		}
		status := "PASS"
		if scenario.Success && scenario.ExpectQualityFailure {
			status = "PASS：已捕获预期中心质量失败"
		} else if !scenario.Success {
			status = "FAIL: " + scenario.FailureReason
		}
		fmt.Fprintf(&b, "- 场景预期校验：**%s**\n\n", status)

		fmt.Fprintln(&b, "| 管线 | 平均实际传输/帧 | 平均读 | 平均写 | 逻辑总量/帧 | 缓存命中率(字节) | 预算超限帧 | 首要热点 | 中心质量 |")
		fmt.Fprintln(&b, "|---|---:|---:|---:|---:|---:|---:|---|---|")
		for _, p := range scenario.Pipelines {
			hotspot := "-"
			if len(p.Hotspots) > 0 {
				h := p.Hotspots[0]
				hotspot = fmt.Sprintf("%s (%s, %.1f%%)", h.Label, FormatBytes(h.TransferBytes/uint64(max1(p.FrameCount))), h.ShareOfTransfer*100)
			}
			quality := "PASS"
			if p.QualityFailedFrames > 0 {
				quality = fmt.Sprintf("FAIL %d/%d 帧", p.QualityFailedFrames, p.FrameCount)
			}
			budget := "0"
			if scenario.Config.Budget.FrameBytes > 0 {
				budget = fmt.Sprintf("%d/%d", p.BudgetExceededFrames, p.FrameCount)
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %.1f%% | %s | %s | %s |\n",
				p.Label,
				FormatBytes(p.Average.TransferTotal()),
				FormatBytes(p.Average.TransferReadBytes),
				FormatBytes(p.Average.TransferWriteBytes),
				FormatBytes(p.Average.LogicalTotal()),
				p.Average.CacheHitRatioBytes*100,
				budget,
				hotspot,
				quality,
			)
		}
		fmt.Fprintln(&b)

		fmt.Fprintln(&b, "### 四类数据的平均实际传输")
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "| 管线 | 颜色 | 深度 | 历史 | 补绘 |")
		fmt.Fprintln(&b, "|---|---:|---:|---:|---:|")
		for _, p := range scenario.Pipelines {
			values := map[string]uint64{}
			for _, class := range p.Classes {
				values[class.Name] = (class.TransferReadBytes + class.TransferWriteBytes) / uint64(max1(p.FrameCount))
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				p.Label,
				FormatBytes(values[string(ClassColor)]),
				FormatBytes(values[string(ClassDepth)]),
				FormatBytes(values[string(ClassHistory)]),
				FormatBytes(values[string(ClassRepaint)]),
			)
		}
		fmt.Fprintln(&b)

		violations := collectViolations(scenario)
		if len(violations) > 0 {
			fmt.Fprintln(&b, "### 质量/预算记录")
			fmt.Fprintln(&b)
			for _, violation := range violations {
				fmt.Fprintf(&b, "- %s\n", violation)
			}
			fmt.Fprintln(&b)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func max1(v int) int {
	if v <= 0 {
		return 1
	}
	return v
}

func collectViolations(scenario *ScenarioReport) []string {
	var out []string
	seen := map[string]bool{}
	for _, frame := range scenario.FrameReports {
		for _, p := range frame.Pipelines {
			for _, v := range p.Quality.Violations {
				line := fmt.Sprintf("[质量][帧 %d][%s] %s", frame.Index, p.ID, v)
				if !seen[line] {
					seen[line] = true
					out = append(out, line)
				}
			}
			for _, v := range p.Quality.ProbeViolations {
				line := fmt.Sprintf("[探针][帧 %d][%s] %s", frame.Index, p.ID, v)
				if !seen[line] {
					seen[line] = true
					out = append(out, line)
				}
			}
			if p.Budget.Exceeded {
				line := fmt.Sprintf("[预算][帧 %d][%s] 实际 %s / 预算 %s，超限 %s",
					frame.Index, p.ID, FormatBytes(p.Budget.ActualBytes), FormatBytes(p.Budget.BudgetBytes), FormatBytes(p.Budget.OverageBytes))
				if !seen[line] {
					seen[line] = true
					out = append(out, line)
				}
			}
		}
	}
	if len(out) > 12 {
		extra := len(out) - 12
		out = append(out[:12], fmt.Sprintf("... 另有 %d 条同类记录，完整数据见 report.json / frames.csv", extra))
	}
	return out
}
