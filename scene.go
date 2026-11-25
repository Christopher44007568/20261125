package main

// Hair 是一根发丝：中心线折线加亚像素半径（半径单位为像素，
// 直径 = 2*radius，可远小于 1 像素）。
type Hair struct {
	radius   float64
	points   []Vec
	segments []Segment
}

func makeHair(radius float64, points []Vec) Hair {
	h := Hair{radius: radius, points: points}
	for i := 0; i+1 < len(points); i++ {
		h.segments = append(h.segments, Segment{points[i], points[i+1]})
	}
	return h
}

// translateHairs 对整组发丝做亚像素平移，保持半径不变。
func translateHairs(hairs []Hair, dx, dy float64) []Hair {
	out := make([]Hair, len(hairs))
	for i, h := range hairs {
		pts := make([]Vec, len(h.points))
		for j, p := range h.points {
			pts[j] = Vec{p.X + dx, p.Y + dy}
		}
		out[i] = makeHair(h.radius, pts)
	}
	return out
}

// closeupHairs 是近景：两根亚像素发丝斜穿像素网格，半径 0.18px，
// 即直径 0.36px，不足一个像素。
func closeupHairs() []Hair {
	return []Hair{
		makeHair(0.18, []Vec{
			{0.4, 6.4},
			{3.1, 4.7},
			{5.9, 2.9},
			{7.6, 1.1},
		}),
		makeHair(0.14, []Vec{
			{0.6, 1.0},
			{2.7, 3.2},
			{4.9, 5.1},
			{7.4, 7.2},
		}),
	}
}

// distantHairs 是远景基础姿态：两根发丝贯穿画面，之后逐帧做亚像素平移。
func distantHairs() []Hair {
	return []Hair{
		makeHair(0.15, []Vec{
			{-6, 34},
			{18, 24},
			{40, 15},
			{70, 5},
		}),
		makeHair(0.13, []Vec{
			{-8, 9},
			{16, 12},
			{42, 18},
			{72, 24},
		}),
	}
}

const (
	distantW = 64
	distantH = 40
	frames   = 16
)

// frameShift 返回第 f 帧相对第 0 帧的亚像素位移。
// 水平 0.125px/帧（共 2px），垂直 0.0625px/帧，故意制造
// 单像素采样最敏感的亚像素相位变化。
func frameShift(f int) (dx, dy float64) {
	return float64(f) * 0.125, float64(f) * 0.0625
}
