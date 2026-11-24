package main

import (
	"errors"
	"fmt"
)

// ShaderProgram is immutable code plus the immutable uniform layout. Both eyes
// intentionally point at the same instance.
type ShaderProgram struct {
	ID         string        `json:"id"`
	EntryPoint string        `json:"entry_point"`
	Source     string        `json:"source"`
	SourceHash string        `json:"source_hash"`
	Layout     UniformLayout `json:"layout"`
}

type UniformField struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

// UniformLayout describes where dynamic values live in a prepared uniform
// block. The offsets are static and shared; the values written at those
// offsets are per-eye.
type UniformLayout struct {
	Version         string         `json:"version"`
	FloatCount      int            `json:"float_count"`
	CameraOffset    int            `json:"camera_offset"`
	CameraLength    int            `json:"camera_length"`
	MaterialOffset  int            `json:"material_offset"`
	MaterialLength  int            `json:"material_length"`
	EyeOffset       int            `json:"eye_offset"`
	EyeLength       int            `json:"eye_length"`
	MaterialFields  []UniformField `json:"material_fields"`
	EyeFields       []UniformField `json:"eye_fields"`
	EyeConstantKeys []string       `json:"eye_constant_keys"`
	Hash            string         `json:"hash"`
}

const defaultShaderSource = `#version 460
layout(std140) uniform EyeFrame {
    vec4 camera_position_eye;
    vec4 camera_angles_fov;
    vec4 material_roughness_alpha_gain_spec;
    vec4 material_tint_rgb_ring;
    vec4 eye_background_horizon_fog_vignette;
    vec4 eye_jitter_viewport_seed;
};
layout(location = 0) out vec4 outColor;

vec3 shade(vec3 base, vec3 world, float roughness, float specGain);
void main() { outColor = vec4(shade(vec3(0), vec3(0), 0.5, 0.5), 1.0); }`

var eyeConstantKeys = []string{
	"background_gain", "horizon", "fog", "vignette",
}

func newProgram() *ShaderProgram {
	layout := UniformLayout{
		Version:        "eye-frame.v1",
		FloatCount:     24,
		CameraOffset:   0,
		CameraLength:   8,
		MaterialOffset: 8,
		MaterialLength: 8,
		EyeOffset:      16,
		EyeLength:      8,
		MaterialFields: []UniformField{
			{Name: "roughness", Offset: 8, Length: 1},
			{Name: "alpha", Offset: 9, Length: 1},
			{Name: "gain", Offset: 10, Length: 1},
			{Name: "spec_gain", Offset: 11, Length: 1},
			{Name: "tint_rgb", Offset: 12, Length: 3},
			{Name: "ring", Offset: 15, Length: 1},
		},
		EyeFields: []UniformField{
			{Name: "background_gain", Offset: 16, Length: 1},
			{Name: "horizon", Offset: 17, Length: 1},
			{Name: "fog", Offset: 18, Length: 1},
			{Name: "vignette", Offset: 19, Length: 1},
			{Name: "jitter", Offset: 20, Length: 3},
			{Name: "viewport_seed", Offset: 23, Length: 1},
		},
		EyeConstantKeys: append([]string(nil), eyeConstantKeys...),
	}
	layout.Hash = hashJSON(layout)
	return &ShaderProgram{
		ID:         "shared.pbr.panel",
		EntryPoint: "main",
		Source:     defaultShaderSource,
		SourceHash: sha256Text(defaultShaderSource),
		Layout:     layout,
	}
}

// Panel is static world-space layout. It contains no per-eye values.
type Panel struct {
	Instance     string  `json:"instance"`
	BaseMaterial string  `json:"base_material"`
	Center       Vec3    `json:"center"`
	Radius       float64 `json:"radius"`
}

type StaticLayout struct {
	Version    string  `json:"version"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	PlaneZ     float64 `json:"plane_z"`
	Panels     []Panel `json:"panels"`
	LayoutHash string  `json:"layout_hash"`
}

func newStaticLayout() *StaticLayout {
	const base = "mat.shared.pbr"
	layout := &StaticLayout{
		Version: "stereo-panels.v1",
		Width:   160,
		Height:  90,
		PlaneZ:  0,
		Panels: []Panel{
			{Instance: "amber-panel", BaseMaterial: base, Center: Vec3{X: -0.72, Y: 0.30}, Radius: 0.27},
			{Instance: "teal-panel", BaseMaterial: base, Center: Vec3{X: -0.22, Y: -0.28}, Radius: 0.30},
			{Instance: "rose-panel", BaseMaterial: base, Center: Vec3{X: 0.28, Y: 0.25}, Radius: 0.25},
			{Instance: "violet-panel", BaseMaterial: base, Center: Vec3{X: 0.76, Y: -0.22}, Radius: 0.26},
		},
	}
	layout.LayoutHash = hashJSON(layout)
	return layout
}

// Scene owns the only program and layout instance used by both eyes. Nothing
// in this type can hold a camera or material override.
type Scene struct {
	Program *ShaderProgram
	Layout  *StaticLayout
}

func newScene() (*Scene, error) {
	scene := &Scene{Program: newProgram(), Layout: newStaticLayout()}
	if scene.Program.ID == "" || scene.Program.SourceHash == "" {
		return nil, errors.New("program is incomplete")
	}
	if scene.Layout.Width <= 0 || scene.Layout.Height <= 0 || len(scene.Layout.Panels) == 0 {
		return nil, errors.New("static layout is incomplete")
	}
	for _, panel := range scene.Layout.Panels {
		if panel.BaseMaterial != scene.Layout.Panels[0].BaseMaterial {
			return nil, errors.New("panels must use the same base material")
		}
	}
	return scene, nil
}

// defaultEyeStates creates two independent snapshots that deliberately share
// program, layout and base material names while differing in every dynamic
// value that matters to shading.
func defaultEyeStates(scene *Scene) ([]EyeState, error) {
	roughnessLeft := []float64{0.18, 0.42, 0.66, 0.29}
	roughnessRight := []float64{0.81, 0.57, 0.23, 0.73}
	alphaLeft := []float64{0.62, 0.78, 0.48, 0.91}
	alphaRight := []float64{0.88, 0.55, 0.72, 0.43}
	tint := [][3]float64{
		{0.92, 0.51, 0.16},
		{0.12, 0.68, 0.64},
		{0.86, 0.26, 0.47},
		{0.49, 0.34, 0.89},
	}
	ring := []float64{0.18, 0.46, 0.74, 0.33}
	ringRight := []float64{0.83, 0.61, 0.27, 0.69}

	states := []EyeState{
		{
			Eye:        EyeLeft,
			SnapshotID: 101,
			Camera: Camera{
				Position:          Vec3{X: -0.16, Y: 0.015, Z: 3.10},
				YawRadians:        0.035,
				PitchRadians:      -0.012,
				VerticalFOVDegree: 52.5,
				Exposure:          1.02,
				NearPlane:         0.05,
			},
			Constants: map[string]float64{
				"background_gain": 0.95,
				"horizon":         0.18,
				"fog":             0.020,
				"vignette":        0.22,
			},
			ViewportSeed: 23,
			Jitter:       Vec3{X: 0.25, Y: -0.10, Z: 0},
		},
		{
			Eye:        EyeRight,
			SnapshotID: 202,
			Camera: Camera{
				Position:          Vec3{X: 0.16, Y: -0.020, Z: 3.08},
				YawRadians:        -0.030,
				PitchRadians:      0.016,
				VerticalFOVDegree: 51.7,
				Exposure:          0.97,
				NearPlane:         0.05,
			},
			Constants: map[string]float64{
				"background_gain": 1.08,
				"horizon":         0.22,
				"fog":             0.035,
				"vignette":        0.16,
			},
			ViewportSeed: 29,
			Jitter:       Vec3{X: -0.20, Y: 0.15, Z: 0.05},
		},
	}

	for eyeIndex := range states {
		state := &states[eyeIndex]
		state.Materials = make([]MaterialState, len(scene.Layout.Panels))
		for panelIndex, panel := range scene.Layout.Panels {
			alpha := alphaLeft[panelIndex]
			roughness := roughnessLeft[panelIndex]
			panelRing := ring[panelIndex]
			gain := 0.92 + float64(panelIndex)*0.04
			spec := 0.66 + float64(panelIndex)*0.03
			if eyeIndex == 1 {
				alpha = alphaRight[panelIndex]
				roughness = roughnessRight[panelIndex]
				panelRing = ringRight[panelIndex]
				gain = 1.04 - float64(panelIndex)*0.03
				spec = 0.51 - float64(panelIndex)*0.04
			}
			state.Materials[panelIndex] = MaterialState{
				Instance:      panel.Instance,
				BaseMaterial:  panel.BaseMaterial,
				Roughness:     roughness,
				AlphaOverride: floatPtr(alpha),
				Constants: map[string]float64{
					"tint_r":     tint[panelIndex][0],
					"tint_g":     tint[panelIndex][1],
					"tint_b":     tint[panelIndex][2],
					"gain":       gain,
					"spec":       spec,
					"ring":       panelRing,
					"base_alpha": 1.0,
				},
			}
		}
		if err := state.Validate(); err != nil {
			return nil, err
		}
	}
	return states, nil
}

func floatPtr(value float64) *float64 {
	return &value
}

func sha256Text(value string) string {
	return hashBytes([]byte(value))
}

func validateScene(scene *Scene) error {
	if scene == nil || scene.Program == nil || scene.Layout == nil {
		return errors.New("scene, program and layout are required")
	}
	if scene.Layout.Width <= 0 || scene.Layout.Height <= 0 {
		return fmt.Errorf("invalid viewport %dx%d", scene.Layout.Width, scene.Layout.Height)
	}
	if scene.Program.Layout.FloatCount <= 0 {
		return errors.New("uniform layout is empty")
	}
	return nil
}
