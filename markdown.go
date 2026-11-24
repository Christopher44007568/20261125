package decallab

import (
	"fmt"
	"io"
	"strings"
)

func (r *Report) WriteMarkdown(w io.Writer) error {
	var b strings.Builder
	b.WriteString("# 双眼贴花 / 接触阴影 / 细节法线世界尺度验证\n\n")
	fmt.Fprintf(&b, "- 模块版本：`%s`\n", r.SchemaVersion)
	fmt.Fprintf(&b, "- Go 运行时：`%s`（仅标准库）\n", r.GoVersion)
	fmt.Fprintf(&b, "- 总结论：**%s**\n", passLabel(r.Pass))
	fmt.Fprintf(&b, "- 场景 / 探针观测 / 阴影边：`%d` / `%d` / `%d`\n",
		r.Summary.ScenarioCount, r.Summary.ObservationCount, r.Summary.ShadowEdgeCount)
	fmt.Fprintf(&b, "- 最大斜视：`%.1f°`；最大双眼像素差：`%.2f px`\n",
		r.Summary.MaxObliquityDeg, r.Summary.MaxBinocularPixelDx)
	fmt.Fprintf(&b, "- 直接复制单眼切线法线的最大误差：`%.3f°`；贴花重投影误差：`%.4f m`\n",
		r.Summary.MaxCopyNormalDeg, r.Summary.MaxCopyDecalErrorM)
	fmt.Fprintf(&b, "- 最大阴影边偏差：`%.5f m`；最大双眼光照差异：`%.3f`\n\n",
		r.Summary.MaxShadowEdgeBiasM, r.Summary.MaxLightingDiff)
	b.WriteString(r.Conclusion + "\n\n")

	b.WriteString("## 检查结果\n\n")
	b.WriteString("| 结果 | 检查 | 观测值 | 阈值 | 证据 |\n")
	b.WriteString("|---|---|---:|---:|---|\n")
	for _, check := range r.Checks {
		threshold := fmt.Sprintf("%.6g %s", check.Threshold, check.Unit)
		if check.Threshold == 0 {
			threshold = check.Unit
		}
		fmt.Fprintf(&b, "| %s | `%s` %s | %.6g %s | %s | %s |\n",
			checkMark(check.Pass), check.ID, check.Title,
			check.Observed, check.Unit, threshold, escapeTable(check.Evidence))
	}

	b.WriteString("\n## 场景记录\n\n")
	for _, scenario := range r.Scenarios {
		b.WriteString(scenario.scenarioMarkdown())
	}

	b.WriteString("\n## 记录文件\n\n")
	b.WriteString("- `report.json`：完整机器可读记录。\n")
	b.WriteString("- `normal_probes.csv`：每个法线探针的切线/世界法线、LOD、波长和采样间距。\n")
	b.WriteString("- `decal_scales.csv`：贴花 UV、米制宽高、曲面雅可比尺度和缓存键。\n")
	b.WriteString("- `shadow_edges.csv`：阴影边方向、理论半径、稀疏采样半径和 10%–90% 过渡宽度。\n")
	b.WriteString("- `binocular_diff.csv`：同像素双眼射线、视差位移、切线复制误差和光照差异。\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func (s ScenarioReport) scenarioMarkdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n\n", s.Title)
	fmt.Fprintf(&b, "- ID：`%s`；标签：`%s`\n", s.ID, strings.Join(s.Tags, "`, `"))
	fmt.Fprintf(&b, "- 表面：`%s` / `%s`；贴花标称尺寸：`%.3f × %.3f m`，方向 `%.1f°`\n",
		s.Surface.ID, s.Surface.Kind, s.Decal.WidthM, s.Decal.HeightM, s.Decal.AngleDeg)
	fmt.Fprintf(&b, "- 最大斜视：`%.1f°`；最大采样间距：`%.4f m`；最大曲率尺度变化：`%.2f%%`\n",
		s.Stats.MaxViewObliquityDeg, s.Stats.MaxFootprintM, s.Stats.MaxCurvatureStretchPct)
	fmt.Fprintf(&b, "- 最大双眼会聚角：`%.3f°`；最大水平视差：`%.2f px`\n",
		s.Stats.MaxBinocularAngleDeg, s.Stats.MaxPixelDisparityPx)
	fmt.Fprintf(&b, "- 共享格点复用：`%.1f%%`；独立/共享位置差：`%.5f m`；法线差：`%.3f°`；尺度雅可比差：`%.3f%%`；阴影 α 差：`%.4f`\n",
		s.Cache.ReusePct, s.Cache.MaxSharedPointErrM, s.Cache.MaxSharedNormalDeg,
		s.Cache.MaxArcScaleDiffPct, s.Cache.MaxShadowAlphaDiff)

	maxNormalCopy, maxDecalCopy, maxRadiance := 0.0, 0.0, 0.0
	detected := 0
	for _, copy := range s.CopyComparisons {
		maxNormalCopy = maxFloat(maxNormalCopy, copy.NormalCopy.TangentReuseErrorDeg)
		maxDecalCopy = maxFloat(maxDecalCopy, copy.DecalCopy.ReprojectionM)
		maxRadiance = maxFloat(maxRadiance, copy.CopiedRadianceError)
		if copy.Detected {
			detected++
		}
	}
	fmt.Fprintf(&b, "- 单眼结果直接复制：`%d/%d` 条超过捕获阈值；最大法线误差 `%.3f°`，贴花误差 `%.4f m`，着色误差 `%.3f`\n",
		detected, len(s.CopyComparisons), maxNormalCopy, maxDecalCopy, maxRadiance)

	maxEdgeBias := 0.0
	for _, edge := range s.ShadowEdges {
		maxEdgeBias = maxFloat(maxEdgeBias, edge.EdgeAbsBiasM)
	}
	fmt.Fprintf(&b, "- 阴影边记录：`%d` 条；最大边缘偏差 `%.5f m`\n", len(s.ShadowEdges), maxEdgeBias)

	if len(s.LODSwitches) > 0 {
		maxLODDelta := 0.0
		for _, change := range s.LODSwitches {
			maxLODDelta = maxFloat(maxLODDelta, change.NormalAngleDeltaDeg)
		}
		fmt.Fprintf(&b, "- LOD 切换：`%d` 条；保留波长一致，最大法线变化 `%.3f°`\n",
			len(s.LODSwitches), maxLODDelta)
	}
	if s.Backside != nil {
		fmt.Fprintf(&b, "- 透明背面：hit=`%t` side=`%s` u/v=`%.3f/%.3f`\n",
			s.Backside.Hit, s.Backside.Side, s.Backside.U, s.Backside.V)
	}
	b.WriteString("\n")
	return b.String()
}

func passLabel(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}

func checkMark(pass bool) string {
	if pass {
		return "✅"
	}
	return "❌"
}

func escapeTable(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
