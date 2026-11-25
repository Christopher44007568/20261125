package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

const (
	sheetPad   = 28
	colGap     = 26
	frameW     = sceneW
	frameH     = sceneH
	sheetW     = sheetPad*2 + frameW*3 + colGap*2
	sheetH     = 704
	imageY     = 140
	histTitleY = 516
	histXPad   = 0
	histTop    = 536
	histBottom = 666
	axisY      = 675
)

var (
	colBackground = color.NRGBA{R: 13, G: 16, B: 22, A: 255}
	colPanel      = color.NRGBA{R: 20, G: 24, B: 33, A: 255}
	colBorder     = color.NRGBA{R: 62, G: 72, B: 91, A: 255}
	colText       = color.NRGBA{R: 232, G: 237, B: 245, A: 255}
	colMuted      = color.NRGBA{R: 151, G: 163, B: 184, A: 255}
	colGrayMark   = color.NRGBA{R: 255, G: 210, B: 74, A: 255}
)

func columnX(index int) int {
	return sheetPad + index*(frameW+colGap)
}

func BuildSceneSheet(scene *Scene, frames []*DisplayFrame) *image.NRGBA {
	sheet := image.NewNRGBA(image.Rect(0, 0, sheetW, sheetH))
	fillRect(sheet, 0, 0, sheetW, sheetH, colBackground)

	drawText(sheet, sheetPad, 24, 3, scene.Title, colText)
	drawText(sheet, sheetPad+2, 61, 2, scene.Subtitle, colMuted)
	anchor := "NEUTRAL GRAY: 18 NIT -> CODE 118"
	drawText(sheet, sheetW-sheetPad-textWidth(anchor, 2), 29, 2, anchor, colGrayMark)
	fillRect(sheet, sheetPad, 91, sheetW-2*sheetPad, 1, colBorder)

	for i, frame := range frames {
		x := columnX(i)
		drawColumn(sheet, scene, frame, x)
	}
	return sheet
}

func drawColumn(sheet *image.NRGBA, scene *Scene, frame *DisplayFrame, x int) {
	accent := nrgbaFromRGB(frame.Spec.Accent)
	fillRect(sheet, x, 99, 7, 23, accent)
	drawText(sheet, x+17, 96, 2, frame.Spec.Label, colText)
	drawText(sheet, x+18, 122, 1, frame.Spec.Standard, colMuted)

	fillRect(sheet, x-2, imageY-2, frameW+4, frameH+4, colBorder)
	draw.Draw(sheet, image.Rect(x, imageY, x+frameW, imageY+frameH),
		frame.Image, image.Point{}, draw.Src)
	drawGrayCardMarker(sheet, scene, x)

	drawText(sheet, x, histTitleY, 1, "PROOF LUMA HISTOGRAM, 64 BINS", colMuted)
	scaleNote := "SQRT COUNT"
	drawText(sheet, x+frameW-textWidth(scaleNote, 1), histTitleY, 1, scaleNote, colMuted)
	drawHistogram(sheet, frame, x)
}

func drawGrayCardMarker(sheet *image.NRGBA, scene *Scene, colX int) {
	r := scene.Gray
	x := colX + int(r.X0)
	y := imageY + int(r.Y0)
	w := int(r.X1 - r.X0)
	h := int(r.Y1 - r.Y0)
	drawCornerBrackets(sheet, x-5, y-5, w+10, h+10, 11, colGrayMark)
	label := "GRAY 18 NIT"
	drawText(sheet, x-3, y-17, 1, label, color.NRGBA{R: 0, G: 0, B: 0, A: 220})
	drawText(sheet, x-4, y-18, 1, label, colGrayMark)
}

func drawCornerBrackets(img *image.NRGBA, x, y, w, h, len int, col color.NRGBA) {
	thick := 2
	fillRect(img, x, y, len, thick, col)
	fillRect(img, x, y, thick, len, col)
	fillRect(img, x+w-len, y, len, thick, col)
	fillRect(img, x+w-thick, y, thick, len, col)
	fillRect(img, x, y+h-thick, len, thick, col)
	fillRect(img, x, y+h-len, thick, len, col)
	fillRect(img, x+w-len, y+h-thick, len, thick, col)
	fillRect(img, x+w-thick, y+h-len, thick, len, col)
}

func drawHistogram(sheet *image.NRGBA, frame *DisplayFrame, x int) {
	histH := histBottom - histTop
	fillRect(sheet, x, histTop, frameW, histH, colPanel)
	for i := 1; i < 4; i++ {
		gy := histTop + histH*i/4
		fillRect(sheet, x, gy, frameW, 1, color.NRGBA{R: 31, G: 38, B: 51, A: 255})
	}

	const groups = 64
	var counts [groups]int
	maxCount := 1
	for code, n := range frame.Histogram {
		g := code * groups / 256
		if g >= groups {
			g = groups - 1
		}
		counts[g] += n
		if counts[g] > maxCount {
			maxCount = counts[g]
		}
	}

	slot := frameW / groups
	barW := slot - 2
	for g, count := range counts {
		ratio := math.Sqrt(float64(count) / float64(maxCount))
		barH := int(ratio * float64(histH-16))
		if count > 0 && barH < 2 {
			barH = 2
		}
		bx := x + g*slot + 1
		by := histBottom - barH
		v := uint8(g*4 + 2)
		barCol := color.NRGBA{R: v, G: v, B: v, A: 255}
		fillRect(sheet, bx, by, barW, barH, barCol)
	}

	grayX := x + int(math.Round(grayProofCode/255.0*float64(frameW)))
	fillRect(sheet, grayX-1, histTop-5, 2, histH+10, colGrayMark)
	label := "GRAY=118"
	drawText(sheet, grayX+7, histTop+7, 1, label, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	drawText(sheet, grayX+6, histTop+6, 1, label, colGrayMark)

	ticks := []int{0, 64, 128, 192, 255}
	for _, value := range ticks {
		tx := x + int(float64(value)/255*float64(frameW))
		fillRect(sheet, tx, histBottom, 1, 5, colMuted)
		label := intToString(value)
		offset := 0
		if value == 0 {
			offset = 0
		} else if value == 255 {
			offset = textWidth(label, 1)
		} else {
			offset = textWidth(label, 1) / 2
		}
		drawText(sheet, tx-offset, axisY, 1, label, colMuted)
	}
	drawRectOutline(sheet, x, histTop, frameW, histH, colBorder)
}

func drawRectOutline(img *image.NRGBA, x, y, w, h int, col color.NRGBA) {
	fillRect(img, x, y, w, 1, col)
	fillRect(img, x, y+h-1, w, 1, col)
	fillRect(img, x, y, 1, h, col)
	fillRect(img, x+w-1, y, 1, h, col)
}

func nrgbaFromRGB(c RGB) color.NRGBA {
	return color.NRGBA{R: clampByte(c[0]), G: clampByte(c[1]), B: clampByte(c[2]), A: 255}
}

func clampByte(v float64) uint8 {
	return clamp255(clamp01(v / 255.0))
}

func intToString(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
