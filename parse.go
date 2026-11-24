package eyestage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

type parser struct {
	ds       *Dataset
	frameSeq int
}

func Parse(r io.Reader) (*Dataset, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("input is empty")
	}

	p := &parser{
		ds: &Dataset{
			Unit:        "ns",
			UnitSource:  "default:ns",
			Diagnostics: make([]Diagnostic, 0),
		},
	}
	root, err := decodeAny(data)
	if err != nil {
		if lineErr := p.parseJSONLines(data); lineErr != nil {
			return nil, fmt.Errorf("decode JSON: %w", err)
		}
	} else {
		p.parseNode(root, "")
	}

	if len(p.ds.Frames) == 0 {
		return nil, errors.New("no frames found")
	}
	return p.ds, nil
}

func (p *parser) parseJSONLines(data []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	lineNo := 0
	found := false
	for scanner.Scan() {
		lineNo++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		value, err := decodeAny(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNo, err)
		}
		p.parseNode(value, "")
		found = true
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if !found {
		return errors.New("no JSON values")
	}
	return nil
}

func decodeAny(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("multiple JSON values; use JSONL input")
		}
		return nil, err
	}
	return value, nil
}

func (p *parser) parseNode(value any, inheritedUnit string) {
	switch node := value.(type) {
	case []any:
		for _, item := range node {
			p.parseNode(item, inheritedUnit)
		}
	case map[string]any:
		p.parseObject(node, inheritedUnit)
	default:
		p.addDiagnostic(Diagnostic{
			Code:    "unsupported_record",
			Message: "record must be a JSON object or array",
		})
	}
}

func (p *parser) parseObject(obj map[string]any, inheritedUnit string) {
	unit := inheritedUnit
	if raw, ok := lookup(obj, "unit", "time_unit"); ok {
		unit = p.consumeUnit(raw)
	}

	if rawFrames, ok := lookup(obj, "frames", "samples", "records"); ok {
		p.parseFramesContainer(rawFrames, unit)
		return
	}

	if _, ok := lookup(obj, "eye", "eye_side"); ok {
		p.parseFlatRecord(obj, unit)
		return
	}
	p.parseFrameFromObject(obj, unit)
}

func (p *parser) parseFramesContainer(raw any, unit string) {
	switch value := raw.(type) {
	case []any:
		for _, item := range value {
			if object, ok := item.(map[string]any); ok {
				if _, flat := lookup(object, "eye", "eye_side"); flat {
					p.parseFlatRecord(object, unit)
				} else {
					p.parseFrameObjectWithID(object, unit, "")
				}
			} else {
				p.parseNode(item, unit)
			}
		}
	case map[string]any:
		if hasEyeFields(value) {
			p.parseFrameObjectWithID(value, unit, "")
			return
		}
		ids := make([]string, 0, len(value))
		for id := range value {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			item := value[id]
			object, ok := item.(map[string]any)
			if !ok {
				p.addDiagnostic(Diagnostic{
					Code:    "unsupported_record",
					Message: "frame entry must be an object",
					Frame:   id,
				})
				continue
			}
			if _, flat := lookup(object, "eye", "eye_side"); flat {
				p.parseFlatRecordWithID(object, unit, id)
			} else {
				p.parseFrameObjectWithID(object, unit, id)
			}
		}
	default:
		p.addDiagnostic(Diagnostic{
			Code:    "unsupported_record",
			Message: "frames must be an array or object",
		})
	}
}

func (p *parser) consumeUnit(raw any) string {
	text, ok := scalarText(raw)
	if !ok {
		p.addDiagnostic(Diagnostic{
			Code:           "unsupported_unit",
			Message:        "time unit must be a string; using ns fallback",
			Field:          "unit",
			Raw:            rawText(raw),
			FallbackSource: "fallback:default_ns",
		})
		p.ds.Unit = "ns"
		p.ds.UnitSource = "fallback:default_ns"
		return "ns"
	}
	normalized, ok := normalizeUnit(text)
	if !ok {
		p.addDiagnostic(Diagnostic{
			Code:           "unsupported_unit",
			Message:        fmt.Sprintf("unsupported time unit %q; using ns fallback", text),
			Field:          "unit",
			Raw:            text,
			FallbackSource: "fallback:unknown_unit_ns",
		})
		p.ds.Unit = "ns"
		p.ds.UnitSource = "fallback:unknown_unit_ns"
		return "ns"
	}
	p.ds.Unit = normalized
	p.ds.UnitSource = "explicit:" + normalized
	return normalized
}

func (p *parser) parseFrameFromObject(obj map[string]any, unit string) {
	frameID := ""
	if raw, ok := lookup(obj, "frame_id", "frame", "id", "frameId"); ok {
		if text, ok := scalarText(raw); ok {
			frameID = text
		}
	}
	p.parseFrameObjectWithID(obj, unit, frameID)
}

func (p *parser) parseFrameObjectWithID(obj map[string]any, unit, frameID string) {
	if frameID == "" {
		if raw, ok := lookup(obj, "frame_id", "frame", "id", "frameId"); ok {
			if text, ok := scalarText(raw); ok {
				frameID = text
			}
		}
	}
	if frameID == "" {
		p.frameSeq++
		frameID = fmt.Sprintf("frame-%d", p.frameSeq)
		p.addDiagnostic(Diagnostic{
			Code:    "missing_frame_id",
			Message: "frame id missing; assigned sequential id",
			Frame:   frameID,
		})
	}
	frame := p.ensureFrame(frameID)

	if rawEyes, ok := lookup(obj, "eyes"); ok {
		if eyes, ok := rawEyes.(map[string]any); ok {
			for key, value := range eyes {
				p.parseEyeValue(frame, key, value, unit)
			}
		} else {
			p.addDiagnostic(Diagnostic{
				Code:    "unsupported_record",
				Message: "eyes must be an object",
				Frame:   frameID,
			})
		}
	}

	for key, value := range obj {
		eye, ok := normalizeEye(key)
		if !ok {
			continue
		}
		p.parseEyeValue(frame, eye, value, unit)
	}

	if len(frame.Eyes) == 0 {
		for key := range obj {
			if _, ok := normalizeStage(key); ok {
				p.addDiagnostic(Diagnostic{
					Code:    "unassigned_stage",
					Message: "stage timings are not assigned to an eye",
					Frame:   frameID,
					Stage:   key,
				})
				break
			}
		}
	}
}

func (p *parser) parseFlatRecord(obj map[string]any, unit string) {
	p.parseFlatRecordWithID(obj, unit, "")
}

func (p *parser) parseFlatRecordWithID(obj map[string]any, unit, frameID string) {
	if raw, ok := lookup(obj, "frame_id", "frame", "id", "frameId"); ok {
		if text, ok := scalarText(raw); ok {
			frameID = text
		}
	}
	if frameID == "" {
		p.frameSeq++
		frameID = fmt.Sprintf("frame-%d", p.frameSeq)
		p.addDiagnostic(Diagnostic{
			Code:    "missing_frame_id",
			Message: "frame id missing; assigned sequential id",
			Frame:   frameID,
		})
	}
	eyeRaw, _ := lookup(obj, "eye", "eye_side")
	eyeText, _ := scalarText(eyeRaw)
	eye, ok := normalizeEye(eyeText)
	if !ok {
		p.addDiagnostic(Diagnostic{
			Code:    "unsupported_eye",
			Message: fmt.Sprintf("unsupported eye %q", eyeText),
			Frame:   frameID,
			Field:   "eye",
			Raw:     eyeText,
		})
		eye = strings.ToLower(strings.TrimSpace(eyeText))
	}
	frame := p.ensureFrame(frameID)

	if rawStage, ok := lookup(obj, "stage", "stage_name"); ok {
		if text, ok := scalarText(rawStage); ok {
			p.parseStageValue(frame, eye, text, obj, unit)
			return
		}
	}
	if rawStages, ok := lookup(obj, "stages", "timestamps"); ok {
		p.parseEyeValue(frame, eye, rawStages, unit)
		return
	}
	p.addDiagnostic(Diagnostic{
		Code:    "missing_stage",
		Message: "flat record has no stage field",
		Frame:   frameID,
		Eye:     eye,
	})
}

func (p *parser) parseEyeValue(frame *Frame, eye string, value any, unit string) {
	object, ok := value.(map[string]any)
	if !ok {
		p.addDiagnostic(Diagnostic{
			Code:    "unsupported_record",
			Message: "eye timing must be an object",
			Frame:   frame.ID,
			Eye:     eye,
		})
		return
	}
	if normalized, valid := normalizeEye(eye); valid {
		eye = normalized
	}
	if raw, ok := lookup(object, "eye", "eye_side"); ok {
		if text, ok := scalarText(raw); ok {
			if normalized, valid := normalizeEye(text); valid {
				eye = normalized
			}
		}
	}
	if eye == "" {
		p.addDiagnostic(Diagnostic{
			Code:    "unassigned_stage",
			Message: "stage timings are not assigned to an eye",
			Frame:   frame.ID,
		})
		return
	}
	for key, stageValue := range object {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "eye", "eye_side", "stages", "timestamps":
			continue
		}
		stage, ok := normalizeStage(key)
		if !ok {
			continue
		}
		p.parseStageValue(frame, eye, stage, stageValue, unit)
	}
	if raw, ok := lookup(object, "stages", "timestamps"); ok {
		if stages, ok := raw.(map[string]any); ok {
			for key, stageValue := range stages {
				stage, valid := normalizeStage(key)
				if valid {
					p.parseStageValue(frame, eye, stage, stageValue, unit)
				}
			}
		}
	}
}
