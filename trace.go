package main

import (
	"encoding/json"
	"io"
)

// WriteTrace emits one JSON object per recorded frame across all scenarios.
func WriteTrace(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	for _, sr := range report.Scenarios {
		for _, f := range sr.Frames {
			if err := enc.Encode(f); err != nil {
				return err
			}
		}
	}
	return nil
}
