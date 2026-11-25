package hair

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

func writeCoverageChart(report DollyReport, path string) error {
	const width, height = 920, 340
	const left, right, top, bottom = 62, 24, 44, 54
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 8, G: 12, B: 19, A: 255})
		}
	}
	plotW := width - left - right
	plotH := height - top - bottom
	axis := color.RGBA{R: 90, G: 104, B: 122, A: 255}
	grid := color.RGBA{R: 28, G: 39, B: 52, A: 255}
	for i := 0; i <= 4; i++ {
		y := top + plotH*i/4
		drawHLine(img, left, width-right, y, grid)
	}
	for i := 0; i <= 4; i++ {
		x := left + plotW*i/4
		drawVLine(img, x, top, top+plotH, grid)
	}
	drawHLine(img, left, width-right, top+plotH, axis)
	drawVLine(img, left, top, top+plotH, axis)

	coverageColor := color.RGBA{R: 67, G: 226, B: 205, A: 255}
	iouColor := color.RGBA{R: 244, G: 184, B: 70, A: 255}
	prevCov := image.Point{}
	prevIoU := image.Point{}
	for i, sample := range report.Samples {
		x := left
		if len(report.Samples) > 1 {
			x += plotW * i / (len(report.Samples) - 1)
		}
		covY := top + int(math.Round((1-clamp(sample.CoverageRatio, 0, 1))*float64(plotH)))
		iouY := top + int(math.Round((1-clamp(sample.IoU, 0, 1))*float64(plotH)))
		if i > 0 {
			drawLine(img, prevCov.X, prevCov.Y, x, covY, coverageColor)
			drawLine(img, prevIoU.X, prevIoU.Y, x, iouY, iouColor)
		}
		drawDisc(img, x, covY, 2, coverageColor)
		drawDisc(img, x, iouY, 2, iouColor)
		prevCov = image.Pt(x, covY)
		prevIoU = image.Pt(x, iouY)
	}
	drawNumber(img, 18, top-12, "1", 2, axis)
	drawNumber(img, 6, top+plotH/2-8, "0.5", 1, axis)
	drawNumber(img, 18, top+plotH-8, "0", 2, axis)
	drawHLine(img, width-168, width-144, 18, coverageColor)
	drawNumber(img, width-138, 13, "C", 2, coverageColor)
	drawHLine(img, width-82, width-58, 18, iouColor)
	drawNumber(img, width-52, 13, "I", 2, iouColor)
	if len(report.Samples) > 0 {
		first := report.Samples[0].Distance
		last := report.Samples[len(report.Samples)-1].Distance
		drawNumber(img, left-8, top+plotH+12, fmt.Sprintf("%.0f", first), 1, axis)
		label := fmt.Sprintf("%.0f", last)
		drawNumber(img, width-right-len(label)*6, top+plotH+12, label, 1, axis)
	}
	return WritePNG(path, img)
}

func drawHLine(img *image.RGBA, x0, x1, y int, c color.RGBA) {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	for x := x0; x <= x1; x++ {
		if image.Pt(x, y).In(img.Bounds()) {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawVLine(img *image.RGBA, x, y0, y1 int, c color.RGBA) {
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		if image.Pt(x, y).In(img.Bounds()) {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
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

func drawDisc(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= radius*radius && image.Pt(x, y).In(img.Bounds()) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

var tinyDigits = map[rune][7]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'.': {"00000", "00000", "00000", "00000", "00000", "00110", "00110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'I': {"01110", "00100", "00100", "00100", "00100", "00100", "01110"},
}

func drawNumber(img *image.RGBA, x, y int, text string, scale int, c color.RGBA) {
	if scale < 1 {
		scale = 1
	}
	cursor := x
	for _, r := range text {
		glyph, ok := tinyDigits[r]
		if !ok {
			cursor += 6 * scale
			continue
		}
		for row, bits := range glyph {
			for col, bit := range bits {
				if bit != '1' {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						px := cursor + col*scale + dx
						py := y + row*scale + dy
						if image.Pt(px, py).In(img.Bounds()) {
							img.SetRGBA(px, py, c)
						}
					}
				}
			}
		}
		cursor += 6 * scale
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
