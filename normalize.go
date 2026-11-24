package eyestage

import (
	"encoding/json"
	"fmt"
	"strings"
)

func normalizeEye(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "left", "l", "left_eye", "lefteye", "os", "左", "左眼":
		return EyeLeft, true
	case "right", "r", "right_eye", "righteye", "od", "右", "右眼":
		return EyeRight, true
	default:
		return "", false
	}
}

func normalizeStage(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "capture_scene", "scene_capture", "capture", "scene", "采集场景":
		return StageCaptureScene, true
	case "reprojection", "reproject", "reprojected", "重投影":
		return StageReprojection, true
	case "distortion", "distort", "undistortion", "undistort", "畸变":
		return StageDistortion, true
	case "composition", "composite", "compose", "synthesis", "合成":
		return StageComposition, true
	default:
		return "", false
	}
}

func normalizeUnit(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ns", "nsec", "nsecs", "nanosecond", "nanoseconds":
		return "ns", true
	case "us", "µs", "μs", "usec", "usecs", "microsecond", "microseconds":
		return "us", true
	case "ms", "msec", "msecs", "millisecond", "milliseconds":
		return "ms", true
	case "s", "sec", "secs", "second", "seconds":
		return "s", true
	default:
		return "", false
	}
}

func lookup(object map[string]any, names ...string) (any, bool) {
	for _, name := range names {
		if value, ok := object[name]; ok {
			return value, true
		}
	}
	for key, value := range object {
		lowerKey := strings.ToLower(key)
		for _, name := range names {
			if lowerKey == strings.ToLower(name) {
				return value, true
			}
		}
	}
	return nil, false
}

func lookupField(object map[string]any, names ...string) (string, any, bool) {
	for _, name := range names {
		if value, ok := object[name]; ok {
			return name, value, true
		}
	}
	for key, value := range object {
		lowerKey := strings.ToLower(key)
		for _, name := range names {
			if lowerKey == strings.ToLower(name) {
				return key, value, true
			}
		}
	}
	return "", nil, false
}

func scalarText(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case json.Number:
		return typed.String(), true
	case float64:
		return fmt.Sprintf("%v", typed), true
	case float32:
		return fmt.Sprintf("%v", typed), true
	case int:
		return fmt.Sprintf("%d", typed), true
	case int64:
		return fmt.Sprintf("%d", typed), true
	case int32:
		return fmt.Sprintf("%d", typed), true
	case uint64:
		return fmt.Sprintf("%d", typed), true
	case bool:
		return fmt.Sprintf("%t", typed), true
	default:
		return "", false
	}
}

func rawText(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	if len(data) > 160 {
		return string(data[:157]) + "..."
	}
	return string(data)
}

func hasEyeFields(object map[string]any) bool {
	for key := range object {
		if _, ok := normalizeEye(key); ok {
			return true
		}
	}
	_, ok := lookup(object, "eyes")
	return ok
}
