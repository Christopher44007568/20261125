package scene

import (
	"fmt"

	"meshscope/internal/analysis"
	"meshscope/internal/mesh"
)

func buildNearPlanar() *Scene {
	m := mesh.NearPlanarPanel()
	res := analysis.Classify(m)
	base := &Scene{
		ID:          "near-planar",
		Index:       "03",
		Title:       "Near-Planar Cover",
		TitleCN:     "近似平面盖板",
		Subtitle:    "3 shallow facet slopes / sub-mm form",
		Code:        "P-03 · FORGED COVER",
		Severity:    "WATCH",
		Description: "顶面由三组近似共面的浅坡度分区构成。亚毫米级形貌偏差落在平面拟合阈值内，分类色很容易掩盖真实褶皱。",
		Camera:      Camera{Azimuth: -24, Elevation: 21, FieldOfView: 29},
		AuditNote:   "实体模式保留低角度光与轮廓；几何审计模式将分类色替换为带符号形貌误差热图，误差不会被颜色层掩盖。",
	}
	base.Findings = []Finding{
		{Tone: "warn", Title: "最大形貌偏差 / FORM", Value: fmt.Sprintf("%.3f mm", res.MaxAbsDev)},
		{Tone: "warn", Title: "RMS 形貌误差 / RMS", Value: fmt.Sprintf("%.3f mm", res.RMSDev)},
		{Tone: "warn", Title: "浅坡度 / SLOPE BREAK", Value: "4–7° local facets"},
		{Tone: "good", Title: "边界置信 / BOUNDARY", Value: fmtConf(res.BoundaryAvg)},
	}
	return buildScene(base, m, res)
}
