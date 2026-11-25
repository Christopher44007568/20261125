package main

import (
	"fmt"
	"path/filepath"
)

type StageResult struct {
	Label string
	Opt   FitOptions
	Fit   *FitResult
	Audit Audit
}

func RenderConvergence(path string, stages []StageResult, selected int) error {
	s := NewSVG(1600, 1100)
	pageShell(s, "06 / 06", "CONVERGENCE", "Resolution ladder and acceptance",
		"A low control-point residual is not accepted as proof. Height, silhouette, normals and curvature are audited independently.")
	panel(s, 48, 190, 720, 418, "HEIGHT + SILHOUETTE", "LOWER IS BETTER")
	panel(s, 792, 190, 744, 418, "CURVATURE + RIPPLE", "PLATEAU CHECK")
	drawConvergenceA(s, 92, 262, 636, 278, stages, selected)
	drawConvergenceB(s, 836, 262, 662, 278, stages, selected)
	drawStageTable(s, 48, 632, 1488, 408, stages, selected)
	return s.Save(filepath.Clean(path))
}

func chartY(v, y, h, max float64) float64 { return y + h - clamp(v/max, 0, 1)*h }

func chartX(i, n int, x, w float64) float64 {
	if n <= 1 {
		return x + w*0.5
	}
	return x + float64(i)/float64(n-1)*w
}

func drawChartFrame(s *SVG, x, y, w, h, max float64, format string) {
	for i := 0; i <= 4; i++ {
		v := max * float64(i) / 4
		yy := chartY(v, y, h, max)
		s.Line(x, yy, x+w, yy, colGrid, 1, 0.75)
		s.Mono(x-10, yy+3, 8, colFaint, "400", "end", fmt.Sprintf(format, v))
	}
	s.Line(x, y, x, y+h, colGrid2, 1, 0.9)
	s.Line(x, y+h, x+w, y+h, colGrid2, 1, 0.9)
}

func drawSeries(s *SVG, x, y, w, h, max float64, vals []float64, tone string, dashed bool) {
	var pts []Vec2
	n := len(vals)
	for i, v := range vals {
		pts = append(pts, Vec2{chartX(i, n, x, w), chartY(v, y, h, max)})
	}
	if dashed {
		for i := 0; i < len(pts)-1; i++ {
			s.DashedLine(pts[i].X, pts[i].Y, pts[i+1].X, pts[i+1].Y, tone, 1.8, 1, "6 6")
		}
	} else {
		s.Segmented(pts, tone, 2.1, 1)
	}
	for _, p := range pts {
		s.Circle(p.X, p.Y, 4.2, colPanel, tone, 1.8, 1)
	}
}

func drawConvergenceA(s *SVG, x, y, w, h float64, stages []StageResult, selected int) {
	max := 1.0
	drawChartFrame(s, x, y, w, h, max, "%.1f")
	n := len(stages)
	for i := range stages {
		xx := chartX(i, n, x, w)
		s.Line(xx, y, xx, y+h, colGrid, 0.7, 0.3)
		s.Mono(xx, y+h+20, 8.5, colMuted, "500", "middle", stages[i].Label)
	}
	var dense, ctrl, cham []float64
	for _, st := range stages {
		dense = append(dense, st.Audit.DenseRMSE)
		ctrl = append(ctrl, st.Fit.InlierRMS)
		cham = append(cham, st.Audit.SilhouetteChamfer)
	}
	drawSeries(s, x, y, w, h, max, ctrl, colOrange, true)
	drawSeries(s, x, y, w, h, max, dense, colCyan, false)
	drawSeries(s, x, y, w, h, max, cham, colMagenta, false)
	sx := chartX(selected, n, x, w)
	s.Line(sx, y-4, sx, y+h+2, colGold, 1.1, 0.75)
	s.Mono(sx+7, y+2, 8, colGold, "600", "start", "SELECTED")
	lx := x + 8
	lx = legendChip(s, lx, y-14, "noise", "CONTROL RMS")
	lx = legendChip(s, lx, y-14, "clean", "DENSE HEIGHT")
	_ = legendChip(s, lx, y-14, "hole", "CHAMFER")
	s.Mono(x+w, y+h+38, 8.5, colFaint, "400", "end", "CONTROL NET →")
}

func drawConvergenceB(s *SVG, x, y, w, h float64, stages []StageResult, selected int) {
	max := 50.0
	drawChartFrame(s, x, y, w, h, max, "%.0f%%")
	n := len(stages)
	for i := range stages {
		xx := chartX(i, n, x, w)
		s.Line(xx, y, xx, y+h, colGrid, 0.7, 0.3)
		s.Mono(xx, y+h+20, 8.5, colMuted, "500", "middle", stages[i].Label)
	}
	var curv, ripple, normal []float64
	for _, st := range stages {
		curv = append(curv, st.Audit.CurvatureRelative)
		ripple = append(ripple, st.Audit.RippleRatio)
		normal = append(normal, st.Audit.NormalP95)
	}
	drawSeries(s, x, y, w, h, max, curv, colGold, false)
	drawSeries(s, x, y, w, h, max, ripple, colMagenta, false)
	// The normal series uses a 0–8 degree right axis.
	var npts []Vec2
	for i, v := range normal {
		npts = append(npts, Vec2{chartX(i, n, x, w), chartY(v, y, h, 8)})
	}
	s.Segmented(npts, colCyan, 1.6, 0.9)
	for _, p := range npts {
		s.Circle(p.X, p.Y, 3, colPanel, colCyan, 1.2, 1)
	}
	for i := 0; i <= 4; i++ {
		deg := 8 * float64(i) / 4
		yy := chartY(deg, y, h, 8)
		s.Mono(x+w+9, yy+3, 8, colFaint, "400", "start", fmt.Sprintf("%.0f°", deg))
	}
	sx := chartX(selected, n, x, w)
	s.Line(sx, y-4, sx, y+h+2, colGold, 1.1, 0.75)
	lx := x + 8
	lx = legendChip(s, lx, y-14, "noise", "CURVATURE %")
	lx = legendChip(s, lx, y-14, "hole", "RIPPLE %")
	_ = legendChip(s, lx, y-14, "clean", "NORMAL p95")
	s.Mono(x+w, y+h+38, 8.5, colFaint, "400", "end", "RIGHT AXIS: NORMAL °")
}

func drawStageTable(s *SVG, x, y, w, h float64, stages []StageResult, selected int) {
	s.RectEx(x, y, w, h, colPanel, colGrid, 1, 0.96, 16)
	s.Text(x+22, y+30, 11, colMuted, "700", "start", "INDEPENDENT ACCEPTANCE TABLE")
	s.Mono(x+w-22, y+30, 9, colCyan, "600", "end",
		fmt.Sprintf("DENSE REFERENCE / %d SAMPLES", stages[selected].Audit.DenseSamples))
	s.Line(x+22, y+46, x+w-22, y+46, colGrid, 1, 1)
	cols := []struct {
		Title string
		Off   float64
	}{
		{"CONTROL NET", 22}, {"INLIER RMS", 196}, {"DENSE RMSE", 318}, {"NORMAL p95", 448},
		{"CURVATURE", 590}, {"RIPPLE", 714}, {"SILHOUETTE IoU", 828}, {"CHAMFER", 994},
		{"DECISION", 1130},
	}
	for _, c := range cols {
		s.Text(x+c.Off, y+70, 8.5, colFaint, "600", "start", c.Title)
	}
	rowY := y + 92
	for i, st := range stages {
		if i == selected {
			s.RectEx(x+14, rowY-20, w-28, 49, "#12323a", "", 0, 0.75, 6)
		}
		decision, tone := "AUDIT", colFaint
		if i == selected {
			decision, tone = "SELECTED", colGold
		}
		if i == len(stages)-1 && i != selected {
			decision, tone = "REJECTED", colOrange
		}
		cells := []string{
			st.Label,
			fmt.Sprintf("%.3f", st.Fit.InlierRMS),
			fmt.Sprintf("%.3f", st.Audit.DenseRMSE),
			fmt.Sprintf("%.2f°", st.Audit.NormalP95),
			fmt.Sprintf("%.1f%%", st.Audit.CurvatureRelative),
			fmt.Sprintf("%.1f%%", st.Audit.RippleRatio),
			fmt.Sprintf("%.5f", st.Audit.SilhouetteIoU),
			fmt.Sprintf("%.3f", st.Audit.SilhouetteChamfer),
			decision,
		}
		for j, cell := range cells {
			clr := colText
			weight := "500"
			switch j {
			case 0:
				clr, weight = colMuted, "600"
			case 4, 5:
				clr = colGold
			case 8:
				clr = tone
			}
			s.Mono(x+cols[j].Off, rowY, 10, clr, weight, "start", cell)
		}
		if i < len(stages)-1 {
			s.Line(x+22, rowY+28, x+w-22, rowY+28, colGrid, 1, 0.55)
		}
		rowY += 52
	}
	last := stages[len(stages)-1]
	if selected >= 0 && selected < len(stages)-1 {
		cur := stages[selected]
		dh := (cur.Audit.DenseRMSE - last.Audit.DenseRMSE) / cur.Audit.DenseRMSE * 100
		dc := (last.Audit.CurvatureRelative - cur.Audit.CurvatureRelative) / cur.Audit.CurvatureRelative * 100
		dr := (last.Audit.RippleRatio - cur.Audit.RippleRatio) / cur.Audit.RippleRatio * 100
		text := fmt.Sprintf("SELECTION RULE  ·  next grid: height %.1f%%, curvature %+.1f%%, ripple %+.1f%%, silhouette %.5f → %.5f",
			-dh, dc, dr, cur.Audit.SilhouetteIoU, last.Audit.SilhouetteIoU)
		s.Mono(x+22, y+h-76, 9, colMuted, "400", "start", text)
	}
	s.Rect(x+22, y+h-58, w-44, 1, colGrid)
	s.Text(x+22, y+h-28, 11, colGold, "600", "start",
		fmt.Sprintf("SELECTED %s", stages[selected].Label))
	s.Text(x+150, y+h-28, 11, colMuted, "400", "start",
		fmt.Sprintf("control RMS %.3f mm  ·  dense RMSE %.3f mm  ·  curvature %.1f%%  ·  ripple %.1f%%",
			stages[selected].Fit.InlierRMS, stages[selected].Audit.DenseRMSE,
			stages[selected].Audit.CurvatureRelative, stages[selected].Audit.RippleRatio))
	s.Text(x+w-22, y+h-28, 9, colRed, "700", "end",
		"LOW POINT RESIDUAL ALONE IS NOT ACCEPTED")
}
