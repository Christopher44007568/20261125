package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
)

type FrameMetrics struct {
	T                   float64 `json:"t"`
	StrandCount         int     `json:"strand_count"`
	MaxTipRadius        float64 `json:"max_tip_radius"`
	MaxTipOverBound     float64 `json:"max_tip_over_bound"`
	TipBoundViolations  int     `json:"tip_bound_violations"`
	MaxControlOverBound float64 `json:"max_control_over_bound"`
	ControlViolations   int     `json:"control_bound_violations"`
	MaxSpanOverBound    float64 `json:"max_segment_over_bound"`
	SpanViolations      int     `json:"segment_bound_violations"`
	MaxMigrationMM      float64 `json:"max_migration_mm"`
	MeanMigrationMM     float64 `json:"mean_migration_mm"`
	Finite              bool    `json:"finite"`
}

type ScenarioReport struct {
	ID                  string         `json:"id"`
	Title               string         `json:"title"`
	GroomA              string         `json:"groom_a"`
	GroomB              string         `json:"groom_b"`
	StrandsA            int            `json:"strands_a"`
	StrandsB            int            `json:"strands_b"`
	BlendStrands        int            `json:"blend_strands"`
	WorldScaleMM        float64        `json:"world_scale_mm"`
	Pairing             string         `json:"pairing"`
	MigrationScaleMM    float64        `json:"migration_scale_mm"`
	MaxTipOverBound     float64        `json:"max_tip_over_bound"`
	MaxControlOverBound float64        `json:"max_control_over_bound"`
	MaxSpanOverBound    float64        `json:"max_segment_over_bound"`
	TipEnvelopeRatio    float64        `json:"tip_envelope_ratio"`
	MaxTipRadiusA       float64        `json:"max_tip_radius_a"`
	MaxTipRadiusB       float64        `json:"max_tip_radius_b"`
	Frames              []FrameMetrics `json:"frames"`
	DeterminismHash     string         `json:"determinism_hash"`
}

type ScenarioRun struct {
	Scenario Scenario
	Report   ScenarioReport
	Frames   []Frame
}

func maxTipRadius(g Groom) float64 {
	best := 0.0
	for _, strand := range g.Strands {
		best = math.Max(best, strand.Tip().Dist(strand.Root()))
	}
	return best
}

func frameMetrics(frame Frame, tipEnvelope float64) FrameMetrics {
	m := FrameMetrics{T: frame.T, StrandCount: len(frame.Strands), Finite: true}
	for i, strand := range frame.Strands {
		link := frame.Links[i]
		root := strand.Controls[0]
		maxRadius := 0.0
		for level, p := range strand.Controls {
			if !finiteVec(p) {
				m.Finite = false
			}
			dOut := p.Dist(root)
			dA := link.aControls[level].Dist(link.aRoot)
			dB := link.bControls[level].Dist(link.bRoot)
			bound := lerp(dA, dB, frame.T) + 1e-12
			ratio := dOut / bound
			m.MaxControlOverBound = math.Max(m.MaxControlOverBound, ratio-1)
			m.ControlViolations += boolInt(ratio > 1+1e-9)
			maxRadius = math.Max(maxRadius, dOut)
		}
		tipLevel := len(strand.Controls) - 1
		tipOut := strand.Tip().Dist(root)
		tipA := link.aControls[tipLevel].Dist(link.aRoot)
		tipB := link.bControls[tipLevel].Dist(link.bRoot)
		tipRatio := tipOut / (lerp(tipA, tipB, frame.T) + 1e-12)
		m.MaxTipOverBound = math.Max(m.MaxTipOverBound, tipRatio-1)
		m.TipBoundViolations += boolInt(tipRatio > 1+1e-9)
		for level := 1; level < len(strand.Controls); level++ {
			spanOut := strand.Controls[level].Dist(strand.Controls[level-1])
			spanA := link.aControls[level].Dist(link.aControls[level-1])
			spanB := link.bControls[level].Dist(link.bControls[level-1])
			ratio := spanOut / (lerp(spanA, spanB, frame.T) + 1e-12)
			m.MaxSpanOverBound = math.Max(m.MaxSpanOverBound, ratio-1)
			m.SpanViolations += boolInt(ratio > 1+1e-9)
		}
		m.MaxTipRadius = math.Max(m.MaxTipRadius, maxRadius)
		m.MaxMigrationMM = math.Max(m.MaxMigrationMM, link.MaxMigration)
	}
	if len(frame.Links) > 0 {
		total := 0.0
		for _, link := range frame.Links {
			total += link.MeanMigration
		}
		m.MeanMigrationMM = total / float64(len(frame.Links))
	}
	if tipEnvelope > 0 {
		m.MaxTipRadius /= tipEnvelope
	}
	return m
}

func finiteVec(v Vec3) bool {
	return !math.IsNaN(v.X) && !math.IsNaN(v.Y) && !math.IsNaN(v.Z) &&
		!math.IsInf(v.X, 0) && !math.IsInf(v.Y, 0) && !math.IsInf(v.Z, 0)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func analyzeScenario(sc Scenario) (ScenarioRun, error) {
	interp, err := NewInterpolator(sc.A, sc.B)
	if err != nil {
		return ScenarioRun{}, err
	}
	envelope := math.Max(maxTipRadius(sc.A), maxTipRadius(sc.B))
	report := ScenarioReport{
		ID:            sc.ID,
		Title:         sc.Title,
		GroomA:        sc.A.Name,
		GroomB:        sc.B.Name,
		StrandsA:      len(sc.A.Strands),
		StrandsB:      len(sc.B.Strands),
		WorldScaleMM:  worldScaleMM,
		Pairing:       "Hilbert scalp atlas quantile resampling; deterministic, no random pairing",
		MaxTipRadiusA: maxTipRadius(sc.A),
		MaxTipRadiusB: maxTipRadius(sc.B),
	}
	const steps = 32
	for i := 0; i <= steps; i++ {
		t := float64(i) / steps
		frame := interp.Frame(t)
		m := frameMetrics(frame, envelope)
		report.Frames = append(report.Frames, m)
		report.MaxTipOverBound = math.Max(report.MaxTipOverBound, m.MaxTipOverBound)
		report.MaxControlOverBound = math.Max(report.MaxControlOverBound, m.MaxControlOverBound)
		report.MaxSpanOverBound = math.Max(report.MaxSpanOverBound, m.MaxSpanOverBound)
		report.MigrationScaleMM = math.Max(report.MigrationScaleMM, m.MaxMigrationMM)
		if i == steps/2 {
			report.BlendStrands = m.StrandCount
		}
	}
	report.TipEnvelopeRatio = 0
	for _, m := range report.Frames {
		report.TipEnvelopeRatio = math.Max(report.TipEnvelopeRatio, m.MaxTipRadius)
	}
	run := ScenarioRun{Scenario: sc, Report: report}
	for i := 0; i <= 16; i++ {
		run.Frames = append(run.Frames, interp.Frame(float64(i)/16))
	}
	report.DeterminismHash = interpolationHash(interp)
	run.Report = report
	return run, nil
}

func interpolationHash(it *Interpolator) string {
	h := sha256.New()
	for i := 0; i <= 64; i++ {
		frame := it.Frame(float64(i) / 64)
		fmt.Fprintf(h, "f|%.17g|%d|%.17g|%.17g|%.17g|%.17g|%.17g|%.17g\n",
			frame.T, len(frame.Strands),
			frame.Pose.Yaw, frame.Pose.Pitch, frame.Pose.Roll,
			frame.Pose.Translate.X, frame.Pose.Translate.Y, frame.Pose.Translate.Z)
		for j, strand := range frame.Strands {
			fmt.Fprintf(h, "s|%d|%.17g|%.17g|%.17g|%.17g", j,
				strand.Width, strand.Controls[0].X, strand.Controls[0].Y, strand.Controls[0].Z)
			for _, p := range strand.Controls[1:] {
				fmt.Fprintf(h, "|%.17g,%.17g,%.17g", p.X, p.Y, p.Z)
			}
			link := frame.Links[j]
			fmt.Fprintf(h, "|a%d,%.17g,%d,%.17g|b%d,%.17g,%d,%.17g\n",
				link.A.Indices[0], link.A.Weights[0], link.A.Indices[1], link.A.Weights[1],
				link.B.Indices[0], link.B.Weights[0], link.B.Indices[1], link.B.Weights[1])
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
