package render

import (
	"image/color"

	"handlate/internal/sim"
	"handlate/internal/strategy"
	"handlate/internal/vec3"
)

const (
	PlotW = PanelW*3 + 32
	PlotH = 260
)

// DrawErrorChart plots per-frame mean hand-point error against time.
func DrawErrorChart(c *Canvas, ox, oy int, scen sim.Scenario, tl *sim.Timeline) {
	c.FillRect(ox, oy, ox+PlotW-1, oy+PlotH-1, colBackground)
	c.Rect(ox, oy, ox+PlotW-1, oy+PlotH-1, colText)
	c.Text(ox+8, oy+12, "HAND POINT ERROR (M) VS TIME (MS)", colText)

	const (
		gx0, gy0 = 54, 30
		gw, gh   = PlotW - 74, PlotH - 74
	)
	tMax := tl.Displays[len(tl.Displays)-1].NowMs
	kinds := []strategy.Kind{strategy.WorldSync, strategy.LateCompose, strategy.Freeze}

	series := make(map[string][][2]float64)
	var errMax float64
	for _, k := range kinds {
		for _, df := range tl.Displays {
			est := strategy.Build(k, df)
			if est.Points == nil {
				continue
			}
			var sum float64
			for i, p := range est.Points {
				sum += vec3.Dist(p, df.Ground.Points[i])
			}
			e := sum / float64(len(est.Points))
			errMax = maxF(errMax, e)
			series[k.Name()] = append(series[k.Name()], [2]float64{float64(df.NowMs), e})
		}
	}
	if errMax < 0.05 {
		errMax = 0.05
	}

	px := func(t float64) int { return ox + gx0 + int(t/float64(tMax)*gw) }
	py := func(e float64) int { return oy + gy0 + gh - int(e/errMax*gh) }

	c.Rect(ox+gx0, oy+gy0, ox+gx0+gw, oy+gy0+gh, colPanel)
	for g := 0; g <= 4; g++ {
		val := errMax * float64(g) / 4
		yy := py(val)
		c.Line(ox+gx0, yy, ox+gx0+gw, yy, colPanelFill)
		c.Text(ox+8, yy+3, trimFloat(val), colText)
	}
	for g := 0; g <= 5; g++ {
		tv := tMax * g / 5
		xx := px(float64(tv))
		c.Text(xx-10, oy+gy0+gh+14, itoa(tv), colText)
	}

	if scen.ID() == "loss" {
		c.FillRect(px(1800), oy+gy0, px(2400), oy+gy0+gh, color.RGBA{R: 70, G: 55, B: 40, A: 255})
		c.Text(px(1800)+4, oy+gy0+10, "LOSS", colOverHide)
	}

	for _, k := range kinds {
		drawSeries(c, series[k.Name()], px, py, strategyColor(k.Name()))
	}

	legY := oy + PlotH - 12
	for i, k := range kinds {
		lx := ox + 60 + i*200
		col := strategyColor(k.Name())
		c.Line(lx, legY, lx+18, legY, col)
		c.Text(lx+22, legY+3, k.Name(), colText)
	}
}

func drawSeries(c *Canvas, data [][2]float64, px func(float64) int, py func(float64) int, col color.Color) {
	for i := 1; i < len(data); i++ {
		c.Line(px(data[i-1][0]), py(data[i-1][1]), px(data[i][0]), py(data[i][1]), col)
	}
}

func trimFloat(v float64) string {
	scaled := int(v*1000 + 0.5)
	return itoa(scaled/1000) + "." + pad3(scaled%1000)
}

func pad3(n int) string {
	b := []byte{'0', '0', '0'}
	for i := 2; i >= 0; i-- {
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b)
}

func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
