package main

import (
	"fmt"
	"image"
	"sort"
)

type Report struct {
	Schema         string          `json:"schema"`
	Program        ProgramReport   `json:"program"`
	Shared         SharedReport    `json:"shared"`
	Isolation      IsolationReport `json:"isolation"`
	Eyes           []EyeReport     `json:"eyes"`
	Pipeline       PipelineReport  `json:"pipeline"`
	FaultInjection FaultReport     `json:"fault_injection"`
}

type ProgramReport struct {
	ID                string        `json:"id"`
	EntryPoint        string        `json:"entry_point"`
	SourceHash        string        `json:"source_hash"`
	Source            string        `json:"source"`
	UniformLayoutHash string        `json:"uniform_layout_hash"`
	Layout            UniformLayout `json:"uniform_layout"`
}

type SharedReport struct {
	CodeAndLayoutShared bool     `json:"code_and_layout_shared"`
	StaticLayoutShared  bool     `json:"static_layout_shared"`
	DynamicStateShared  bool     `json:"dynamic_state_shared"`
	SameBaseMaterial    bool     `json:"same_base_material"`
	Viewport            [2]int   `json:"viewport"`
	StaticLayoutHash    string   `json:"static_layout_hash"`
	PanelCount          int      `json:"panel_count"`
	Evidence            []string `json:"evidence"`
}

type IsolationReport struct {
	Clean                     bool     `json:"clean"`
	CamerasDistinct           bool     `json:"cameras_distinct"`
	MaterialStatesDistinct    bool     `json:"material_states_distinct"`
	AlphaOverridesDistinct    bool     `json:"alpha_overrides_distinct"`
	MaterialConstantsDistinct bool     `json:"material_constants_distinct"`
	EyeConstantsDistinct      bool     `json:"eye_constants_distinct"`
	OtherDynamicDistinct      bool     `json:"other_dynamic_values_distinct"`
	SnapshotsStable           bool     `json:"snapshots_stable"`
	IsolatedMatchesReference  bool     `json:"isolated_matches_reference"`
	FrameHashesDiffer         bool     `json:"frame_hashes_differ"`
	Evidence                  []string `json:"evidence"`
}

type EyeReport struct {
	Eye                Eye            `json:"eye"`
	ParameterSnapshot  EyeState       `json:"parameter_snapshot"`
	SnapshotHash       string         `json:"snapshot_hash"`
	ReferenceHash      string         `json:"reference_hash"`
	ImageHash          string         `json:"image_hash"`
	ImageFile          string         `json:"image_file"`
	FirstPollutedPixel *Pixel         `json:"first_polluted_pixel"`
	PollutedPixels     int            `json:"polluted_pixels"`
	FaultProbe         FaultEyeReport `json:"fault_probe"`
}

type PipelineReport struct {
	Config          PipelineConfig       `json:"config"`
	BatchCount      int                  `json:"batch_count"`
	Batches         []BatchRecord        `json:"batches"`
	Cache           CacheStats           `json:"cache"`
	CacheEntries    []CacheSnapshotEntry `json:"cache_entries"`
	SnapshotsStable bool                 `json:"snapshots_stable"`
}

type FaultReport struct {
	Enabled             bool             `json:"enabled"`
	Description         string           `json:"description"`
	BrokenKeyFields     []string         `json:"broken_key_fields"`
	OmittedDynamicKeys  []string         `json:"omitted_dynamic_keys"`
	Cache               CacheStats       `json:"cache"`
	HashMismatchBatches int              `json:"hash_mismatch_batches"`
	Eyes                []FaultEyeReport `json:"eyes"`
	AnyPollution        bool             `json:"any_pollution"`
}

type FaultEyeReport struct {
	Eye                Eye       `json:"eye"`
	ReferenceHash      string    `json:"reference_hash"`
	ImageHash          string    `json:"image_hash"`
	ImageFile          string    `json:"image_file"`
	DiffFile           string    `json:"diff_file"`
	FirstPollutedPixel *Pixel    `json:"first_polluted_pixel"`
	PollutedPixels     int       `json:"polluted_pixels"`
	ReferenceRGBA      *[4]uint8 `json:"reference_rgba"`
	ActualRGBA         *[4]uint8 `json:"actual_rgba"`
}

func buildReport(
	scene *Scene,
	frames []EyeState,
	reference map[Eye]*image.RGBA,
	isolated *RunResult,
	fault *RunResult,
) (Report, error) {
	if isolated == nil || fault == nil {
		return Report{}, fmt.Errorf("both isolated and fault runs are required")
	}
	if len(frames) != 2 {
		return Report{}, fmt.Errorf("two eye snapshots are required")
	}

	shared := buildSharedReport(scene, frames)
	isolation, err := buildIsolationReport(frames, reference, isolated)
	if err != nil {
		return Report{}, err
	}
	eyes, err := buildEyeReports(reference, isolated, fault)
	if err != nil {
		return Report{}, err
	}
	faultReport := buildFaultReport(reference, fault)
	return Report{
		Schema:    "eye-dynamic-isolation/v1",
		Program:   buildProgramReport(scene),
		Shared:    shared,
		Isolation: isolation,
		Eyes:      eyes,
		Pipeline: PipelineReport{
			Config:          isolated.Config,
			BatchCount:      len(isolated.Batches),
			Batches:         isolated.Batches,
			Cache:           isolated.Cache,
			CacheEntries:    isolated.CacheEntries,
			SnapshotsStable: isolated.SnapshotsStable,
		},
		FaultInjection: faultReport,
	}, nil
}

func buildProgramReport(scene *Scene) ProgramReport {
	return ProgramReport{
		ID:                scene.Program.ID,
		EntryPoint:        scene.Program.EntryPoint,
		SourceHash:        scene.Program.SourceHash,
		Source:            scene.Program.Source,
		UniformLayoutHash: scene.Program.Layout.Hash,
		Layout:            scene.Program.Layout,
	}
}

func buildSharedReport(scene *Scene, frames []EyeState) SharedReport {
	sameBase := true
	base := ""
	for _, frame := range frames {
		for _, material := range frame.Materials {
			if base == "" {
				base = material.BaseMaterial
			}
			if material.BaseMaterial != base {
				sameBase = false
			}
		}
	}
	return SharedReport{
		CodeAndLayoutShared: true,
		StaticLayoutShared:  true,
		DynamicStateShared:  false,
		SameBaseMaterial:    sameBase,
		Viewport:            [2]int{scene.Layout.Width, scene.Layout.Height},
		StaticLayoutHash:    scene.Layout.LayoutHash,
		PanelCount:          len(scene.Layout.Panels),
		Evidence: []string{
			"one immutable *ShaderProgram is used for every draw of both eyes",
			"one immutable *StaticLayout owns all panel positions and viewport size",
			"each draw receives a per-eye uniform block interpreted with the shared layout",
		},
	}
}

func buildIsolationReport(
	frames []EyeState,
	reference map[Eye]*image.RGBA,
	run *RunResult,
) (IsolationReport, error) {
	left, right, err := orderedFrames(frames)
	if err != nil {
		return IsolationReport{}, err
	}
	materialDistinct, alphaDistinct, constantsDistinct := compareMaterials(left, right)
	otherDistinct := left.SnapshotID != right.SnapshotID &&
		left.ViewportSeed != right.ViewportSeed &&
		left.Jitter != right.Jitter
	camerasDistinct := hashJSON(left.Camera) != hashJSON(right.Camera)
	eyeConstantsDistinct := hashJSON(left.Constants) != hashJSON(right.Constants)
	matchesReference := imagesEqual(reference[EyeLeft], run.Images[EyeLeft]) &&
		imagesEqual(reference[EyeRight], run.Images[EyeRight])
	hashDiffer := run.Hashes[EyeLeft] != run.Hashes[EyeRight]
	clean := matchesReference && run.SnapshotsStable
	evidence := []string{
		"eye states are deep-cloned before the engine sees them",
		"isolated cache keys include eye, instance and dynamic_hash",
		"left and right use different camera, material and eye-level constants",
		"every batch reads back the dynamic hash it prepared",
	}
	if matchesReference {
		evidence = append(evidence, "both isolated images match the uncached per-eye reference byte-for-byte")
	}
	return IsolationReport{
		Clean:                     clean,
		CamerasDistinct:           camerasDistinct,
		MaterialStatesDistinct:    materialDistinct,
		AlphaOverridesDistinct:    alphaDistinct,
		MaterialConstantsDistinct: constantsDistinct,
		EyeConstantsDistinct:      eyeConstantsDistinct,
		OtherDynamicDistinct:      otherDistinct,
		SnapshotsStable:           run.SnapshotsStable,
		IsolatedMatchesReference:  matchesReference,
		FrameHashesDiffer:         hashDiffer,
		Evidence:                  evidence,
	}, nil
}

func buildEyeReports(
	reference map[Eye]*image.RGBA,
	isolated *RunResult,
	fault *RunResult,
) ([]EyeReport, error) {
	faultByEye := make(map[Eye]FaultEyeReport, len(fault.Images))
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		faultByEye[eye] = diffEye(eye, reference[eye], fault.Images[eye], faultFilename(eye))
	}
	reports := make([]EyeReport, 0, len(isolated.Snapshots))
	for _, frame := range isolated.Snapshots {
		ref := reference[frame.Eye]
		actual := isolated.Images[frame.Eye]
		if ref == nil || actual == nil {
			return nil, fmt.Errorf("missing image for %s", frame.Eye)
		}
		reports = append(reports, EyeReport{
			Eye:                frame.Eye,
			ParameterSnapshot:  frame,
			SnapshotHash:       frame.SnapshotHash(),
			ReferenceHash:      HashImage(ref),
			ImageHash:          HashImage(actual),
			ImageFile:          imageFilename(frame.Eye),
			FirstPollutedPixel: FirstPollutedPixel(ref, actual),
			PollutedPixels:     CountPollutedPixels(ref, actual),
			FaultProbe:         faultByEye[frame.Eye],
		})
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].Eye < reports[j].Eye })
	return reports, nil
}

func buildFaultReport(reference map[Eye]*image.RGBA, fault *RunResult) FaultReport {
	eyes := make([]FaultEyeReport, 0, 2)
	anyPollution := false
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		result := diffEye(eye, reference[eye], fault.Images[eye], faultFilename(eye))
		if result.PollutedPixels > 0 {
			anyPollution = true
		}
		eyes = append(eyes, result)
	}
	mismatches := 0
	for _, batch := range fault.Batches {
		if !batch.HashMatched {
			mismatches++
		}
	}
	return FaultReport{
		Enabled:             true,
		Description:         "cache key aliases every eye and panel sharing one base material",
		BrokenKeyFields:     []string{"program_id", "layout_hash", "base_material"},
		OmittedDynamicKeys:  []string{"eye", "instance", "dynamic_hash"},
		Cache:               fault.Cache,
		HashMismatchBatches: mismatches,
		Eyes:                eyes,
		AnyPollution:        anyPollution,
	}
}

func diffEye(eye Eye, reference, actual *image.RGBA, imageFile string) FaultEyeReport {
	first := FirstPollutedPixel(reference, actual)
	return FaultEyeReport{
		Eye:                eye,
		ReferenceHash:      HashImage(reference),
		ImageHash:          HashImage(actual),
		ImageFile:          imageFile,
		DiffFile:           diffFilename(eye),
		FirstPollutedPixel: first,
		PollutedPixels:     CountPollutedPixels(reference, actual),
		ReferenceRGBA:      PixelRGBA(reference, first),
		ActualRGBA:         PixelRGBA(actual, first),
	}
}

func orderedFrames(frames []EyeState) (EyeState, EyeState, error) {
	var left, right EyeState
	var foundLeft, foundRight bool
	for _, frame := range frames {
		switch frame.Eye {
		case EyeLeft:
			left, foundLeft = frame, true
		case EyeRight:
			right, foundRight = frame, true
		}
	}
	if !foundLeft || !foundRight {
		return EyeState{}, EyeState{}, fmt.Errorf("left and right snapshots are required")
	}
	return left, right, nil
}

func compareMaterials(left, right EyeState) (bool, bool, bool) {
	statesDistinct, alphaDistinct, constantsDistinct := true, true, true
	for _, leftMaterial := range left.Materials {
		rightMaterial, ok := findMaterial(right, leftMaterial.Instance)
		if !ok {
			return false, false, false
		}
		if leftMaterial.MaterialHash() == rightMaterial.MaterialHash() {
			statesDistinct = false
		}
		if floatPtrEqual(leftMaterial.AlphaOverride, rightMaterial.AlphaOverride) {
			alphaDistinct = false
		}
		if hashJSON(leftMaterial.Constants) == hashJSON(rightMaterial.Constants) {
			constantsDistinct = false
		}
	}
	return statesDistinct, alphaDistinct, constantsDistinct
}

func floatPtrEqual(left, right *float64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func imagesEqual(left, right *image.RGBA) bool {
	if left == nil || right == nil || left.Bounds() != right.Bounds() {
		return false
	}
	for i := range left.Pix {
		if left.Pix[i] != right.Pix[i] {
			return false
		}
	}
	return true
}

func imageFilename(eye Eye) string {
	return fmt.Sprintf("%s.png", eye)
}

func faultFilename(eye Eye) string {
	return fmt.Sprintf("fault-%s.png", eye)
}

func diffFilename(eye Eye) string {
	return fmt.Sprintf("fault-%s-diff.png", eye)
}
