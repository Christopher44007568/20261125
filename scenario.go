package stereorollback

// InvariantCheck is one named correctness assertion evaluated on a scenario.
type InvariantCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// ScenarioReport contains the full deterministic output of one named case.
type ScenarioReport struct {
	ID              string           `json:"id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Counters        Counters         `json:"counters"`
	Frames          []FrameRecord    `json:"frames"`
	PoolResources   []ResourceInfo   `json:"pool_resources_after"`
	InvariantChecks []InvariantCheck `json:"invariant_checks"`
	Clean           bool             `json:"clean"`
}

func runEyeScenario(id, title, description string, failed Eye) (*ScenarioReport, error) {
	script := Script{EyeFaults: map[uint64]map[Eye]FaultType{
		2: {failed: FaultAttachmentCreate},
		3: {failed: FaultMaterialParameter},
		4: {failed: FaultReprojectionTimeout},
	}}
	p := NewPipeline(Config{Script: script})
	if err := p.Bootstrap(); err != nil {
		return nil, err
	}

	report := &ScenarioReport{ID: id, Title: title, Description: description}
	counters := NewCounters()
	for frame := uint64(1); frame <= 5; frame++ {
		fr := p.RunFrame(frame)
		report.Frames = append(report.Frames, fr)
		counters.RecordFrame(fr)
	}
	report.Counters = counters
	report.PoolResources = p.Pool().List()
	report.InvariantChecks = checkEyeScenario(report, p, failed)
	report.Clean = allChecksHold(report.InvariantChecks)
	return report, nil
}

// RunLeftEyeFailure runs the case where only the left eye fails.
func RunLeftEyeFailure() (*ScenarioReport, error) {
	return runEyeScenario(
		"left_eye_failure",
		"Left-eye failure preserves the right eye",
		"Left eye hits attachment creation, material parameter and reprojection faults; right eye keeps rendering and shared resources survive.",
		EyeLeft,
	)
}

// RunRightEyeFailure mirrors RunLeftEyeFailure for the right eye.
func RunRightEyeFailure() (*ScenarioReport, error) {
	return runEyeScenario(
		"right_eye_failure",
		"Right-eye failure preserves the left eye",
		"Right eye hits attachment creation, material parameter and reprojection faults; left eye keeps rendering and shared resources survive.",
		EyeRight,
	)
}

// RunSharedTextureCorruption runs the distinct shared-texture corruption case.
func RunSharedTextureCorruption() (*ScenarioReport, error) {
	script := Script{CorruptSharedAt: map[uint64]ResourceID{2: ResSharedAtlas}}
	p := NewPipeline(Config{Script: script})
	if err := p.Bootstrap(); err != nil {
		return nil, err
	}

	report := &ScenarioReport{
		ID:          "shared_texture_corruption",
		Title:       "Shared texture corruption handled separately",
		Description: "The shared atlas is corrupted; it is quarantined and restored in place while all users and unrelated resources survive.",
	}
	counters := NewCounters()
	for frame := uint64(1); frame <= 4; frame++ {
		fr := p.RunFrame(frame)
		report.Frames = append(report.Frames, fr)
		counters.RecordFrame(fr)
	}
	report.Counters = counters
	report.PoolResources = p.Pool().List()
	report.InvariantChecks = checkSharedScenario(report, p)
	report.Clean = allChecksHold(report.InvariantChecks)
	return report, nil
}

func allChecksHold(checks []InvariantCheck) bool {
	for _, c := range checks {
		if !c.OK {
			return false
		}
	}
	return true
}

func hasStage(c Counters, stage Stage) bool {
	for key, n := range c.FailuresByStage {
		if Stage(key) == stage && n > 0 {
			return true
		}
	}
	return false
}
