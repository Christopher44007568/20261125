package xrpath

type pathKey struct {
	Motion int
	Pos    Vec3
	Yaw    float64
	Pitch  float64
}

var fixedPathKeys = []pathKey{
	{Motion: 0, Pos: V(0, 1.60, 2.80), Yaw: 0, Pitch: -0.03},
	{Motion: 11, Pos: V(0, 1.60, 2.80), Yaw: 0, Pitch: -0.03},
	{Motion: 12, Pos: V(0, 1.60, 2.80), Yaw: 0, Pitch: -0.03},
	{Motion: 14, Pos: V(0, 1.60, 2.80), Yaw: -0.86, Pitch: -0.04},
	{Motion: 16, Pos: V(0, 1.60, 2.80), Yaw: 0.86, Pitch: -0.04},
	{Motion: 18, Pos: V(0, 1.60, 2.80), Yaw: -0.28, Pitch: -0.03},
	{Motion: 20, Pos: V(0, 1.60, 2.80), Yaw: 0, Pitch: -0.03},
	{Motion: 27, Pos: V(0, 1.60, 1.00), Yaw: 0, Pitch: -0.05},
	{Motion: 33, Pos: V(0, 1.60, -0.80), Yaw: 0, Pitch: -0.07},
	{Motion: 38, Pos: V(-0.85, 1.61, -0.92), Yaw: -0.16, Pitch: -0.07},
	{Motion: 42, Pos: V(-2.10, 1.62, -1.05), Yaw: -0.31, Pitch: -0.06},
	{Motion: 47, Pos: V(-2.10, 1.62, -1.05), Yaw: -0.31, Pitch: -0.06},
	{Motion: 52, Pos: V(-1.92, 1.61, -1.22), Yaw: -0.22, Pitch: -0.05},
	{Motion: 58, Pos: V(-1.65, 1.60, -1.38), Yaw: -0.15, Pitch: -0.04},
	{Motion: 67, Pos: V(-1.65, 1.60, -1.38), Yaw: -0.15, Pitch: -0.04},
}

func MotionFrameForOutput(frame int) int {
	switch {
	case frame < PauseAtFrame:
		return frame
	case frame < ResumeAtFrame:
		return PauseAtFrame - 1
	default:
		return frame - (ResumeAtFrame - PauseAtFrame)
	}
}

func PhaseForMotion(motion int) string {
	switch {
	case motion <= 11:
		return "static"
	case motion <= 19:
		return "fast_rotation"
	case motion <= 33:
		return "forward"
	case motion <= 47:
		return "occlusion_reveal"
	default:
		return "relocalization"
	}
}

func PathPoseAtMotion(motion int) Pose {
	if motion <= fixedPathKeys[0].Motion {
		k := fixedPathKeys[0]
		return poseFromKey(k)
	}
	last := fixedPathKeys[len(fixedPathKeys)-1]
	if motion >= last.Motion {
		return poseFromKey(last)
	}

	for i := 0; i < len(fixedPathKeys)-1; i++ {
		a := fixedPathKeys[i]
		b := fixedPathKeys[i+1]
		if motion < a.Motion || motion > b.Motion || a.Motion == b.Motion {
			continue
		}
		if motion <= b.Motion {
			t := smoothStep(float64(motion-a.Motion) / float64(b.Motion-a.Motion))
			yaw := a.Yaw + angleDelta(a.Yaw, b.Yaw)*t
			pitch := lerp(a.Pitch, b.Pitch, t)
			pos := lerpVec(a.Pos, b.Pos, t)
			return Pose{
				Position:    pos,
				Orientation: QuatFromYawPitch(yaw, pitch),
				Yaw:         yaw,
				Pitch:       pitch,
			}
		}
	}
	return poseFromKey(last)
}

func poseFromKey(k pathKey) Pose {
	return Pose{
		Position:    k.Pos,
		Orientation: QuatFromYawPitch(k.Yaw, k.Pitch),
		Yaw:         k.Yaw,
		Pitch:       k.Pitch,
	}
}

func TrackingForFrame(frame, motion int) TrackingState {
	lastGood := PauseAtFrame - 1
	if motion >= 48 {
		lastGood = 51
	}
	switch {
	case motion < 48:
		return TrackingState{
			Status: "tracking", Quality: 0.985, LastGoodFrame: frame,
			FeatureCount: 168 + (frame*7)%31,
		}
	case motion <= 50:
		return TrackingState{Status: "lost", Quality: 0, LastGoodFrame: lastGood, FeatureCount: 0}
	case motion == 51:
		return TrackingState{
			Status: "relocalizing", Quality: 0.34, LastGoodFrame: lastGood,
			MatchedKeyframe: "kf-atrium-07", FeatureCount: 21,
		}
	case motion == 52:
		return TrackingState{
			Status: "relocalized", Quality: 0.86, LastGoodFrame: frame,
			RelocalizedFrame: frame, MatchedKeyframe: "kf-atrium-07", FeatureCount: 96,
		}
	default:
		return TrackingState{
			Status: "tracking", Quality: 0.972, LastGoodFrame: frame,
			RelocalizedFrame: 56, MatchedKeyframe: "kf-atrium-07", FeatureCount: 154 + (frame*5)%23,
		}
	}
}

func FixedPathHash() string {
	return canonicalSHA256(struct {
		FrameCount  int       `json:"frame_count"`
		FrameRateHz int       `json:"frame_rate_hz"`
		PauseAt     int       `json:"pause_at"`
		ResumeAt    int       `json:"resume_at"`
		Keys        []pathKey `json:"keys"`
	}{FrameCount, FrameRateHz, PauseAtFrame, ResumeAtFrame, fixedPathKeys})
}

func angularVelocityBetween(a, b Pose, dt float64) Vec3 {
	dyaw := angleDelta(a.Yaw, b.Yaw) / dt
	dpitch := angleDelta(a.Pitch, b.Pitch) / dt
	return V(dpitch, dyaw, 0)
}
