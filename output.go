package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

const (
	closeCells = 8
	closeScale = 56
	headerH    = 26
	panelGap   = 16
)

var (
	gridColor    = color.RGBA{48, 52, 62, 255}
	outlineColor = color.RGBA{90, 170, 255, 255}
	labelLight   = color.RGBA{12, 14, 18, 255}
	labelDark    = color.RGBA{220, 224, 230, 255}
)

// renderCloseup 输出近景放大对比：无覆盖率二元采样、加粗二元采样、
// 逐像素裁剪的解析覆盖率。每个逻辑像素放大 56 倍并叠加网格。
func renderCloseup() *image.RGBA {
	hairs := closeupHairs()
	panelPx := closeCells * closeScale
	width := panelPx*3 + panelGap*4
	height := headerH + panelPx + panelGap
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(img, 0, 0, width, height, color.RGBA{10, 11, 15, 255})

	titles := []string{
		"A BINARY R.18",
		"B THICK R.50",
		"C ANALYTIC",
	}
	modes := []Mode{ModeBinary, ModeThick, ModeAnalytic}
	for i, mode := range modes {
		ox := panelGap + i*(panelPx+panelGap)
		drawText(img, ox+2, 8, titles[i], labelDark)
		r := newRaster(closeCells, closeCells)
		r.Rasterize(hairs, mode)
		frame := rasterToImage(r)
		big := nearestScale(frame, closeScale)
		for y := 0; y < panelPx; y++ {
			for x := 0; x < panelPx; x++ {
				img.SetRGBA(ox+x, headerH+y, big.RGBAAt(x, y))
			}
		}
		for g := 0; g <= closeCells; g++ {
			vLine(img, ox+g*closeScale, headerH, headerH+panelPx, gridColor)
			hLine(img, ox, ox+panelPx, headerH+g*closeScale, gridColor)
		}
		drawRibbonOutlines(img, hairs, ox, headerH)
		if mode == ModeAnalytic {
			drawCoverageLabels(img, r, ox, headerH)
		}
	}
	return img
}

func drawRibbonOutlines(img *image.RGBA, hairs []Hair, ox, oy int) {
	for _, h := range hairs {
		for _, s := range h.segments {
			quad := ribbonQuad(s, h.radius)
			for i := range quad {
				a, b := quad[i], quad[(i+1)%len(quad)]
				drawLine(img, ox, oy, a, b, outlineColor)
			}
		}
	}
}

func drawCoverageLabels(img *image.RGBA, r *Raster, ox, oy int) {
	for y := 0; y < closeCells; y++ {
		for x := 0; x < closeCells; x++ {
			cov := r.Cov[r.idx(x, y)]
			if cov < 0.02 {
				continue
			}
			txt := fmt.Sprintf("%.2f", cov)
			tx := ox + x*closeScale + (closeScale-5*len(txt)+len(txt)-1)/2
			ty := oy + y*closeScale + (closeScale-7)/2
			col := labelDark
			if cov > 0.55 {
				col = labelLight
			}
			drawText(img, tx, ty, txt, col)
		}
	}
}

// drawLine 在放大面板内画一条蓝色几何轮廓线（Bresenham，仅用于标注）。
func drawLine(img *image.RGBA, ox, oy int, a, b Vec, c color.RGBA) {
	x0 := int(math.Round(a.X*closeScale)) + ox
	y0 := int(math.Round(a.Y*closeScale)) + oy
	x1 := int(math.Round(b.X*closeScale)) + ox
	y1 := int(math.Round(b.Y*closeScale)) + oy
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
		img.SetRGBA(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
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

// FrameStats 保存单帧单像素的覆盖率时序与全帧统计。
type FrameStats struct {
	coverage [][]float64 // [frame][pixel]
	total    []float64
	minPixel []float64
	litCount []int
}

// simulateFrames 对 16 帧做亚像素平移并分别用两种模式光栅化。
func simulateFrames(mode Mode) *FrameStats {
	base := distantHairs()
	fs := &FrameStats{
		coverage: make([][]float64, frames),
		total:    make([]float64, frames),
		minPixel: make([]float64, frames),
	}
	for f := 0; f < frames; f++ {
		dx, dy := frameShift(f)
		r := newRaster(distantW, distantH)
		r.Rasterize(translateHairs(base, dx, dy), mode)
		fs.coverage[f] = append([]float64(nil), r.Cov...)
		fs.total[f] = r.totalCoverage()
		min := 1.0
		lit := 0
		for _, c := range r.Cov {
			if c > 0.001 {
				lit++
				if c < min {
					min = c
				}
			}
		}
		if lit == 0 {
			min = 0
		}
		fs.minPixel[f] = min
		fs.litCount = append(fs.litCount, lit)
	}
	return fs
}

// flickerAmplitude 返回每个像素在整段动画里的覆盖率极差。
func flickerAmplitude(fs *FrameStats) []float64 {
	n := distantW * distantH
	out := make([]float64, n)
	for p := 0; p < n; p++ {
		lo, hi := fs.coverage[0][p], fs.coverage[0][p]
		for f := 1; f < frames; f++ {
			v := fs.coverage[f][p]
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
		out[p] = hi - lo
	}
	return out
}

// toggleCount 统计每个像素相邻帧之间覆盖率变化超过阈值的次数。
func toggleCount(fs *FrameStats) []int {
	n := distantW * distantH
	out := make([]int, n)
	for p := 0; p < n; p++ {
		for f := 1; f < frames; f++ {
			if math.Abs(fs.coverage[f][p]-fs.coverage[f-1][p]) > 0.25 {
				out[p]++
			}
		}
	}
	return out
}
