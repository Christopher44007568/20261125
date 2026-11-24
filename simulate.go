package xrpath

import "fmt"

func Simulate(faults ...Fault) RunResult {
	world := DefaultWorld()
	records := make([]FrameRecord, 0, FrameCount)
	images := make(map[int]*StereoImage, FrameCount)
	allEvents := []Event{}
	previousPhase := ""
	previousTracking := ""

	for frame := 0; frame < FrameCount; frame++ {
		motion := MotionFrameForOutput(frame)
		phase := PhaseForMotion(motion)
		lifecycle := "running"
		if frame >= PauseAtFrame && frame < ResumeAtFrame {
			lifecycle = "paused"
		}

		params := DefaultParameters()
		if frame >= ResumeAtFrame {
			params = ReloadedParameters()
		}
		tracking := TrackingForFrame(frame, motion)
		pose := PathPoseAtMotion(motion)
		missing := map[string]bool{}
		frameFaults := faultsAtFrame(faults, frame)

		for _, fault := range frameFaults {
			switch fault.Kind {
			case FaultPoseDrift:
				pose.Position = pose.Position.Add(V(0.24, 0, 0.16))
				pose.Yaw += 0.11
				pose.Orientation = QuatFromYawPitch(pose.Yaw, pose.Pitch)
			case FaultResourceMissing:
				target := fault.Target
				if target == "" {
					target = "tex-crate-blue"
				}
				missing[target] = true
			case FaultLifecycleSkipPause:
				lifecycle = "running"
			case FaultParametersStale:
				params = DefaultParameters()
			case FaultTrackingWrongKeyframe:
				tracking.MatchedKeyframe = "kf-unknown-13"
			}
		}

		if frame > 0 {
			previousPose := records[frame-1].Pose
			dt := 1.0 / float64(FrameRateHz)
			pose.LinearVelocity = pose.Position.Sub(previousPose.Position).Scale(1 / dt)
			pose.AngularVelocity = angularVelocityBetween(previousPose, pose, dt)
		}

		stereo := RenderStereo(pose, params, world, missing)
		applyImageFaults(frameFaults, stereo)

		leftRef, err := encodeRenderedEye(frame, stereo.Left)
		if err != nil {
			panic(fmt.Sprintf("encode left frame %d: %v", frame, err))
		}
		rightRef, err := encodeRenderedEye(frame, stereo.Right)
		if err != nil {
			panic(fmt.Sprintf("encode right frame %d: %v", frame, err))
		}

		visibility := VisibilitySet{Left: stereo.Left.Hits, Right: stereo.Right.Hits}
		resources := BuildResourceStates(world, params, visibility, tracking, missing)
		for _, fault := range frameFaults {
			if fault.Kind == FaultVisibilityDrop {
				target := fault.Target
				if target == "" {
					target = "occluder-wall"
				}
				visibility.Left = removeVisibleObject(visibility.Left, target)
				visibility.Right = removeVisibleObject(visibility.Right, target)
			}
		}
		previousStereo := images[frame-1]
		leftSource := BuildReprojection("left", frame, phase, lifecycle, pose, params, tracking, leftRef, stereo.Left, records, previousStereo)
		rightSource := BuildReprojection("right", frame, phase, lifecycle, pose, params, tracking, rightRef, stereo.Right, records, previousStereo)
		applyReprojectionFaults(frameFaults, frame, records, &leftSource, &rightSource)

		events := []Event{}
		if frame == 0 {
			events = append(events, Event{Frame: 0, AtSeconds: 0, Type: "session_start", Detail: "fixed XR path recording"})
		}
		if phase != previousPhase {
			events = append(events, Event{
				Frame: frame, AtSeconds: float64(frame) / FrameRateHz,
				Type: "phase_changed", Detail: phase,
			})
			previousPhase = phase
		}
		events = append(events, scriptedEventsForFrame(frame)...)
		if tracking.Status != previousTracking && tracking.Status != "tracking" {
			events = append(events, Event{
				Frame: frame, AtSeconds: float64(frame) / FrameRateHz,
				Type: "tracking_" + tracking.Status, Detail: "state transition recorded separately",
			})
		}
		if tracking.Status == "tracking" && previousTracking == "relocalized" {
			events = append(events, Event{
				Frame: frame, AtSeconds: float64(frame) / FrameRateHz,
				Type: "tracking_recovered", Detail: tracking.MatchedKeyframe,
			})
		}
		previousTracking = tracking.Status
		allEvents = append(allEvents, events...)

		record := FrameRecord{
			Schema: SchemaVersion, Frame: frame, MotionFrame: motion,
			AtSeconds: float64(frame) / FrameRateHz, Phase: phase, Lifecycle: lifecycle,
			Pose: pose, Tracking: tracking, Parameters: params, Events: events,
			Resources: resources, Visibility: visibility,
			Reprojection: [2]ReprojectionSource{leftSource, rightSource},
			Images:       ImagePair{Left: leftRef, Right: rightRef},
		}
		record.FrameSHA256 = canonicalSHA256(record)
		records = append(records, record)
		images[frame] = stereo
	}

	return RunResult{
		Manifest: buildManifest(records, allEvents),
		Records:  records,
		Images:   images,
	}
}

func faultsAtFrame(faults []Fault, frame int) []Fault {
	var result []Fault
	for _, fault := range faults {
		if fault.Frame == frame {
			result = append(result, fault)
		}
	}
	return result
}

func applyImageFaults(faults []Fault, stereo *StereoImage) {
	for _, fault := range faults {
		switch fault.Kind {
		case FaultLeftImagePixel:
			idx := (fault.Frame%stereo.Left.Image.Bounds().Max.Y)*stereo.Left.Image.Stride + 40*4
			stereo.Left.Image.Pix[idx] ^= 0xff
		case FaultRightImagePixel:
			idx := (fault.Frame%stereo.Right.Image.Bounds().Max.Y)*stereo.Right.Image.Stride + 40*4
			stereo.Right.Image.Pix[idx] ^= 0xff
		}
	}
}

func removeVisibleObject(objects []VisibleObject, id string) []VisibleObject {
	result := objects[:0]
	for _, obj := range objects {
		if obj.ID != id {
			result = append(result, obj)
		}
	}
	return result
}

func applyReprojectionFaults(faults []Fault, frame int, history []FrameRecord, left, right *ReprojectionSource) {
	for _, fault := range faults {
		if fault.Kind != FaultReprojectionStale {
			continue
		}
		target := left
		if fault.Target == "right" {
			target = right
		}
		staleFrame := frame - 8
		if staleFrame < 0 {
			staleFrame = 0
		}
		target.SourceFrame = staleFrame
		target.Mode = "injected_stale_pose"
		target.Reason = "fault_injection: source frame not updated"
		target.Confidence = 0.18
		if staleFrame < len(history) {
			src := history[staleFrame]
			target.SourcePosition = src.Pose.Position
			target.SourceOrientation = src.Pose.Orientation
			if target.Eye == "left" {
				target.SourceImageSHA256 = src.Images.Left.PixelSHA256
			} else {
				target.SourceImageSHA256 = src.Images.Right.PixelSHA256
			}
		}
	}
}
