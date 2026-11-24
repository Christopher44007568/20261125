package eyebw

type StageBudgetReport struct {
	Stage        StageID `json:"stage"`
	BudgetBytes  uint64  `json:"budget_bytes"`
	ActualBytes  uint64  `json:"actual_bytes"`
	OverageBytes uint64  `json:"overage_bytes"`
	Exceeded     bool    `json:"exceeded"`
	Utilization  float64 `json:"utilization"`
}

type BudgetReport struct {
	BudgetBytes    uint64              `json:"budget_bytes"`
	ActualBytes    uint64              `json:"actual_bytes"`
	RemainingBytes uint64              `json:"remaining_bytes"`
	OverageBytes   uint64              `json:"overage_bytes"`
	Exceeded       bool                `json:"exceeded"`
	Utilization    float64             `json:"utilization"`
	StageOverrides []StageBudgetReport `json:"stage_overrides,omitempty"`
	ExceededStages []StageBudgetReport `json:"exceeded_stages,omitempty"`
}

func CheckBudget(budget Budget, actual uint64, stages []NamedTotal) BudgetReport {
	out := BudgetReport{BudgetBytes: budget.FrameBytes, ActualBytes: actual}
	if budget.FrameBytes > 0 {
		out.Utilization = float64(actual) / float64(budget.FrameBytes)
		if actual > budget.FrameBytes {
			out.Exceeded = true
			out.OverageBytes = actual - budget.FrameBytes
		} else {
			out.RemainingBytes = budget.FrameBytes - actual
		}
	}
	for _, stage := range stages {
		limit, ok := budget.StageBytes[StageID(stage.Name)]
		if !ok || limit == 0 {
			continue
		}
		actualStage := stage.TransferTotal()
		row := StageBudgetReport{
			Stage:       StageID(stage.Name),
			BudgetBytes: limit,
			ActualBytes: actualStage,
			Utilization: float64(actualStage) / float64(limit),
		}
		if actualStage > limit {
			row.Exceeded = true
			row.OverageBytes = actualStage - limit
			out.ExceededStages = append(out.ExceededStages, row)
		}
		out.StageOverrides = append(out.StageOverrides, row)
	}
	return out
}
