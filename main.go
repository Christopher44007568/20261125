package main

import (
	"fmt"
	"math"
	"os"
)

type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Add(o Vec3) Vec3      { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3      { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Scale(k float64) Vec3 { return Vec3{v.X * k, v.Y * k, v.Z * k} }
func (v Vec3) Dot(o Vec3) float64   { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }
func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{v.Y*o.Z - v.Z*o.Y, v.Z*o.X - v.X*o.Z, v.X*o.Y - v.Y*o.X}
}
func (v Vec3) Norm() float64    { return math.Sqrt(v.Dot(v)) }
func (v Vec3) Normalized() Vec3 { n := v.Norm(); return v.Scale(1 / n) }

type Station struct {
	Param     float64
	Point     Vec3
	Tangent   Vec3
	Normal    Vec3
	Binormal  Vec3
	E1        Vec3
	E2        Vec3
	Arc       float64
	Curvature float64
	Torsion   float64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	stations, length := sampleTrefoil(sampleCount)

	frenet := buildFrenet(stations)
	parallel, holonomy, closureResidual := buildParallelTransport(stations)
	fixed, reference, referenceMargin := buildFixedFrame(stations)
	metrics := []Metric{
		measureFrame("Frenet", frenet),
		measureFrame("parallel-transport", parallel),
		measureFrame("fixed-reference", fixed),
	}

	if err := writeReport(metrics, holonomy, closureResidual, reference, referenceMargin); err != nil {
		return err
	}

	svg := renderSVG([]FrameSet{
		{Name: "Frenet 标架", Stations: frenet},
		{Name: "平行传输标架", Stations: parallel, Note: fmt.Sprintf("holonomy=%.2f°，均布校正；闭合残差 %.1e", radToDeg(holonomy), closureResidual)},
		{Name: "固定参考标架", Stations: fixed, Note: fmt.Sprintf("动态固定轴；最小 |T·R|=%.3f，无平行退化", referenceMargin)},
	}, length)

	return os.WriteFile("artifacts/frame-comparison.svg", []byte(svg), 0o644)
}
