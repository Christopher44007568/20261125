package spectralresample

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
)

// SVGPlot is a minimal line-chart writer: axes, grid, legend and several
// polyline series, enough for spectral curves without third-party code.
type SVGPlot struct {
	Width  int
	Height int
	Title  string
	XLabel string
	YLabel string
	XMin   float64
	XMax   float64
	YMin   float64
	YMax   float64
	Series []SVGSeries
	XTicks []float64
	YTicks []float64
}

// SVGSeries is one polyline with a legend label and stroke colour.
type SVGSeries struct {
	Label  string
	Color  string
	X      []float64
	Y      []float64
	Width  float64
	Dashed bool
}

const (
	svgLeft   = 78.0
	svgRight  = 24.0
	svgTop    = 52.0
	svgBottom = 64.0
)

func (p SVGPlot) xPx(x float64) float64 {
	return svgLeft + (x-p.XMin)/(p.XMax-p.XMin)*(float64(p.Width)-svgLeft-svgRight)
}

func (p SVGPlot) yPx(y float64) float64 {
	return float64(p.Height) - svgBottom - (y-p.YMin)/(p.YMax-p.YMin)*(float64(p.Height)-svgTop-svgBottom)
}

// WriteSVG emits a standalone SVG 1.1 document.
func (p SVGPlot) WriteSVG(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", p.Width, p.Height, p.Width, p.Height)
	fmt.Fprintf(w, `<rect width="100%%" height="100%%" fill="white"/>`+"\n")
	fmt.Fprintf(w, `<text x="%g" y="30" font-family="monospace" font-size="20" font-weight="bold">%s</text>`+"\n", svgLeft, escapeXML(p.Title))

	plotL, plotR := svgLeft, float64(p.Width)-svgRight
	plotT, plotB := svgTop, float64(p.Height)-svgBottom

	// Curves are drawn before the border and legend so overlays never hide
	// the data inside the plot region.
	for _, series := range p.Series {
		dash := ""
		if series.Dashed {
			dash = ` stroke-dasharray="6 4"`
		}
		width := series.Width
		if width == 0 {
			width = 2
		}
		fmt.Fprintf(w, `<polyline fill="none" stroke="%s" stroke-width="%g"%s points="`, series.Color, width, dash)
		for i := range series.X {
			if i > 0 {
				fmt.Fprint(w, " ")
			}
			fmt.Fprintf(w, "%g,%g", p.xPx(series.X[i]), p.yPx(series.Y[i]))
		}
		fmt.Fprint(w, `"/>`+"\n")
	}

	for _, tick := range p.YTicks {
		y := p.yPx(tick)
		fmt.Fprintf(w, `<line x1="%g" y1="%g" x2="%g" y2="%g" stroke="#dddddd"/>`+"\n", plotL, y, plotR, y)
		fmt.Fprintf(w, `<text x="%g" y="%g" font-family="monospace" font-size="12" text-anchor="end">%s</text>`+"\n", plotL-6, y+4, formatTick(tick))
	}
	for _, tick := range p.XTicks {
		x := p.xPx(tick)
		fmt.Fprintf(w, `<line x1="%g" y1="%g" x2="%g" y2="%g" stroke="#dddddd"/>`+"\n", x, plotT, x, plotB)
		fmt.Fprintf(w, `<text x="%g" y="%g" font-family="monospace" font-size="12" text-anchor="middle">%s</text>`+"\n", x, plotB+18, formatTick(tick))
	}
	fmt.Fprintf(w, `<rect x="%g" y="%g" width="%g" height="%g" fill="none" stroke="#333333"/>`+"\n", plotL, plotT, plotR-plotL, plotB-plotT)

	fmt.Fprintf(w, `<text x="%g" y="%g" font-family="monospace" font-size="13" text-anchor="middle">%s</text>`+"\n", (plotL+plotR)/2, plotB+16, escapeXML(p.XLabel))
	fmt.Fprintf(w, `<text x="20" y="%g" font-family="monospace" font-size="13" transform="rotate(-90 20 %g)" text-anchor="middle">%s</text>`+"\n", (plotT+plotB)/2, (plotT+plotB)/2, escapeXML(p.YLabel))

	columns := 1
	if len(p.Series) > 8 {
		columns = 2
	}
	rowsPerColumn := (len(p.Series) + columns - 1) / columns
	legendWidth := 170.0 * float64(columns)
	legendHeight := 18.0*float64(rowsPerColumn) + 10
	legendX := plotL
	legendY := plotB + 28
	fmt.Fprintf(w, `<rect x="%g" y="%g" width="%g" height="%g" fill="white" stroke="#999999"/>`+"\n", legendX, legendY-14, legendWidth, legendHeight)
	for i, series := range p.Series {
		col := i / rowsPerColumn
		row := i % rowsPerColumn
		x := legendX + 8 + float64(col)*170
		y := legendY + float64(row)*18
		dash := ""
		if series.Dashed {
			dash = ` stroke-dasharray="6 4"`
		}
		fmt.Fprintf(w, `<line x1="%g" y1="%g" x2="%g" y2="%g" stroke="%s" stroke-width="2"%s/>`+"\n", x, y, x+26, y, series.Color, dash)
		fmt.Fprintf(w, `<text x="%g" y="%g" font-family="monospace" font-size="12">%s</text>`+"\n", x+32, y+4, escapeXML(series.Label))
	}
	fmt.Fprintln(w, "</svg>")
	return w.Flush()
}

func formatTick(value float64) string {
	if math.Abs(value-math.Round(value)) < 1e-9 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

func escapeXML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}
