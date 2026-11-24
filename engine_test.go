package main

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func renderFixture(t *testing.T) (*Scene, []EyeState, map[Eye]*image.RGBA) {
	t.Helper()
	scene, err := newScene()
	if err != nil {
		t.Fatal(err)
	}
	frames, err := defaultEyeStates(scene)
	if err != nil {
		t.Fatal(err)
	}
	reference := make(map[Eye]*image.RGBA, len(frames))
	for _, frame := range frames {
		rendered, err := renderReference(scene, frame)
		if err != nil {
			t.Fatal(err)
		}
		reference[frame.Eye] = rendered
	}
	return scene, frames, reference
}

func TestShuffledCacheStressIsIsolated(t *testing.T) {
	scene, frames, reference := renderFixture(t)
	config := defaultPipelineConfig()
	run, err := executePipeline(scene, frames, config, ModeIsolated)
	if err != nil {
		t.Fatal(err)
	}
	if !run.SnapshotsStable {
		t.Fatal("dynamic snapshots changed during render")
	}
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		if !imagesEqual(reference[eye], run.Images[eye]) {
			t.Fatalf("%s isolated render differs from uncached reference", eye)
		}
		if run.Hashes[eye] != HashImage(reference[eye]) {
			t.Fatalf("%s image hash does not describe rendered pixels", eye)
		}
	}
	if run.Cache.Hits == 0 || run.Cache.Misses == 0 {
		t.Fatalf("expected both hits and misses, got %+v", run.Cache)
	}
	if run.Cache.Evictions == 0 {
		t.Fatalf("expected cache eviction, got %+v", run.Cache)
	}
	if run.Cache.Entries != config.CacheCapacity {
		t.Fatalf("final cache entries=%d, want %d", run.Cache.Entries, config.CacheCapacity)
	}
	for _, batch := range run.Batches {
		if !batch.HashMatched {
			t.Fatalf("batch %d returned a foreign dynamic block", batch.Sequence)
		}
		if batch.PreparedEye != batch.Eye {
			t.Fatalf("batch %d prepared %s for %s target", batch.Sequence, batch.PreparedEye, batch.Eye)
		}
	}

	secondConfig := config
	secondConfig.Seed++
	second, err := executePipeline(scene, frames, secondConfig, ModeIsolated)
	if err != nil {
		t.Fatal(err)
	}
	if run.Hashes[EyeLeft] != second.Hashes[EyeLeft] ||
		run.Hashes[EyeRight] != second.Hashes[EyeRight] {
		t.Fatal("batch shuffle order changed the isolated output")
	}
}

func TestFaultProbeFindsFirstPollutedPixel(t *testing.T) {
	scene, frames, reference := renderFixture(t)
	config := defaultPipelineConfig()
	fault, err := executePipeline(scene, frames, config, ModeFault)
	if err != nil {
		t.Fatal(err)
	}
	for _, eye := range []Eye{EyeLeft, EyeRight} {
		first := FirstPollutedPixel(reference[eye], fault.Images[eye])
		if first == nil {
			t.Fatalf("%s fault injection produced no pollution", eye)
		}
		if count := CountPollutedPixels(reference[eye], fault.Images[eye]); count == 0 {
			t.Fatalf("%s fault injection count is zero", eye)
		}
		if fault.Hashes[eye] == HashImage(reference[eye]) {
			t.Fatalf("%s fault hash equals reference", eye)
		}
	}
	report, err := buildReport(scene, frames, reference, mustIsolated(t, scene, frames, config), fault)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Isolation.Clean {
		t.Fatal("report marked isolated run dirty")
	}
	if !report.FaultInjection.AnyPollution {
		t.Fatal("report did not record fault pollution")
	}
	for _, eye := range report.Eyes {
		if eye.SnapshotHash == "" || eye.ImageHash == "" {
			t.Fatalf("%s report is missing hashes", eye.Eye)
		}
		if eye.FirstPollutedPixel != nil || eye.PollutedPixels != 0 {
			t.Fatalf("%s isolated report is polluted", eye.Eye)
		}
		if eye.FaultProbe.FirstPollutedPixel == nil || eye.FaultProbe.PollutedPixels == 0 {
			t.Fatalf("%s fault probe is missing pollution details", eye.Eye)
		}
	}
}

func mustIsolated(t *testing.T, scene *Scene, frames []EyeState, config PipelineConfig) *RunResult {
	t.Helper()
	run, err := executePipeline(scene, frames, config, ModeIsolated)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

func TestReportBundleRecordsSnapshotsHashesAndPollution(t *testing.T) {
	scene, frames, reference := renderFixture(t)
	config := defaultPipelineConfig()
	isolated := mustIsolated(t, scene, frames, config)
	fault, err := executePipeline(scene, frames, config, ModeFault)
	if err != nil {
		t.Fatal(err)
	}
	report, err := buildReport(scene, frames, reference, isolated, fault)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, key := range []string{
		`"parameter_snapshot"`,
		`"image_hash"`,
		`"first_polluted_pixel"`,
		`"fault_injection"`,
		`"dynamic_state_shared":false`,
	} {
		if !strings.Contains(text, key) {
			t.Fatalf("report JSON is missing %s", key)
		}
	}

	outDir := t.TempDir()
	if err := writeBundle(outDir, report, reference, isolated, fault); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"report.json", "report.md",
		imageFilename(EyeLeft), imageFilename(EyeRight),
		faultFilename(EyeLeft), faultFilename(EyeRight),
		diffFilename(EyeLeft), diffFilename(EyeRight),
	} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("missing output %s: %v", name, err)
		}
	}
}

func TestFirstPollutedPixelUsesRasterOrder(t *testing.T) {
	reference := newRGBA(3, 2)
	actual := newRGBA(3, 2)
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			setColor(reference, x, y, 0.1, 0.2, 0.3, 1)
			setColor(actual, x, y, 0.1, 0.2, 0.3, 1)
		}
	}
	setColor(actual, 2, 0, 0.9, 0.2, 0.3, 1)
	setColor(actual, 0, 1, 0.1, 0.9, 0.3, 1)

	first := FirstPollutedPixel(reference, actual)
	if first == nil || first.X != 2 || first.Y != 0 {
		t.Fatalf("first polluted pixel=%+v, want (2,0)", first)
	}
	if count := CountPollutedPixels(reference, actual); count != 2 {
		t.Fatalf("polluted pixel count=%d, want 2", count)
	}
	if rgba := PixelRGBA(actual, first); rgba == nil || rgba[0] == 0 {
		t.Fatalf("missing actual RGBA for first polluted pixel: %v", rgba)
	}
	if FirstPollutedPixel(reference, reference) != nil {
		t.Fatal("identical images reported pollution")
	}
}

func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache, err := newUniformCache(2)
	if err != nil {
		t.Fatal(err)
	}
	first := &PreparedUniforms{Instance: "a", DynamicHash: "a"}
	second := &PreparedUniforms{Instance: "b", DynamicHash: "b"}
	third := &PreparedUniforms{Instance: "c", DynamicHash: "c"}
	cache.put(CacheKey{Instance: "a"}, first)
	cache.put(CacheKey{Instance: "b"}, second)
	if _, ok := cache.get(CacheKey{Instance: "a"}); !ok {
		t.Fatal("expected cache hit for a")
	}
	cache.put(CacheKey{Instance: "c"}, third)
	if _, ok := cache.get(CacheKey{Instance: "b"}); ok {
		t.Fatal("least-recently-used entry b was not evicted")
	}
	if cache.snapshotStats().Evictions != 1 {
		t.Fatalf("evictions=%d, want 1", cache.snapshotStats().Evictions)
	}
}
