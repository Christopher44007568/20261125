package main

import (
	"fmt"
	"math"
	"os"
)

type Metric struct {
	Name             string
	MinCurvature     float64
	MaxBendDeg       float64
	SeamSegmentDeg   float64
	MaxOrthogonality float64
	MaxFlipDeg       float64
	TotalTwistDeg    float64
	ClosedTwistDeg   float64
}

func radToDeg(v float64) float64 { return v * 180 / math.Pi }

func measureFrame(name string, stations []Station) Metric {
	n := len(stations)
	m := Metric{Name: name, MinCurvature: math.Inf(1), MaxFlipDeg: -math.Inf(1)}
	totalTwist := 0.0

	for i, s := range stations {
		m.MinCurvature = math.Min(m.MinCurvature, s.Curvature)
		m.MaxOrthogonality = math.Max(m.MaxOrthogonality, math.Abs(s.E1.Dot(s.Tangent)))
		m.MaxOrthogonality = math.Max(m.MaxOrthogonality, math.Abs(s.E2.Dot(s.Tangent)))
		m.MaxOrthogonality = math.Max(m.MaxOrthogonality, math.Abs(s.E1.Dot(s.E2)))

		next := stations[(i+1)%n]
		bend := math.Acos(math.Max(-1, math.Min(1, s.Tangent.Dot(next.Tangent))))
		m.MaxBendDeg = math.Max(m.MaxBendDeg, radToDeg(bend))
	}
	angles := unwrapFrameAngles(stations)
	for i, angle := range angles {
		m.MaxFlipDeg = math.Max(m.MaxFlipDeg, radToDeg(math.Abs(angle)))
		totalTwist += angle
		if i == n-1 {
			m.SeamSegmentDeg = radToDeg(math.Abs(angle))
		}
	}
	m.TotalTwistDeg = radToDeg(totalTwist)
	m.ClosedTwistDeg = radToDeg(closedFrameTwist(stations))
	return m
}

func writeReport(metrics []Metric, holonomy, closureResidual float64, reference Vec3, referenceMargin float64) error {
	var b []byte
	b = append(b, []byte("# 椭圆管扫掠标架对比\n\n")...)
	b = append(b, []byte(fmt.Sprintf("- 平行传输闭合 holonomy: %.4f°，已按站点均布校正\n", radToDeg(holonomy)))...)
	b = append(b, []byte(fmt.Sprintf("- 平行传输虚拟闭合点残差: %.2e\n", closureResidual))...)
	b = append(b, []byte(fmt.Sprintf("- 固定参考轴: (%.4f, %.4f, %.4f)\n", reference.X, reference.Y, reference.Z))...)
	b = append(b, []byte(fmt.Sprintf("- 固定参考最小非平行裕度 |T·R|: %.6f\n", referenceMargin))...)
	b = append(b, []byte("\n| 标架 | 最小曲率 | 最大相邻转角 | 接缝采样段旋转 | 最大正交误差 | 最大逐站旋转 | 局部累计角 | 闭合总扭转 |\n")...)
	b = append(b, []byte("|---|---:|---:|---:|---:|---:|---:|---:|\n")...)
	for _, m := range metrics {
		b = append(b, []byte(fmt.Sprintf("| %s | %.5f | %.3f° | %.6f° | %.2e | %.3f° | %.2f° | %.2f° |\n",
			m.Name, m.MinCurvature, m.MaxBendDeg, m.SeamSegmentDeg, m.MaxOrthogonality, m.MaxFlipDeg, m.TotalTwistDeg, m.ClosedTwistDeg))...)
	}
	b = append(b, []byte("\n渲染文件: `artifacts/frame-comparison.svg`\n")...)
	return os.WriteFile("artifacts/report.md", b, 0o644)
}
