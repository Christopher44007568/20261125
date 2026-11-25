package main

import "math"

// Mode 选择光栅化方式。
type Mode int

const (
	// ModeBinary 是没有覆盖率的单点采样：命中即整像素不透明。
	ModeBinary Mode = iota
	// ModeThick 同样是单点采样，但把半宽强制抬到 0.5 像素，
	// 用来演示“靠加粗掩盖闪烁”的错误做法。
	ModeThick
	// ModeAnalytic 对每个像素做几何裁剪，取真实相交面积作为覆盖率。
	ModeAnalytic
)

// Raster 保存一张浮点覆盖率图，取值范围 [0,1]。
type Raster struct {
	W, H int
	Cov  []float64
}

func newRaster(w, h int) *Raster {
	return &Raster{W: w, H: h, Cov: make([]float64, w*h)}
}

func (r *Raster) idx(x, y int) int { return y*r.W + x }

// Rasterize 按指定模式把多段发丝光栅化到覆盖率图。
// 多段发丝在同一像素交叠时按 1-(1-a)(1-b) 合并，
// 覆盖率永远不会超过 1，也不会被简单相加抹出硬边。
func (r *Raster) Rasterize(hairs []Hair, mode Mode) {
	for _, h := range hairs {
		for _, s := range h.segments {
			switch mode {
			case ModeBinary:
				r.rasterSegmentBinary(s, h.radius, false)
			case ModeThick:
				r.rasterSegmentBinary(s, h.radius, true)
			case ModeAnalytic:
				r.rasterSegmentAnalytic(s, h.radius)
			}
		}
	}
}

func (r *Raster) rasterSegmentBinary(s Segment, radius float64, thick bool) {
	sampleR := radius
	if thick {
		sampleR = math.Max(radius, 0.5)
	}
	minX, minY, maxX, maxY := segmentPixelBox(s, sampleR)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= r.W || y >= r.H {
				continue
			}
			center := Vec{float64(x) + 0.5, float64(y) + 0.5}
			if pointInRibbon(s, sampleR, center) {
				r.Cov[r.idx(x, y)] = 1
			}
		}
	}
}

func (r *Raster) rasterSegmentAnalytic(s Segment, radius float64) {
	quad := ribbonQuad(s, radius)
	minX, minY, maxX, maxY := polygonPixelBox(quad)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if x < 0 || y < 0 || x >= r.W || y >= r.H {
				continue
			}
			clipped := clipRect(quad, pixelRect(x, y))
			if len(clipped) < 3 {
				continue
			}
			area := polygonArea(clipped)
			if area <= 0 {
				continue
			}
			if area > 1 {
				area = 1
			}
			i := r.idx(x, y)
			r.Cov[i] = 1 - (1-r.Cov[i])*(1-area)
		}
	}
}

// segmentPixelBox 返回可能与膨胀后线段相交的整数像素包围盒。
func segmentPixelBox(s Segment, r float64) (minX, minY, maxX, maxY int) {
	loX := math.Min(s.P.X, s.Q.X) - r
	hiX := math.Max(s.P.X, s.Q.X) + r
	loY := math.Min(s.P.Y, s.Q.Y) - r
	hiY := math.Max(s.P.Y, s.Q.Y) + r
	return int(math.Floor(loX)), int(math.Floor(loY)),
		int(math.Ceil(hiX)) - 1, int(math.Ceil(hiY)) - 1
}

// polygonPixelBox 返回多边形包围盒覆盖的整数像素范围。
func polygonPixelBox(poly []Vec) (minX, minY, maxX, maxY int) {
	loX, hiX := poly[0].X, poly[0].X
	loY, hiY := poly[0].Y, poly[0].Y
	for _, v := range poly[1:] {
		loX = math.Min(loX, v.X)
		hiX = math.Max(hiX, v.X)
		loY = math.Min(loY, v.Y)
		hiY = math.Max(hiY, v.Y)
	}
	return int(math.Floor(loX)), int(math.Floor(loY)),
		int(math.Ceil(hiX)) - 1, int(math.Ceil(hiY)) - 1
}

// totalCoverage 对全图覆盖率求和，用于校验解析光栅化的面积守恒：
// 发丝做亚像素平移时，总覆盖率只应随出/入画面边界小幅变化。
func (r *Raster) totalCoverage() float64 {
	var sum float64
	for _, c := range r.Cov {
		sum += c
	}
	return sum
}
