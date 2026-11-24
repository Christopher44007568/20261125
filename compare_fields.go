package xrpath

import "reflect"

const geometryEpsilon = 1e-7

func compareLifecycle(c *comparator, frame int, want, got FrameRecord) {
	c.equalString(frame, SubsystemLifecycle, "frame", "phase", want.Phase, got.Phase)
	c.equalString(frame, SubsystemLifecycle, "frame", "lifecycle", want.Lifecycle, got.Lifecycle)
	c.equalInt(frame, SubsystemLifecycle, "frame", "motion_frame", want.MotionFrame, got.MotionFrame)
	c.equalFloat(frame, SubsystemLifecycle, "frame", "at_seconds", want.AtSeconds, got.AtSeconds, 1e-9)
	c.check(SubsystemLifecycle)
	if !reflect.DeepEqual(want.Events, got.Events) {
		c.add(Mismatch{
			Frame: frame, Subsystem: SubsystemLifecycle, Item: "events", Field: "event_stream",
			Expected: want.Events, Actual: got.Events, Detail: "pause/resume/reload/tracking event differs",
		})
	}
}

func compareParameters(c *comparator, frame int, want, got Parameters) {
	s := SubsystemParameters
	c.equalInt(frame, s, "tuning", "revision", want.Revision, got.Revision)
	c.equalInt(frame, s, "viewport", "width", want.Width, got.Width)
	c.equalInt(frame, s, "viewport", "height", want.Height, got.Height)
	c.equalFloat(frame, s, "frustum", "fov_deg", want.FOVDeg, got.FOVDeg, 1e-9)
	c.equalFloat(frame, s, "frustum", "near", want.Near, got.Near, 1e-9)
	c.equalFloat(frame, s, "frustum", "far", want.Far, got.Far, 1e-9)
	c.equalFloat(frame, s, "stereo", "ipd", want.IPD, got.IPD, 1e-9)
	c.equalFloat(frame, s, "render", "exposure", want.Exposure, got.Exposure, 1e-9)
	c.equalString(frame, s, "reprojection", "policy", want.ReprojectionPolicy, got.ReprojectionPolicy)
	c.check(s)
	if want.MaxResourceBytes != got.MaxResourceBytes {
		c.add(Mismatch{Frame: frame, Subsystem: s, Item: "budget", Field: "max_resource_bytes", Expected: want.MaxResourceBytes, Actual: got.MaxResourceBytes})
	}
}

func compareTracking(c *comparator, frame int, want, got TrackingState) {
	s := SubsystemTracking
	c.equalString(frame, s, "tracking", "status", want.Status, got.Status)
	c.equalFloat(frame, s, "tracking", "quality", want.Quality, got.Quality, 1e-9)
	c.equalString(frame, s, "keyframe", "matched_keyframe", want.MatchedKeyframe, got.MatchedKeyframe)
	c.equalInt(frame, s, "tracking", "last_good_frame", want.LastGoodFrame, got.LastGoodFrame)
	c.equalInt(frame, s, "tracking", "relocalized_frame", want.RelocalizedFrame, got.RelocalizedFrame)
	c.equalInt(frame, s, "tracking", "feature_count", want.FeatureCount, got.FeatureCount)
}

func comparePose(c *comparator, frame int, want, got Pose) {
	s := SubsystemPose
	c.equalFloat(frame, s, "position", "x", want.Position.X, got.Position.X, geometryEpsilon)
	c.equalFloat(frame, s, "position", "y", want.Position.Y, got.Position.Y, geometryEpsilon)
	c.equalFloat(frame, s, "position", "z", want.Position.Z, got.Position.Z, geometryEpsilon)
	c.equalFloat(frame, s, "orientation", "w", want.Orientation.W, got.Orientation.W, geometryEpsilon)
	c.equalFloat(frame, s, "orientation", "x", want.Orientation.X, got.Orientation.X, geometryEpsilon)
	c.equalFloat(frame, s, "orientation", "y", want.Orientation.Y, got.Orientation.Y, geometryEpsilon)
	c.equalFloat(frame, s, "orientation", "z", want.Orientation.Z, got.Orientation.Z, geometryEpsilon)
	c.equalFloat(frame, s, "euler", "yaw_rad", want.Yaw, got.Yaw, geometryEpsilon)
	c.equalFloat(frame, s, "euler", "pitch_rad", want.Pitch, got.Pitch, geometryEpsilon)
	c.equalFloat(frame, s, "euler", "roll_rad", want.Roll, got.Roll, geometryEpsilon)
	c.equalFloat(frame, s, "linear_velocity", "x", want.LinearVelocity.X, got.LinearVelocity.X, 1e-6)
	c.equalFloat(frame, s, "linear_velocity", "y", want.LinearVelocity.Y, got.LinearVelocity.Y, 1e-6)
	c.equalFloat(frame, s, "linear_velocity", "z", want.LinearVelocity.Z, got.LinearVelocity.Z, 1e-6)
	c.equalFloat(frame, s, "angular_velocity", "x", want.AngularVelocity.X, got.AngularVelocity.X, 1e-6)
	c.equalFloat(frame, s, "angular_velocity", "y", want.AngularVelocity.Y, got.AngularVelocity.Y, 1e-6)
	c.equalFloat(frame, s, "angular_velocity", "z", want.AngularVelocity.Z, got.AngularVelocity.Z, 1e-6)
}

func compareResources(c *comparator, frame int, want, got []ResourceState) {
	s := SubsystemResource
	wantMap := map[string]ResourceState{}
	gotMap := map[string]ResourceState{}
	wantIDs := []string{}
	gotIDs := []string{}
	for _, item := range want {
		wantMap[item.ID] = item
		wantIDs = append(wantIDs, item.ID)
	}
	for _, item := range got {
		gotMap[item.ID] = item
		gotIDs = append(gotIDs, item.ID)
	}
	c.sliceEqual(frame, s, "catalog", "resource_ids", wantIDs, gotIDs)

	for _, id := range wantIDs {
		w, ok := wantMap[id]
		g, exists := gotMap[id]
		if !ok || !exists {
			continue
		}
		item := "resource:" + id
		c.equalString(frame, s, item, "kind", w.Kind, g.Kind)
		c.equalInt(frame, s, item, "revision", w.Revision, g.Revision)
		c.equalString(frame, s, item, "sha256", w.SHA256, g.SHA256)
		c.equalString(frame, s, item, "status", w.Status, g.Status)
		c.equalBool(frame, s, item, "resident", w.Resident, g.Resident)
		c.check(s)
		if w.Bytes != g.Bytes {
			c.add(Mismatch{Frame: frame, Subsystem: s, Item: item, Field: "bytes", Expected: w.Bytes, Actual: g.Bytes})
		}
		c.equalInt(frame, s, item, "reference_count", w.ReferenceCount, g.ReferenceCount)
		c.sliceEqual(frame, s, item, "used_by", w.UsedBy, g.UsedBy)
	}
}

func compareVisibility(c *comparator, frame int, want, got VisibilitySet) {
	compareEyeVisibility(c, frame, "left", want.Left, got.Left)
	compareEyeVisibility(c, frame, "right", want.Right, got.Right)
}

func compareEyeVisibility(c *comparator, frame int, eye string, want, got []VisibleObject) {
	s := SubsystemVisibility
	wantMap := map[string]VisibleObject{}
	gotMap := map[string]VisibleObject{}
	wantIDs := []string{}
	gotIDs := []string{}
	for _, obj := range want {
		wantMap[obj.ID] = obj
		wantIDs = append(wantIDs, obj.ID)
	}
	for _, obj := range got {
		gotMap[obj.ID] = obj
		gotIDs = append(gotIDs, obj.ID)
	}
	c.sliceEqual(frame, s, eye, "visible_ids", wantIDs, gotIDs)

	for _, id := range wantIDs {
		w, ok := wantMap[id]
		g, exists := gotMap[id]
		if !ok || !exists {
			continue
		}
		item := eye + ":" + id
		c.equalString(frame, s, item, "resource_id", w.ResourceID, g.ResourceID)
		c.equalFloat(frame, s, item, "center.x", w.Center.X, g.Center.X, geometryEpsilon)
		c.equalFloat(frame, s, item, "center.y", w.Center.Y, g.Center.Y, geometryEpsilon)
		c.equalFloat(frame, s, item, "center.z", w.Center.Z, g.Center.Z, geometryEpsilon)
		c.equalFloat(frame, s, item, "half_extents.x", w.HalfExtents.X, g.HalfExtents.X, geometryEpsilon)
		c.equalFloat(frame, s, item, "half_extents.y", w.HalfExtents.Y, g.HalfExtents.Y, geometryEpsilon)
		c.equalFloat(frame, s, item, "half_extents.z", w.HalfExtents.Z, g.HalfExtents.Z, geometryEpsilon)
		c.equalFloat(frame, s, item, "depth", w.Depth, g.Depth, 1e-6)
		c.equalInt(frame, s, item, "pixels", w.Pixels, g.Pixels)
		c.equalInt(frame, s, item, "screen.min_x", w.Screen.MinX, g.Screen.MinX)
		c.equalInt(frame, s, item, "screen.min_y", w.Screen.MinY, g.Screen.MinY)
		c.equalInt(frame, s, item, "screen.max_x", w.Screen.MaxX, g.Screen.MaxX)
		c.equalInt(frame, s, item, "screen.max_y", w.Screen.MaxY, g.Screen.MaxY)
	}
}

func compareReprojection(c *comparator, frame int, want, got [2]ReprojectionSource) {
	for i := 0; i < 2; i++ {
		w := want[i]
		g := got[i]
		item := w.Eye
		if item == "" {
			item = g.Eye
		}
		s := SubsystemReprojection
		c.equalString(frame, s, item, "eye", w.Eye, g.Eye)
		c.equalString(frame, s, item, "mode", w.Mode, g.Mode)
		c.equalInt(frame, s, item, "source_frame", w.SourceFrame, g.SourceFrame)
		c.equalString(frame, s, item, "source_keyframe", w.SourceKeyframe, g.SourceKeyframe)
		c.equalFloat(frame, s, item, "source_position.x", w.SourcePosition.X, g.SourcePosition.X, geometryEpsilon)
		c.equalFloat(frame, s, item, "source_position.y", w.SourcePosition.Y, g.SourcePosition.Y, geometryEpsilon)
		c.equalFloat(frame, s, item, "source_position.z", w.SourcePosition.Z, g.SourcePosition.Z, geometryEpsilon)
		c.equalFloat(frame, s, item, "source_orientation.w", w.SourceOrientation.W, g.SourceOrientation.W, geometryEpsilon)
		c.equalFloat(frame, s, item, "source_orientation.x", w.SourceOrientation.X, g.SourceOrientation.X, geometryEpsilon)
		c.equalFloat(frame, s, item, "source_orientation.y", w.SourceOrientation.Y, g.SourceOrientation.Y, geometryEpsilon)
		c.equalFloat(frame, s, item, "source_orientation.z", w.SourceOrientation.Z, g.SourceOrientation.Z, geometryEpsilon)
		c.equalString(frame, s, item, "source_image_sha256", w.SourceImageSHA256, g.SourceImageSHA256)
		c.equalString(frame, s, item, "warp_mesh_sha256", w.WarpMeshSHA256, g.WarpMeshSHA256)
		c.equalFloat(frame, s, item, "confidence", w.Confidence, g.Confidence, 1e-9)
		c.equalFloat(frame, s, item, "valid_pixel_ratio", w.ValidPixelRatio, g.ValidPixelRatio, 1e-9)
		c.equalInt(frame, s, item, "disocclusion_pixels", w.DisocclusionPixels, g.DisocclusionPixels)
		c.equalString(frame, s, item, "reason", w.Reason, g.Reason)
	}
}

func compareImageRefs(c *comparator, frame int, want, got ImagePair) {
	compareOneImageRef(c, frame, SubsystemLeftImage, want.Left, got.Left)
	compareOneImageRef(c, frame, SubsystemRightImage, want.Right, got.Right)
}

func compareOneImageRef(c *comparator, frame int, subsystem string, want, got ImageRef) {
	c.equalString(frame, subsystem, want.Eye, "path", want.Path, got.Path)
	c.equalInt(frame, subsystem, want.Eye, "width", want.Width, got.Width)
	c.equalInt(frame, subsystem, want.Eye, "height", want.Height, got.Height)
	c.equalString(frame, subsystem, want.Eye, "pixel_sha256", want.PixelSHA256, got.PixelSHA256)
	c.equalString(frame, subsystem, want.Eye, "png_sha256", want.PNGSHA256, got.PNGSHA256)
}

func compareFrameHash(c *comparator, frame int, want, got string) {
	c.equalString(frame, SubsystemFrameHash, "record", "frame_sha256", want, got)
}
