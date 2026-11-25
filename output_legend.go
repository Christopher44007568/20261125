package main

import (
	"image"
	"image/color"
	"math"
	"strconv"
)

var font5x7 = map[rune][7]byte{
	' ': {0, 0, 0, 0, 0, 0, 0},
	'A': {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'B': {0b11110, 0b10001, 0b10001, 0b11110, 0b10001, 0b10001, 0b11110},
	'C': {0b01111, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b01111},
	'D': {0b11110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b11110},
	'E': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
	'F': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000},
	'G': {0b01111, 0b10000, 0b10000, 0b10111, 0b10001, 0b10001, 0b01110},
	'H': {0b10001, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'I': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b11111},
	'J': {0b00111, 0b00010, 0b00010, 0b00010, 0b10010, 0b10010, 0b01100},
	'K': {0b10001, 0b10010, 0b10100, 0b11000, 0b10100, 0b10010, 0b10001},
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'M': {0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001},
	'N': {0b10001, 0b11001, 0b10101, 0b10011, 0b10001, 0b10001, 0b10001},
	'O': {0b01110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'P': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'Q': {0b01110, 0b10001, 0b10001, 0b10001, 0b10101, 0b10010, 0b01101},
	'R': {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	'S': {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	'T': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100},
	'U': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'V': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'W': {0b10001, 0b10001, 0b10001, 0b10101, 0b10101, 0b10101, 0b01010},
	'X': {0b10001, 0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b10001},
	'Y': {0b10001, 0b10001, 0b01010, 0b00100, 0b00100, 0b00100, 0b00100},
	'Z': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b11111},
	'0': {0b01110, 0b10001, 0b10011, 0b10101, 0b11001, 0b10001, 0b01110},
	'1': {0b00100, 0b01100, 0b00100, 0b00100, 0b00100, 0b00100, 0b11111},
	'2': {0b01110, 0b10001, 0b00001, 0b00010, 0b00100, 0b01000, 0b11111},
	'3': {0b11110, 0b00001, 0b00001, 0b01110, 0b00001, 0b00001, 0b11110},
	'4': {0b00010, 0b00110, 0b01010, 0b10010, 0b11111, 0b00010, 0b00010},
	'5': {0b11111, 0b10000, 0b10000, 0b11110, 0b00001, 0b00001, 0b11110},
	'6': {0b01110, 0b10000, 0b10000, 0b11110, 0b10001, 0b10001, 0b01110},
	'7': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b01000, 0b01000},
	'8': {0b01110, 0b10001, 0b10001, 0b01110, 0b10001, 0b10001, 0b01110},
	'9': {0b01110, 0b10001, 0b10001, 0b01111, 0b00001, 0b00001, 0b01110},
	'=': {0b00000, 0b00000, 0b11111, 0b00000, 0b11111, 0b00000, 0b00000},
	'/': {0b00001, 0b00010, 0b00010, 0b00100, 0b01000, 0b01000, 0b10000},
	'-': {0b00000, 0b00000, 0b00000, 0b11111, 0b00000, 0b00000, 0b00000},
	'.': {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00110, 0b00110},
}

func drawText(img *image.NRGBA, x, y int, text string, scale int, c color.NRGBA) {
	cursor := x
	for _, ch := range text {
		g, ok := font5x7[ch]
		if !ok {
			g = font5x7[' ']
		}
		for row := 0; row < 7; row++ {
			for col := 0; col < 5; col++ {
				if g[row]&(1<<uint(4-col)) == 0 {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						blendPixel(img, cursor+col*scale+sx, y+row*scale+sy, c)
					}
				}
			}
		}
		cursor += 6 * scale
	}
}

func drawTextShadow(img *image.NRGBA, x, y int, text string, scale int, c color.NRGBA) {
	drawText(img, x+1, y+1, text, scale, color.NRGBA{R: 0, G: 0, B: 0, A: 180})
	drawText(img, x, y, text, scale, c)
}

func drawMaskLegend(img *image.NRGBA) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	x0, x1 := 30, minInt(458, w-30)
	y0, y1 := h-92, h-28
	fillPanel(img, x0, y0, x1, y1, color.NRGBA{R: 4, G: 9, B: 16, A: 205})
	drawLine(img, Vec2{X: float64(x0), Y: float64(y0)}, Vec2{X: float64(x1), Y: float64(y0)},
		color.NRGBA{R: 72, G: 124, B: 145, A: 190}, 0)
	drawTextShadow(img, x0+14, y0+11, "TRANSPARENT LAYER ORDER", 2,
		color.NRGBA{R: 222, G: 238, B: 239, A: 255})
	barX0, barX1 := x0+14, minInt(x0+294, x1-16)
	barY0, barY1 := y0+38, y0+49
	for x := barX0; x < barX1; x++ {
		t := float64(x-barX0) / float64(barX1-barX0-1)
		c := orderPalette(t)
		for y := barY0; y < barY1; y++ {
			img.Pix[img.PixOffset(x, y)+0] = encodeSRGB(c.R)
			img.Pix[img.PixOffset(x, y)+1] = encodeSRGB(c.G)
			img.Pix[img.PixOffset(x, y)+2] = encodeSRGB(c.B)
			img.Pix[img.PixOffset(x, y)+3] = 255
		}
	}
	drawText(img, barX0, barY1+4, "BACK", 1, color.NRGBA{R: 150, G: 176, B: 184, A: 255})
	drawText(img, barX1-25, barY1+4, "FRONT", 1, color.NRGBA{R: 235, G: 194, B: 184, A: 255})
	drawText(img, barX1+20, barY1+4, "DEPTH", 1, color.NRGBA{R: 184, G: 206, B: 209, A: 210})
}

func drawFlowKey(img *image.NRGBA, scale float64, res *RenderResult) {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	x0, x1 := w-258, w-24
	y0, y1 := h-122, h-24
	fillPanel(img, x0, y0, x1, y1, color.NRGBA{R: 4, G: 9, B: 16, A: 205})
	drawLine(img, Vec2{X: float64(x0), Y: float64(y0)}, Vec2{X: float64(x1), Y: float64(y0)},
		color.NRGBA{R: 65, G: 122, B: 143, A: 185}, 0)
	drawTextShadow(img, x0+12, y0+9, "MOTION VECTORS", 2,
		color.NRGBA{R: 224, G: 239, B: 235, A: 255})
	cx, cy, rad := x0+47, y0+69, 26
	for y := -rad; y <= rad; y++ {
		for x := -rad; x <= rad; x++ {
			d := math.Hypot(float64(x), float64(y))
			if d > float64(rad) {
				continue
			}
			angle := math.Atan2(float64(y), float64(x))
			if angle < 0 {
				angle += 2 * math.Pi
			}
			c := hsvToRGB(angle/(2*math.Pi), 0.78, 0.95)
			blendPixel(img, cx+x, cy+y, color.NRGBA{
				R: encodeSRGB(c.R), G: encodeSRGB(c.G), B: encodeSRGB(c.B), A: 235,
			})
		}
	}
	drawText(img, x0+84, y0+53, "HUE = DIRECTION", 1,
		color.NRGBA{R: 198, G: 220, B: 218, A: 255})
	drawText(img, x0+84, y0+65, "VALUE = SPEED", 1,
		color.NRGBA{R: 198, G: 220, B: 218, A: 255})
	maxText := "MAX " + strconv.Itoa(int(math.Round(res.MaxSpeed))) + " PX/S"
	drawText(img, x0+84, y0+77, maxText, 1,
		color.NRGBA{R: 135, G: 177, B: 185, A: 245})
	_ = scale
}
