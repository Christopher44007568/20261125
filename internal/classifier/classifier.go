// Package classifier assigns per-marker, per-eye causal labels for the four
// transparent-material failure classes and records why each label fired.
package classifier

import (
	"fmt"
	"sort"

	"eyediag/internal/features"
	"eyediag/internal/model"
)

// Label is one fired cause with its evidence trail.
type Label struct {
	Cause    model.Cause `json:"cause"`
	Score    float64     `json:"score"`
	Evidence []string    `json:"evidence"`
}

// EyeVerdict is the classification result for one eye of one marker.
type EyeVerdict struct {
	Eye            model.Eye `json:"eye"`
	Correct        bool      `json:"correct"`
	Labels         []Label   `json:"labels"`
	DegradedBy     []string  `json:"degradedBy,omitempty"`
	TotalPixelDiff float64   `json:"totalPixelDiff"`
}

// MarkerVerdict groups both eyes and the localized stages.
type MarkerVerdict struct {
	MarkerID string                   `json:"markerId"`
	Eyes     map[model.Eye]EyeVerdict `json:"eyes"`
	Stages   []model.Stage            `json:"stages"`
}

// Classify scores all four causes independently for each eye. Multiple
// labels can fire for the same eye; TotalPixelDiff is reported for context
// only and never participates in any score.
func Classify(m model.Marker, th model.Thresholds) MarkerVerdict {
	th = th.Resolve()
	v := MarkerVerdict{MarkerID: m.ID, Eyes: map[model.Eye]EyeVerdict{}}
	stageSet := map[model.Stage]bool{}

	for _, eye := range []model.Eye{model.Left, model.Right} {
		obs, ok := m.Eyes[eye]
		if !ok {
			continue
		}
		var other *model.Observation
		if o, ok2 := m.Eyes[otherEye(eye)]; ok2 {
			other = &o
		}
		f := features.Extract(m, eye, other, th)

		labels := []Label{
			sortingScore(f, th),
			parallaxScore(f, th),
			alphaScore(f, th),
			aaScore(f, m, obs, th),
		}
		fired := []Label{}
		for _, l := range labels {
			if l.Score >= th.FireScore {
				fired = append(fired, l)
			}
		}
		sort.SliceStable(fired, func(i, j int) bool { return fired[i].Score > fired[j].Score })

		verdict := EyeVerdict{
			Eye:            eye,
			Correct:        len(fired) == 0,
			Labels:         fired,
			DegradedBy:     budgetFactors(m.Budget, fired),
			TotalPixelDiff: pixelDiff(obs.Color, m.ExpectedColor),
		}
		v.Eyes[eye] = verdict
		for _, l := range fired {
			stageSet[stageFor(l.Cause)] = true
		}
	}

	for s := range stageSet {
		v.Stages = append(v.Stages, s)
	}
	sort.Slice(v.Stages, func(i, j int) bool { return v.Stages[i] < v.Stages[j] })
	return v
}

func sortingScore(f features.EyeFeatures, th model.Thresholds) Label {
	l := Label{Cause: model.Sorting}
	colorSig := saturate(f.ColorDist / (th.ColorEps * 4))
	if f.OrderMismatchOK {
		l.Score += 0.5
		l.Evidence = append(l.Evidence,
			fmt.Sprintf("透明图层绘制顺序与期望不符（错位 %.0f%%）", f.OrderMismatch*100))
	}
	if f.DepthAliasOK {
		l.Score += 0.25
		l.Evidence = append(l.Evidence, "互换图层深度几乎相同，属于深度相等导致的排序歧义")
	}
	if colorSig > 0.25 {
		l.Score += 0.2 * colorSig
		l.Evidence = append(l.Evidence, "合成颜色偏差与错误叠加顺序一致")
	}
	if !f.CoverOK {
		l.Score += 0.05
	}
	return l
}

func parallaxScore(f features.EyeFeatures, th model.Thresholds) Label {
	l := Label{Cause: model.Parallax}
	vertexSig := saturate(f.VertexDelta / (th.DepthEps * 5))
	if vertexSig > 0.2 {
		l.Score += 0.4 * vertexSig
		l.Evidence = append(l.Evidence,
			fmt.Sprintf("顶点插值深度偏离期望值 %.4f", f.VertexDelta))
	}
	if !f.ParallaxOK {
		l.Score += 0.35 * saturate(f.ParallaxErr)
		l.Evidence = append(l.Evidence,
			fmt.Sprintf("单眼视差/UV 偏移偏离镜像基准（误差 %.2f）", f.ParallaxErr))
	}
	if f.DisparityErr > th.ParallaxEps {
		l.Score += 0.25 * saturate(f.DisparityErr)
		l.Evidence = append(l.Evidence, "双眼深度视差不满足镜像对称")
	}
	if f.DepthDelta > th.DepthEps {
		l.Score += 0.1
	}
	// Guards: sorting/coverage signatures argue against a parallax root cause.
	if f.OrderMismatchOK {
		l.Score *= 0.6
	}
	if !f.CoverOK {
		l.Score *= 0.8
	}
	return l
}

func alphaScore(f features.EyeFeatures, th model.Thresholds) Label {
	l := Label{Cause: model.Alpha}
	if !f.CoverOK {
		l.Score += 0.4 * saturate(f.CoverageDelta/(th.CoverEps*4))
		l.Evidence = append(l.Evidence,
			fmt.Sprintf("Alpha 覆盖率偏差 %.3f（期望容差 %.3f）", f.CoverageDelta, th.CoverEps))
	}
	if f.AlphaLeakFit > 0.5 {
		l.Score += 0.35 * f.AlphaLeakFit
		l.Evidence = append(l.Evidence, "颜色偏差可由混合公式反演解释，几何深度未变，指向混合/覆盖率泄漏")
	}
	if f.OverPremul > 0.1 {
		l.Score += 0.25 * f.OverPremul
		l.Evidence = append(l.Evidence, "检测到预乘 Alpha 处理不一致")
	}
	if f.ColorDist > th.ColorEps {
		l.Score += 0.1
	}
	// Guards only suppress weak, ambiguous evidence. A strong coverage
	// error must remain visible even when sorting is also wrong, so that
	// genuinely co-occurring multi-cause faults can both fire.
	if f.OrderMismatchOK && f.CoverageDelta < 0.1 {
		l.Score *= 0.75
	}
	if !f.DepthOK && f.CoverageDelta < th.CoverEps*2 {
		l.Score *= 0.85
	}
	return l
}

func aaScore(f features.EyeFeatures, m model.Marker, obs model.Observation, th model.Thresholds) Label {
	l := Label{Cause: model.AA}
	if !f.SampleOK {
		l.Score += 0.4 * saturate(f.SampleDeficit)
		l.Evidence = append(l.Evidence,
			fmt.Sprintf("实际采样数 %d/期望 %d，采样不足", obs.SampleCount, m.ExpectedSamples))
	}
	edgeSig := saturate(f.EdgeDelta / 0.15)
	if edgeSig > 0.2 {
		l.Score += 0.35 * edgeSig
		l.Evidence = append(l.Evidence, "边缘覆盖率呈锯齿状偏差，符合抗锯齿欠采样特征")
	}
	if m.Budget.AlphaToCoverage {
		l.Score += 0.15
		l.Evidence = append(l.Evidence, "Alpha-to-Coverage 已启用，边缘质量直接受 MSAA 采样数影响")
	}
	if f.ColorDist > th.ColorEps && f.CoverOK {
		l.Score += 0.1
	}
	return l
}

// stageFor maps a confirmed cause to the pipeline stage that must be fixed.
func stageFor(c model.Cause) model.Stage {
	switch c {
	case model.Parallax:
		return model.VertexStage
	case model.Sorting, model.Alpha:
		return model.LayerStage
	case model.AA:
		return model.SampleStage
	}
	return model.LayerStage
}

// budgetFactors annotates which budget/degradation conditions plausibly
// contributed to each fired label. Degradation never fabricates a label;
// it only contextualizes labels that already fired on evidence.
func budgetFactors(b model.BudgetStatus, fired []Label) []string {
	if !b.Degraded {
		return nil
	}
	causes := map[model.Cause]bool{}
	for _, l := range fired {
		causes[l.Cause] = true
	}
	out := []string{}
	if b.DepthPrepassOff && (causes[model.Sorting] || causes[model.Parallax]) {
		out = append(out, "深度预载通道关闭，排序/深度歧义未被消解")
	}
	if b.OverdrawRatio > 1.5 && causes[model.Sorting] {
		out = append(out, fmt.Sprintf("过载比 %.1f 超出预算，透明层被合并重排", b.OverdrawRatio))
	}
	if (b.MSAAHalved || b.SampleBudget < 4) && causes[model.AA] {
		out = append(out, "MSAA 预算被削减，采样阶段降级")
	}
	if b.AlphaToCoverage && causes[model.Alpha] {
		out = append(out, "Alpha-to-Coverage 降级改变了有效覆盖率")
	}
	return out
}

func pixelDiff(a, b model.RGBA) float64 {
	dr, dg, db, da := a.R-b.R, a.G-b.G, a.B-b.B, a.A-b.A
	return dr*dr + dg*dg + db*db + da*da
}

func saturate(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func otherEye(e model.Eye) model.Eye {
	if e == model.Left {
		return model.Right
	}
	return model.Left
}
