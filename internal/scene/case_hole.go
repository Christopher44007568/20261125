package scene

import (
	"fmt"
	"math"

	"meshscope/internal/analysis"
	"meshscope/internal/mesh"
)

func buildObliqueHole() *Scene {
	m := mesh.ObliqueHole()
	res := analysis.Classify(m)
	base := &Scene{
		ID:          "oblique-hole",
		Index:       "02",
		Title:       "Oblique Through-Hole",
		TitleCN:     "倾斜贯穿孔",
		Subtitle:    "28° axis / elliptical rim",
		Code:        "H-02 · 5-AXIS DRILL",
		Severity:    "NOMINAL",
		Description: "孔轴偏离表面法线 28°。顶部与底部的椭圆口沿、孔壁和板面三组几何分别拟合，入口边界沿真实椭圆环追踪。",
		Camera:      Camera{Azimuth: -31, Elevation: 43, FieldOfView: 30},
		AuditNote:   "孔壁被单独着色，但孔的深度、椭圆入口和孔轴遮挡仍由实体明暗完整表达。",
	}
	axisDeg, span, cylConf := 0.0, 0.0, 0.0
	for _, r := range res.Regions {
		if r.Kind != mesh.Hole {
			continue
		}
		axisDeg = math.Acos(math.Abs(r.Axis.Y)) * 180 / math.Pi
		span = r.Span
		cylConf = r.Confidence
		break
	}
	base.Findings = []Finding{
		{Tone: "good", Title: "孔轴 / AXIS", Value: fmt.Sprintf("%.1f° from normal", axisDeg)},
		{Tone: "good", Title: "孔壁覆盖 / COVERAGE", Value: fmt.Sprintf("%.0f° ring", span)},
		{Tone: "good", Title: "孔识别置信 / CONF", Value: fmtConf(cylConf)},
		{Tone: "good", Title: "圆柱度 / CYLINDRICITY", Value: fmt.Sprintf("%.3f mm RMS", res.RMSDev)},
	}
	return buildScene(base, m, res)
}
