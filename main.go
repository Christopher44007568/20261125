package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type frameArtifacts struct {
	bundle      *CaptureBundle
	fused       FusionResult
	virtual     *VirtualLayer
	visibility  VisibilityStats
	conf        []float64
	calibration ColorCalibration
	passthrough *image.NRGBA
	composite   *image.NRGBA
}

func main() {
	cfg := Config{}
	flag.IntVar(&cfg.Width, "width", defaultWidth, "output width in pixels")
	flag.IntVar(&cfg.Height, "height", defaultHeight, "output height in pixels")
	flag.IntVar(&cfg.Frames, "frames", defaultFrames, "number of captured frames used for latency statistics")
	flag.Int64Var(&cfg.Seed, "seed", 20261125, "deterministic scene seed")
	flag.StringVar(&cfg.OutDir, "out", "out", "output directory")
	flag.DurationVar(&cfg.LatencyBudget, "budget", defaultFrameBudget, "per-frame latency budget")
	flag.Parse()

	report, err := run(cfg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d images + metrics to %s\n", 5, cfg.OutDir)
	fmt.Printf("latency mean %.2f ms, p95 %.2f ms, budget %.2f ms\n",
		report.Latency.MeanMS, report.Latency.P95MS, report.Latency.BudgetMS)
	fmt.Printf("depth invalid %.1f%%, sphere visible %.1f%%, occluded %.1f%%, invalid %.1f%%\n",
		report.Depth.InvalidFraction*100,
		report.Depth.VisibleSphereFraction*100,
		report.Depth.OccludedSphereFraction*100,
		report.Depth.InvalidInSphereFraction*100)
	fmt.Printf("max exposure deviation %.3f EV, color gain [%.3f %.3f %.3f]\n",
		report.Exposure.MaxAbsDeviationEV,
		report.Color.GainR, report.Color.GainG, report.Color.GainB)
}

func run(cfg Config) (Report, error) {
	if cfg.Width < 64 || cfg.Height < 36 {
		return Report{}, fmt.Errorf("resolution must be at least 64x36")
	}
	if cfg.Frames < 1 {
		return Report{}, fmt.Errorf("frames must be positive")
	}
	if cfg.LatencyBudget <= 0 {
		return Report{}, fmt.Errorf("latency budget must be positive")
	}
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return Report{}, fmt.Errorf("create output directory: %w", err)
	}

	k := intrinsicsFor(cfg.Width, cfg.Height, 58)
	latencies := make([]time.Duration, 0, cfg.Frames)
	var last *frameArtifacts
	for i := 0; i < cfg.Frames; i++ {
		rng := rand.New(rand.NewSource(cfg.Seed + int64(i)*7919))
		bundle := captureBackground(cfg.Width, cfg.Height, k, rng)
		bundle.Sequence = i
		bundle.CapturedAt = time.Now()
		started := bundle.CapturedAt

		fused := fuseExposures(bundle)
		calibration := calibrateColor(fused.Camera, bundle.DepthValid, bundle.WhiteBalance)
		virtual, visibility := renderVirtualLayer(bundle, calibration)
		conf := buildConfidence(bundle, fused)
		passthrough := imageFromLinear(fused.Camera)
		composed := composite(fused.Camera, passthrough, virtual, bundle.SystemLayer)
		latencies = append(latencies, time.Since(started))

		last = &frameArtifacts{
			bundle:      bundle,
			fused:       fused,
			virtual:     virtual,
			visibility:  visibility,
			conf:        conf,
			calibration: calibration,
			passthrough: passthrough,
			composite:   composed,
		}
	}
	if last == nil {
		return Report{}, fmt.Errorf("no frames captured")
	}

	paths := outputPaths(cfg.OutDir)
	images := map[string]image.Image{
		"passthrough": last.passthrough,
		"virtual":     imageFromVirtual(last.virtual),
		"composite":   last.composite,
		"confidence":  confidenceImage(last.conf, cfg.Width, cfg.Height),
		"depth":       depthPreview(last.bundle),
	}
	for name, img := range images {
		if err := writePNG(paths[name], img); err != nil {
			return Report{}, fmt.Errorf("write %s: %w", paths[name], err)
		}
	}

	report := Report{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Resolution:     [2]int{cfg.Width, cfg.Height},
		FramesRendered: cfg.Frames,
		Latency:        latencyReport(latencies, cfg.LatencyBudget),
		Depth:          depthReport(last.bundle, last.visibility, last.conf),
		Exposure:       exposureReport(last.fused),
		Color: ColorReport{
			Method:           last.calibration.Method,
			GainR:            last.calibration.Gains.R,
			GainG:            last.calibration.Gains.G,
			GainB:            last.calibration.Gains.B,
			BlendSpace:       "linear-srgb",
			AppliedBeforeMix: true,
		},
		SystemLayer: systemReport(last.bundle.SystemLayer),
		Outputs:     make(map[string]string, len(paths)),
	}
	for name, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			report.Outputs[name] = path
			continue
		}
		report.Outputs[name] = abs
	}
	if err := writeReport(paths["metrics"], report); err != nil {
		return Report{}, fmt.Errorf("write report: %w", err)
	}
	return report, nil
}
