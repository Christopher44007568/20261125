package colorlab

import "testing"

func TestStandardChartRows(t *testing.T) {
	chart := StandardChart()
	if len(chart.Rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(chart.Rows))
	}
	wantNames := []string{"neutral gray steps", "primary/secondary bars", "skin tones"}
	for i, row := range chart.Rows {
		if row.Name != wantNames[i] {
			t.Errorf("row %d name = %q, want %q", i, row.Name, wantNames[i])
		}
		if len(row.Patches) != ChartColumns {
			t.Errorf("row %d has %d patches, want %d", i, len(row.Patches), ChartColumns)
		}
	}
	for i, patch := range chart.Rows[0].Patches {
		want := float64(i) / float64(ChartColumns-1)
		if patch.R != want || patch.G != want || patch.B != want {
			t.Errorf("gray patch %d = %+v, want neutral %.6g", i, patch, want)
		}
	}
	if chart.Rows[1].Patches[0] != (RGB{1, 0, 0}) {
		t.Errorf("first primary = %+v, want red", chart.Rows[1].Patches[0])
	}
}

func TestRenderChartDimensions(t *testing.T) {
	chart := StandardChart()
	img := RenderChart(chart, SRGB, 7)
	if img.Width != ChartColumns*7 || img.Height != 3*7 {
		t.Fatalf("image is %dx%d, want %dx%d",
			img.Width, img.Height, ChartColumns*7, 3*7)
	}

	x, y := 5*7+3, 0*7+3
	got := RGB{
		float64(img.Pix[(y*img.Width+x)*3+0]) / 255.0,
		float64(img.Pix[(y*img.Width+x)*3+1]) / 255.0,
		float64(img.Pix[(y*img.Width+x)*3+2]) / 255.0,
	}
	want := RGB{0.5, 0.5, 0.5}
	for _, delta := range channelDeltas(got, want) {
		if delta > 1.0/255.0 {
			t.Fatalf("gray pixel = %+v, want %+v", got, want)
		}
	}
}

func TestChainMosaicBands(t *testing.T) {
	chart := StandardChart()
	cell, gutter := 5, 2
	img := RenderChainMosaic(chart, SRGB, Rec2020, cell, gutter)
	chartHeight := len(chart.Rows) * cell
	wantHeight := chartHeight*3 + gutter*2
	if img.Width != ChartColumns*cell || img.Height != wantHeight {
		t.Fatalf("mosaic is %dx%d, want %dx%d",
			img.Width, img.Height, ChartColumns*cell, wantHeight)
	}

	top := img.Pix[0 : (chartHeight)*img.Width*3]
	bottomStart := (chartHeight + gutter) * 2 * img.Width * 3
	bottom := img.Pix[bottomStart:]
	for i := range top {
		delta := int(top[i]) - int(bottom[i])
		if delta < 0 {
			delta = -delta
		}
		if delta > 1 {
			t.Fatalf("round-trip band differs from source by %d at byte %d", delta, i)
		}
	}
	if len(img.Pix) != img.Width*img.Height*3 {
		t.Fatalf("pixel buffer has %d bytes, want %d",
			len(img.Pix), img.Width*img.Height*3)
	}
}
