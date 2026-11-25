package main

import (
	"image"
	"image/color"
	"image/draw"
)

type cv struct {
	img  *image.RGBA
	clip image.Rectangle
}

func newCanvas(w, h int, bg color.RGBA) *cv {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	return &cv{img: img, clip: img.Bounds()}
}

func (c *cv) px(x, y int, col color.RGBA) {
	p := image.Pt(x, y)
	if !p.In(c.clip) {
		return
	}
	c.img.SetRGBA(x, y, col)
}

func (c *cv) withClip(r image.Rectangle, fn func()) {
	old := c.clip
	c.clip = c.clip.Intersect(r)
	fn()
	c.clip = old
}

func (c *cv) fill(x0, y0, x1, y1 int, col color.RGBA) {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	r := image.Rect(x0, y0, x1, y1)
	r = r.Intersect(c.img.Bounds())
	r = r.Intersect(c.clip)
	if r.Empty() {
		return
	}
	draw.Draw(c.img, r, &image.Uniform{C: col}, image.Point{}, draw.Src)
}

func (c *cv) strokeRect(x0, y0, x1, y1, t int, col color.RGBA) {
	c.fill(x0, y0, x1, y0+t, col)
	c.fill(x0, y1-t, x1, y1, col)
	c.fill(x0, y0, x0+t, y1, col)
	c.fill(x1-t, y0, x1, y1, col)
}

func (c *cv) line(x0, y0, x1, y1 int, col color.RGBA) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		c.px(x0, y0, col)
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

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (c *cv) dashedRect(x0, y0, x1, y1, dash, gap, t int, col color.RGBA) {
	for x := x0; x < x1; x += dash + gap {
		x2 := minInt(x+dash, x1)
		c.fill(x, y0, x2, y0+t, col)
		c.fill(x, y1-t, x2, y1, col)
	}
	for y := y0; y < y1; y += dash + gap {
		y2 := minInt(y+dash, y1)
		c.fill(x0, y, x0+t, y2, col)
		c.fill(x1-t, y, x1, y2, col)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fontAdvance(f bitmapGlyph) int { return f.w + 1 }

func textWidth(f bitmapGlyph, s string) int {
	if s == "" {
		return 0
	}
	return len(s) * fontAdvance(f)
}

// text draws s with top-left at (x,y) using an embedded bitmap font.
func (c *cv) text(x, y int, f bitmapGlyph, col color.RGBA, s string) int {
	adv := fontAdvance(f)
	bpr := (f.w + 7) / 8
	cx := x
	for _, ch := range s {
		glyph, ok := f.rows[int(ch)]
		if ok {
			for row := 0; row < f.h; row++ {
				for bitX := 0; bitX < f.w; bitX++ {
					idx := row*bpr + bitX/8
					if idx >= len(glyph) {
						break
					}
					bits := glyph[idx]
					if bits&(0x80>>uint(bitX%8)) != 0 {
						c.px(cx+bitX, y+row, col)
					}
				}
			}
		}
		cx += adv
	}
	return cx
}

func (c *cv) textRight(xRight, y int, f bitmapGlyph, col color.RGBA, s string) {
	c.text(xRight-textWidth(f, s), y, f, col, s)
}

func (c *cv) textCentered(xCenter, y int, f bitmapGlyph, col color.RGBA, s string) {
	c.text(xCenter-textWidth(f, s)/2, y, f, col, s)
}
