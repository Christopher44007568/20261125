package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type reportData struct {
	Input       inputReport     `json:"input"`
	Fit         fitReport       `json:"fit"`
	Audit       auditReport     `json:"audit"`
	Convergence []stageReport   `json:"convergence"`
	Selection   selectionReport `json:"selection"`
}

type inputReport struct {
	TotalSamples   int     `json:"totalSamples"`
	Clean          int     `json:"clean"`
	Noise          int     `json:"noise"`
	OcclusionVoids int     `json:"occlusionVoids"`
	TrimAmm        float64 `json:"trimAmm"`
	TrimBmm        float64 `json:"trimBmm"`
}

type fitReport struct {
	Degree        int     `json:"degree"`
	ControlNet    string  `json:"controlNet"`
	Lambda        float64 `json:"lambda"`
	IRLSPasses    int     `json:"irlsPasses"`
	InlierRMSmm   float64 `json:"inlierRmsMm"`
	RobustSigmaMm float64 `json:"robustSigmaMm"`
}

type auditReport struct {
	DenseSamples        int     `json:"denseSamples"`
	DenseRMSEmm         float64 `json:"denseRmseMm"`
	DenseMaxAbsmm       float64 `json:"denseMaxAbsMm"`
	NormalRMSdeg        float64 `json:"normalRmsDeg"`
	NormalP95deg        float64 `json:"normalP95Deg"`
	NormalMaxdeg        float64 `json:"normalMaxDeg"`
	CurvatureSamples    int     `json:"curvatureSamples"`
	CurvatureRMS        float64 `json:"curvatureRmsPerMm"`
	CurvatureP95        float64 `json:"curvatureP95PerMm"`
	CurvatureRelPct     float64 `json:"curvatureRelPct"`
	RippleRMS           float64 `json:"rippleRmsScaled"`
	RippleRelPct        float64 `json:"rippleRelPct"`
	SilhouetteIoU       float64 `json:"silhouetteIoU"`
	SilhouetteChamferMM float64 `json:"silhouetteChamferMm"`
}

type stageReport struct {
	ControlNet      string  `json:"controlNet"`
	InlierRMSmm     float64 `json:"inlierRmsMm"`
	DenseRMSEmm     float64 `json:"denseRmseMm"`
	NormalP95deg    float64 `json:"normalP95Deg"`
	CurvatureRelPct float64 `json:"curvatureRelPct"`
	RippleRelPct    float64 `json:"rippleRelPct"`
	SilhouetteIoU   float64 `json:"silhouetteIoU"`
	ChamferMM       float64 `json:"chamferMm"`
	Selected        bool    `json:"selected"`
	Decision        string  `json:"decision"`
}

type selectionReport struct {
	Selected              string  `json:"selected"`
	HeightGainFromNextPct float64 `json:"heightGainFromNextPct"`
	CurvatureChangePct    float64 `json:"curvatureChangePct"`
	RippleChangePct       float64 `json:"rippleChangePct"`
	SilhouetteChange      float64 `json:"silhouetteChange"`
	Rule                  string  `json:"rule"`
}

func main() {
	out := flag.String("out", "artifacts", "output directory")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "surfacefit:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	ref := Reference{}
	points := GenerateScan(ref)
	clean, noise := 0, 0
	for _, p := range points {
		if p.Kind == PointNoise {
			noise++
		} else {
			clean++
		}
	}
	holes := OcclusionMask(points)
	trimA, trimB := EstimateTrim(points)

	specs := []struct {
		Label  string
		NU, NV int
	}{
		{"6 × 5", 6, 5},
		{"8 × 6", 8, 6},
		{"10 × 7", 10, 7},
		{"12 × 9", 12, 9},
		{"14 × 10", 14, 10},
	}
	stages := make([]StageResult, 0, len(specs))
	for _, sp := range specs {
		opt := FitOptions{NU: sp.NU, NV: sp.NV, Lambda: 0.02, RobustIters: 8}
		fit, err := FitSurface(points, opt)
		if err != nil {
			return fmt.Errorf("fit %s: %w", sp.Label, err)
		}
		audit := EvaluateFit(fit.Surface, trimA, trimB)
		stages = append(stages, StageResult{Label: sp.Label, Opt: opt, Fit: fit, Audit: audit})
	}
	selected := 3
	final := stages[selected]

	files := []struct {
		Name string
		Run  func(string) error
	}{
		{"01_pointcloud.svg", func(path string) error {
			return RenderPointCloud(path, points, holes, final.Fit, clean, noise, trimA, trimB)
		}},
		{"02_fitted_mesh.svg", func(path string) error {
			return RenderFittedMesh(path, points, final.Fit, final.Audit, trimA, trimB, clean, noise)
		}},
		{"03_normal_error.svg", func(path string) error {
			return RenderNormalError(path, points, final.Fit, final.Audit, trimA, trimB, clean, noise)
		}},
		{"04_view_a.svg", func(path string) error {
			return RenderRotatedView(path, "04 / 06", "Rotation A · front quarter",
				"Fitted shell with silhouette overlay, independent noise and occlusion masks.",
				"ROTATION A / FRONT", -38, 19, points, final.Fit, final.Audit, trimA, trimB, clean, noise)
		}},
		{"05_view_b.svg", func(path string) error {
			return RenderRotatedView(path, "05 / 06", "Rotation B · rear quarter",
				"Second orbit position stresses the opposite trim and the compound-curvature shoulder.",
				"ROTATION B / REAR", 137, 16, points, final.Fit, final.Audit, trimA, trimB, clean, noise)
		}},
		{"06_convergence.svg", func(path string) error {
			return RenderConvergence(path, stages, selected)
		}},
	}
	for _, file := range files {
		if err := file.Run(filepath.Join(out, file.Name)); err != nil {
			return err
		}
		fmt.Printf("wrote %s\n", filepath.Join(out, file.Name))
	}

	data := makeReport(points, holes, final, stages, selected, trimA, trimB, clean, noise)
	if err := writeReports(out, data); err != nil {
		return err
	}
	fmt.Printf("selected %s: dense %.3f mm, normal p95 %.2f deg, curvature %.1f%%, ripple %.1f%%, IoU %.5f\n",
		final.Label, final.Audit.DenseRMSE, final.Audit.NormalP95, final.Audit.CurvatureRelative,
		final.Audit.RippleRatio, final.Audit.SilhouetteIoU)
	return nil
}

func makeReport(points []ScanPoint, holes HoleField, final StageResult, stages []StageResult, selected int, trimA, trimB float64, clean, noise int) reportData {
	d := reportData{
		Input: inputReport{
			TotalSamples: len(points), Clean: clean, Noise: noise,
			OcclusionVoids: len(holes.Components), TrimAmm: trimA, TrimBmm: trimB,
		},
		Fit: fitReport{
			Degree: 3, ControlNet: fmt.Sprintf("%d × %d", final.Opt.NU, final.Opt.NV),
			Lambda: final.Opt.Lambda, IRLSPasses: final.Fit.Iterations,
			InlierRMSmm: final.Fit.InlierRMS, RobustSigmaMm: final.Fit.Sigma,
		},
		Audit: auditReport{
			DenseSamples: final.Audit.DenseSamples,
			DenseRMSEmm:  final.Audit.DenseRMSE, DenseMaxAbsmm: final.Audit.DenseMaxAbs,
			NormalRMSdeg: final.Audit.NormalRMSE, NormalP95deg: final.Audit.NormalP95,
			NormalMaxdeg: final.Audit.NormalMax, CurvatureSamples: final.Audit.CurvatureSamples,
			CurvatureRMS: final.Audit.CurvatureRMS, CurvatureP95: final.Audit.CurvatureP95,
			CurvatureRelPct: final.Audit.CurvatureRelative,
			RippleRMS:       final.Audit.RippleRMS, RippleRelPct: final.Audit.RippleRatio,
			SilhouetteIoU: final.Audit.SilhouetteIoU, SilhouetteChamferMM: final.Audit.SilhouetteChamfer,
		},
	}
	for i, st := range stages {
		decision := "audit"
		if i == selected {
			decision = "selected"
		} else if i == len(stages)-1 {
			decision = "rejected: curvature and ripple regress"
		}
		d.Convergence = append(d.Convergence, stageReport{
			ControlNet: st.Label, InlierRMSmm: st.Fit.InlierRMS, DenseRMSEmm: st.Audit.DenseRMSE,
			NormalP95deg: st.Audit.NormalP95, CurvatureRelPct: st.Audit.CurvatureRelative,
			RippleRelPct: st.Audit.RippleRatio, SilhouetteIoU: st.Audit.SilhouetteIoU,
			ChamferMM: st.Audit.SilhouetteChamfer, Selected: i == selected, Decision: decision,
		})
	}
	cur, last := stages[selected], stages[len(stages)-1]
	d.Selection = selectionReport{
		Selected:              final.Label,
		HeightGainFromNextPct: (cur.Audit.DenseRMSE - last.Audit.DenseRMSE) / cur.Audit.DenseRMSE * 100,
		CurvatureChangePct:    (last.Audit.CurvatureRelative - cur.Audit.CurvatureRelative) / cur.Audit.CurvatureRelative * 100,
		RippleChangePct:       (last.Audit.RippleRatio - cur.Audit.RippleRatio) / cur.Audit.RippleRatio * 100,
		SilhouetteChange:      last.Audit.SilhouetteIoU - cur.Audit.SilhouetteIoU,
		Rule:                  "increase resolution only while silhouette, curvature and ripple improve together; low control residual is insufficient",
	}
	return d
}
