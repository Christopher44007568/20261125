package xrpath

import "math"

func BuildReprojection(
	eye string,
	frame int,
	phase string,
	lifecycle string,
	pose Pose,
	params Parameters,
	tracking TrackingState,
	currentRef ImageRef,
	currentEye *RenderedEye,
	history []FrameRecord,
	previousStereo *StereoImage,
) ReprojectionSource {
	sourceFrame := frame
	mode := "current_raster"
	reason := "first_frame_has_no_history"
	keyframe := ""
	confidence := 1.0
	validRatio := 1.0
	disocclusion := 0

	if frame > 0 {
		sourceFrame = frame - 1
		mode = "depth_aware_warp"
		reason = "continuous_pose"
		confidence = 0.975
	}

	switch {
	case frame == 0:
		// Keep first-frame defaults.
	case lifecycle == "paused":
		mode = "pose_hold"
		reason = "simulation_paused_pose_unchanged"
		confidence = 0.999
		validRatio = 1
	case tracking.Status == "lost":
		sourceFrame = tracking.LastGoodFrame
		mode = "last_good_frame_warp"
		reason = "tracking_lost_using_last_good_depth"
		confidence = 0.42
		validRatio = 0.91
	case tracking.Status == "relocalizing":
		sourceFrame = tracking.LastGoodFrame
		keyframe = tracking.MatchedKeyframe
		mode = "relocalization_keyframe"
		reason = "matching_map_keyframe"
		confidence = 0.58
		validRatio = 0.88
	case tracking.Status == "relocalized":
		sourceFrame = tracking.LastGoodFrame
		keyframe = tracking.MatchedKeyframe
		mode = "keyframe_pose_blend"
		reason = "relocalization_accepted"
		confidence = 0.88
		validRatio = 0.94
	case phase == "static":
		mode = "same_pose_warp"
		reason = "static_identity_warp"
		confidence = 0.999
		validRatio = 1
	case phase == "fast_rotation":
		mode = "time_warp"
		reason = "high_angular_velocity"
		confidence = clampFloat(0.96-math.Abs(pose.AngularVelocity.Y)*0.01, 0.72, 0.96)
		validRatio = 0.965
	case phase == "forward":
		mode = "depth_warp"
		reason = "forward_translation_requires_depth"
		confidence = 0.965
		validRatio = 0.955
	case phase == "occlusion_reveal":
		mode = "disocclusion_aware_warp"
		reason = "newly_revealed_pixels_rasterized"
		disocclusion = revealDisocclusionPixels(eye, currentEye, previousStereo)
		validRatio = clampFloat(1-float64(disocclusion)/float64(params.Width*params.Height), 0.82, 1)
		confidence = 0.94
	case phase == "relocalization":
		mode = "post_relocalization_depth_warp"
		keyframe = tracking.MatchedKeyframe
		reason = "settling_after_relocalization"
		confidence = 0.93
		validRatio = 0.96
	}

	sourcePose := pose
	sourceHash := currentRef.PixelSHA256
	if sourceFrame != frame && sourceFrame >= 0 && sourceFrame < len(history) {
		src := history[sourceFrame]
		sourcePose = src.Pose
		if eye == "left" {
			sourceHash = src.Images.Left.PixelSHA256
		} else {
			sourceHash = src.Images.Right.PixelSHA256
		}
	}

	warpHash := canonicalSHA256(struct {
		Policy      string     `json:"policy"`
		Mode        string     `json:"mode"`
		Frame       int        `json:"frame"`
		SourceFrame int        `json:"source_frame"`
		Keyframe    string     `json:"keyframe"`
		SourcePose  Pose       `json:"source_pose"`
		TargetPose  Pose       `json:"target_pose"`
		Params      Parameters `json:"params"`
	}{params.ReprojectionPolicy, mode, frame, sourceFrame, keyframe, sourcePose, pose, params})

	return ReprojectionSource{
		Eye: eye, Mode: mode, SourceFrame: sourceFrame, SourceKeyframe: keyframe,
		SourcePosition: sourcePose.Position, SourceOrientation: sourcePose.Orientation,
		SourceImageSHA256: sourceHash, WarpMeshSHA256: warpHash,
		Confidence: confidence, ValidPixelRatio: validRatio,
		DisocclusionPixels: disocclusion, Reason: reason,
	}
}

func revealDisocclusionPixels(eye string, current *RenderedEye, previous *StereoImage) int {
	if current == nil || previous == nil {
		return 0
	}
	var prior *RenderedEye
	if eye == "left" {
		prior = previous.Left
	} else {
		prior = previous.Right
	}
	if prior == nil {
		return 0
	}
	currentRelic := 0
	previousRelic := 0
	for _, id := range current.ObjectIDs {
		if id == "reveal-relic" {
			currentRelic++
		}
	}
	for _, id := range prior.ObjectIDs {
		if id == "reveal-relic" {
			previousRelic++
		}
	}
	if currentRelic <= previousRelic {
		return 0
	}
	return currentRelic - previousRelic
}
