package glass

import (
	"fmt"
	"math"
	"sort"
)

type Eye string

const (
	EyeLeft  Eye = "left"
	EyeRight Eye = "right"
)

var Eyes = []Eye{EyeLeft, EyeRight}

type SceneKind string

const (
	SceneIntersection   SceneKind = "intersection"
	SceneNearCoplanar   SceneKind = "near-coplanar"
	SceneSharedMaterial SceneKind = "shared-material"
)

type RGB struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
}

type PremultipliedRGBA struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
	A float64 `json:"a"`
}

type Material struct {
	ID    string `json:"id"`
	Color RGB    `json:"color"`
}

type Layer struct {
	ID         string          `json:"id"`
	MaterialID string          `json:"material_id"`
	Depth      map[Eye]float64 `json:"depth"`
	Opacity    float64         `json:"opacity"`
}

type Scene struct {
	Kind      SceneKind  `json:"kind"`
	Materials []Material `json:"materials"`
	Layers    []Layer    `json:"layers"`
}

type Budget struct {
	Left  int `json:"left"`
	Right int `json:"right"`
}

func (b Budget) For(eye Eye) int {
	if eye == EyeLeft {
		return b.Left
	}
	return b.Right
}

type Options struct {
	DepthEpsilon float64           `json:"depth_epsilon"`
	Background   PremultipliedRGBA `json:"background"`
}

const DefaultDepthEpsilon = 1e-6

func (s Scene) Validate() error {
	if len(s.Layers) == 0 {
		return fmt.Errorf("scene has no layers")
	}

	materials := make(map[string]Material, len(s.Materials))
	for i, material := range s.Materials {
		if material.ID == "" {
			return fmt.Errorf("material %d has an empty id", i)
		}
		if _, exists := materials[material.ID]; exists {
			return fmt.Errorf("duplicate material id %q", material.ID)
		}
		if err := validateUnitRange("material "+material.ID+" red", material.Color.R); err != nil {
			return err
		}
		if err := validateUnitRange("material "+material.ID+" green", material.Color.G); err != nil {
			return err
		}
		if err := validateUnitRange("material "+material.ID+" blue", material.Color.B); err != nil {
			return err
		}
		materials[material.ID] = material
	}

	layers := make(map[string]struct{}, len(s.Layers))
	for i, layer := range s.Layers {
		if layer.ID == "" {
			return fmt.Errorf("layer %d has an empty id", i)
		}
		if _, exists := layers[layer.ID]; exists {
			return fmt.Errorf("duplicate layer id %q", layer.ID)
		}
		layers[layer.ID] = struct{}{}
		if _, exists := materials[layer.MaterialID]; !exists {
			return fmt.Errorf("layer %q references unknown material %q", layer.ID, layer.MaterialID)
		}
		if err := validateUnitRange("layer "+layer.ID+" opacity", layer.Opacity); err != nil {
			return err
		}
		for _, eye := range Eyes {
			depth, exists := layer.Depth[eye]
			if !exists {
				return fmt.Errorf("layer %q has no %s-eye depth", layer.ID, eye)
			}
			if math.IsNaN(depth) || math.IsInf(depth, 0) || depth < 0 {
				return fmt.Errorf("layer %q has invalid %s-eye depth %v", layer.ID, eye, depth)
			}
		}
	}

	return nil
}

func validateUnitRange(name string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
		return fmt.Errorf("%s must be in [0,1], got %v", name, value)
	}
	return nil
}

func validateBudget(budget Budget) error {
	if budget.Left < 0 {
		return fmt.Errorf("left-eye budget must be non-negative, got %d", budget.Left)
	}
	if budget.Right < 0 {
		return fmt.Errorf("right-eye budget must be non-negative, got %d", budget.Right)
	}
	return nil
}

func normalizeOptions(options Options) (Options, error) {
	if options.DepthEpsilon == 0 {
		options.DepthEpsilon = DefaultDepthEpsilon
	}
	if math.IsNaN(options.DepthEpsilon) || math.IsInf(options.DepthEpsilon, 0) || options.DepthEpsilon < 0 {
		return Options{}, fmt.Errorf("depth epsilon must be positive or zero for the default, got %v", options.DepthEpsilon)
	}
	for name, value := range map[string]float64{
		"background red":   options.Background.R,
		"background green": options.Background.G,
		"background blue":  options.Background.B,
		"background alpha": options.Background.A,
	} {
		if err := validateUnitRange(name, value); err != nil {
			return Options{}, err
		}
	}
	return options, nil
}

func canonicalLayerIDs(scene Scene) []string {
	ids := make([]string, 0, len(scene.Layers))
	for _, layer := range scene.Layers {
		ids = append(ids, layer.ID)
	}
	sort.Strings(ids)
	return ids
}
