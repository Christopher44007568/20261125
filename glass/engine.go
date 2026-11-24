package glass

import (
	"fmt"
	"math"
	"sort"
)

const (
	DecisionKept       = "kept"
	DecisionOverBudget = "unprocessed-budget"
	DepthWriteReason   = "transparent-pass: depth writes disabled"
)

type LayerOrderEntry struct {
	ID                  string            `json:"id"`
	MaterialID          string            `json:"material_id"`
	Rank                int               `json:"rank"`
	Depth               float64           `json:"depth"`
	DepthBucket         int64             `json:"depth_bucket"`
	Opacity             float64           `json:"opacity"`
	Decision            string            `json:"decision"`
	SourcePremultiplied PremultipliedRGBA `json:"source_premultiplied"`
}

type DepthWriteEntry struct {
	LayerID string `json:"layer_id"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

type EyeAudit struct {
	Eye               Eye               `json:"eye"`
	Budget            int               `json:"budget"`
	DepthOrder        []LayerOrderEntry `json:"depth_order"`
	KeptIDs           []string          `json:"kept_ids"`
	UnprocessedIDs    []string          `json:"unprocessed_ids"`
	ProcessedMask     string            `json:"processed_mask"`
	UnprocessedMask   string            `json:"unprocessed_mask"`
	RenderOrder       []string          `json:"render_order"`
	DepthWriteEnabled bool              `json:"depth_write_enabled"`
	DepthWriteReason  string            `json:"depth_write_reason"`
	DepthWrites       []DepthWriteEntry `json:"depth_writes"`
	Premultiplied     PremultipliedRGBA `json:"premultiplied"`
}

type Audit struct {
	Scene           SceneKind  `json:"scene"`
	DepthEpsilon    float64    `json:"depth_epsilon"`
	Budget          Budget     `json:"budget"`
	SubmissionOrder []string   `json:"submission_order"`
	MaskBitOrder    []string   `json:"mask_bit_order"`
	Eyes            []EyeAudit `json:"eyes"`
}

func AuditScene(scene Scene, budget Budget, options Options) (Audit, error) {
	if err := scene.Validate(); err != nil {
		return Audit{}, err
	}
	if err := validateBudget(budget); err != nil {
		return Audit{}, err
	}
	normalized, err := normalizeOptions(options)
	if err != nil {
		return Audit{}, err
	}

	materials := make(map[string]Material, len(scene.Materials))
	for _, material := range scene.Materials {
		materials[material.ID] = material
	}

	bitOrder := canonicalLayerIDs(scene)
	audit := Audit{
		Scene:           scene.Kind,
		DepthEpsilon:    normalized.DepthEpsilon,
		Budget:          budget,
		SubmissionOrder: layerIDs(scene.Layers),
		MaskBitOrder:    bitOrder,
		Eyes:            make([]EyeAudit, 0, len(Eyes)),
	}
	for _, eye := range Eyes {
		audit.Eyes = append(audit.Eyes, auditEye(scene, materials, bitOrder, budget.For(eye), eye, normalized))
	}
	return audit, nil
}

func auditEye(
	scene Scene,
	materials map[string]Material,
	bitOrder []string,
	budget int,
	eye Eye,
	options Options,
) EyeAudit {
	order := sortedForEye(scene.Layers, eye, options.DepthEpsilon)
	selected := selectedSet(order, budget)
	unprocessed := complementSet(bitOrder, selected)

	result := EyeAudit{
		Eye:               eye,
		Budget:            budget,
		DepthOrder:        make([]LayerOrderEntry, 0, len(order)),
		KeptIDs:           make([]string, 0, min(budget, len(order))),
		UnprocessedIDs:    make([]string, 0, len(order)-min(budget, len(order))),
		ProcessedMask:     encodeMask(bitOrder, selected),
		UnprocessedMask:   encodeMask(bitOrder, unprocessed),
		RenderOrder:       make([]string, 0, min(budget, len(order))),
		DepthWriteEnabled: false,
		DepthWriteReason:  DepthWriteReason,
		DepthWrites:       make([]DepthWriteEntry, 0, min(budget, len(order))),
	}

	for index, layer := range order {
		material := materials[layer.MaterialID]
		entry := LayerOrderEntry{
			ID:                  layer.ID,
			MaterialID:          layer.MaterialID,
			Rank:                index + 1,
			Depth:               layer.Depth[eye],
			DepthBucket:         depthBucket(layer.Depth[eye], options.DepthEpsilon),
			Opacity:             layer.Opacity,
			SourcePremultiplied: premultiply(material.Color, layer.Opacity),
		}
		if selected[layer.ID] {
			entry.Decision = DecisionKept
			result.KeptIDs = append(result.KeptIDs, layer.ID)
		} else {
			entry.Decision = DecisionOverBudget
			result.UnprocessedIDs = append(result.UnprocessedIDs, layer.ID)
		}
		result.DepthOrder = append(result.DepthOrder, entry)
	}

	for index := len(result.KeptIDs) - 1; index >= 0; index-- {
		id := result.KeptIDs[index]
		result.RenderOrder = append(result.RenderOrder, id)
		result.DepthWrites = append(result.DepthWrites, DepthWriteEntry{
			LayerID: id,
			Enabled: false,
			Reason:  DepthWriteReason,
		})
	}

	result.Premultiplied = composite(result.RenderOrder, order, materials, options.Background)
	return result
}

func sortedForEye(layers []Layer, eye Eye, epsilon float64) []Layer {
	order := append([]Layer(nil), layers...)
	sort.Slice(order, func(i, j int) bool {
		left := order[i]
		right := order[j]
		leftBucket := depthBucket(left.Depth[eye], epsilon)
		rightBucket := depthBucket(right.Depth[eye], epsilon)
		if leftBucket != rightBucket {
			return leftBucket < rightBucket
		}
		return left.ID < right.ID
	})
	return order
}

func depthBucket(depth, epsilon float64) int64 {
	bucket := math.Floor(depth/epsilon + 0.5)
	if bucket >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(bucket)
}

func premultiply(color RGB, opacity float64) PremultipliedRGBA {
	return PremultipliedRGBA{
		R: color.R * opacity,
		G: color.G * opacity,
		B: color.B * opacity,
		A: opacity,
	}
}

func composite(
	renderOrder []string,
	layers []Layer,
	materials map[string]Material,
	background PremultipliedRGBA,
) PremultipliedRGBA {
	byID := make(map[string]Layer, len(layers))
	for _, layer := range layers {
		byID[layer.ID] = layer
	}

	result := background
	for _, id := range renderOrder {
		layer := byID[id]
		source := premultiply(materials[layer.MaterialID].Color, layer.Opacity)
		remaining := 1 - source.A
		result.R = source.R + result.R*remaining
		result.G = source.G + result.G*remaining
		result.B = source.B + result.B*remaining
		result.A = source.A + result.A*remaining
	}
	return result
}

func layerIDs(layers []Layer) []string {
	ids := make([]string, 0, len(layers))
	for _, layer := range layers {
		ids = append(ids, layer.ID)
	}
	return ids
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func (a Audit) ValidateEvidence() error {
	if len(a.Eyes) != len(Eyes) {
		return fmt.Errorf("expected %d eye audits, got %d", len(Eyes), len(a.Eyes))
	}
	for _, eye := range a.Eyes {
		if len(eye.KeptIDs)+len(eye.UnprocessedIDs) != len(a.MaskBitOrder) {
			return fmt.Errorf("%s eye did not account for every submitted layer", eye.Eye)
		}
		if eye.DepthWriteEnabled {
			return fmt.Errorf("%s eye unexpectedly enabled transparent-pass depth writes", eye.Eye)
		}
	}
	return nil
}
