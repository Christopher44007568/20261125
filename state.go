package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// Eye identifies one half of a stereo pair. Dynamic state is never indexed by
// this value alone; caches must also include the dynamic snapshot fingerprint.
type Eye string

const (
	EyeLeft  Eye = "left"
	EyeRight Eye = "right"
)

func (e Eye) Valid() bool {
	return e == EyeLeft || e == EyeRight
}

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{X: v.X + o.X, Y: v.Y + o.Y, Z: v.Z + o.Z}
}

func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{X: v.X - o.X, Y: v.Y - o.Y, Z: v.Z - o.Z}
}

func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		X: v.Y*o.Z - v.Z*o.Y,
		Y: v.Z*o.X - v.X*o.Z,
		Z: v.X*o.Y - v.Y*o.X,
	}
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

func (v Vec3) Normalize() Vec3 {
	n := v.Length()
	if n == 0 {
		return Vec3{}
	}
	return v.Scale(1 / n)
}

// Camera is entirely per-eye. It is copied into every draw snapshot so a
// shuffled batch can never observe the camera of the other eye.
type Camera struct {
	Position          Vec3    `json:"position"`
	YawRadians        float64 `json:"yaw_radians"`
	PitchRadians      float64 `json:"pitch_radians"`
	VerticalFOVDegree float64 `json:"vertical_fov_degrees"`
	Exposure          float64 `json:"exposure"`
	NearPlane         float64 `json:"near_plane"`
}

func (c Camera) Validate() error {
	if !finite(c.Position.X) || !finite(c.Position.Y) || !finite(c.Position.Z) {
		return errors.New("camera position is not finite")
	}
	if !finite(c.YawRadians) || !finite(c.PitchRadians) {
		return errors.New("camera orientation is not finite")
	}
	if c.VerticalFOVDegree <= 0 || c.VerticalFOVDegree >= 180 {
		return fmt.Errorf("camera vertical FOV must be in (0, 180), got %g", c.VerticalFOVDegree)
	}
	if c.Exposure <= 0 || !finite(c.Exposure) {
		return fmt.Errorf("camera exposure must be positive, got %g", c.Exposure)
	}
	if c.NearPlane <= 0 || !finite(c.NearPlane) {
		return fmt.Errorf("camera near plane must be positive, got %g", c.NearPlane)
	}
	return nil
}

// MaterialState is the mutable material data for one eye and one static
// layout panel. BaseMaterial may be shared, but every field below belongs to
// exactly one eye snapshot.
type MaterialState struct {
	Instance      string             `json:"instance"`
	BaseMaterial  string             `json:"base_material"`
	Roughness     float64            `json:"roughness"`
	AlphaOverride *float64           `json:"alpha_override"`
	Constants     map[string]float64 `json:"constants"`
}

func (m MaterialState) Clone() MaterialState {
	out := m
	if m.AlphaOverride != nil {
		alpha := *m.AlphaOverride
		out.AlphaOverride = &alpha
	}
	out.Constants = cloneConstants(m.Constants)
	return out
}

func (m MaterialState) Validate() error {
	if m.Instance == "" {
		return errors.New("material instance is empty")
	}
	if m.BaseMaterial == "" {
		return fmt.Errorf("material %q has no base material", m.Instance)
	}
	if m.Roughness < 0 || m.Roughness > 1 || !finite(m.Roughness) {
		return fmt.Errorf("material %q roughness must be in [0,1], got %g", m.Instance, m.Roughness)
	}
	if m.AlphaOverride != nil {
		if *m.AlphaOverride < 0 || *m.AlphaOverride > 1 || !finite(*m.AlphaOverride) {
			return fmt.Errorf("material %q alpha override must be in [0,1], got %g", m.Instance, *m.AlphaOverride)
		}
	}
	for key, value := range m.Constants {
		if !finite(value) {
			return fmt.Errorf("material %q constant %q is not finite", m.Instance, key)
		}
	}
	return nil
}

// EyeState is an immutable-by-convention snapshot of all per-eye dynamic
// inputs. Clone performs a deep copy, including AlphaOverride and Constants.
type EyeState struct {
	Eye          Eye                `json:"eye"`
	SnapshotID   uint64             `json:"snapshot_id"`
	Camera       Camera             `json:"camera"`
	Materials    []MaterialState    `json:"materials"`
	Constants    map[string]float64 `json:"constants"`
	ViewportSeed uint64             `json:"viewport_seed"`
	Jitter       Vec3               `json:"jitter"`
}

func (e EyeState) Clone() EyeState {
	out := e
	out.Materials = make([]MaterialState, len(e.Materials))
	for i := range e.Materials {
		out.Materials[i] = e.Materials[i].Clone()
	}
	out.Constants = cloneConstants(e.Constants)
	return out
}

func (e EyeState) Validate() error {
	if !e.Eye.Valid() {
		return fmt.Errorf("invalid eye %q", e.Eye)
	}
	if err := e.Camera.Validate(); err != nil {
		return fmt.Errorf("%s camera: %w", e.Eye, err)
	}
	if len(e.Materials) == 0 {
		return fmt.Errorf("%s has no materials", e.Eye)
	}
	seen := make(map[string]struct{}, len(e.Materials))
	for _, material := range e.Materials {
		if err := material.Validate(); err != nil {
			return fmt.Errorf("%s: %w", e.Eye, err)
		}
		if _, ok := seen[material.Instance]; ok {
			return fmt.Errorf("%s has duplicate material instance %q", e.Eye, material.Instance)
		}
		seen[material.Instance] = struct{}{}
	}
	for key, value := range e.Constants {
		if !finite(value) {
			return fmt.Errorf("%s constant %q is not finite", e.Eye, key)
		}
	}
	return nil
}

// SnapshotHash fingerprints every dynamic field on this eye.
func (e EyeState) SnapshotHash() string {
	return hashJSON(e)
}

// MaterialHash fingerprints only one material instance. The caller should
// still include the eye and camera hash when preparing a cache key.
func (m MaterialState) MaterialHash() string {
	return hashJSON(m)
}

func cloneConstants(in map[string]float64) map[string]float64 {
	if in == nil {
		return nil
	}
	out := make(map[string]float64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func hashJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("hash JSON: %v", err))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
