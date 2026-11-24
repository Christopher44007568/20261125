package stereorollback

import (
	"math"
)

// MaterialParams is the small material parameter block validated before draw.
type MaterialParams struct {
	Roughness float64    `json:"roughness"`
	Metallic  float64    `json:"metallic"`
	Exposure  float64    `json:"exposure"`
	Atlas     ResourceID `json:"atlas"`
}

// DefaultMaterialParams returns a known-good parameter block.
func DefaultMaterialParams(atlas ResourceID) MaterialParams {
	return MaterialParams{Roughness: 0.55, Metallic: 0.1, Exposure: 1.0, Atlas: atlas}
}

// Validate returns the offending field and ErrMaterialParameter when invalid.
func (m MaterialParams) Validate() (field string, err error) {
	bad := func(field string) (string, error) {
		return field, &StageError{Stage: StageMaterial, Fault: FaultMaterialParameter, Err: ErrMaterialParameter, Field: field}
	}
	if math.IsNaN(m.Roughness) || m.Roughness < 0 || m.Roughness > 1 {
		return bad("roughness")
	}
	if math.IsNaN(m.Metallic) || m.Metallic < 0 || m.Metallic > 1 {
		return bad("metallic")
	}
	if math.IsNaN(m.Exposure) || m.Exposure <= 0 || m.Exposure > 8 {
		return bad("exposure")
	}
	if m.Atlas == "" {
		return bad("atlas_binding")
	}
	return "", nil
}

// injectedBadMaterial returns a deterministic invalid block for injection.
func injectedBadMaterial(atlas ResourceID) MaterialParams {
	m := DefaultMaterialParams(atlas)
	m.Roughness = -0.25
	return m
}
