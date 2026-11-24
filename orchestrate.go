package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type frameSet struct {
	left, right *Frame
}

func shiftScene(sc Scene, dx, dy float64) Scene {
	out := sc
	out.Offset = Point{sc.Offset.X + dx, sc.Offset.Y + dy}
	return out
}

func runStrategy(cfg Config, base Scene, mot Motion, strategy string, gt []frameSet) (
	work []frameSet, results []EyeMetrics, interEye float64, falseStab bool) {

	eyes := bothEyes(cfg)
	analyzers := []*analyzer{newAnalyzer(cfg, eyes[0].Name), newAnalyzer(cfg, eyes[1].Name)}
	var basePeriphVar float64
	if strategy == stratBlurred {
		baseGT := gt
		_, baseEyes, _, _ := runStrategy(cfg, base, mot, stratIndependent, baseGT)
		for _, e := range baseEyes {
			basePeriphVar += e.PeriphVar
		}
		basePeriphVar /= float64(len(baseEyes))
	}
	var gtSharpSum, sharpSum float64

	for t := 0; t < cfg.Frames; t++ {
		dx, dy := mot.Shift(t)
		sc := shiftScene(base, dx, dy)
		var fl, fr *Frame
		switch strategy {
		case stratGT:
			fl = renderGroundTruth(cfg, sc, eyes[0], t)
			fr = renderGroundTruth(cfg, sc, eyes[1], t)
		case stratIndependent:
			fl = renderFoveated(cfg, sc, eyes[0], t)
			fr = renderFoveated(cfg, sc, eyes[1], t)
		case stratShared:
			fl, fr = renderShared(cfg, sc, t)
		case stratBlurred:
			fl = boxBlur5(renderFoveated(cfg, sc, eyes[0], t))
			fr = boxBlur5(renderFoveated(cfg, sc, eyes[1], t))
		}
		work = append(work, frameSet{fl, fr})
		analyzers[0].addFrame(t, gt[t].left, fl)
		analyzers[1].addFrame(t, gt[t].right, fr)
	}

	gtAn := []*analyzer{newAnalyzer(cfg, eyes[0].Name), newAnalyzer(cfg, eyes[1].Name)}
	for t := 0; t < cfg.Frames; t++ {
		gtAn[0].addFrame(t, gt[t].left, gt[t].left)
		gtAn[1].addFrame(t, gt[t].right, gt[t].right)
	}
	gtSharp := (gtAn[0].grad/float64(max(gtAn[0].gradN, 1)) +
		gtAn[1].grad/float64(max(gtAn[1].gradN, 1))) / 2

	for _, an := range analyzers {
		results = append(results, an.finalize(gtSharp))
		sharpSum += an.grad / float64(max(an.gradN, 1))
	}
	sharp := sharpSum / 2
	gtSharpSum = gtSharp

	var lf, rf, lgt, rgt []*Frame
	for t := range work {
		lf, rf = append(lf, work[t].left), append(rf, work[t].right)
		lgt, rgt = append(lgt, gt[t].left), append(rgt, gt[t].right)
	}
	interEye = interEyeResidual(cfg, lf, rf, lgt, rgt)
	rmse := (results[0].RMSE + results[1].RMSE) / 2
	periphVar := (results[0].PeriphVar + results[1].PeriphVar) / 2
	falseStab = blurFalseStability(strategy, rmse, gtSharpSum, sharp, periphVar, basePeriphVar)
	if strategy == stratShared {
		if d := interEyeDivergence(cfg, lf, rf); d != 0 {
			panic("shared coverage must produce identical buffers for both eyes")
		}
	}
	return
}

func renderGroundTruthTrack(cfg Config, base Scene, mot Motion) []frameSet {
	eyes := bothEyes(cfg)
	out := make([]frameSet, cfg.Frames)
	for t := 0; t < cfg.Frames; t++ {
		dx, dy := mot.Shift(t)
		sc := shiftScene(base, dx, dy)
		out[t] = frameSet{
			renderGroundTruth(cfg, sc, eyes[0], t),
			renderGroundTruth(cfg, sc, eyes[1], t),
		}
	}
	return out
}

func runAll(cfg Config) []SceneResult {
	var all []SceneResult
	for _, sceneName := range allScenes() {
		base := buildScene(sceneName, cfg)
		for _, mot := range allMotions() {
			gt := renderGroundTruthTrack(cfg, base, mot)
			for _, strategy := range allStrategies() {
				work, eyes, inter, falseStab := runStrategy(cfg, base, mot, strategy, gt)
				all = append(all, SceneResult{
					Scene: sceneName, Motion: mot.Name, Strategy: strategy,
					Eyes: eyes, InterEyeResidual: inter, FalseStability: falseStab,
				})
				if cfg.OutDir != "" {
					writeArtifacts(cfg, base, mot, strategy, gt, work)
				}
			}
		}
	}
	return all
}

func ensureDir(p string) {
	if err := os.MkdirAll(p, 0o755); err != nil {
		panic(fmt.Sprintf("mkdir %s: %v", p, err))
	}
}

func writeArtifacts(cfg Config, base Scene, mot Motion, strategy string, gt, work []frameSet) {
	dir := filepath.Join(cfg.OutDir, base.Name, mot.Name, strategy)
	ensureDir(dir)
	writePNG(filepath.Join(dir, "frame-last-left.png"), work[len(work)-1].left)

	eyes := bothEyes(cfg)
	an := newAnalyzer(cfg, eyes[0].Name)
	for t := range work {
		an.addFrame(t, gt[t].left, work[t].left)
	}
	writeFlickerHeatmap(filepath.Join(dir, "flicker-heatmap-left.png"), an)
	writeTimeseries(filepath.Join(dir, "timeseries-left.csv"), cfg, an)
}
