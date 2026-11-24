package eyestage

import (
	"math"
	"strconv"
	"strings"
	"time"
)

type timestampField struct {
	name string
	unit string
}

var startFields = []timestampField{
	{"start_ns", "ns"}, {"begin_ns", "ns"}, {"start_nanos", "ns"}, {"start_nanoseconds", "ns"},
	{"start_us", "us"}, {"begin_us", "us"}, {"start_micros", "us"}, {"start_microseconds", "us"},
	{"start_ms", "ms"}, {"begin_ms", "ms"}, {"start_millis", "ms"}, {"start_milliseconds", "ms"},
	{"start_s", "s"}, {"begin_s", "s"}, {"start_sec", "s"}, {"start_seconds", "s"},
	{"started_at", ""}, {"start_at", ""}, {"start_time", ""}, {"start_ts", ""},
	{"start", ""}, {"begin", ""}, {"from", ""},
}

var endFields = []timestampField{
	{"end_ns", "ns"}, {"finish_ns", "ns"}, {"end_nanos", "ns"}, {"end_nanoseconds", "ns"},
	{"end_us", "us"}, {"finish_us", "us"}, {"end_micros", "us"}, {"end_microseconds", "us"},
	{"end_ms", "ms"}, {"finish_ms", "ms"}, {"end_millis", "ms"}, {"end_milliseconds", "ms"},
	{"end_s", "s"}, {"finish_s", "s"}, {"end_sec", "s"}, {"end_seconds", "s"},
	{"ended_at", ""}, {"finished_at", ""}, {"end_at", ""}, {"end_time", ""}, {"end_ts", ""},
	{"end", ""}, {"finish", ""}, {"to", ""},
}

var durationFields = []timestampField{
	{"duration_ns", "ns"}, {"elapsed_ns", "ns"},
	{"duration_us", "us"}, {"elapsed_us", "us"},
	{"duration_ms", "ms"}, {"elapsed_ms", "ms"},
	{"duration_s", "s"}, {"elapsed_s", "s"},
	{"duration", ""}, {"elapsed", ""},
}

func (p *parser) lookupTimestampField(object map[string]any, fields []timestampField) (string, any, string, bool) {
	for _, field := range fields {
		if key, value, ok := lookupField(object, field.name); ok {
			return key, value, field.unit, true
		}
	}
	return "", nil, "", false
}

func (p *parser) parseTimestamp(value any, explicitUnit, unit, field, frameID, eye, stage string) (*Timestamp, bool) {
	if value == nil {
		return nil, false
	}
	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			p.unsupportedTimestamp(field, value, frameID, eye, stage, "empty timestamp")
			return nil, false
		}
		if parsed, source, ok := parseRFC3339(text); ok {
			ts := &Timestamp{NS: parsed, Source: source, Raw: text}
			p.noteFallback(ts, field, frameID, eye, stage)
			return ts, true
		}
		if numeric, ok := parseNumericTimestamp(text, explicitUnit, unit, p.ds.UnitSource); ok {
			ts := &Timestamp{NS: numeric.NS, Source: numeric.Source, Raw: text}
			p.noteFallback(ts, field, frameID, eye, stage)
			return ts, true
		}
		p.unsupportedTimestamp(field, value, frameID, eye, stage, "unsupported timestamp string")
		return nil, false
	}

	text, ok := scalarText(value)
	if !ok {
		p.unsupportedTimestamp(field, value, frameID, eye, stage, "timestamp must be a number or string")
		return nil, false
	}
	numeric, ok := parseNumericTimestamp(text, explicitUnit, unit, p.ds.UnitSource)
	if !ok {
		p.unsupportedTimestamp(field, value, frameID, eye, stage, "unsupported numeric timestamp")
		return nil, false
	}
	ts := &Timestamp{NS: numeric.NS, Source: numeric.Source, Raw: text}
	p.noteFallback(ts, field, frameID, eye, stage)
	return ts, true
}

type numericTime struct {
	NS     int64
	Source string
}

func parseNumericTimestamp(text, explicitUnit, unit, unitSource string) (numericTime, bool) {
	selected := explicitUnit
	source := ""
	if selected != "" {
		source = "explicit:" + selected
	} else {
		selected = unit
		if selected == "" {
			selected = "ns"
			source = "fallback:default_ns"
		} else if strings.HasPrefix(unitSource, "fallback:") {
			source = unitSource
		} else {
			source = "explicit:" + selected
		}
	}
	ns, integral, ok := convertToNS(text, selected)
	if !ok {
		return numericTime{}, false
	}
	if !integral {
		source = "explicit:" + selected + ":float"
	}
	return numericTime{NS: ns, Source: source}, true
}

func convertToNS(text, unit string) (int64, bool, bool) {
	scale, ok := unitScale(unit)
	if !ok {
		return 0, false, false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false, false
	}
	if !strings.ContainsAny(text, ".eE") {
		value, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return 0, false, false
		}
		if value > math.MaxInt64/scale || value < math.MinInt64/scale {
			return 0, false, false
		}
		return value * scale, true, true
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false, false
	}
	scaled := value * float64(scale)
	limit := math.Ldexp(1, 63)
	if scaled >= limit || scaled < -limit {
		return 0, false, false
	}
	return int64(math.Round(scaled)), false, true
}

func unitScale(unit string) (int64, bool) {
	switch unit {
	case "ns":
		return 1, true
	case "us":
		return 1_000, true
	case "ms":
		return 1_000_000, true
	case "s":
		return 1_000_000_000, true
	default:
		return 0, false
	}
}

func parseRFC3339(text string) (int64, string, bool) {
	zoned := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999Z07:00",
	}
	for _, layout := range zoned {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed.UnixNano(), "explicit:rfc3339", true
		}
	}
	withoutZone := []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05.999999999",
	}
	for _, layout := range withoutZone {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed.UnixNano(), "fallback:no_timezone_utc", true
		}
	}
	return 0, "", false
}

func (p *parser) parseDuration(value any, explicitUnit, unit, field, frameID, eye, stage string) (*int64, bool) {
	text, ok := scalarText(value)
	if !ok {
		p.addDiagnostic(Diagnostic{
			Code:    "unsupported_timestamp",
			Message: "duration must be a number",
			Frame:   frameID,
			Eye:     eye,
			Stage:   stage,
			Field:   field,
			Raw:     rawText(value),
		})
		return nil, false
	}
	selected := explicitUnit
	source := ""
	if selected == "" {
		selected = unit
		if selected == "" {
			selected = "ns"
			source = "fallback:default_ns"
		} else if strings.HasPrefix(p.ds.UnitSource, "fallback:") {
			source = p.ds.UnitSource
		} else {
			source = "explicit:" + selected
		}
	} else {
		source = "explicit:" + selected
	}
	ns, _, valid := convertToNS(text, selected)
	if !valid || ns < 0 {
		p.addDiagnostic(Diagnostic{
			Code:    "invalid_duration",
			Message: "duration is negative or not representable",
			Frame:   frameID,
			Eye:     eye,
			Stage:   stage,
			Field:   field,
			Raw:     text,
		})
		return nil, false
	}
	if strings.HasPrefix(source, "fallback:") {
		p.addDiagnostic(Diagnostic{
			Code:           "timestamp_fallback",
			Message:        "duration unit missing; fallback source used",
			Frame:          frameID,
			Eye:            eye,
			Stage:          stage,
			Field:          field,
			Raw:            text,
			FallbackSource: source,
		})
	}
	return &ns, true
}

func (p *parser) noteFallback(ts *Timestamp, field, frameID, eye, stage string) {
	if ts == nil || !strings.HasPrefix(ts.Source, "fallback:") {
		return
	}
	p.addDiagnostic(Diagnostic{
		Code:           "timestamp_fallback",
		Message:        "timestamp unit missing or unsupported; fallback source used",
		Frame:          frameID,
		Eye:            eye,
		Stage:          stage,
		Field:          field,
		Raw:            ts.Raw,
		FallbackSource: ts.Source,
	})
}

func (p *parser) unsupportedTimestamp(field string, value any, frameID, eye, stage, reason string) {
	p.addDiagnostic(Diagnostic{
		Code:           "unsupported_timestamp",
		Message:        reason,
		Frame:          frameID,
		Eye:            eye,
		Stage:          stage,
		Field:          field,
		Raw:            rawText(value),
		FallbackSource: "none",
	})
}
