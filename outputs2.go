package main

import (
	"fmt"
	"image"
	"image/color"
)

const (
	filmScale  = 3
	leftColumn = 150
)

// renderFilmstrip 输出远景连续帧：上排二元采样、下排解析覆盖率，
// 每帧 64x40，最近邻放大 3 倍，共 16 帧。
func renderFilmstrip(bin, aa *FrameStats) *image.RGBA {
	fw, fh := distantW*filmScale, distantH*filmScale
	gap := 6
	width := leftColumn + frames*(fw+gap) + 10
	rowH := 16 + fh
	height := 14 + 2*rowH + 14
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(img, 0, 0, width, height, color.RGBA{10, 11, 15, 255})

	rows := []struct {
		title string
		stats *FrameStats
	}{
		{"A BINARY", bin},
		{"B ANALYTIC", aa},
	}
	for row, item := range rows {
		ry := 14 + row*(rowH+14)
		drawText(img, 8, ry+2, item.title, labelDark)
		for f := 0; f < frames; f++ {
			fx := leftColumn + f*(fw+gap)
			num := fmt.Sprintf("%02d", f)
			drawText(img, fx+2, ry+2, num, labelDark)
			r := &Raster{W: distantW, H: distantH, Cov: item.stats.coverage[f]}
			frame := nearestScale(rasterToImage(r), filmScale)
			for y := 0; y < fh; y++ {
				for x := 0; x < fw; x++ {
					img.SetRGBA(fx+x, ry+16+y, frame.RGBAAt(x, y))
				}
			}
		}
	}
	return img
}

// renderFlickerMap 输出闪烁像素热力图：二元覆盖率极差、解析覆盖率极差、
// 两者差值。采用统一归一化尺度，颜色越亮表示时序闪烁越强。
func renderFlickerMap(binAmp, aaAmp []float64) *image.RGBA {
	const scale = 4
	pw, ph := distantW*scale, distantH*scale
	gap := 16
	width := gap*4 + pw*3
	height := headerH + ph + gap
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(img, 0, 0, width, height, color.RGBA{10, 11, 15, 255})

	peak := 0.0
	for _, v := range binAmp {
		if v > peak {
			peak = v
		}
	}
	for _, v := range aaAmp {
		if v > peak {
			peak = v
		}
	}
	if peak == 0 {
		peak = 1
	}
	diff := make([]float64, len(binAmp))
	for i := range diff {
		if d := binAmp[i] - aaAmp[i]; d > 0 {
			diff[i] = d
		}
	}
	panels := []struct {
		title string
		data  []float64
	}{
		{"A BINARY AMP", binAmp},
		{"B ANALYTIC AMP", aaAmp},
		{"C AMP DELTA", diff},
	}
	for i, p := range panels {
		ox := gap + i*(pw+gap)
		drawText(img, ox+2, 8, p.title, labelDark)
		for y := 0; y < distantH; y++ {
			for x := 0; x < distantW; x++ {
				c := heatColor(p.data[y*distantW+x] / peak)
				fillRect(img, ox+x*scale, headerH+y*scale,
					ox+(x+1)*scale, headerH+(y+1)*scale, c)
			}
		}
	}
	return img
}
