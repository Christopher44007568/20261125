package main

import "math"

type Mask struct {
	W, H           int
	A              []bool
	X0, Y0, X1, Y1 int
}

type Bounds struct{ X0, Y0, X1, Y1 int }

func (b Bounds) W() int { return b.X1 - b.X0 + 1 }
func (b Bounds) H() int { return b.Y1 - b.Y0 + 1 }

func maskFromAlpha(c *Canvas, threshold float64) Mask {
	m := Mask{W: c.W, H: c.H, A: make([]bool, c.W*c.H)}
	m.X0, m.Y0, m.X1, m.Y1 = c.W, c.H, -1, -1
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			if c.Pix[y*c.W+x].A <= threshold {
				continue
			}
			m.A[y*c.W+x] = true
			m.X0 = min(m.X0, x)
			m.Y0 = min(m.Y0, y)
			m.X1 = max(m.X1, x)
			m.Y1 = max(m.Y1, y)
		}
	}
	return m
}

func unionBounds(a, b Mask) Bounds {
	return Bounds{
		X0: max(0, min(a.X0, b.X0)-10),
		Y0: max(0, min(a.Y0, b.Y0)-10),
		X1: min(a.W-1, max(a.X1, b.X1)+10),
		Y1: min(a.H-1, max(a.Y1, b.Y1)+10),
	}
}

func (m Mask) At(x, y int) bool {
	return x >= 0 && y >= 0 && x < m.W && y < m.H && m.A[y*m.W+x]
}

func edgeMask(m Mask) []bool {
	e := make([]bool, len(m.A))
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			if !m.A[y*m.W+x] {
				continue
			}
			if !m.At(x-1, y) || !m.At(x+1, y) || !m.At(x, y-1) || !m.At(x, y+1) {
				e[y*m.W+x] = true
			}
		}
	}
	return e
}

type MaskStats struct {
	IoU, Xor, AreaDelta float64
}

func compareMasks(a, b Mask) MaskStats {
	areaA, areaB, inter, uni := 0, 0, 0, 0
	for i := range a.A {
		if a.A[i] {
			areaA++
		}
		if b.A[i] {
			areaB++
		}
		if a.A[i] || b.A[i] {
			uni++
			if a.A[i] && b.A[i] {
				inter++
			}
		}
	}
	if areaA == 0 || uni == 0 {
		return MaskStats{}
	}
	return MaskStats{
		IoU:       float64(inter) / float64(uni),
		Xor:       float64(uni-inter) / float64(uni),
		AreaDelta: float64(areaB-areaA) / float64(areaA),
	}
}

type Panel struct{ X, Y, W, H int }

func (p Panel) mapToMask(b Bounds, px, py float64) (int, int, bool) {
	scale := math.Min(float64(p.W)/float64(b.W()), float64(p.H)/float64(b.H()))
	sw := float64(b.W()) * scale
	sh := float64(b.H()) * scale
	ox := float64(p.X) + (float64(p.W)-sw)*0.5
	oy := float64(p.Y) + (float64(p.H)-sh)*0.5
	x := int(float64(b.X0) + (px-ox)/scale)
	y := int(float64(b.Y0) + (py-oy)/scale)
	return x, y, x >= b.X0 && x <= b.X1 && y >= b.Y0 && y <= b.Y1
}

func fillRect(c *Canvas, p Panel, col RGB, alpha float64) {
	for y := max(0, p.Y); y < min(c.H, p.Y+p.H); y++ {
		for x := max(0, p.X); x < min(c.W, p.X+p.W); x++ {
			c.Pix[y*c.W+x] = over(c.Pix[y*c.W+x], premul(col, alpha))
		}
	}
}

func strokeRect(c *Canvas, p Panel, col RGB, alpha float64, width int) {
	for w := 0; w < width; w++ {
		for x := p.X; x < p.X+p.W; x++ {
			c.Set(x, p.Y+w, over(c.At(x, p.Y+w), premul(col, alpha)))
			c.Set(x, p.Y+p.H-1-w, over(c.At(x, p.Y+p.H-1-w), premul(col, alpha)))
		}
		for y := p.Y; y < p.Y+p.H; y++ {
			c.Set(p.X+w, y, over(c.At(p.X+w, y), premul(col, alpha)))
			c.Set(p.X+p.W-1-w, y, over(c.At(p.X+p.W-1-w, y), premul(col, alpha)))
		}
	}
}

func drawMaskPanel(c *Canvas, m Mask, b Bounds, p Panel, top, bottom RGB, edge RGB) {
	e := edgeMask(m)
	for py := p.Y; py < p.Y+p.H; py++ {
		for px := p.X; px < p.X+p.W; px++ {
			cov, edgeCov := 0.0, 0.0
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					x, y, ok := p.mapToMask(b, float64(px)+0.25+float64(sx)*0.5, float64(py)+0.25+float64(sy)*0.5)
					if !ok {
						continue
					}
					if m.At(x, y) {
						cov += 0.25
						if e[y*m.W+x] {
							edgeCov += 0.25
						}
					}
				}
			}
			if cov <= 0.001 {
				continue
			}
			t := sat(float64(py-p.Y) / float64(max(1, p.H-1)))
			col := top.Lerp(bottom, t)
			col = col.Add(edge.Mul(edgeCov * 0.65))
			i := py*c.W + px
			c.Pix[i] = over(c.Pix[i], premul(col, cov*0.92))
		}
	}
}

func drawDiffPanel(c *Canvas, shell, strand Mask, b Bounds, p Panel, shellCol, strandCol, bothCol RGB) {
	for py := p.Y; py < p.Y+p.H; py++ {
		for px := p.X; px < p.X+p.W; px++ {
			var cs, ct float64
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					x, y, ok := p.mapToMask(b, float64(px)+0.25+float64(sx)*0.5, float64(py)+0.25+float64(sy)*0.5)
					if !ok {
						continue
					}
					if shell.At(x, y) {
						cs += 0.25
					}
					if strand.At(x, y) {
						ct += 0.25
					}
				}
			}
			if cs <= 0.001 && ct <= 0.001 {
				continue
			}
			var col RGB
			alpha := 0.0
			if cs > 0 && ct > 0 {
				col = bothCol.Mul(math.Min(cs, ct))
				alpha = math.Min(cs, ct) * 0.70
			}
			if cs > ct {
				col = col.Add(shellCol.Mul((cs - ct) * 0.95))
				alpha = max(alpha, cs*0.90)
			} else if ct > cs {
				col = col.Add(strandCol.Mul((ct - cs) * 0.95))
				alpha = max(alpha, ct*0.90)
			}
			if alpha <= 0 {
				continue
			}
			i := py*c.W + px
			c.Pix[i] = over(c.Pix[i], premul(col, alpha))
		}
	}
}

func drawLegendSwatch(c *Canvas, x, y, w, h int, col RGB, label string, alpha float64) {
	fillRect(c, Panel{X: x, Y: y, W: w, H: h}, col, alpha)
	drawText(c, x+w+10, y-2, label, 2, col, 0.92, 2)
}
