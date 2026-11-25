package colorlab

import "fmt"

// ChartColumns is the width of the test chart in patches. Every row is padded
// to this width so the mosaic stays rectangular.
const ChartColumns = 11

// Row is one horizontal color bar group in the chart.
type Row struct {
	Name    string
	Patches []RGB
}

// Chart is a display-referred sRGB test pattern. Colors are encoded sRGB
// values, which keeps the reference chart independent of the conversion path.
type Chart struct {
	Rows []Row
}

// StandardChart returns neutral gray steps, saturated primary/secondary bars,
// and a broad range of skin tones.
func StandardChart() Chart {
	gray := make([]RGB, 0, ChartColumns)
	for i := 0; i < ChartColumns; i++ {
		v := float64(i) / float64(ChartColumns-1)
		gray = append(gray, RGB{v, v, v})
	}

	primaries := []RGB{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
		{0, 1, 1},
		{1, 0, 1},
		{1, 1, 0},
		{1, 1, 1},
		{0, 0, 0},
		{0.75, 0, 0},
		{0, 0.75, 0},
		{0, 0, 0.75},
	}

	skin := []RGB{
		{1.000, 0.878, 0.769},
		{0.945, 0.761, 0.490},
		{0.922, 0.706, 0.549},
		{0.878, 0.675, 0.412},
		{0.824, 0.588, 0.373},
		{0.776, 0.525, 0.259},
		{0.706, 0.471, 0.294},
		{0.627, 0.392, 0.235},
		{0.553, 0.333, 0.141},
		{0.451, 0.275, 0.176},
		{0.294, 0.176, 0.118},
	}

	return Chart{Rows: []Row{
		{Name: "neutral gray steps", Patches: gray},
		{Name: "primary/secondary bars", Patches: primaries},
		{Name: "skin tones", Patches: skin},
	}}
}

// RenderChart converts every patch from sRGB into the requested space and
// writes the encoded values directly as RGB bytes.
func RenderChart(chart Chart, space Space, cell int) *Image {
	if cell < 1 {
		cell = 1
	}
	img := NewImage(ChartColumns*cell, len(chart.Rows)*cell)
	for rowIndex, row := range chart.Rows {
		for col := 0; col < ChartColumns; col++ {
			source := RGB{}
			if col < len(row.Patches) {
				source = row.Patches[col]
			}
			encoded := Convert(source, SRGB, space)
			img.FillRect(col*cell, rowIndex*cell, cell, cell, encoded)
		}
	}
	return img
}

// RenderChainMosaic renders three bands: the chart encoded in from, the
// forward result encoded in to, and the round-trip result encoded in from.
func RenderChainMosaic(chart Chart, from, to Space, cell, gutter int) *Image {
	if cell < 1 {
		cell = 1
	}
	if gutter < 0 {
		gutter = 0
	}

	width := ChartColumns * cell
	chartHeight := len(chart.Rows) * cell
	img := NewImage(width, chartHeight*3+gutter*2)
	bands := []struct {
		y     int
		space Space
		round bool
	}{
		{y: 0, space: from},
		{y: chartHeight + gutter, space: to},
		{y: 2 * (chartHeight + gutter), space: from, round: true},
	}

	for rowIndex, row := range chart.Rows {
		for col := 0; col < ChartColumns; col++ {
			source := RGB{}
			if col < len(row.Patches) {
				source = row.Patches[col]
			}
			for _, band := range bands {
				value := Convert(source, SRGB, band.space)
				if band.round {
					value = Convert(Convert(Convert(source, SRGB, from), from, to), to, from)
				} else if band.space == to {
					value = Convert(Convert(source, SRGB, from), from, to)
				}
				img.FillRect(col*cell, band.y+rowIndex*cell, cell, cell, value)
			}
		}
	}
	return img
}

// ChainFileName returns the stable PPM name for an A -> B -> A round trip.
func ChainFileName(from, to Space) string {
	return fmt.Sprintf("chain_%s_to_%s_to_%s.ppm", from.Slug(), to.Slug(), from.Slug())
}
