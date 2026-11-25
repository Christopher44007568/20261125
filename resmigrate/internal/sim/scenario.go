package sim

// FaultSpec injects a deterministic upgrade failure.
type FaultSpec struct {
	Eye       EyeIndex `json:"eye"`
	OnAttempt int      `json:"on_attempt"`
	// Phase is "remap" for one-shot strategies and "stage" for staged.
	Phase string `json:"phase"`
	// Stage is the zero-based band index used for staged mid-migration faults.
	Stage int `json:"stage,omitempty"`
}

// ScenarioSpec fully describes one deterministic simulation run.
type ScenarioSpec struct {
	ID       string
	Title    string
	Strategy Strategy
	Initial  Size
	Target   Size
	SPP      int
	Budget   BudgetMode

	// Warmup/stable/post frame counts around the upgrade window.
	WarmupFrames int
	StableFrames int
	PostFrames   int

	RetryAllowed bool
	Fault        *FaultSpec

	// Discard skips history remapping on upgrade (control group).
	Discard bool
}

// Scenarios returns the full deterministic scenario suite.
func Scenarios() []ScenarioSpec {
	sudden := Size{320, 180}
	suddenTarget := Size{640, 360}
	small := Size{160, 90}
	extremeTarget := Size{960, 540}
	midTarget := Size{320, 180}

	specs := []ScenarioSpec{
		{ID: "sudden_inplace", Title: "尺寸突变 320x180 → 640x360 / 原地重建", Strategy: StrategyInplace, Initial: sudden, Target: suddenTarget, SPP: 4, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5},
		{ID: "sudden_doublebuf", Title: "尺寸突变 320x180 → 640x360 / 双缓冲切换", Strategy: StrategyDoubleBuf, Initial: sudden, Target: suddenTarget, SPP: 4, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5},
		{ID: "sudden_staged", Title: "尺寸突变 320x180 → 640x360 / 分阶段迁移", Strategy: StrategyStaged, Initial: sudden, Target: suddenTarget, SPP: 4, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5},
		{ID: "sudden_inplace_discard", Title: "对照：原地清空重建（不重映射）", Strategy: StrategyInplace, Initial: sudden, Target: suddenTarget, SPP: 4, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Discard: true},

		{ID: "extreme_inplace", Title: "极端尺寸突变 160x90 → 960x540 / 原地重建", Strategy: StrategyInplace, Initial: small, Target: extremeTarget, SPP: 2, Budget: BudgetFixedSPP, WarmupFrames: 1, StableFrames: 1, PostFrames: 3},
		{ID: "extreme_doublebuf", Title: "极端尺寸突变 160x90 → 960x540 / 双缓冲切换", Strategy: StrategyDoubleBuf, Initial: small, Target: extremeTarget, SPP: 2, Budget: BudgetFixedSPP, WarmupFrames: 1, StableFrames: 1, PostFrames: 3},
		{ID: "extreme_staged", Title: "极端尺寸突变 160x90 → 960x540 / 分阶段迁移", Strategy: StrategyStaged, Initial: small, Target: extremeTarget, SPP: 2, Budget: BudgetFixedSPP, WarmupFrames: 1, StableFrames: 1, PostFrames: 3},

		{ID: "fault_inplace_left_rollback", Title: "一眼首试失败并回滚 / 原地重建 / 左眼", Strategy: StrategyInplace, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Fault: &FaultSpec{Eye: Left, OnAttempt: 1, Phase: "remap"}},
		{ID: "fault_inplace_right_rollback", Title: "一眼首试失败并回滚 / 原地重建 / 右眼", Strategy: StrategyInplace, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Fault: &FaultSpec{Eye: Right, OnAttempt: 1, Phase: "remap"}},
		{ID: "fault_inplace_retry", Title: "一眼首试失败后重试成功 / 原地重建", Strategy: StrategyInplace, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, RetryAllowed: true, Fault: &FaultSpec{Eye: Right, OnAttempt: 1, Phase: "remap"}},

		{ID: "fault_doublebuf_left_rollback", Title: "一眼首试失败并回滚 / 双缓冲切换 / 左眼", Strategy: StrategyDoubleBuf, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Fault: &FaultSpec{Eye: Left, OnAttempt: 1, Phase: "remap"}},
		{ID: "fault_doublebuf_right_rollback", Title: "一眼首试失败并回滚 / 双缓冲切换 / 右眼", Strategy: StrategyDoubleBuf, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Fault: &FaultSpec{Eye: Right, OnAttempt: 1, Phase: "remap"}},
		{ID: "fault_doublebuf_retry", Title: "一眼首试失败后重试成功 / 双缓冲切换", Strategy: StrategyDoubleBuf, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, RetryAllowed: true, Fault: &FaultSpec{Eye: Left, OnAttempt: 1, Phase: "remap"}},

		{ID: "fault_staged_left_rollback", Title: "一眼首试失败并回滚 / 分阶段迁移 / 左眼", Strategy: StrategyStaged, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Fault: &FaultSpec{Eye: Left, OnAttempt: 1, Phase: "stage", Stage: 0}},
		{ID: "fault_staged_right_rollback", Title: "一眼首试失败并回滚 / 分阶段迁移 / 右眼", Strategy: StrategyStaged, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Fault: &FaultSpec{Eye: Right, OnAttempt: 1, Phase: "stage", Stage: 0}},
		{ID: "fault_staged_retry", Title: "一眼首试失败后重试成功 / 分阶段迁移", Strategy: StrategyStaged, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, RetryAllowed: true, Fault: &FaultSpec{Eye: Right, OnAttempt: 1, Phase: "stage", Stage: 0}},
		{ID: "fault_staged_left_midrollback", Title: "分阶段迁移中途左眼失败并回滚", Strategy: StrategyStaged, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Fault: &FaultSpec{Eye: Left, OnAttempt: 1, Phase: "stage", Stage: 2}},
		{ID: "fault_staged_right_midrollback", Title: "分阶段迁移中途右眼失败并回滚", Strategy: StrategyStaged, Initial: small, Target: midTarget, SPP: 3, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5, Fault: &FaultSpec{Eye: Right, OnAttempt: 1, Phase: "stage", Stage: 2}},

		{ID: "budget_fixed_spp", Title: "固定 SPP：预算随像素数迁移", Strategy: StrategyDoubleBuf, Initial: sudden, Target: suddenTarget, SPP: 6, Budget: BudgetFixedSPP, WarmupFrames: 2, StableFrames: 1, PostFrames: 5},
		{ID: "budget_fixed_total", Title: "固定总预算：SPP 随分辨率收缩", Strategy: StrategyDoubleBuf, Initial: sudden, Target: suddenTarget, SPP: 8, Budget: BudgetFixedTotal, WarmupFrames: 2, StableFrames: 1, PostFrames: 5},
	}
	return specs
}
