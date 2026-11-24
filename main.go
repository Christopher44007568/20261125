package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type Options struct {
	OutDir    string
	Params    Params
	SweepFrom float64
	SweepTo   float64
	SweepStep float64
}

func defaultOptions() Options {
	return Options{
		OutDir: "artifacts", Params: DefaultParams(),
		SweepFrom: 0, SweepTo: 3, SweepStep: 0.025,
	}
}

func main() {
	opts := defaultOptions()
	flag.StringVar(&opts.OutDir, "out", opts.OutDir, "output directory")
	flag.Float64Var(&opts.Params.Offset, "offset", opts.Params.Offset,
		"normal offset distance in mm")
	flag.Float64Var(&opts.Params.CornerRadius, "corner-radius",
		opts.Params.CornerRadius, "convex elbow radius in mm")
	flag.Float64Var(&opts.Params.ArmLength, "arm-length",
		opts.Params.ArmLength, "straight arm length in mm")
	flag.Float64Var(&opts.Params.Height, "height", opts.Params.Height,
		"extrusion height in mm")
	flag.IntVar(&opts.Params.ArcSegments, "arc-segments",
		opts.Params.ArcSegments, "quarter-arc segments")
	flag.Float64Var(&opts.SweepFrom, "sweep-from", opts.SweepFrom,
		"volume sweep start offset")
	flag.Float64Var(&opts.SweepTo, "sweep-to", opts.SweepTo,
		"volume sweep end offset")
	flag.Float64Var(&opts.SweepStep, "sweep-step", opts.SweepStep,
		"volume sweep step")
	flag.Parse()
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(opts Options) error {
	if !opts.Params.valid() {
		return fmt.Errorf("invalid parameters: %+v", opts.Params)
	}
	if opts.SweepStep <= 0 || opts.SweepTo < opts.SweepFrom {
		return fmt.Errorf("invalid sweep range")
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}

	profile := buildProfile(opts.Params)
	mesh := buildMesh(profile)
	if err := mesh.Validate(); err != nil {
		return err
	}
	diag := analyzeSection(profile)
	outer := renderSurface(profile, false)
	inner := renderSurface(profile, true)
	section := renderSection(profile, diag)

	paths := Files{
		InnerSurface: filepath.Join(opts.OutDir, "inner.ppm"),
		OuterSurface: filepath.Join(opts.OutDir, "outer.ppm"),
		ReportJSON:   filepath.Join(opts.OutDir, "report.json"),
		Section:      filepath.Join(opts.OutDir, "section.ppm"),
		VolumeCSV:    filepath.Join(opts.OutDir, "volume_curve.csv"),
	}
	if err := outer.SavePPM(paths.OuterSurface); err != nil {
		return err
	}
	if err := inner.SavePPM(paths.InnerSurface); err != nil {
		return err
	}
	if err := section.SavePPM(paths.Section); err != nil {
		return err
	}

	samples := sweep(opts.Params, opts.SweepFrom, opts.SweepTo, opts.SweepStep)
	signed := mesh.SignedVolume()
	if math.Abs(signed) < 1e-9 {
		signed = 0
	}
	report := Report{
		Unit:                      "millimetres; volume is mm^3",
		RequestedOffsetMM:         opts.Params.Offset,
		Parameters:                opts.Params,
		Topology:                  mesh.Topology(),
		InvertedSectionTriangles:  diag.Inverted,
		SelfIntersections:         len(diag.Intersections),
		SelfIntersectionLoops:     diag.LoopCount,
		RedPixelsInSection:        section.CountRedPixels(),
		VolumeMM3:                 math.Abs(signed),
		SignedVolumeMM3:           signed,
		AnalyticRetainedVolumeMM3: profile.AnalyticVolume(opts.Params.Offset),
		Continuity: continuityFrom(samples, opts.SweepFrom, opts.SweepTo,
			opts.SweepStep),
		Sweep: samples,
		Files: paths,
		Policy: []string{
			"All offset triangles are retained; red diagnostics recolour but never delete geometry.",
			"Fixed vertex and triangle counts make signed volume continuous in offset distance.",
			"The combinatorial shell is watertight; self-intersection and inversion are reported as geometric defects rather than hidden.",
		},
	}
	if err := writeReport(paths.ReportJSON, report); err != nil {
		return err
	}
	csvFile, err := os.Create(paths.VolumeCSV)
	if err != nil {
		return err
	}
	if err := writeSweepCSV(csvFile, samples); err != nil {
		csvFile.Close()
		return err
	}
	if err := csvFile.Close(); err != nil {
		return err
	}
	fmt.Printf("wrote %s, %s, %s\n", paths.OuterSurface,
		paths.InnerSurface, paths.Section)
	fmt.Printf("offset=%.3f mm volume=%.6f mm^3 inverted=%d loops=%d\n",
		opts.Params.Offset, report.VolumeMM3, diag.Inverted, diag.LoopCount)
	return nil
}
