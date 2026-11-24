package eyestage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"text/tabwriter"
)

func WriteJSON(w io.Writer, report *Report) error {
	if err := validateReport(report); err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func WriteText(w io.Writer, report *Report) error {
	if err := validateReport(report); err != nil {
		return err
	}
	out := bufio.NewWriter(w)

	fmt.Fprintln(out, "眼别阶段时间归因")
	fmt.Fprintln(out, strings.Repeat("=", 72))
	fmt.Fprintf(out, "帧数: %d  配对: %d  未配对: %d\n",
		report.Summary.Frames, report.Summary.PairedFrames, report.Summary.UnpairedFrames)
	fmt.Fprintf(out, "阶段缺失: %d  时间戳回退/倒退: %d  不支持时间戳: %d  回退来源: %d  长尾帧: %d  长尾阶段: %d\n",
		report.Summary.MissingStages,
		report.Summary.TimestampRegressions,
		report.Summary.UnsupportedTimes,
		report.Summary.FallbackTimes,
		report.Summary.LongTailFrames,
		report.Summary.LongTailStages,
	)

	writeFrameOverview(out, report)
	writeHistograms(out, report)
	writeLongTail(out, report)
	writeOverlaps(out, report)
	writeIdleWait(out, report)
	writeDiagnostics(out, report)
	return out.Flush()
}

func validateReport(report *Report) error {
	if report == nil {
		return errors.New("report is nil")
	}
	return nil
}

func writeFrameOverview(out *bufio.Writer, report *Report) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "逐眼帧概览（阶段耗时来自逐阶段归因，不用单帧总时长替代）")
	for _, frame := range report.Frames {
		fmt.Fprintf(out, "帧 %s  帧跨度=%s（仅上下文）\n", frame.FrameID, formatNS(frame.FrameSpanNS))
		for _, eye := range frame.Eyes {
			status := "缺失"
			if eye.Present {
				status = "存在"
			}
			fmt.Fprintf(out, "  %-10s %s  完整阶段=%d  阶段耗时=%s  活跃跨度=%s",
				eyeLabel(eye.Eye), status, eye.CompleteStages, formatNS(eye.StageDurationNS), formatNS(eye.ActiveSpanNS))
			if eye.IdleWaitNS != nil {
				fmt.Fprintf(out, "  空等=%s", formatNS(*eye.IdleWaitNS))
			}
			fmt.Fprintln(out)
			if len(eye.MissingStages) > 0 {
				fmt.Fprintf(out, "    缺失阶段: %s\n", strings.Join(eye.MissingStages, ", "))
			}
		}
	}
}

func writeHistograms(out *bufio.Writer, report *Report) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "阶段耗时直方图（按眼、按阶段）")
	for _, stat := range report.StageStats {
		fmt.Fprintf(out, "\n[%s] %s n=%d", eyeLabel(stat.Eye), stageLabel(stat.Stage), stat.Count)
		if stat.Count == 0 {
			fmt.Fprintln(out, "  无可用耗时")
			continue
		}
		fmt.Fprintf(out, "  min=%s p50=%s p95=%s p99=%s max=%s\n",
			formatNS(stat.MinNS), formatNS(stat.P50NS), formatNS(stat.P95NS), formatNS(stat.P99NS), formatNS(stat.MaxNS))
		renderHistogram(out, stat.Histogram)
	}
}

func renderHistogram(out *bufio.Writer, buckets []HistogramBucket) {
	if len(buckets) == 0 {
		return
	}
	maxCount := 0
	for _, bucket := range buckets {
		if bucket.Count > maxCount {
			maxCount = bucket.Count
		}
	}
	const maximumBar = 40
	for _, bucket := range buckets {
		width := 0
		if maxCount > 0 && bucket.Count > 0 {
			width = int(math.Round(float64(bucket.Count) / float64(maxCount) * maximumBar))
			if width < 1 {
				width = 1
			}
		}
		if bucket.UpperNS <= bucket.LowerNS+1 {
			fmt.Fprintf(out, "  %-23s | %-40s %d\n",
				formatNS(bucket.LowerNS), strings.Repeat("#", width), bucket.Count)
			continue
		}
		fmt.Fprintf(out, "  %10s..%-10s | %-40s %d\n",
			formatNS(bucket.LowerNS), formatNS(bucket.UpperNS), strings.Repeat("#", width), bucket.Count)
	}
}

func writeLongTail(out *bufio.Writer, report *Report) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "长尾帧（逐眼逐阶段 P%.0f 阈值归因，不使用单帧总时长分类）\n", report.Config.TailPercentile)
	if len(report.LongTail) == 0 {
		fmt.Fprintln(out, "  无")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, frame := range report.LongTailFrames {
		fmt.Fprintf(tw, "  帧 %s\t最长阶段=%s\t阶段数=%d\n", frame.FrameID, formatNS(frame.LongestNS), frame.StageCount)
		for _, entry := range frame.Stages {
			fmt.Fprintf(tw, "    %s\t%s\t%s\t阈值=%s\n",
				eyeLabel(entry.Eye), stageLabel(entry.Stage),
				formatNS(entry.DurationNS), formatNS(entry.ThresholdNS))
		}
	}
	tw.Flush()
}

func writeOverlaps(out *bufio.Writer, report *Report) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "跨眼对应阶段重叠")
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  阶段\t比较帧\t有重叠帧\t左眼耗时\t右眼耗时\t重叠\t仅左\t仅右\t重叠率")
	for _, overlap := range report.StageOverlaps {
		fmt.Fprintf(tw, "  %s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%.1f%%\n",
			stageLabel(overlap.Stage), overlap.FramesCompared, overlap.FramesOverlap,
			formatNS(overlap.LeftNS), formatNS(overlap.RightNS),
			formatNS(overlap.OverlapNS), formatNS(overlap.LeftOnlyNS), formatNS(overlap.RightOnlyNS),
			overlap.OverlapRatio*100)
	}
	tw.Flush()
	if len(report.CrossStageOverlaps) == 0 {
		fmt.Fprintln(out, "  跨阶段错位重叠: 无")
		return
	}
	fmt.Fprintln(out, "  跨阶段错位重叠:")
	for _, overlap := range report.CrossStageOverlaps {
		fmt.Fprintf(out, "    左眼/%s <-> 右眼/%s  重叠=%s  帧数=%d\n",
			stageLabel(overlap.LeftStage), stageLabel(overlap.RightStage),
			formatNS(overlap.OverlapNS), overlap.FrameCount)
	}
}

func writeIdleWait(out *bufio.Writer, report *Report) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "单眼空等（对侧眼执行阶段、本眼无阶段活动）")
	for _, idle := range report.IdleWaits {
		fmt.Fprintf(out, "  %-10s 有等待帧=%d  总等待=%s  中位=%s  p95=%s  最大=%s\n",
			eyeLabel(idle.Eye), idle.FramesWithWait, formatNS(idle.TotalWaitNS),
			formatNS(idle.MedianWaitNS), formatNS(idle.P95WaitNS), formatNS(idle.MaxWaitNS))
	}
}

func writeDiagnostics(out *bufio.Writer, report *Report) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "诊断")
	if len(report.Diagnostics) == 0 {
		fmt.Fprintln(out, "  无")
		return
	}
	for _, diagnostic := range report.Diagnostics {
		context := make([]string, 0, 3)
		if diagnostic.Frame != "" {
			context = append(context, "frame="+diagnostic.Frame)
		}
		if diagnostic.Eye != "" {
			context = append(context, "eye="+diagnostic.Eye)
		}
		if diagnostic.Stage != "" {
			context = append(context, "stage="+diagnostic.Stage)
		}
		if diagnostic.Field != "" {
			context = append(context, "field="+diagnostic.Field)
		}
		if diagnostic.FallbackSource != "" {
			context = append(context, "fallback_source="+diagnostic.FallbackSource)
		}
		if diagnostic.Raw != "" {
			context = append(context, "raw="+diagnostic.Raw)
		}
		fmt.Fprintf(out, "  [%s] %s %s\n", diagnostic.Code, strings.Join(context, " "), diagnostic.Message)
	}
}

func eyeLabel(eye string) string {
	switch eye {
	case EyeLeft:
		return "左眼(left)"
	case EyeRight:
		return "右眼(right)"
	default:
		return eye
	}
}

func stageLabel(stage string) string {
	switch stage {
	case StageCaptureScene:
		return "capture_scene(采集场景)"
	case StageReprojection:
		return "reprojection(重投影)"
	case StageDistortion:
		return "distortion(畸变)"
	case StageComposition:
		return "composition(合成)"
	default:
		return stage
	}
}
