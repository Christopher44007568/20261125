package sim

// Size is a render target size in pixels.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (s Size) pixels() int { return s.Width * s.Height }

func (s Size) equals(o Size) bool { return s.Width == o.Width && s.Height == o.Height }

// Equals reports whether two sizes match.
func (s Size) Equals(o Size) bool { return s.equals(o) }

func (s Size) String() string {
	return itoa(s.Width) + "x" + itoa(s.Height)
}

// Strategy selects how the stereo rig migrates to a new size.
type Strategy string

const (
	StrategyInplace   Strategy = "inplace"
	StrategyDoubleBuf Strategy = "doublebuf"
	StrategyStaged    Strategy = "staged"
)

func strategyName(s Strategy) string {
	switch s {
	case StrategyInplace:
		return "原地重建"
	case StrategyDoubleBuf:
		return "双缓冲切换"
	case StrategyStaged:
		return "分阶段迁移"
	}
	return string(s)
}

// Outcome of an upgrade attempt sequence.
type Outcome string

const (
	OutcomeUpgraded   Outcome = "upgraded"
	OutcomeRolledBack Outcome = "rolled_back"
)

// EyeIndex: 0 = left, 1 = right.
type EyeIndex int

const (
	Left  EyeIndex = 0
	Right EyeIndex = 1
)

func (e EyeIndex) String() string {
	if e == Right {
		return "right"
	}
	return "left"
}

// BudgetMode controls how the sampling budget follows resolution.
type BudgetMode string

const (
	// BudgetFixedSPP keeps samples per pixel; total budget grows with pixels.
	BudgetFixedSPP BudgetMode = "fixed_spp"
	// BudgetFixedTotal caps the per-eye per-frame sample total; SPP shrinks.
	BudgetFixedTotal BudgetMode = "fixed_total"
)

const (
	jumpThresholdLuma = 0.015
	frameBudgetMS     = 16.7
	unitsPerMS        = 150000.0
	historyBytesPerPx = 32
	imageBytesPerPx   = 24
	stagedBands       = 6
	// sceneTimeStep advances the continuous scene one unit per presented frame.
	sceneTimeStep = 1.0
)
