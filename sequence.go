package eyebw

import "fmt"

type Scenario struct {
	Name                   string      `json:"name"`
	Description            string      `json:"description"`
	Config                 FrameConfig `json:"config"`
	Frames                 int         `json:"frames"`
	DynamicGaze            bool        `json:"dynamic_gaze"`
	ExpectQualityFailure   bool        `json:"expect_quality_failure"`
	ExpectedFailedPipeline PipelineID  `json:"expected_failed_pipeline,omitempty"`
}

type PipelineSequenceSummary struct {
	ID                     PipelineID   `json:"id"`
	Label                  string       `json:"label_zh"`
	FrameCount             int          `json:"frame_count"`
	Total                  Totals       `json:"total"`
	Average                Totals       `json:"average"`
	PeakFrameTransferBytes uint64       `json:"peak_frame_transfer_bytes"`
	BudgetExceededFrames   int          `json:"budget_exceeded_frames"`
	QualityFailedFrames    int          `json:"quality_failed_frames"`
	AllQualityPassed       bool         `json:"all_quality_passed"`
	Hotspots               []Hotspot    `json:"hotspots"`
	Classes                []NamedTotal `json:"classes"`
	BudgetExceeded         bool         `json:"budget_exceeded"`
	OverageBytes           uint64       `json:"overage_bytes"`
}

type ScenarioReport struct {
	Scenario
	FrameReports  []*FrameReport            `json:"frame_reports"`
	Pipelines     []PipelineSequenceSummary `json:"pipelines"`
	Success       bool                      `json:"success"`
	FailureReason string                    `json:"failure_reason,omitempty"`
}

func RunScenario(scenario Scenario) (*ScenarioReport, error) {
	if scenario.Frames <= 0 {
		scenario.Frames = 12
	}
	sim, err := NewSimulator(scenario.Config)
	if err != nil {
		return nil, fmt.Errorf("scenario %s: %w", scenario.Name, err)
	}
	out := &ScenarioReport{Scenario: scenario, Success: true}
	for i := 0; i < scenario.Frames; i++ {
		gazes := [2]Gaze{}
		if scenario.DynamicGaze {
			gazes = SweepGaze(i, scenario.Frames)
		}
		frame, err := sim.Step(FrameInput{Index: i, Gazes: gazes})
		if err != nil {
			return nil, err
		}
		out.FrameReports = append(out.FrameReports, frame)
	}
	out.Pipelines = summarizeSequence(scenario, out.FrameReports)
	out.Success, out.FailureReason = evaluateScenarioExpectations(scenario, out.Pipelines)
	return out, nil
}

func summarizeSequence(scenario Scenario, frames []*FrameReport) []PipelineSequenceSummary {
	out := make([]PipelineSequenceSummary, 0, len(AllPipelines()))
	for _, id := range AllPipelines() {
		summary := PipelineSequenceSummary{
			ID:               id,
			Label:            id.LabelZH(),
			FrameCount:       len(frames),
			AllQualityPassed: true,
		}
		classTotals := make(map[DataClass]*Totals)
		stageTotals := make(map[StageID]*Totals)
		for _, frame := range frames {
			report := frame.Get(id)
			if report == nil {
				continue
			}
			summary.Total = summary.Total.add(report.Totals)
			frameTransfer := report.TransferTotal()
			if frameTransfer > summary.PeakFrameTransferBytes {
				summary.PeakFrameTransferBytes = frameTransfer
			}
			if report.Budget.Exceeded {
				summary.BudgetExceededFrames++
				summary.BudgetExceeded = true
				summary.OverageBytes += report.Budget.OverageBytes
			}
			if !report.Quality.Passed {
				summary.QualityFailedFrames++
				summary.AllQualityPassed = false
			}
			for _, class := range report.Classes {
				id := DataClass(class.Name)
				if classTotals[id] == nil {
					classTotals[id] = &Totals{}
				}
				*classTotals[id] = classTotals[id].add(class.Totals)
			}
			for _, stage := range report.Stages {
				id := StageID(stage.Name)
				if stageTotals[id] == nil {
					stageTotals[id] = &Totals{}
				}
				*stageTotals[id] = stageTotals[id].add(stage.Totals)
			}
		}
		if summary.FrameCount > 0 {
			summary.Average = divideTotals(summary.Total, uint64(summary.FrameCount))
		}
		for _, class := range AllClasses() {
			totals := classTotals[class]
			value := Totals{}
			if totals != nil {
				value = *totals
			}
			summary.Classes = append(summary.Classes, NamedTotal{Name: string(class), Label: class.LabelZH(), Totals: value})
		}
		stageNames := make([]NamedTotal, 0, len(stageTotals))
		for id, totals := range stageTotals {
			stageNames = append(stageNames, NamedTotal{Name: string(id), Label: stageLabelZH(id), Totals: *totals})
		}
		summary.Hotspots = BuildHotspots(stageNames)
		out = append(out, summary)
	}
	return out
}

func divideTotals(t Totals, divisor uint64) Totals {
	if divisor == 0 {
		return t
	}
	hitRatio := t.CacheHitRatioBytes
	t.LogicalReadBytes /= divisor
	t.LogicalWriteBytes /= divisor
	t.TransferReadBytes /= divisor
	t.TransferWriteBytes /= divisor
	t.CacheHitBytes /= divisor
	t.CacheMissBytes /= divisor
	t.NonCacheableBytes /= divisor
	t.CacheableReadBytes /= divisor
	t.CacheHitRatioBytes = hitRatio
	return t
}

func evaluateScenarioExpectations(scenario Scenario, summaries []PipelineSequenceSummary) (bool, string) {
	byID := make(map[PipelineID]PipelineSequenceSummary, len(summaries))
	for _, summary := range summaries {
		byID[summary.ID] = summary
	}
	if scenario.ExpectQualityFailure {
		expected := scenario.ExpectedFailedPipeline
		if expected == "" {
			expected = PipelineSingleReuse
		}
		summary, ok := byID[expected]
		if !ok || summary.QualityFailedFrames == 0 {
			return false, fmt.Sprintf("expected quality failure for %s was not recorded", expected)
		}
		for id, candidate := range byID {
			if id != expected && !candidate.AllQualityPassed {
				return false, fmt.Sprintf("unexpected quality failure for %s", id)
			}
		}
		return true, ""
	}
	for _, summary := range summaries {
		if !summary.AllQualityPassed {
			return false, fmt.Sprintf("pipeline %s failed center quality gate", summary.ID)
		}
	}
	return true, ""
}
