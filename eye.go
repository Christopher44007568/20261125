package main

import (
	"fmt"
	"math"
)

type EyeID string

const (
	LeftEye  EyeID = "left"
	RightEye EyeID = "right"
)

// FOV stores positive tangent-plane distances for the four frustum edges.
// Left/right are the magnitudes of X/Z at the left and right edges.
type FOV struct {
	Left  float64 `json:"leftTan"`
	Right float64 `json:"rightTan"`
	Up    float64 `json:"upTan"`
	Down  float64 `json:"downTan"`
}

func (f FOV) HorizontalTan() float64 { return f.Left + f.Right }
func (f FOV) VerticalTan() float64   { return f.Up + f.Down }

type TrackingState struct {
	State       string  `json:"state"`
	Confidence  float64 `json:"confidence"`
	SafeInsetPx float64 `json:"safeInsetPx"`
	Fallback    string  `json:"fallback"`
}

type Eye struct {
	ID       EyeID         `json:"id"`
	Width    int           `json:"width"`
	Height   int           `json:"height"`
	IPD      float64       `json:"ipdMeters"`
	Pose     Pose          `json:"pose"`
	FOV      FOV           `json:"fov"`
	Tracking TrackingState `json:"tracking"`
	LeadSec  float64       `json:"predictionLeadSeconds"`
}

type Projection struct {
	Point            ScreenPoint `json:"point"`
	Visible          bool        `json:"visible"`
	DepthM           float64     `json:"depthMeters"`
	Tangent          ScreenPoint `json:"tangent"`
	PixelsPerMeter   float64     `json:"pixelsPerMeter"`
	PixelsPerRadianX float64     `json:"pixelsPerRadianX"`
	PixelsPerRadianY float64     `json:"pixelsPerRadianY"`
	ProjectionPath   string      `json:"projectionPath"`
}

func (e Eye) focalX() float64 {
	return float64(e.Width) / e.FOV.HorizontalTan()
}

func (e Eye) focalY() float64 {
	return float64(e.Height) / e.FOV.VerticalTan()
}

func (e Eye) principalX() float64 {
	return float64(e.Width) * e.FOV.Left / e.FOV.HorizontalTan()
}

func (e Eye) principalY() float64 {
	return float64(e.Height) * e.FOV.Up / e.FOV.VerticalTan()
}

// Project always runs the full per-eye frustum transform. Callers must never
// substitute the returned coordinate from the other eye.
func (e Eye) Project(world Vec3) Projection {
	eyePoint := e.Pose.ToEye(world)
	fx, fy := e.focalX(), e.focalY()
	cx, cy := e.principalX(), e.principalY()
	if eyePoint.Z <= 0.02 {
		return Projection{
			Visible:        false,
			DepthM:         eyePoint.Z,
			ProjectionPath: "independent_eye_off_axis_frustum",
		}
	}

	tanX := eyePoint.X / eyePoint.Z
	tanY := eyePoint.Y / eyePoint.Z
	p := ScreenPoint{X: cx + fx*tanX, Y: cy - fy*tanY}
	visible := p.X >= 0 && p.X <= float64(e.Width) &&
		p.Y >= 0 && p.Y <= float64(e.Height)

	return Projection{
		Point:            p,
		Visible:          visible,
		DepthM:           eyePoint.Z,
		Tangent:          ScreenPoint{X: tanX, Y: tanY},
		PixelsPerMeter:   (fx + fy) * 0.5 / eyePoint.Z,
		PixelsPerRadianX: fx,
		PixelsPerRadianY: fy,
		ProjectionPath:   "independent_eye_off_axis_frustum",
	}
}

func (e Eye) ImageRect() FloatRect {
	return FloatRect{MinX: 0, MinY: 0, MaxX: float64(e.Width), MaxY: float64(e.Height)}
}

func (e Eye) SafeRect() FloatRect {
	inset := e.Tracking.SafeInsetPx
	return e.ImageRect().Inset(inset, inset)
}

type TrackingSpec struct {
	State       string
	Confidence  float64
	SafeInsetPx float64
	Fallback    string
}

type Scenario struct {
	ID                   string
	TitleZH              string
	HeadYaw              float64
	YawVelocityRadPerSec float64
	LeadSeconds          [2]float64
	FOV                  [2]FOV
	Tracking             [2]TrackingSpec
	PersistenceSeconds   float64
	TrackingErrorArcMin  float64
}

func NominalFOV(eye EyeID) FOV {
	if eye == LeftEye {
		return FOV{Left: 1.08, Right: 0.78, Up: 0.78, Down: 0.78}
	}
	return FOV{Left: 0.78, Right: 1.08, Up: 0.78, Down: 0.78}
}

func StrongAsymmetricFOV(eye EyeID) FOV {
	if eye == LeftEye {
		return FOV{Left: 0.62, Right: 1.24, Up: 0.72, Down: 0.84}
	}
	return FOV{Left: 1.22, Right: 0.64, Up: 0.82, Down: 0.70}
}

func TrackedSpec() TrackingSpec {
	return TrackingSpec{State: "tracked", Confidence: 0.99, SafeInsetPx: 18, Fallback: "none"}
}

func PredictedSpec() TrackingSpec {
	return TrackingSpec{State: "predicted", Confidence: 0.86, SafeInsetPx: 24, Fallback: "velocity_prediction"}
}

func LostFallbackSpec() TrackingSpec {
	return TrackingSpec{
		State:       "lost_fallback",
		Confidence:  0.22,
		SafeInsetPx: 42,
		Fallback:    "last_known_ipd_plus_head_pose_safe_area",
	}
}

func Scenarios() []Scenario {
	return []Scenario{
		{
			ID:                  "baseline",
			TitleZH:             "基线",
			LeadSeconds:         [2]float64{0, 0},
			FOV:                 [2]FOV{NominalFOV(LeftEye), NominalFOV(RightEye)},
			Tracking:            [2]TrackingSpec{TrackedSpec(), TrackedSpec()},
			PersistenceSeconds:  0.0012,
			TrackingErrorArcMin: 0.8,
		},
		{
			ID:                   "fast_head_turn",
			TitleZH:              "快速转头",
			HeadYaw:              0.020,
			YawVelocityRadPerSec: 5.4,
			LeadSeconds:          [2]float64{0.015, 0.015},
			FOV:                  [2]FOV{NominalFOV(LeftEye), NominalFOV(RightEye)},
			Tracking:             [2]TrackingSpec{PredictedSpec(), PredictedSpec()},
			PersistenceSeconds:   0.0010,
			TrackingErrorArcMin:  5.5,
		},
		{
			ID:                  "asymmetric_fov",
			TitleZH:             "非对称 FOV",
			LeadSeconds:         [2]float64{0, 0},
			FOV:                 [2]FOV{StrongAsymmetricFOV(LeftEye), StrongAsymmetricFOV(RightEye)},
			Tracking:            [2]TrackingSpec{TrackedSpec(), TrackedSpec()},
			PersistenceSeconds:  0.0012,
			TrackingErrorArcMin: 1.0,
		},
		{
			ID:                   "mono_tracking_lost",
			TitleZH:              "单眼追踪失效",
			HeadYaw:              0.010,
			YawVelocityRadPerSec: 1.1,
			LeadSeconds:          [2]float64{0.012, 0.012},
			FOV:                  [2]FOV{NominalFOV(LeftEye), NominalFOV(RightEye)},
			Tracking:             [2]TrackingSpec{TrackedSpec(), LostFallbackSpec()},
			PersistenceSeconds:   0.0014,
			TrackingErrorArcMin:  11.0,
		},
		{
			ID:                   "combined_stress",
			TitleZH:              "快速转头+非对称FOV+单眼失效",
			HeadYaw:              0.020,
			YawVelocityRadPerSec: 5.4,
			LeadSeconds:          [2]float64{0.015, 0.015},
			FOV:                  [2]FOV{StrongAsymmetricFOV(LeftEye), StrongAsymmetricFOV(RightEye)},
			Tracking:             [2]TrackingSpec{PredictedSpec(), LostFallbackSpec()},
			PersistenceSeconds:   0.0010,
			TrackingErrorArcMin:  12.0,
		},
	}
}

// BuildEyes predicts each eye independently. A lost eye still receives its own
// last-known IPD offset, FOV and head pose; it is never populated by copying
// the other eye's screen coordinate.
func (s Scenario) BuildEyes(width, height int, ipd float64) [2]Eye {
	ids := [2]EyeID{LeftEye, RightEye}
	var eyes [2]Eye
	for i, id := range ids {
		lead := s.LeadSeconds[i]
		yaw := s.HeadYaw + s.YawVelocityRadPerSec*lead
		head := Pose{Pos: Vec3{}, Yaw: yaw}
		eyeSign := -1.0
		if id == RightEye {
			eyeSign = 1.0
		}
		offset := RotateY(Vec3{X: eyeSign * ipd * 0.5}, yaw)
		spec := s.Tracking[i]
		fallback := spec.Fallback
		if spec.State == "" {
			spec = TrackedSpec()
			fallback = spec.Fallback
		}
		eyes[i] = Eye{
			ID:      id,
			Width:   width,
			Height:  height,
			IPD:     ipd,
			Pose:    Pose{Pos: head.Pos.Add(offset), Yaw: yaw},
			FOV:     s.FOV[i],
			LeadSec: lead,
			Tracking: TrackingState{
				State:       spec.State,
				Confidence:  spec.Confidence,
				SafeInsetPx: spec.SafeInsetPx,
				Fallback:    fallback,
			},
		}
	}
	return eyes
}

func arcMinutes(rad float64) float64 {
	return rad * 180 / math.Pi * 60
}

func degrees(rad float64) float64 {
	return rad * 180 / math.Pi
}

func formatCoord(p ScreenPoint) string {
	return fmt.Sprintf("(%.2f,%.2f)", p.X, p.Y)
}
