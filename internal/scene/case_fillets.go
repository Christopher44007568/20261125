package scene

import (
	"fmt"

	"meshscope/internal/analysis"
	"meshscope/internal/mesh"
)

func buildAdjacentFillets() *Scene {
	m := mesh.AdjacentFillets()
	res := analysis.Classify(m)
	base := &Scene{
		ID:          "adjacent-fillets",
		Index:       "01",
		Title:       "Adjacent Fillet Chain",
		TitleCN:     "邻接圆角链",
		Subtitle:    "G1-tangent ogee / 相切双圆角",
		Code:        "F-01 · SIDE MILL",
		Severity:    "REVIEW",
		Description: "两个等半径圆角在切点共享同一条法线。分类器以曲率中心而非硬边拆开两侧圆角，并把低置信度保留在相切缝上。",
		Camera:      Camera{Azimuth: 37, Elevation: 25, FieldOfView: 30},
		AuditNote:   "分类色仅作为受光材质叠加；圆角轮廓、高光和法线变化始终参与实体着色。",
	}
	filletRegions, filletConf, seamConf := 0, 0.0, 0.0
	for _, r := range res.Regions {
		if r.Kind == mesh.Fillet {
			filletRegions++
			filletConf += r.Confidence
		}
	}
	if filletRegions > 0 {
		filletConf /= float64(filletRegions)
	}
	for _, c := range res.Callouts {
		if c.KindA == mesh.Fillet && c.KindB == mesh.Fillet {
			seamConf = c.Confidence
			break
		}
	}
	base.Findings = []Finding{
		{Tone: "warn", Title: "G1 相切缝 / SEAM", Value: fmtConf(seamConf)},
		{Tone: "good", Title: "圆角拆分 / SPLIT", Value: fmt.Sprintf("%d regions · %s", filletRegions, fmtConf(filletConf))},
		{Tone: "warn", Title: "曲率切换 / CURVATURE", Value: "convex → concave"},
		{Tone: "good", Title: "拟合残差 / FIT", Value: fmt.Sprintf("%.3f mm", res.RMSDev)},
	}
	base.AuditNote += fmt.Sprintf(" 当前相邻圆角边界置信度 %.2f，低于锐边检测阈值。", seamConf)
	return buildScene(base, m, res)
}
