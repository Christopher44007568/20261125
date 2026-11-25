package main

import (
	"image"
	"image/color"
)

// glyph3x5 is a compact hand-built 3x5 bitmap font (one byte per row,
// low three bits). Sufficient for labels and numeric reports.
var glyph3x5 = map[rune][5]byte{
	'A': {0b010, 0b101, 0b111, 0b101, 0b101},
	'B': {0b110, 0b101, 0b110, 0b101, 0b110},
	'C': {0b011, 0b100, 0b100, 0b100, 0b011},
	'D': {0b110, 0b101, 0b101, 0b101, 0b110},
	'E': {0b111, 0b100, 0b110, 0b100, 0b111},
	'F': {0b111, 0b100, 0b110, 0b100, 0b100},
	'G': {0b011, 0b100, 0b101, 0b101, 0b011},
	'H': {0b101, 0b101, 0b111, 0b101, 0b101},
	'I': {0b111, 0b010, 0b010, 0b010, 0b111},
	'K': {0b101, 0b110, 0b100, 0b110, 0b101},
	'L': {0b100, 0b100, 0b100, 0b100, 0b111},
	'M': {0b101, 0b111, 0b111, 0b101, 0b101},
	'N': {0b101, 0b111, 0b111, 0b111, 0b101},
	'O': {0b010, 0b101, 0b101, 0b101, 0b010},
	'P': {0b110, 0b101, 0b110, 0b100, 0b100},
	'R': {0b110, 0b101, 0b110, 0b101, 0b101},
	'S': {0b011, 0b100, 0b010, 0b001, 0b110},
	'T': {0b111, 0b010, 0b010, 0b010, 0b010},
	'U': {0b101, 0b101, 0b101, 0b101, 0b111},
	'V': {0b101, 0b101, 0b101, 0b101, 0b010},
	'X': {0b101, 0b101, 0b010, 0b101, 0b101},
	'Y': {0b101, 0b101, 0b010, 0b010, 0b010},
	'0': {0b111, 0b101, 0b101, 0b101, 0b111},
	'1': {0b010, 0b110, 0b010, 0b010, 0b111},
	'2': {0b110, 0b001, 0b010, 0b100, 0b111},
	'3': {0b110, 0b001, 0b010, 0b001, 0b110},
	'4': {0b101, 0b101, 0b111, 0b001, 0b001},
	'5': {0b111, 0b100, 0b110, 0b001, 0b110},
	'6': {0b011, 0b100, 0b110, 0b101, 0b010},
	'7': {0b111, 0b001, 0b010, 0b010, 0b010},
	'8': {0b010, 0b101, 0b010, 0b101, 0b010},
	'9': {0b010, 0b101, 0b011, 0b001, 0b110},
	'.': {0b000, 0b000, 0b000, 0b000, 0b010},
	',': {0b000, 0b000, 0b000, 0b010, 0b100},
	':': {0b000, 0b010, 0b000, 0b010, 0b000},
	'-': {0b000, 0b000, 0b111, 0b000, 0b000},
	'_': {0b000, 0b000, 0b000, 0b000, 0b111},
	'/': {0b001, 0b001, 0b010, 0b100, 0b100},
	'(': {0b010, 0b100, 0b100, 0b100, 0b010},
	')': {0b010, 0b001, 0b001, 0b001, 0b010},
	'%': {0b101, 0b001, 0b010, 0b100, 0b101},
	' ': {0, 0, 0, 0, 0},
}

// drawText renders a string at (x,y) with pixel scale s in the given color.
func drawText(img *image.RGBA, x, y, s int, text string, col color.RGBA) {
	cx := x
	for _, r := range text {
		glyph, ok := glyph3x5[r]
		if !ok {
			glyph = glyph3x5[' ']
		}
		for row := 0; row < 5; row++ {
			bits := glyph[row]
			for colBit := 0; colBit < 3; colBit++ {
				if bits&(1<<(2-colBit)) == 0 {
					continue
				}
				for dy := 0; dy < s; dy++ {
					for dx := 0; dx < s; dx++ {
						px, py := cx+colBit*s+dx, y+row*s+dy
						if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
							img.Set(px, py, col)
						}
					}
				}
			}
		}
		cx += 4 * s
	}
}
