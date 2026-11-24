package eyebw

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Report struct {
	Scenarios []*ScenarioReport `json:"scenarios"`
	Success   bool              `json:"success"`
}

func RunScenarios(scenarios []Scenario) (*Report, error) {
	report := &Report{Success: true}
	for _, scenario := range scenarios {
		result, err := RunScenario(scenario)
		if err != nil {
			return nil, err
		}
		report.Scenarios = append(report.Scenarios, result)
		if !result.Success {
			report.Success = false
		}
	}
	return report, nil
}

func DefaultReport() (*Report, error) {
	return RunScenarios(DefaultScenarios())
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

func WriteReport(dir string, report *Report, writeEvents bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := WriteJSON(filepath.Join(dir, "report.json"), report); err != nil {
		return err
	}
	if err := WriteFramesCSV(filepath.Join(dir, "frames.csv"), report); err != nil {
		return err
	}
	if err := WriteStagesCSV(filepath.Join(dir, "stages.csv"), report); err != nil {
		return err
	}
	if err := WriteClassesCSV(filepath.Join(dir, "classes.csv"), report); err != nil {
		return err
	}
	if writeEvents {
		if err := WriteEventsCSV(filepath.Join(dir, "events.csv"), report); err != nil {
			return err
		}
	}
	return WriteMarkdown(filepath.Join(dir, "summary.md"), report)
}
