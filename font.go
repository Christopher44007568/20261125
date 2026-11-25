package main

import (
	"image"
	"image/color"
	"strings"
)

var font3x5 = map[rune][5]byte{
	' ': {0, 0, 0, 0, 0},
	'-': {0, 0, 0b111, 0, 0},
	'=': {0, 0b111, 0, 0b111, 0},
	'+': {0, 0b010, 0b111, 0b010, 0},
	':': {0, 0b010, 0, 0b010, 0},
	'.': {0, 0, 0, 0, 0b010},
	'?': {0b110, 0b001, 0b010, 0, 0b010},
	'_': {0, 0, 0, 0, 0b111},
	'/': {0b001, 0b001, 0b010, 0b100, 0b100},
	'0': {0b111, 0b101, 0b101, 0b101, 0b111},
	'1': {0b010, 0b110, 0b010, 0b010, 0b111},
	'2': {0b111, 0b001, 0b111, 0b100, 0b111},
	'3': {0b111, 0b001, 0b111, 0b001, 0b111},
	'4': {0b101, 0b101, 0b111, 0b001, 0b001},
	'5': {0b111, 0b100, 0b111, 0b001, 0b111},
	'6': {0b111, 0b100, 0b111, 0b101, 0b111},
	'7': {0b111, 0b001, 0b010, 0b010, 0b010},
	'8': {0b111, 0b101, 0b111, 0b101, 0b111},
	'9': {0b111, 0b101, 0b111, 0b001, 0b111},
	'A': {0b010, 0b101, 0b111, 0b101, 0b101},
	'B': {0b110, 0b101, 0b110, 0b101, 0b110},
	'C': {0b011, 0b100, 0b100, 0b100, 0b011},
	'D': {0b110, 0b101, 0b101, 0b101, 0b110},
	'E': {0b111, 0b100, 0b110, 0b100, 0b111},
	'F': {0b111, 0b100, 0b110, 0b100, 0b100},
	'G': {0b011, 0b100, 0b101, 0b101, 0b011},
	'H': {0b101, 0b101, 0b111, 0b101, 0b101},
	'I': {0b111, 0b010, 0b010, 0b010, 0b111},
	'J': {0b111, 0b001, 0b001, 0b101, 0b010},
	'K': {0b101, 0b110, 0b100, 0b110, 0b101},
	'L': {0b100, 0b100, 0b100, 0b100, 0b111},
	'M': {0b101, 0b111, 0b111, 0b101, 0b101},
	'N': {0b101, 0b111, 0b111, 0b111, 0b101},
	'O': {0b111, 0b101, 0b101, 0b101, 0b111},
	'P': {0b110, 0b101, 0b110, 0b100, 0b100},
	'Q': {0b111, 0b101, 0b101, 0b111, 0b001},
	'R': {0b110, 0b101, 0b110, 0b110, 0b101},
	'S': {0b011, 0b100, 0b010, 0b001, 0b110},
	'T': {0b111, 0b010, 0b010, 0b010, 0b010},
	'U': {0b101, 0b101, 0b101, 0b101, 0b111},
	'V': {0b101, 0b101, 0b101, 0b101, 0b010},
	'W': {0b101, 0b101, 0b111, 0b111, 0b101},
	'X': {0b101, 0b101, 0b010, 0b101, 0b101},
	'Y': {0b101, 0b101, 0b010, 0b010, 0b010},
	'Z': {0b111, 0b001, 0b010, 0b100, 0b111},
}

func DrawText(img *image.RGBA, text string, x, y, scale int, c color.RGBA) int {
	if scale < 1 {
		scale = 1
	}
	cx := x
	for _, r := range strings.ToUpper(text) {
		if r == '\n' {
			cx = x
			y += 7 * scale
			continue
		}
		glyph, ok := font3x5[r]
		if !ok {
			glyph = font3x5['?']
		}
		for row, bits := range glyph {
			for col := 0; col < 3; col++ {
				if bits&(1<<(2-col)) == 0 {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						px := cx + col*scale + dx
						py := y + row*scale + dy
						off := img.PixOffset(px, py)
						if px >= 0 && py >= 0 && px < img.Bounds().Dx() && py < img.Bounds().Dy() {
							img.Pix[off+0] = c.R
							img.Pix[off+1] = c.G
							img.Pix[off+2] = c.B
							img.Pix[off+3] = 255
						}
					}
				}
			}
		}
		cx += 4 * scale
	}
	return cx
}

func TextWidth(text string, scale int) int {
	if scale < 1 {
		scale = 1
	}
	return len(text) * 4 * scale
}
