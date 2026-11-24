package eyestage

import "sort"

func buildStageStats(values map[stageKey][]stageValue, config Config, longTail *[]LongTail) []StageStat {
	stats := make([]StageStat, 0, len(Eyes)*len(Stages))
	for _, eye := range Eyes {
		for _, stage := range Stages {
			key := stageKey{Eye: eye, Stage: stage}
			records := values[key]
			numbers := make([]int64, 0, len(records))
			for _, record := range records {
				numbers = append(numbers, record.NS)
			}
			stat := summarize(numbers, config.TailPercentile, config.HistogramBuckets)
			stat.Eye = eye
			stat.Stage = stage
			stats = append(stats, stat)
			if stat.Count == 0 {
				continue
			}
			for _, record := range records {
				if record.NS >= stat.TailCutoff && record.NS > stat.P50NS {
					*longTail = append(*longTail, LongTail{
						FrameID:     record.FrameID,
						Eye:         record.Eye,
						Stage:       record.Stage,
						DurationNS:  record.NS,
						ThresholdNS: stat.TailCutoff,
						Percentile:  config.TailPercentile,
					})
				}
			}
		}
	}
	sort.SliceStable(*longTail, func(i, j int) bool {
		if (*longTail)[i].DurationNS == (*longTail)[j].DurationNS {
			if (*longTail)[i].FrameID == (*longTail)[j].FrameID {
				if (*longTail)[i].Eye == (*longTail)[j].Eye {
					return stageIndex((*longTail)[i].Stage) < stageIndex((*longTail)[j].Stage)
				}
				return (*longTail)[i].Eye < (*longTail)[j].Eye
			}
			return (*longTail)[i].FrameID < (*longTail)[j].FrameID
		}
		return (*longTail)[i].DurationNS > (*longTail)[j].DurationNS
	})
	return stats
}

func stageIndex(stage string) int {
	for index, candidate := range Stages {
		if candidate == stage {
			return index
		}
	}
	return len(Stages)
}

func groupLongTailFrames(entries []LongTail) []LongTailFrame {
	byFrame := make(map[string]*LongTailFrame)
	for _, entry := range entries {
		frame, ok := byFrame[entry.FrameID]
		if !ok {
			frame = &LongTailFrame{FrameID: entry.FrameID, Stages: make([]LongTail, 0)}
			byFrame[entry.FrameID] = frame
		}
		frame.Stages = append(frame.Stages, entry)
		frame.StageCount++
		if entry.DurationNS > frame.LongestNS {
			frame.LongestNS = entry.DurationNS
		}
	}
	result := make([]LongTailFrame, 0, len(byFrame))
	for _, frame := range byFrame {
		result = append(result, *frame)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].LongestNS == result[j].LongestNS {
			return result[i].FrameID < result[j].FrameID
		}
		return result[i].LongestNS > result[j].LongestNS
	})
	return result
}

func flattenCrossOverlaps(values map[[2]string]*CrossStageOverlap) []CrossStageOverlap {
	result := make([]CrossStageOverlap, 0, len(values))
	for _, value := range values {
		result = append(result, *value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].LeftStage == result[j].LeftStage {
			return stageIndex(result[i].RightStage) < stageIndex(result[j].RightStage)
		}
		return stageIndex(result[i].LeftStage) < stageIndex(result[j].LeftStage)
	})
	return result
}

func summarizeIdle(values map[string][]int64, percentileValue float64) []IdleSummary {
	result := make([]IdleSummary, 0, len(Eyes))
	for _, eye := range Eyes {
		numbers := values[eye]
		summary := IdleSummary{Eye: eye}
		if len(numbers) > 0 {
			sorted := append([]int64(nil), numbers...)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
			summary.MedianWaitNS = percentile(sorted, 50)
			summary.P95WaitNS = percentile(sorted, percentileValue)
			summary.MaxWaitNS = sorted[len(sorted)-1]
			for _, value := range sorted {
				summary.TotalWaitNS += value
				if value > 0 {
					summary.FramesWithWait++
				}
			}
		}
		result = append(result, summary)
	}
	return result
}

func sortDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	sorted := append([]Diagnostic(nil), diagnostics...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left, right := sorted[i], sorted[j]
		if left.Frame != right.Frame {
			return left.Frame < right.Frame
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Eye != right.Eye {
			return left.Eye < right.Eye
		}
		if left.Stage != right.Stage {
			return stageIndex(left.Stage) < stageIndex(right.Stage)
		}
		return left.Field < right.Field
	})
	return sorted
}

func recountDiagnostics(report *Report) {
	report.Summary.MissingStages = 0
	report.Summary.TimestampRegressions = 0
	report.Summary.UnsupportedTimes = 0
	report.Summary.FallbackTimes = 0
	for _, diagnostic := range report.Diagnostics {
		switch diagnostic.Code {
		case "missing_stage":
			report.Summary.MissingStages++
		case "timestamp_regression":
			report.Summary.TimestampRegressions++
		case "unsupported_timestamp":
			report.Summary.UnsupportedTimes++
		case "timestamp_fallback":
			report.Summary.FallbackTimes++
		}
	}
}
