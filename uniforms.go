package main

import (
	"fmt"
	"math"
)

// PreparedUniforms contains only per-draw dynamic values. The program and
// layout used to interpret these values are immutable and shared.
type PreparedUniforms struct {
	ProgramID    string    `json:"program_id"`
	LayoutHash   string    `json:"layout_hash"`
	Eye          Eye       `json:"eye"`
	Instance     string    `json:"instance"`
	BaseMaterial string    `json:"base_material"`
	DynamicHash  string    `json:"dynamic_hash"`
	Values       []float64 `json:"values"`
}

type CacheKey struct {
	ProgramID   string `json:"program_id"`
	LayoutHash  string `json:"layout_hash"`
	Eye         Eye    `json:"eye"`
	Instance    string `json:"instance"`
	DynamicHash string `json:"dynamic_hash"`
}

func correctCacheKey(prepared *PreparedUniforms) CacheKey {
	return CacheKey{
		ProgramID:   prepared.ProgramID,
		LayoutHash:  prepared.LayoutHash,
		Eye:         prepared.Eye,
		Instance:    prepared.Instance,
		DynamicHash: prepared.DynamicHash,
	}
}

// faultCacheKey deliberately omits eye, instance and dynamic fingerprint. It
// is used only by the regression probe to demonstrate how a program/material
// key can accidentally alias unrelated dynamic state.
func faultCacheKey(prepared *PreparedUniforms) CacheKey {
	return CacheKey{
		ProgramID:  prepared.ProgramID,
		LayoutHash: prepared.LayoutHash,
		Instance:   prepared.BaseMaterial,
	}
}

func prepareDraw(scene *Scene, eye EyeState, material MaterialState) (*PreparedUniforms, error) {
	if err := validateScene(scene); err != nil {
		return nil, err
	}
	if !eye.Eye.Valid() {
		return nil, fmt.Errorf("invalid eye %q", eye.Eye)
	}
	if err := eye.Camera.Validate(); err != nil {
		return nil, err
	}
	if err := material.Validate(); err != nil {
		return nil, err
	}

	layout := scene.Program.Layout
	values := make([]float64, layout.FloatCount)
	camera := values[layout.CameraOffset : layout.CameraOffset+layout.CameraLength]
	camera[0] = eye.Camera.Position.X
	camera[1] = eye.Camera.Position.Y
	camera[2] = eye.Camera.Position.Z
	camera[3] = eye.Camera.YawRadians
	camera[4] = eye.Camera.PitchRadians
	camera[5] = eye.Camera.VerticalFOVDegree * math.Pi / 180
	camera[6] = eye.Camera.Exposure
	camera[7] = eye.Camera.NearPlane

	materialBlock := values[layout.MaterialOffset : layout.MaterialOffset+layout.MaterialLength]
	materialBlock[0] = material.Roughness
	materialBlock[1] = effectiveAlpha(material)
	materialBlock[2] = material.Constants["gain"]
	materialBlock[3] = material.Constants["spec"]
	materialBlock[4] = material.Constants["tint_r"]
	materialBlock[5] = material.Constants["tint_g"]
	materialBlock[6] = material.Constants["tint_b"]
	materialBlock[7] = material.Constants["ring"]

	eyeBlock := values[layout.EyeOffset : layout.EyeOffset+layout.EyeLength]
	eyeBlock[0] = eye.Constants["background_gain"]
	eyeBlock[1] = eye.Constants["horizon"]
	eyeBlock[2] = eye.Constants["fog"]
	eyeBlock[3] = eye.Constants["vignette"]
	eyeBlock[4] = eye.Jitter.X
	eyeBlock[5] = eye.Jitter.Y
	eyeBlock[6] = eye.Jitter.Z
	eyeBlock[7] = float64(eye.ViewportSeed)

	payload := struct {
		Eye          Eye                `json:"eye"`
		Camera       Camera             `json:"camera"`
		Material     MaterialState      `json:"material"`
		Constants    map[string]float64 `json:"constants"`
		Jitter       Vec3               `json:"jitter"`
		ViewportSeed uint64             `json:"viewport_seed"`
	}{
		Eye:          eye.Eye,
		Camera:       eye.Camera,
		Material:     material.Clone(),
		Constants:    cloneConstants(eye.Constants),
		Jitter:       eye.Jitter,
		ViewportSeed: eye.ViewportSeed,
	}

	return &PreparedUniforms{
		ProgramID:    scene.Program.ID,
		LayoutHash:   scene.Program.Layout.Hash,
		Eye:          eye.Eye,
		Instance:     material.Instance,
		BaseMaterial: material.BaseMaterial,
		DynamicHash:  hashJSON(payload),
		Values:       values,
	}, nil
}

type DecodedUniforms struct {
	Camera       Camera
	Roughness    float64
	Alpha        float64
	Gain         float64
	SpecGain     float64
	Tint         Vec3
	Ring         float64
	EyeConstants map[string]float64
	Jitter       Vec3
	ViewportSeed uint64
}

func decodeUniforms(scene *Scene, prepared *PreparedUniforms) (DecodedUniforms, error) {
	if prepared == nil {
		return DecodedUniforms{}, fmt.Errorf("nil prepared uniforms")
	}
	layout := scene.Program.Layout
	if prepared.LayoutHash != layout.Hash ||
		len(prepared.Values) < layout.CameraOffset+layout.CameraLength ||
		len(prepared.Values) < layout.MaterialOffset+layout.MaterialLength ||
		len(prepared.Values) < layout.EyeOffset+layout.EyeLength {
		return DecodedUniforms{}, fmt.Errorf("uniform block does not match shared layout %s", layout.Hash)
	}
	camera := prepared.Values[layout.CameraOffset : layout.CameraOffset+layout.CameraLength]
	material := prepared.Values[layout.MaterialOffset : layout.MaterialOffset+layout.MaterialLength]
	eye := prepared.Values[layout.EyeOffset : layout.EyeOffset+layout.EyeLength]
	return DecodedUniforms{
		Camera: Camera{
			Position:          Vec3{X: camera[0], Y: camera[1], Z: camera[2]},
			YawRadians:        camera[3],
			PitchRadians:      camera[4],
			VerticalFOVDegree: camera[5] * 180 / math.Pi,
			Exposure:          camera[6],
			NearPlane:         camera[7],
		},
		Roughness: material[0],
		Alpha:     material[1],
		Gain:      material[2],
		SpecGain:  material[3],
		Tint:      Vec3{X: material[4], Y: material[5], Z: material[6]},
		Ring:      material[7],
		EyeConstants: map[string]float64{
			"background_gain": eye[0],
			"horizon":         eye[1],
			"fog":             eye[2],
			"vignette":        eye[3],
		},
		Jitter:       Vec3{X: eye[4], Y: eye[5], Z: eye[6]},
		ViewportSeed: uint64(eye[7]),
	}, nil
}

func effectiveAlpha(material MaterialState) float64 {
	if material.AlphaOverride != nil {
		return *material.AlphaOverride
	}
	return material.Constants["base_alpha"]
}
