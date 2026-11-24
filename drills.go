package xrpath

import "encoding/json"

type FaultDrill struct {
	Fault             Fault     `json:"fault"`
	ExpectedSubsystem string    `json:"expected_subsystem"`
	Detected          bool      `json:"detected"`
	FirstDivergence   *Mismatch `json:"first_divergence,omitempty"`
	MismatchCount     int       `json:"mismatch_count"`
}

type DrillSuite struct {
	OK     bool         `json:"ok"`
	Drills []FaultDrill `json:"drills"`
}

type HashBlindDrill struct {
	Name               string    `json:"name"`
	OK                 bool      `json:"ok"`
	FirstDivergence    *Mismatch `json:"first_divergence,omitempty"`
	PoseMismatches     int       `json:"pose_mismatches"`
	ResourceMismatches int       `json:"resource_mismatches"`
	Report             Report    `json:"report"`
}

func ExpectedSubsystemForFault(kind string) string {
	switch kind {
	case FaultPoseDrift:
		return SubsystemPose
	case FaultResourceMissing:
		return SubsystemResource
	case FaultVisibilityDrop:
		return SubsystemVisibility
	case FaultReprojectionStale:
		return SubsystemReprojection
	case FaultLeftImagePixel:
		return SubsystemLeftImage
	case FaultRightImagePixel:
		return SubsystemRightImage
	case FaultLifecycleSkipPause:
		return SubsystemLifecycle
	case FaultParametersStale:
		return SubsystemParameters
	case FaultTrackingWrongKeyframe:
		return SubsystemTracking
	default:
		return ""
	}
}

func RunFaultDrills(baseline RunResult) DrillSuite {
	suite := DrillSuite{OK: true, Drills: []FaultDrill{}}
	for _, fault := range CanonicalFaultMatrix() {
		candidate := Simulate(fault)
		report := CompareRuns(baseline, candidate)
		expected := ExpectedSubsystemForFault(fault.Kind)
		drill := FaultDrill{
			Fault: fault, ExpectedSubsystem: expected,
			MismatchCount:   report.MismatchCount,
			FirstDivergence: report.FirstDivergence,
		}
		if report.FirstDivergence != nil {
			drill.Detected = report.FirstDivergence.Frame == fault.Frame &&
				report.FirstDivergence.Subsystem == expected
		}
		if !drill.Detected {
			suite.OK = false
		}
		suite.Drills = append(suite.Drills, drill)
	}
	return suite
}

func RunHashBlindDrills(baseline []FrameRecord) ([]HashBlindDrill, error) {
	geometryCandidate, err := CloneRecords(baseline)
	if err != nil {
		return nil, err
	}
	geometryCandidate[31].Pose.Position.X += 0.31
	geometryCandidate[31].Pose.Orientation = QuatFromYawPitch(geometryCandidate[31].Pose.Yaw+0.2, geometryCandidate[31].Pose.Pitch)
	geometryReport := CompareRecords(baseline, geometryCandidate)

	resourceCandidate, err := CloneRecords(baseline)
	if err != nil {
		return nil, err
	}
	resourceCandidate[31].Resources[0].Status = "corrupt_hash_mask_test"
	resourceCandidate[31].Resources[0].Resident = false
	resourceReport := CompareRecords(baseline, resourceCandidate)

	return []HashBlindDrill{
		summarizeHashBlind("same_image_hash_but_pose_drift", geometryReport, SubsystemPose),
		summarizeHashBlind("same_image_hash_but_resource_change", resourceReport, SubsystemResource),
	}, nil
}

func summarizeHashBlind(name string, report Report, expectedFirst string) HashBlindDrill {
	poseCount := report.MismatchesBySystem[SubsystemPose]
	resourceCount := report.MismatchesBySystem[SubsystemResource]
	ok := !report.OK && report.FirstDivergence != nil && report.FirstDivergence.Subsystem == expectedFirst
	return HashBlindDrill{
		Name: name, OK: ok, FirstDivergence: report.FirstDivergence,
		PoseMismatches: poseCount, ResourceMismatches: resourceCount, Report: report,
	}
}

func CloneRecords(records []FrameRecord) ([]FrameRecord, error) {
	data, err := json.Marshal(records)
	if err != nil {
		return nil, err
	}
	var cloned []FrameRecord
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}
