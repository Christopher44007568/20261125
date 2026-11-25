package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// GenerateAll writes degrade_report.json/.md, trace.jsonl and shot PNGs.
func GenerateAll(outDir string, cfg Config) (*Report, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	report := BuildReport(cfg)

	jsonRaw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "degrade_report.json"),
		append(jsonRaw, '\n'), 0o644); err != nil {
		return nil, err
	}

	md := RenderMarkdown(report)
	if err := os.WriteFile(filepath.Join(outDir, "degrade_report.md"),
		[]byte(md), 0o644); err != nil {
		return nil, err
	}

	tf, err := os.Create(filepath.Join(outDir, "trace.jsonl"))
	if err != nil {
		return nil, err
	}
	if err := WriteTrace(tf, report); err != nil {
		tf.Close()
		return nil, err
	}
	tf.Close()

	for _, sr := range report.Scenarios {
		for _, shot := range sr.Shots {
			pngRaw, err := RenderShotPNG(sr, shot.Frame)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(outDir, shot.File), pngRaw, 0o644); err != nil {
				return nil, err
			}
		}
	}
	return report, nil
}
