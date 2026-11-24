package xrpath

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type FaultArtifact struct {
	OK              bool             `json:"ok"`
	FaultSuite      DrillSuite       `json:"fault_suite"`
	HashBlindDrills []HashBlindDrill `json:"hash_blind_drills"`
}

func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func WriteMarkdown(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
