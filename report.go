package notify

import (
	"encoding/json"
	"io"
)

// Check is one machine-readable invariant result.
type Check struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

// CaseReport contains the requested versions and differences for one injection case.
type CaseReport struct {
	ID                   string          `json:"id"`
	Title                string          `json:"title"`
	Description          string          `json:"description"`
	NotificationID       string          `json:"notification_id"`
	Old                  State           `json:"old"`
	AnchorFrame          int             `json:"anchor_frame"`
	Committed            State           `json:"committed"`
	Final                State           `json:"final"`
	CommittedDiff        Diff            `json:"committed_diff"`
	FinalDiff            Diff            `json:"final_diff"`
	CommittedToFinalDiff Diff            `json:"committed_to_final_diff"`
	Frames               []*Frame        `json:"frames"`
	Events               []*InputRecord  `json:"events"`
	WorldReceipts        []*WorldReceipt `json:"world_receipts,omitempty"`
	Checks               []Check         `json:"checks"`
	Pass                 bool            `json:"pass"`
}

// Report is the complete deterministic simulation record.
type Report struct {
	NotificationID string        `json:"notification_id"`
	Cases          []*CaseReport `json:"cases"`
	Pass           bool          `json:"pass"`
}

func newCaseReport(id, title, description string, old State, p *Pipeline, anchorFrame int) *CaseReport {
	frames := p.Frames()
	report := &CaseReport{
		ID:             id,
		Title:          title,
		Description:    description,
		NotificationID: old.ID,
		Old:            old,
		AnchorFrame:    anchorFrame,
		Frames:         frames,
		Events:         p.Events(),
		WorldReceipts:  p.WorldReceipts(),
		Pass:           true,
	}

	for _, frame := range frames {
		if frame.Number == anchorFrame {
			report.Committed = frame.States[old.ID]
		}
	}
	if len(frames) > 0 {
		last := frames[len(frames)-1]
		report.Final = last.States[old.ID]
	}

	report.CommittedDiff = DiffStates(report.Old, report.Committed)
	report.FinalDiff = DiffStates(report.Old, report.Final)
	report.CommittedToFinalDiff = DiffStates(report.Committed, report.Final)
	return report
}

func (r *CaseReport) check(name string, pass bool, detail string) {
	r.Checks = append(r.Checks, Check{Name: name, Pass: pass, Detail: detail})
	if !pass {
		r.Pass = false
	}
}

// DefaultReport runs all requested deterministic injection scenarios.
func DefaultReport() *Report {
	cases := []*CaseReport{
		beforeDeadlineCase(),
		lateCase(),
		continuousCase(),
		closeCase(),
	}
	report := &Report{NotificationID: DefaultNotificationID, Cases: cases, Pass: true}
	for _, scenario := range cases {
		if !scenario.Pass {
			report.Pass = false
		}
	}
	return report
}

// WriteJSON writes an indented JSON report.
func (r *Report) WriteJSON(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}
