package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Width       int
	Height      int
	Supersample int
	Shutter     float64
	Samples     int
	FrameTime   float64
	Seed        int64
	OutDir      string
}

func main() {
	cfg := Config{}
	flag.IntVar(&cfg.Width, "width", 1280, "output width")
	flag.IntVar(&cfg.Height, "height", 800, "output height")
	flag.IntVar(&cfg.Supersample, "ss", 2, "spatial supersampling factor")
	flag.IntVar(&cfg.Samples, "samples", 20, "temporal shutter samples")
	flag.Float64Var(&cfg.Shutter, "shutter", 0.060, "shutter duration in seconds")
	flag.Float64Var(&cfg.FrameTime, "time", 0.665, "frame center time in seconds")
	flag.Int64Var(&cfg.Seed, "seed", 20261125, "deterministic scene seed")
	flag.StringVar(&cfg.OutDir, "out", "artifacts", "output directory")
	flag.Parse()

	if cfg.Width < 320 || cfg.Height < 200 || cfg.Supersample < 1 || cfg.Samples < 1 {
		fmt.Fprintln(os.Stderr, "invalid resolution or sample count")
		os.Exit(2)
	}
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	start := time.Now()
	result, err := Render(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := WriteOutputs(filepath.Clean(cfg.OutDir), cfg, result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("rendered %dx%d in %s -> %s\n", cfg.Width, cfg.Height,
		time.Since(start).Round(time.Millisecond), cfg.OutDir)
}
