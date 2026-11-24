package main

import (
	"fmt"
	"image"
	"math/rand"
)

type RunMode string

const (
	ModeIsolated RunMode = "isolated"
	ModeFault    RunMode = "fault-injection"
)

type PipelineConfig struct {
	Seed          int64 `json:"seed"`
	Rounds        int   `json:"rounds"`
	CacheCapacity int   `json:"cache_capacity"`
}

func defaultPipelineConfig() PipelineConfig {
	return PipelineConfig{Seed: 20261125, Rounds: 4, CacheCapacity: 3}
}

func (c PipelineConfig) Validate() error {
	if c.Rounds <= 0 {
		return fmt.Errorf("rounds must be positive, got %d", c.Rounds)
	}
	if c.CacheCapacity <= 0 {
		return fmt.Errorf("cache capacity must be positive, got %d", c.CacheCapacity)
	}
	return nil
}

type Batch struct {
	Sequence      int    `json:"sequence"`
	Round         int    `json:"round"`
	Eye           Eye    `json:"eye"`
	Instance      string `json:"instance"`
	MaterialIndex int    `json:"material_index"`
}

type BatchRecord struct {
	Sequence     int      `json:"sequence"`
	Round        int      `json:"round"`
	Eye          Eye      `json:"eye"`
	Instance     string   `json:"instance"`
	CacheHit     bool     `json:"cache_hit"`
	CacheKey     CacheKey `json:"cache_key"`
	PreparedEye  Eye      `json:"prepared_eye"`
	PreparedHash string   `json:"prepared_hash"`
	ExpectedHash string   `json:"expected_hash"`
	HashMatched  bool     `json:"hash_matched"`
}

type RunResult struct {
	Mode            RunMode              `json:"mode"`
	Config          PipelineConfig       `json:"config"`
	Snapshots       []EyeState           `json:"snapshots"`
	SnapshotHashes  map[Eye]string       `json:"snapshot_hashes"`
	SnapshotsStable bool                 `json:"snapshots_stable"`
	Hashes          map[Eye]string       `json:"image_hashes"`
	Images          map[Eye]*image.RGBA  `json:"-"`
	Batches         []BatchRecord        `json:"batches"`
	Cache           CacheStats           `json:"cache"`
	CacheEntries    []CacheSnapshotEntry `json:"cache_entries"`
}

func executePipeline(scene *Scene, input []EyeState, config PipelineConfig, mode RunMode) (*RunResult, error) {
	if err := validateScene(scene); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if mode != ModeIsolated && mode != ModeFault {
		return nil, fmt.Errorf("unknown run mode %q", mode)
	}

	frames, err := cloneAndValidateFrames(input)
	if err != nil {
		return nil, err
	}
	before := snapshotHashes(frames)
	batches, err := buildBatches(frames, scene, config)
	if err != nil {
		return nil, err
	}
	cache, err := newUniformCache(config.CacheCapacity)
	if err != nil {
		return nil, err
	}

	images := make(map[Eye]*image.RGBA, len(frames))

	frameByEye := make(map[Eye]EyeState, len(frames))
	for _, frame := range frames {
		frameByEye[frame.Eye] = frame
	}

	records := make([]BatchRecord, 0, len(batches))
	currentRound := -1
	for _, batch := range batches {
		if batch.Round != currentRound {
			images = newRenderTargets(scene, frames)
			currentRound = batch.Round
		}
		frame := frameByEye[batch.Eye]
		material := frame.Materials[batch.MaterialIndex]
		prepared, err := prepareDraw(scene, frame, material)
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", batch.Sequence, err)
		}
		key := correctCacheKey(prepared)
		if mode == ModeFault {
			key = faultCacheKey(prepared)
		}
		cached, hit := cache.get(key)
		if !hit {
			cache.put(key, prepared)
			cached = prepared
		}
		if mode == ModeIsolated && cached.DynamicHash != prepared.DynamicHash {
			return nil, fmt.Errorf("cache returned %s dynamic state for %s batch", cached.Eye, frame.Eye)
		}
		panel := scene.Layout.Panels[batch.MaterialIndex]
		if err := renderPanel(images[batch.Eye], scene, panel, cached); err != nil {
			return nil, fmt.Errorf("batch %d: %w", batch.Sequence, err)
		}
		records = append(records, BatchRecord{
			Sequence:     batch.Sequence,
			Round:        batch.Round,
			Eye:          batch.Eye,
			Instance:     batch.Instance,
			CacheHit:     hit,
			CacheKey:     key,
			PreparedEye:  cached.Eye,
			PreparedHash: cached.DynamicHash,
			ExpectedHash: prepared.DynamicHash,
			HashMatched:  cached.DynamicHash == prepared.DynamicHash,
		})
	}

	after := snapshotHashes(frames)
	hashes := make(map[Eye]string, len(images))
	for eye, img := range images {
		hashes[eye] = HashImage(img)
	}
	return &RunResult{
		Mode:            mode,
		Config:          config,
		Snapshots:       frames,
		SnapshotHashes:  before,
		SnapshotsStable: equalEyeHashes(before, after),
		Hashes:          hashes,
		Images:          images,
		Batches:         records,
		Cache:           cache.snapshotStats(),
		CacheEntries:    cache.snapshot(),
	}, nil
}

func newRenderTargets(scene *Scene, frames []EyeState) map[Eye]*image.RGBA {
	images := make(map[Eye]*image.RGBA, len(frames))
	for _, frame := range frames {
		img := newRGBA(scene.Layout.Width, scene.Layout.Height)
		renderBackground(img, frame.Camera, frame.Constants, frame.Jitter)
		images[frame.Eye] = img
	}
	return images
}

func cloneAndValidateFrames(input []EyeState) ([]EyeState, error) {
	if len(input) != 2 {
		return nil, fmt.Errorf("stereo render requires exactly two eyes, got %d", len(input))
	}
	seen := make(map[Eye]bool, 2)
	frames := make([]EyeState, len(input))
	for i := range input {
		frames[i] = input[i].Clone()
		if !frames[i].Eye.Valid() {
			return nil, fmt.Errorf("frame %d has invalid eye %q", i, frames[i].Eye)
		}
		if seen[frames[i].Eye] {
			return nil, fmt.Errorf("duplicate eye %q", frames[i].Eye)
		}
		seen[frames[i].Eye] = true
		if err := frames[i].Validate(); err != nil {
			return nil, err
		}
	}
	return frames, nil
}

func buildBatches(frames []EyeState, scene *Scene, config PipelineConfig) ([]Batch, error) {
	rng := rand.New(rand.NewSource(config.Seed))
	batches := make([]Batch, 0, len(frames)*len(scene.Layout.Panels)*config.Rounds)
	for round := 1; round <= config.Rounds; round++ {
		roundStart := len(batches)
		for _, frame := range frames {
			for _, panel := range scene.Layout.Panels {
				materialIndex := -1
				for index, material := range frame.Materials {
					if material.Instance == panel.Instance {
						materialIndex = index
						break
					}
				}
				if materialIndex < 0 {
					return nil, fmt.Errorf("%s has no state for panel %q", frame.Eye, panel.Instance)
				}
				batches = append(batches, Batch{
					Round:         round,
					Eye:           frame.Eye,
					Instance:      panel.Instance,
					MaterialIndex: materialIndex,
				})
			}
		}
		rng.Shuffle(len(batches)-roundStart, func(i, j int) {
			left := roundStart + i
			right := roundStart + j
			batches[left], batches[right] = batches[right], batches[left]
		})
	}
	for i := range batches {
		batches[i].Sequence = i
	}
	return batches, nil
}

func snapshotHashes(frames []EyeState) map[Eye]string {
	hashes := make(map[Eye]string, len(frames))
	for _, frame := range frames {
		hashes[frame.Eye] = frame.SnapshotHash()
	}
	return hashes
}

func equalEyeHashes(left, right map[Eye]string) bool {
	if len(left) != len(right) {
		return false
	}
	for eye, hash := range left {
		if right[eye] != hash {
			return false
		}
	}
	return true
}
