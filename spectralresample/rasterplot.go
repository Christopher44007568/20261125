package spectralresample

import "math"

// RasterSeries is one curve for the dependency-free PPM line chart.
type RasterSeries struct {
	Label  string
	Color  RGB
	X      []float64
	Y      []float64
	Dashed bool
}

// RasterPlotConfig describes axes and data ranges for a PPM line chart.
type RasterPlotConfig struct {
	Width  int
	Height int
	Title  string
	XLabel string
	YLabel string
	XMin   float64
	XMax   float64
	YMin   float64
	YMax   float64
	XTicks []float64
	YTicks []float64
	Series []RasterSeries
}

const (
	plotLeft   = 84
	plotRight  = 28
	plotTop    = 56
	plotBottom = 118
)

// RenderRasterPlot draws axes, ticks, gridlines and polylines onto a PPM
// canvas so spectral curves can be inspected without an SVG renderer.
func RenderRasterPlot(cfg RasterPlotConfig) Canvas {
	canvas := NewCanvas(cfg.Width, cfg.Height)
	canvas.FillRect(0, 0, cfg.Width-1, cfg.Height-1, RGB{255, 255, 255})
	left, right := plotLeft, cfg.Width-plotRight
	top, bottom := plotTop, cfg.Height-plotBottom

	axisColor := RGB{60, 60, 60}
	gridColor := RGB{224, 224, 224}
	textColor := RGB{30, 30, 30}
	drawFittedTitle(canvas, left, 14, cfg.Width-left-plotRight, cfg.Title, textColor)

	xPx := func(x float64) int {
		return left + int(math.Round((x-cfg.XMin)/(cfg.XMax-cfg.XMin)*float64(right-left)))
	}
	yPx := func(y float64) int {
		return bottom - int(math.Round((y-cfg.YMin)/(cfg.YMax-cfg.YMin)*float64(bottom-top)))
	}

	for _, tick := range cfg.YTicks {
		y := yPx(tick)
		canvas.FillRect(left, y, right, y, gridColor)
		label := formatRasterTick(tick)
		canvas.DrawText(left-TextWidth(label, 1)-6, y-3, 1, label, textColor)
	}
	for _, tick := range cfg.XTicks {
		x := xPx(tick)
		canvas.FillRect(x, top, x, bottom, gridColor)
		label := formatRasterTick(tick)
		canvas.DrawText(x-TextWidth(label, 1)/2, bottom+8, 1, label, textColor)
	}
	canvas.FillRect(left, top, left, bottom, axisColor)
	canvas.FillRect(left, bottom, right, bottom, axisColor)

	for _, series := range cfg.Series {
		for i := 1; i < len(series.X); i++ {
			if series.Dashed && i%2 == 0 {
				continue
			}
			drawLine(canvas, xPx(series.X[i-1]), yPx(series.Y[i-1]), xPx(series.X[i]), yPx(series.Y[i]), series.Color)
		}
	}

	canvas.DrawText((left+right)/2-TextWidth(cfg.XLabel, 1)/2, bottom+26, 1, cfg.XLabel, textColor)
	renderRasterLegend(canvas, left, bottom+42, cfg.Series)
	return canvas
}

func renderRasterLegend(canvas Canvas, x, y int, series []RasterSeries) {
	columns := 1
	if len(series) > 8 {
		columns = 2
	}
	rows := (len(series) + columns - 1) / columns
	for i, entry := range series {
		col := i / rows
		row := i % rows
		cx := x + col*((canvas.Width-plotLeft-plotRight)/2)
		cy := y + row*15
		canvas.FillRect(cx, cy, cx+24, cy, entry.Color)
		canvas.DrawText(cx+30, cy-5, 1, entry.Label, RGB{40, 40, 40})
	}
}

// drawLine rasterises a segment with Bresenham's algorithm.
func drawLine(canvas Canvas, x0, y0, x1, y1 int, color RGB) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		canvas.Set(x0, y0, color)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func formatRasterTick(value float64) string {
	if math.Abs(value-math.Round(value)) < 1e-9 {
		return itoa(int(math.Round(value)))
	}
	scaled := int(math.Round(value * 100))
	return itoa(scaled/100) + "." + itoa(absInt(scaled)%100)
}
