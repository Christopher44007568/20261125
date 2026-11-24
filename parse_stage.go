package eyestage

import "math"

func (p *parser) ensureFrame(id string) *Frame {
	for _, frame := range p.ds.Frames {
		if frame.ID == id {
			return frame
		}
	}
	frame := &Frame{ID: id, Eyes: make(map[string]*EyeTiming)}
	p.ds.Frames = append(p.ds.Frames, frame)
	return frame
}

func (p *parser) ensureEye(frame *Frame, eye string) *EyeTiming {
	if timing, ok := frame.Eyes[eye]; ok {
		return timing
	}
	timing := &EyeTiming{Eye: eye, Stages: make(map[string]*StageTiming)}
	frame.Eyes[eye] = timing
	return timing
}

func (p *parser) addDiagnostic(d Diagnostic) {
	p.ds.Diagnostics = append(p.ds.Diagnostics, d)
}

func (p *parser) parseStageValue(frame *Frame, eye, stageName string, value any, unit string) {
	stage, ok := normalizeStage(stageName)
	if !ok {
		p.addDiagnostic(Diagnostic{
			Code:    "unsupported_stage",
			Message: "unsupported stage name",
			Frame:   frame.ID,
			Eye:     eye,
			Stage:   stageName,
			Raw:     stageName,
		})
		return
	}
	if eye == "" {
		p.addDiagnostic(Diagnostic{
			Code:    "unassigned_stage",
			Message: "stage timings are not assigned to an eye",
			Frame:   frame.ID,
			Stage:   stage,
		})
		return
	}
	timing := p.ensureEye(frame, eye)
	if _, exists := timing.Stages[stage]; exists {
		p.addDiagnostic(Diagnostic{
			Code:    "duplicate_stage",
			Message: "duplicate stage record; later value replaces earlier value",
			Frame:   frame.ID,
			Eye:     eye,
			Stage:   stage,
		})
	}
	timing.Stages[stage] = p.parseStageBody(frame.ID, eye, stage, value, unit)
}

func (p *parser) parseStageBody(frameID, eye, stage string, value any, unit string) *StageTiming {
	result := &StageTiming{Stage: stage}

	if pair, ok := value.([]any); ok {
		if len(pair) != 2 {
			p.addDiagnostic(Diagnostic{
				Code:    "unsupported_record",
				Message: "stage interval array must contain start and end",
				Frame:   frameID,
				Eye:     eye,
				Stage:   stage,
			})
			return result
		}
		result.Start, _ = p.parseTimestamp(pair[0], "", unit, "[0]", frameID, eye, stage)
		result.End, _ = p.parseTimestamp(pair[1], "", unit, "[1]", frameID, eye, stage)
		return p.finishStage(result, frameID, eye, stage, nil, false)
	}

	object, ok := value.(map[string]any)
	if !ok {
		result.Start, _ = p.parseTimestamp(value, "", unit, "value", frameID, eye, stage)
		p.addDiagnostic(Diagnostic{
			Code:    "missing_timestamp",
			Message: "stage has a start value but no end timestamp",
			Frame:   frameID,
			Eye:     eye,
			Stage:   stage,
			Field:   "end",
		})
		return result
	}

	startField, startRaw, startUnit, hasStart := p.lookupTimestampField(object, startFields)
	endField, endRaw, endUnit, hasEnd := p.lookupTimestampField(object, endFields)
	durationField, durationRaw, durationUnit, hasDuration := p.lookupTimestampField(object, durationFields)

	if hasStart {
		result.Start, _ = p.parseTimestamp(startRaw, startUnit, unit, startField, frameID, eye, stage)
	}
	if hasEnd {
		result.End, _ = p.parseTimestamp(endRaw, endUnit, unit, endField, frameID, eye, stage)
	}
	var explicitDuration *int64
	if hasDuration {
		explicitDuration, _ = p.parseDuration(durationRaw, durationUnit, unit, durationField, frameID, eye, stage)
	}
	if !hasStart && !hasEnd && !hasDuration {
		p.addDiagnostic(Diagnostic{
			Code:    "missing_timestamp",
			Message: "stage contains no supported start, end, or duration field",
			Frame:   frameID,
			Eye:     eye,
			Stage:   stage,
		})
	}
	return p.finishStage(result, frameID, eye, stage, explicitDuration, hasDuration)
}

func (p *parser) finishStage(result *StageTiming, frameID, eye, stage string, explicitDuration *int64, hadDuration bool) *StageTiming {
	if result.Start == nil && result.End == nil {
		if explicitDuration != nil {
			result.DurationNS = explicitDuration
			result.DurationSource = "explicit"
			p.addDiagnostic(Diagnostic{
				Code:           "incomplete_timestamp",
				Message:        "duration is available but no absolute timestamp can support overlap analysis",
				Frame:          frameID,
				Eye:            eye,
				Stage:          stage,
				FallbackSource: "none",
			})
		}
		return result
	}

	if result.Start == nil && result.End != nil && explicitDuration != nil {
		if !subSafe(result.End.NS, *explicitDuration, &result.Start) {
			p.addDiagnostic(Diagnostic{
				Code:    "unsupported_timestamp",
				Message: "cannot derive start timestamp from end and duration",
				Frame:   frameID,
				Eye:     eye,
				Stage:   stage,
				Field:   "start",
				Raw:     rawText(explicitDuration),
			})
			return result
		}
		result.Start.Source = "fallback:duration"
		p.timestampFallback(frameID, eye, stage, "start", result.Start.Source)
	}
	if result.End == nil && result.Start != nil && explicitDuration != nil {
		if !addSafe(result.Start.NS, *explicitDuration, &result.End) {
			p.addDiagnostic(Diagnostic{
				Code:    "unsupported_timestamp",
				Message: "cannot derive end timestamp from start and duration",
				Frame:   frameID,
				Eye:     eye,
				Stage:   stage,
				Field:   "end",
				Raw:     rawText(explicitDuration),
			})
			return result
		}
		result.End.Source = "fallback:duration"
		p.timestampFallback(frameID, eye, stage, "end", result.End.Source)
	}

	if result.Start == nil || result.End == nil {
		if result.Start == nil {
			p.addDiagnostic(Diagnostic{
				Code:    "missing_timestamp",
				Message: "stage end exists but start is missing",
				Frame:   frameID,
				Eye:     eye,
				Stage:   stage,
				Field:   "start",
			})
		}
		if result.End == nil {
			p.addDiagnostic(Diagnostic{
				Code:    "missing_timestamp",
				Message: "stage start exists but end is missing",
				Frame:   frameID,
				Eye:     eye,
				Stage:   stage,
				Field:   "end",
			})
		}
		if explicitDuration != nil {
			result.DurationNS = explicitDuration
			result.DurationSource = "explicit"
		}
		return result
	}

	if result.End.NS < result.Start.NS {
		p.addDiagnostic(Diagnostic{
			Code:    "timestamp_regression",
			Message: "stage end timestamp is earlier than start timestamp",
			Frame:   frameID,
			Eye:     eye,
			Stage:   stage,
			Raw:     result.Start.Raw + " -> " + result.End.Raw,
		})
		return result
	}
	duration := result.End.NS - result.Start.NS
	result.DurationNS = &duration
	result.DurationSource = "interval"
	if hadDuration && explicitDuration != nil && *explicitDuration != duration {
		p.addDiagnostic(Diagnostic{
			Code:    "duration_mismatch",
			Message: "explicit duration differs from timestamp interval; interval is used",
			Frame:   frameID,
			Eye:     eye,
			Stage:   stage,
			Raw:     rawText(*explicitDuration),
		})
	}
	return result
}

func (p *parser) timestampFallback(frameID, eye, stage, field, source string) {
	p.addDiagnostic(Diagnostic{
		Code:           "timestamp_fallback",
		Message:        "timestamp missing or unsupported; fallback source used",
		Frame:          frameID,
		Eye:            eye,
		Stage:          stage,
		Field:          field,
		FallbackSource: source,
	})
}

func addSafe(left, right int64, target **Timestamp) bool {
	if right > 0 && left > math.MaxInt64-right {
		return false
	}
	if right < 0 && left < math.MinInt64-right {
		return false
	}
	*target = &Timestamp{NS: left + right}
	return true
}

func subSafe(left, right int64, target **Timestamp) bool {
	if right > 0 && left < math.MinInt64+right {
		return false
	}
	if right < 0 && left > math.MaxInt64+right {
		return false
	}
	*target = &Timestamp{NS: left - right}
	return true
}
