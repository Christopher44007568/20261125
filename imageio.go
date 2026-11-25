package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func Downsample2x(fb *FrameBuffer) *image.RGBA {
	w := fb.W / 2
	h := fb.H / 2
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := x * 2
			sy := y * 2
			var r, g, b uint32
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					o := (sy+dy)*fb.Image.Stride + (sx+dx)*4
					r += uint32(fb.Image.Pix[o+0])
					g += uint32(fb.Image.Pix[o+1])
					b += uint32(fb.Image.Pix[o+2])
				}
			}
			o := y*out.Stride + x*4
			out.Pix[o+0] = uint8(r / 4)
			out.Pix[o+1] = uint8(g / 4)
			out.Pix[o+2] = uint8(b / 4)
			out.Pix[o+3] = 255
		}
	}
	return out
}

func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func WriteContactSheet(path string, frames []*image.RGBA, cols int) error {
	if len(frames) == 0 {
		return nil
	}
	frameW := frames[0].Bounds().Dx()
	frameH := frames[0].Bounds().Dy()
	gap := 6
	rows := (len(frames) + cols - 1) / cols
	sheetW := cols*frameW + (cols+1)*gap
	sheetH := rows*frameH + (rows+1)*gap
	sheet := image.NewRGBA(image.Rect(0, 0, sheetW, sheetH))
	fillRGBA(sheet, color.RGBA{R: 226, G: 230, B: 235, A: 255})

	for i, frame := range frames {
		col := i % cols
		row := i / cols
		x := gap + col*(frameW+gap)
		y := gap + row*(frameH+gap)
		copyRGBA(sheet, frame, x, y)
		drawFrameBadge(sheet, x+6, y+6, i)
	}
	return SavePNG(path, sheet)
}

func fillRGBA(img *image.RGBA, c color.RGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+0] = c.R
		img.Pix[i+1] = c.G
		img.Pix[i+2] = c.B
		img.Pix[i+3] = c.A
	}
}

func copyRGBA(dst, src *image.RGBA, x0, y0 int) {
	for y := 0; y < src.Bounds().Dy(); y++ {
		for x := 0; x < src.Bounds().Dx(); x++ {
			si := y*src.Stride + x*4
			di := (y0+y)*dst.Stride + (x0+x)*4
			dst.Pix[di+0] = src.Pix[si+0]
			dst.Pix[di+1] = src.Pix[si+1]
			dst.Pix[di+2] = src.Pix[si+2]
			dst.Pix[di+3] = 255
		}
	}
}

var font5x7 = map[rune][]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
	'6': {"00110", "01000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00010", "01100"},
	'.': {"00000", "00000", "00000", "00000", "00000", "00110", "00110"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'G': {"01111", "10000", "10000", "10111", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
}

func drawFrameBadge(img *image.RGBA, x, y, frame int) {
	label := itoa3(frame)
	w := len(label)*6 - 1 + 8
	h := 15
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			o := yy*img.Stride + xx*4
			if o >= 0 && o+3 < len(img.Pix) {
				img.Pix[o+3] = 210
				img.Pix[o+0] = 20
				img.Pix[o+1] = 24
				img.Pix[o+2] = 28
			}
		}
	}
	for i, r := range label {
		glyph := font5x7[r]
		for gy, row := range glyph {
			for gx, cell := range row {
				if cell != '1' {
					continue
				}
				px := x + 4 + i*6 + gx
				py := y + 4 + gy
				o := py*img.Stride + px*4
				img.Pix[o+0] = 245
				img.Pix[o+1] = 248
				img.Pix[o+2] = 250
				img.Pix[o+3] = 255
			}
		}
	}
}

func itoa3(v int) string {
	if v >= 100 {
		return string(rune('0'+v/100)) + string(rune('0'+(v/10)%10)) + string(rune('0'+v%10))
	}
	if v >= 10 {
		return string(rune('0'+v/10)) + string(rune('0'+v%10))
	}
	return "00" + string(rune('0'+v))
}
