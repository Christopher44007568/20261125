package spectralresample

import (
	"fmt"
	"math"
	"strings"
)

// Swatch is one rendered patch: display RGB plus its metrics.
type Swatch struct {
	Label  string
	Family string
	RGB    RGB
	DeltaE float64
}

// RenderPatchChart lays out swatches in columns columns with a title strip
// and a short label under each tile.
func RenderPatchChart(title string, swatches []Swatch, columns int) Canvas {
	const (
		tileSize   = 96
		gap        = 12
		labelH     = 20
		titleH     = 44
		margin     = 16
		labelScale = 1
	)
	rows := (len(swatches) + columns - 1) / columns
	width := margin*2 + columns*tileSize + (columns-1)*gap
	height := titleH + margin + rows*(tileSize+labelH+gap)
	canvas := NewCanvas(width, height)
	canvas.FillRect(0, 0, width-1, height-1, RGB{24, 24, 28})
	drawFittedTitle(canvas, margin, 10, width-2*margin, title, RGB{235, 235, 240})

	for i, swatch := range swatches {
		col := i % columns
		row := i / columns
		x0 := margin + col*(tileSize+gap)
		y0 := titleH + margin + row*(tileSize+labelH+gap)
		canvas.FillRect(x0-1, y0-1, x0+tileSize, y0+tileSize, RGB{220, 220, 224})
		canvas.FillRect(x0, y0, x0+tileSize-1, y0+tileSize-1, swatch.RGB)
		label := swatchLabel(i, swatch.Label)
		if TextWidth(label, labelScale) > tileSize {
			label = trimToWidth(label, tileSize, labelScale)
		}
		textX := x0 + (tileSize-TextWidth(label, labelScale))/2
		canvas.DrawText(textX, y0+tileSize+5, labelScale, label, RGB{225, 225, 230})
	}
	return canvas
}

// RenderErrorMap draws one heat tile per swatch using a blue-cyan-yellow-red
// ramp keyed to CIEDE2000, plus the numeric value and a color-bar legend.
// Equal average colour cannot hide these per-patch spectral mismatches.
func RenderErrorMap(title string, swatches []Swatch, columns int) Canvas {
	const (
		tileSize   = 96
		gap        = 12
		labelH     = 30
		titleH     = 44
		margin     = 16
		labelScale = 1
		legendH    = 64
	)
	rows := (len(swatches) + columns - 1) / columns
	width := margin*2 + columns*tileSize + (columns-1)*gap
	height := titleH + margin + rows*(tileSize+labelH+gap) + legendH
	canvas := NewCanvas(width, height)
	canvas.FillRect(0, 0, width-1, height-1, RGB{20, 20, 24})
	drawFittedTitle(canvas, margin, 10, width-2*margin, title, RGB{235, 235, 240})

	maxError := 0.0
	for _, swatch := range swatches {
		maxError = math.Max(maxError, swatch.DeltaE)
	}
	// Residuals below 0.01 dE2000 are perceptually zero and typically
	// floating-point noise in the metamer constraint; suppress them so a
	// matched chart renders with one uniform heat colour.
	const perceptualZero = 0.01
	for i, swatch := range swatches {
		col := i % columns
		row := i / columns
		x0 := margin + col*(tileSize+gap)
		y0 := titleH + margin + row*(tileSize+labelH+gap)
		normalized := 0.0
		if maxError > perceptualZero {
			normalized = swatch.DeltaE / maxError
		}
		heat := heatRamp(normalized)
		canvas.FillRect(x0-1, y0-1, x0+tileSize, y0+tileSize, RGB{220, 220, 224})
		canvas.FillRect(x0, y0, x0+tileSize-1, y0+tileSize-1, heat)
		label := swatchLabel(i, swatch.Label)
		if TextWidth(label, labelScale) > tileSize {
			label = trimToWidth(label, tileSize, labelScale)
		}
		textX := x0 + (tileSize-TextWidth(label, labelScale))/2
		canvas.DrawText(textX, y0+tileSize+4, labelScale, label, RGB{230, 230, 235})
		value := formatDeltaE(swatch.DeltaE)
		valueX := x0 + (tileSize-TextWidth(value, labelScale))/2
		canvas.DrawText(valueX, y0+tileSize+16, labelScale, value, RGB{245, 245, 250})
	}
	renderLegend(canvas, margin, height-legendH+8, width-2*margin, maxError)
	return canvas
}

func renderLegend(canvas Canvas, x, y, width int, maxError float64) {
	bars := 64
	barW := width / bars
	for i := 0; i < bars; i++ {
		color := heatRamp(float64(i) / float64(bars-1))
		canvas.FillRect(x+i*barW, y, x+(i+1)*barW-1, y+16, color)
	}
	canvas.DrawText(x, y+26, 2, "DE2000 0", RGB{230, 230, 235})
	high := "MAX " + formatDeltaE(maxError)
	canvas.DrawText(x+width-TextWidth(high, 2), y+26, 2, high, RGB{230, 230, 235})
}

// heatRamp maps t in [0,1] to a perceptually ordered blue->red ramp.
func heatRamp(t float64) RGB {
	t = Clamp01(t)
	stops := []struct {
		at    float64
		color RGB
	}{
		{0, RGB{20, 38, 140}},
		{0.25, RGB{28, 150, 210}},
		{0.5, RGB{70, 200, 140}},
		{0.75, RGB{245, 210, 60}},
		{1, RGB{225, 55, 45}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].at {
			a, b := stops[i-1], stops[i]
			f := (t - a.at) / (b.at - a.at)
			return RGB{
				R: uint8(math.Round(float64(a.color.R) + f*float64(b.color.R-a.color.R))),
				G: uint8(math.Round(float64(a.color.G) + f*float64(b.color.G-a.color.G))),
				B: uint8(math.Round(float64(a.color.B) + f*float64(b.color.B-a.color.B))),
			}
		}
	}
	return stops[len(stops)-1].color
}

func compactLabel(label string) string {
	label = strings.ToUpper(label)
	if i := strings.Index(label, " ("); i > 0 {
		label = label[:i]
	}
	return label
}

// swatchLabel prefixes the two-digit patch index so neutral ramps stay
// distinguishable even when the text is truncated to the tile width.
func swatchLabel(index int, label string) string {
	return fmt.Sprintf("%02d %s", index+1, compactLabel(label))
}

// drawFittedTitle shrinks the bitmap font scale until the title fits the
// available width.
func drawFittedTitle(canvas Canvas, x, y, maxWidth int, title string, color RGB) {
	scale := 3
	for scale > 1 && TextWidth(title, scale) > maxWidth {
		scale--
	}
	canvas.DrawText(x, y, scale, title, color)
}

func trimToWidth(label string, width, scale int) string {
	for len(label) > 1 && TextWidth(label+"..", scale) > width {
		label = label[:len(label)-1]
	}
	return label + ".."
}

func formatDeltaE(value float64) string {
	scaled := int(math.Round(value * 10))
	return itoa(scaled/10) + "." + itoa(absInt(scaled)%10)
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
