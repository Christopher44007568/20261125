package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "eyeiso: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	config := defaultPipelineConfig()
	outDir := "artifacts/eye-isolation"
	flag.Int64Var(&config.Seed, "seed", config.Seed, "deterministic batch shuffle seed")
	flag.IntVar(&config.Rounds, "rounds", config.Rounds, "number of shuffled render rounds")
	flag.IntVar(&config.CacheCapacity, "cache-capacity", config.CacheCapacity, "LRU uniform-cache capacity")
	flag.StringVar(&outDir, "out", outDir, "output directory for JSON, Markdown and PNG files")
	flag.Parse()

	if err := config.Validate(); err != nil {
		return err
	}
	scene, err := newScene()
	if err != nil {
		return err
	}
	frames, err := defaultEyeStates(scene)
	if err != nil {
		return err
	}

	reference := make(map[Eye]*image.RGBA, len(frames))
	for _, frame := range frames {
		rendered, err := renderReference(scene, frame)
		if err != nil {
			return fmt.Errorf("reference render for %s: %w", frame.Eye, err)
		}
		reference[frame.Eye] = rendered
	}

	isolated, err := executePipeline(scene, frames, config, ModeIsolated)
	if err != nil {
		return err
	}
	fault, err := executePipeline(scene, frames, config, ModeFault)
	if err != nil {
		return err
	}
	report, err := buildReport(scene, frames, reference, isolated, fault)
	if err != nil {
		return err
	}
	if err := writeBundle(outDir, report, reference, isolated, fault); err != nil {
		return err
	}

	absoluteOut, _ := filepath.Abs(outDir)
	fmt.Printf("isolated: clean=%t, matches_reference=%t, cache hits=%d misses=%d evictions=%d\n",
		report.Isolation.Clean, report.Isolation.IsolatedMatchesReference,
		report.Pipeline.Cache.Hits, report.Pipeline.Cache.Misses, report.Pipeline.Cache.Evictions)
	for _, eye := range report.Eyes {
		fmt.Printf("%s: snapshot=%s image=%s isolated_pollution=%d\n",
			eye.Eye, shortHash(eye.SnapshotHash), shortHash(eye.ImageHash), eye.PollutedPixels)
	}
	for _, eye := range report.FaultInjection.Eyes {
		pixel := "none"
		if eye.FirstPollutedPixel != nil {
			pixel = fmt.Sprintf("(%d,%d)", eye.FirstPollutedPixel.X, eye.FirstPollutedPixel.Y)
		}
		fmt.Printf("fault %s: image=%s first_polluted=%s polluted_pixels=%d\n",
			eye.Eye, shortHash(eye.ImageHash), pixel, eye.PollutedPixels)
	}
	fmt.Printf("report: %s\n", filepath.Join(absoluteOut, "report.json"))

	if !report.Isolation.Clean {
		return fmt.Errorf("isolated render was contaminated or snapshots changed")
	}
	if !report.FaultInjection.AnyPollution {
		return fmt.Errorf("fault probe did not reproduce cross-eye contamination")
	}
	return nil
}

func shortHash(hash string) string {
	if len(hash) <= 16 {
		return hash
	}
	return hash[:16]
}
