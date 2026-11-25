package main

import (
	"fmt"
	"math"
	"path/filepath"
)

func meshCamera() Camera {
	c := NewCamera(-36, 22, 1600, 1100, 5.15)
	c.Center = Vec3{0, 0, 29}
	c.PanX = -310
	c.PanY = 40
	return c
}

func RenderFittedMesh(path string, points []ScanPoint, fit *FitResult, audit Audit, trimA, trimB float64, clean, noise int) error {
	s := NewSVG(1600, 1100)
	pageShell(s, "02 / 06", "SURFACE", "Fitted BRep preview shell",
		"Trimmed cubic B-spline surface, tessellated from the control net; patch seams and isoparametric flow remain visible.")
	panel(s, 48, 190, 1074, 850, "FITTED MESH / SHADED TESSELLATION", "56 × 36 QUADS")
	cam := meshCamera()
	faces := buildSurfaceFaces(fit.Surface, Reference{}, trimA, trimB, 56, 36, 0)
	zs := []float64{}
	for iy := 0; iy <= 16; iy++ {
		for ix := 0; ix <= 20; ix++ {
			x := -trimA + float64(ix)/20*2*trimA
			y := -trimB + float64(iy)/16*2*trimB
			if trimRadius(x, y) < 0.98 {
				zs = append(zs, fit.Surface.Z(x, y))
			}
		}
	}
	zmin, zmax := minMax(zs)
	colorFn := func(f SurfaceFace) string {
		t := clamp((f.Center.Z-zmin)/(zmax-zmin+1e-9), 0, 1)
		base := mixColor("#0a4c63", "#6ff0d4", 0.08+0.84*t)
		return base
	}
	drawSurfaceFaces(s, cam, faces, colorFn, "#a6fff5", 0.32, 0.30)
	drawIsoparms(s, cam, fit.Surface, trimA, trimB, 12, "#8ff9ee", 0.22)
	seamColor := colGold
	seam1 := isoCurve(fit.Surface, trimA, trimB, 0.0, 0, 0, 240)
	seam2 := isoCurve(fit.Surface, trimA, trimB, 0.0, 1, 0, 240)
	s.Segmented(projectWorld(cam, seam1), seamColor, 1.05, 0.64)
	s.Segmented(projectWorld(cam, seam2), seamColor, 1.05, 0.64)
	drawTrim(s, cam, fit.Surface, trimA, trimB, "#071017", 6.5, 0.55)
	drawTrim(s, cam, fit.Surface, trimA, trimB, colCyan, 2.0, 0.98)
	drawTrim(s, cam, fit.Surface, trimA, trimB, "#d6fffb", 0.65, 0.9)
	drawHoleOverlay(s, cam, fit.Surface, true)
	drawNoisePoints(s, cam, fit.Surface, points, true)

	seen := 0
	for _, p := range points {
		if p.Kind != PointClean {
			continue
		}
		seen++
		if seen%7 != 0 {
			continue
		}
		pz := fit.Surface.Z(p.X, p.Y) + 0.15
		x, y, _ := cam.Project(Vec3{p.X, p.Y, pz})
		s.Circle(x, y, 1.5, "#b8fff6", "#071017", 0.45, 0.72)
	}
	for i, q := range [][2]float64{{-0.44, -0.38}, {0.44, -0.38}, {-0.44, 0.38}, {0.44, 0.38}} {
		x, y := q[0]*trimA, q[1]*trimB
		sx, sy, _ := cam.Project(Vec3{x, y, fit.Surface.Z(x, y) + 0.8})
		s.Circle(sx, sy, 11, "#07131b", colGold, 1, 0.88)
		s.Mono(sx, sy+3.3, 8, colGold, "700", "middle", fmt.Sprintf("P%d", i+1))
	}
	drawSurfaceLabel(s, 76, 870, []string{
		fmt.Sprintf("PATCH AREA   %.0f mm²", math.Pi*trimA*trimB),
		fmt.Sprintf("HEIGHT RANGE %.1f … %.1f mm", zmin, zmax),
		"SEAMS u=0.5 / v=0.5  ·  4 QUADRANTS",
	}, colCyan)
	drawAxisGizmo(s, cam, 1028, 935, 34)
	lx := 72.0
	lx = legendChip(s, lx, 1002, "clean", "SCAN SAMPLE / 7th")
	lx = legendChip(s, lx, 1002, "noise", "NOISE")
	_ = legendChip(s, lx, 1002, "hole", "OCCLUSION")

	statRail(s, 1145, 190, 391, audit, fit, clean, noise, len(scanHoles))
	s.Mono(1167, 992, 9, colFaint, "400", "start", "CUBIC B-SPLINE SURFACE  /  CLAMPED KNOTS")
	s.Mono(1167, 1010, 9, colFaint, "400", "start", "TESSELLATION RESOLUTION 56 × 36")
	return s.Save(filepath.Clean(path))
}

func minMax(v []float64) (float64, float64) {
	mn, mx := math.MaxFloat64, -math.MaxFloat64
	for _, x := range v {
		mn = math.Min(mn, x)
		mx = math.Max(mx, x)
	}
	return mn, mx
}
