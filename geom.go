package main

import "math"

// Vec 是二维几何向量，所有坐标以像素为单位，允许亚像素值。
type Vec struct {
	X, Y float64
}

func add(a, b Vec) Vec         { return Vec{a.X + b.X, a.Y + b.Y} }
func sub(a, b Vec) Vec         { return Vec{a.X - b.X, a.Y - b.Y} }
func mul(a Vec, k float64) Vec { return Vec{a.X * k, a.Y * k} }
func dot(a, b Vec) float64     { return a.X*b.X + a.Y*b.Y }
func lenVec(a Vec) float64     { return math.Hypot(a.X, a.Y) }

// norm 返回单位方向向量；退化（零长度）输入时返回零向量。
func norm(a Vec) Vec {
	l := lenVec(a)
	if l < 1e-12 {
		return Vec{}
	}
	return Vec{a.X / l, a.Y / l}
}

// perp 返回逆时针旋转 90 度的法向量。
func perp(a Vec) Vec { return Vec{-a.Y, a.X} }

// Rect 是轴对齐矩形，像素 (x,y) 的覆盖区域为 [x,x+1)×[y,y+1)。
type Rect struct {
	X0, Y0, X1, Y1 float64
}

func pixelRect(x, y int) Rect {
	return Rect{float64(x), float64(y), float64(x) + 1, float64(y) + 1}
}

// Segment 表示一段发丝中心线。
type Segment struct {
	P, Q Vec
}

// ribbonQuad 返回线段按半宽 r 膨胀出的有向矩形（平接头），
// 顶点顺序围绕多边形一周。这是发丝的真实几何足迹。
func ribbonQuad(s Segment, r float64) []Vec {
	u := norm(sub(s.Q, s.P))
	n := perp(u)
	return []Vec{
		add(s.P, mul(n, r)),
		add(s.Q, mul(n, r)),
		add(s.Q, mul(n, -r)),
		add(s.P, mul(n, -r)),
	}
}

// pointInRibbon 判断点 v 是否落在有向矩形内（局部坐标系精确判定，
// 不做面积积分，对应无抗锯齿的单点采样光栅化）。
func pointInRibbon(s Segment, r float64, v Vec) bool {
	u := norm(sub(s.Q, s.P))
	rel := sub(v, s.P)
	t := dot(rel, u)
	if t < 0 || t > lenVec(sub(s.Q, s.P)) {
		return false
	}
	return math.Abs(dot(rel, perp(u))) <= r
}

// clipRect 用 Sutherland–Hodgman 算法把任意凸多边形裁剪到轴对齐矩形内，
// 返回裁剪后的凸多边形顶点。多边形完全在矩形外时返回 nil。
func clipRect(poly []Vec, rc Rect) []Vec {
	if len(poly) < 3 {
		return nil
	}
	type edge struct {
		inside func(Vec) bool
		cut    func(a, b Vec) Vec
	}
	edges := []edge{
		{func(v Vec) bool { return v.X >= rc.X0 },
			func(a, b Vec) Vec {
				t := (rc.X0 - a.X) / (b.X - a.X)
				return Vec{rc.X0, a.Y + t*(b.Y-a.Y)}
			}},
		{func(v Vec) bool { return v.X <= rc.X1 },
			func(a, b Vec) Vec {
				t := (rc.X1 - a.X) / (b.X - a.X)
				return Vec{rc.X1, a.Y + t*(b.Y-a.Y)}
			}},
		{func(v Vec) bool { return v.Y >= rc.Y0 },
			func(a, b Vec) Vec {
				t := (rc.Y0 - a.Y) / (b.Y - a.Y)
				return Vec{a.X + t*(b.X-a.X), rc.Y0}
			}},
		{func(v Vec) bool { return v.Y <= rc.Y1 },
			func(a, b Vec) Vec {
				t := (rc.Y1 - a.Y) / (b.Y - a.Y)
				return Vec{a.X + t*(b.X-a.X), rc.Y1}
			}},
	}
	cur := append([]Vec(nil), poly...)
	for _, e := range edges {
		if len(cur) == 0 {
			return nil
		}
		var out []Vec
		for i, b := range cur {
			a := cur[(i+len(cur)-1)%len(cur)]
			ain, bin := e.inside(a), e.inside(b)
			switch {
			case ain && bin:
				out = append(out, b)
			case ain && !bin:
				out = append(out, e.cut(a, b))
			case !ain && bin:
				out = append(out, e.cut(a, b), b)
			}
		}
		cur = out
	}
	return cur
}

// polygonArea 返回多边形面积的绝对值（鞋带公式）。
func polygonArea(poly []Vec) float64 {
	if len(poly) < 3 {
		return 0
	}
	var twice float64
	for i, b := range poly {
		a := poly[(i+len(poly)-1)%len(poly)]
		twice += a.X*b.Y - b.X*a.Y
	}
	return math.Abs(twice) * 0.5
}
