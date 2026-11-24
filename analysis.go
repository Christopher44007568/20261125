package eyestage

import "fmt"

type stageValue struct {
	FrameID string
	Eye     string
	Stage   string
	NS      int64
}

type stageKey struct {
	Eye   string
	Stage string
}

type eyeCalc struct {
	summary        EyeFrameSummary
	stageIntervals map[string]Interval
	coverage       []Interval
	spanStart      int64
	spanEnd        int64
	hasSpan        bool
}

func Analyze(dataset *Dataset, config Config) *Report {
	config = normalizeConfig(config)
	report := &Report{
		Config:      config,
		Diagnostics: make([]Diagnostic, 0),
		Frames:      make([]FrameSummary, 0),
	}
	if dataset == nil {
		return report
	}
	report.Diagnostics = append(report.Diagnostics, dataset.Diagnostics...)
	report.Summary.Frames = len(dataset.Frames)

	values := make(map[stageKey][]stageValue)
	overlaps := make(map[string]*StageOverlap)
	for _, stage := range Stages {
		overlaps[stage] = &StageOverlap{Stage: stage}
	}
	crossOverlaps := make(map[[2]string]*CrossStageOverlap)
	idleValues := map[string][]int64{EyeLeft: {}, EyeRight: {}}
	lastFrameStart := make(map[string]int64)
	lastFrameID := make(map[string]string)

	for _, frame := range dataset.Frames {
		if frame == nil {
			continue
		}
		left := calculateEye(report, frame, EyeLeft, values)
		right := calculateEye(report, frame, EyeRight, values)
		frameSummary := FrameSummary{FrameID: frame.ID, Eyes: []EyeFrameSummary{left.summary, right.summary}}
		frameSummary.FrameSpanNS = frameSpan(left, right)

		bothPresent := frame.Eyes[EyeLeft] != nil && frame.Eyes[EyeRight] != nil
		if bothPresent {
			report.Summary.PairedFrames++
			leftWait := subtractIntervals(right.coverage, left.coverage)
			rightWait := subtractIntervals(left.coverage, right.coverage)
			leftWaitNS := intervalLength(leftWait)
			rightWaitNS := intervalLength(rightWait)
			frameSummary.Eyes[0].IdleWaitNS = &leftWaitNS
			frameSummary.Eyes[0].IdleWindows = leftWait
			frameSummary.Eyes[1].IdleWaitNS = &rightWaitNS
			frameSummary.Eyes[1].IdleWindows = rightWait
			idleValues[EyeLeft] = append(idleValues[EyeLeft], leftWaitNS)
			idleValues[EyeRight] = append(idleValues[EyeRight], rightWaitNS)
			reportSingleEyeWait(report, frame.ID, EyeLeft, leftWaitNS, rightWaitNS, leftWait, config)
			reportSingleEyeWait(report, frame.ID, EyeRight, rightWaitNS, leftWaitNS, rightWait, config)
			aggregateOverlaps(left, right, overlaps, crossOverlaps)
		} else {
			report.Summary.UnpairedFrames++
		}
		checkFrameTimestampOrder(report, frame.ID, left, lastFrameStart, lastFrameID)
		checkFrameTimestampOrder(report, frame.ID, right, lastFrameStart, lastFrameID)
		report.Frames = append(report.Frames, frameSummary)
	}

	report.StageStats = buildStageStats(values, config, &report.LongTail)
	report.Summary.LongTailStages = len(report.LongTail)
	report.LongTailFrames = groupLongTailFrames(report.LongTail)
	report.Summary.LongTailFrames = len(report.LongTailFrames)
	for _, stage := range Stages {
		overlap := overlaps[stage]
		union := overlap.LeftNS + overlap.RightNS - overlap.OverlapNS
		if union > 0 {
			overlap.OverlapRatio = float64(overlap.OverlapNS) / float64(union)
		}
		report.StageOverlaps = append(report.StageOverlaps, *overlap)
	}
	report.CrossStageOverlaps = flattenCrossOverlaps(crossOverlaps)
	report.IdleWaits = summarizeIdle(idleValues, 95)
	reportPersistentWaits(report)
	report.Diagnostics = sortDiagnostics(report.Diagnostics)
	recountDiagnostics(report)
	return report
}

func reportPersistentWaits(report *Report) {
	if report.Summary.PairedFrames < 2 || len(report.IdleWaits) < 2 {
		return
	}
	for _, idle := range report.IdleWaits {
		if idle.FramesWithWait < report.Summary.PairedFrames {
			continue
		}
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code:    "persistent_single_eye_wait",
			Message: fmt.Sprintf("%s eye had idle wait in all %d paired frames", idle.Eye, report.Summary.PairedFrames),
			Eye:     idle.Eye,
			Raw:     fmt.Sprintf("wait_ns=%d frames=%d", idle.TotalWaitNS, idle.FramesWithWait),
		})
	}
}

func frameSpan(left, right eyeCalc) int64 {
	hasSpan := false
	start, end := int64(0), int64(0)
	for _, eye := range []eyeCalc{left, right} {
		if !eye.hasSpan {
			continue
		}
		if !hasSpan || eye.spanStart < start {
			start = eye.spanStart
		}
		if !hasSpan || eye.spanEnd > end {
			end = eye.spanEnd
		}
		hasSpan = true
	}
	if !hasSpan || end < start {
		return 0
	}
	return end - start
}

func normalizeConfig(config Config) Config {
	if config.TailPercentile <= 0 || config.TailPercentile > 100 {
		config.TailPercentile = 95
	}
	if config.HistogramBuckets <= 0 {
		config.HistogramBuckets = 12
	}
	if config.HistogramBuckets > 64 {
		config.HistogramBuckets = 64
	}
	if config.MinIdleWait < 0 {
		config.MinIdleWait = 0
	}
	return config
}

func calculateEye(report *Report, frame *Frame, eye string, values map[stageKey][]stageValue) eyeCalc {
	result := eyeCalc{
		summary: EyeFrameSummary{
			Eye:           eye,
			MissingStages: make([]string, 0),
		},
		stageIntervals: make(map[string]Interval),
	}
	timing, present := frame.Eyes[eye]
	result.summary.Present = present && timing != nil
	if !present || timing == nil {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code:    "missing_eye",
			Message: "eye timing is missing for this frame",
			Frame:   frame.ID,
			Eye:     eye,
		})
	}

	allIntervals := make([]Interval, 0)
	for _, stage := range Stages {
		var stageTiming *StageTiming
		if timing != nil {
			stageTiming = timing.Stages[stage]
		}
		if stageTiming == nil {
			result.summary.MissingStages = append(result.summary.MissingStages, stage)
			report.Diagnostics = append(report.Diagnostics, Diagnostic{
				Code:    "missing_stage",
				Message: "expected stage is absent for this eye",
				Frame:   frame.ID,
				Eye:     eye,
				Stage:   stage,
			})
			continue
		}
		if stageTiming.DurationNS != nil && *stageTiming.DurationNS >= 0 {
			result.summary.StageDurationNS += *stageTiming.DurationNS
			result.summary.CompleteStages++
			values[stageKey{Eye: eye, Stage: stage}] = append(
				values[stageKey{Eye: eye, Stage: stage}],
				stageValue{FrameID: frame.ID, Eye: eye, Stage: stage, NS: *stageTiming.DurationNS},
			)
		}
		result.summary.Stages = append(result.summary.Stages, StageDetail{
			Stage:          stage,
			Start:          stageTiming.Start,
			End:            stageTiming.End,
			DurationNS:     stageTiming.DurationNS,
			DurationSource: stageTiming.DurationSource,
		})
		if stageTiming.Start == nil || stageTiming.End == nil || stageTiming.End.NS < stageTiming.Start.NS {
			continue
		}
		interval := Interval{StartNS: stageTiming.Start.NS, EndNS: stageTiming.End.NS}
		if interval.EndNS < interval.StartNS {
			continue
		}
		result.stageIntervals[stage] = interval
		allIntervals = append(allIntervals, interval)
		if !result.hasSpan || interval.StartNS < result.spanStart {
			result.spanStart = interval.StartNS
		}
		if !result.hasSpan || interval.EndNS > result.spanEnd {
			result.spanEnd = interval.EndNS
		}
		result.hasSpan = true
	}
	result.coverage = mergeIntervals(allIntervals)
	if result.hasSpan {
		result.summary.ActiveSpanNS = result.spanEnd - result.spanStart
	}
	return result
}

func reportSingleEyeWait(report *Report, frameID, eye string, waitNS, otherWaitNS int64, windows []Interval, config Config) {
	if waitNS <= int64(config.MinIdleWait) || waitNS <= 0 {
		return
	}
	other := EyeRight
	if eye == EyeRight {
		other = EyeLeft
	}
	report.Diagnostics = append(report.Diagnostics, Diagnostic{
		Code:    "single_eye_idle_wait",
		Message: fmt.Sprintf("%s eye is idle while %s eye is active (%s vs %s)", eye, other, formatNS(waitNS), formatNS(otherWaitNS)),
		Frame:   frameID,
		Eye:     eye,
		Raw:     fmt.Sprintf("wait_ns=%d windows=%d", waitNS, len(windows)),
	})
}

func checkFrameTimestampOrder(report *Report, frameID string, eye eyeCalc, lastStart map[string]int64, lastFrame map[string]string) {
	if len(eye.coverage) == 0 {
		return
	}
	start := eye.coverage[0].StartNS
	if previous, ok := lastStart[eye.summary.Eye]; ok && start < previous {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code:    "timestamp_regression",
			Message: fmt.Sprintf("frame start regressed versus frame %s", lastFrame[eye.summary.Eye]),
			Frame:   frameID,
			Eye:     eye.summary.Eye,
			Field:   "frame_start",
			Raw:     fmt.Sprintf("%d < %d", start, previous),
		})
	}
	lastStart[eye.summary.Eye] = start
	lastFrame[eye.summary.Eye] = frameID
}

func aggregateOverlaps(left, right eyeCalc, overlaps map[string]*StageOverlap, cross map[[2]string]*CrossStageOverlap) {
	for _, leftStage := range Stages {
		leftInterval, leftOK := left.stageIntervals[leftStage]
		if !leftOK {
			continue
		}
		leftLength := leftInterval.EndNS - leftInterval.StartNS
		for _, rightStage := range Stages {
			rightInterval, rightOK := right.stageIntervals[rightStage]
			if !rightOK {
				continue
			}
			rightLength := rightInterval.EndNS - rightInterval.StartNS
			overlapNS := intersectLength([]Interval{leftInterval}, []Interval{rightInterval})
			if leftStage == rightStage {
				stageOverlap := overlaps[leftStage]
				stageOverlap.FramesCompared++
				stageOverlap.LeftNS += leftLength
				stageOverlap.RightNS += rightLength
				stageOverlap.OverlapNS += overlapNS
				stageOverlap.LeftOnlyNS += leftLength - overlapNS
				stageOverlap.RightOnlyNS += rightLength - overlapNS
				if overlapNS > 0 {
					stageOverlap.FramesOverlap++
				}
				continue
			}
			if overlapNS <= 0 {
				continue
			}
			key := [2]string{leftStage, rightStage}
			entry, ok := cross[key]
			if !ok {
				entry = &CrossStageOverlap{LeftStage: leftStage, RightStage: rightStage}
				cross[key] = entry
			}
			entry.OverlapNS += overlapNS
			entry.FrameCount++
		}
	}
}
