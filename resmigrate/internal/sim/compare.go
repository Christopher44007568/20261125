package sim

// BuildReport assembles the top-level report including comparison rows.
func BuildReport(scenarios []ScenarioReport) *Report {
	rows := make([]ComparisonRow, 0, len(scenarios))
	for _, s := range scenarios {
		rows = append(rows, ComparisonRow{
			ScenarioID:        s.ScenarioID,
			Title:             s.Title,
			Strategy:          s.Strategy,
			StrategyName:      s.StrategyName,
			Outcome:           s.Outcome,
			FinalSize:         s.FinalSize,
			PeakOverBaseline:  s.Memory.PeakOverBaseline,
			PeakMemoryBytes:   s.Memory.PeakBytes,
			HistoryPeakBytes:  s.Memory.HistoryPeakBytes,
			InvalidationWorst: s.Invalidation.WorstRatio,
			JumpEvents:        s.Jumps.EventCount,
			MaxJump:           s.Jumps.MaxStereoJump,
			DeadlineMisses:    s.Work.DeadlineMisses,
			RemapApplied:      s.RemapApplied,
			StereoConsistent:  s.FinalSize.equals(s.EyeFinal(0)) && s.FinalSize.equals(s.EyeFinal(1)),
			OK:                s.OK,
		})
	}
	return &Report{
		Schema:            "resmigrate-report/v1",
		JumpThresholdLuma: jumpThresholdLuma,
		Scenarios:         scenarios,
		Comparison:        rows,
	}
}

// EyeFinal returns the final size of eye 0 (left) or 1 (right).
func (s ScenarioReport) EyeFinal(i int) Size {
	return s.Eyes[i].FinalSize
}
