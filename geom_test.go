package main

import (
	"math"
	"testing"
)

func approxEq(t *testing.T, got, want, eps float64, name string) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Fatalf("%s = %.6f, want %.6f (eps %.1e)", name, got, want, eps)
	}
}

func TestClipRectAreaMatchesRibbonFootprint(t *testing.T) {
	// 完全落在 10x10 画面内的水平发丝（半宽 0.2，长 6），
	// 逐像素裁剪面积之和必须等于带状四边形的真实面积。
	s := Segment{Vec{2, 5}, Vec{8, 5}}
	r := 0.2
	quad := ribbonQuad(s, r)
	want := polygonArea(quad)

	var sum float64
	raster := newRaster(10, 10)
	raster.rasterSegmentAnalytic(s, r)
	for _, c := range raster.Cov {
		sum += c
	}
	approxEq(t, sum, want, 1e-9, "clipped area sum")
	approxEq(t, want, 2*r*6, 1e-9, "ribbon area")
}

func TestAnalyticSubpixelCoverageNeverFullyErased(t *testing.T) {
	// 直径 0.3px 的发丝在 1px 平移周期内，任何一帧都必须留下可见覆盖，
	// 证明细发不会被完全抹掉。
	s := Segment{Vec{0.5, 5}, Vec{9.5, 5}}
	for shift := 0; shift < 20; shift++ {
		moved := Segment{
			P: Vec{s.P.X + float64(shift)*0.05, s.P.Y},
			Q: Vec{s.Q.X + float64(shift)*0.05, s.Q.Y},
		}
		raster := newRaster(10, 10)
		raster.rasterSegmentAnalytic(moved, 0.15)
		if raster.totalCoverage() <= 0 {
			t.Fatalf("shift %.2f: hair erased entirely", float64(shift)*0.05)
		}
		for _, c := range raster.Cov {
			if c > 1+1e-12 || c < 0 {
				t.Fatalf("coverage out of [0,1]: %.4f", c)
			}
			if c > 0 && c >= 1 {
				// 亚像素发丝不应产生任何完全不透明像素。
				t.Fatalf("subpixel hair produced fully opaque pixel %.4f", c)
			}
		}
	}
}

func TestBinaryFlickersAnalyticDoesNot(t *testing.T) {
	bin := simulateFrames(ModeBinary)
	aa := simulateFrames(ModeAnalytic)

	if sumInts(toggleCount(bin)) == 0 {
		t.Fatal("binary rasterizer expected frame toggles, got zero")
	}
	if got := sumInts(toggleCount(aa)); got != 0 {
		t.Fatalf("analytic rasterizer had %d toggles, want 0", got)
	}

	binCV := coefficientOfVariation(bin.total)
	aaCV := coefficientOfVariation(aa.total)
	if aaCV >= binCV {
		t.Fatalf("analytic CV %.4f should be smaller than binary CV %.4f", aaCV, binCV)
	}
	if aaCV > 0.01 {
		t.Fatalf("analytic total coverage unstable: CV %.4f", aaCV)
	}
}

func TestThickeningIsWiderThanRealGeometry(t *testing.T) {
	hairs := closeupHairs()
	bin := newRaster(closeCells, closeCells)
	thick := newRaster(closeCells, closeCells)
	aa := newRaster(closeCells, closeCells)
	bin.Rasterize(hairs, ModeBinary)
	thick.Rasterize(hairs, ModeThick)
	aa.Rasterize(hairs, ModeAnalytic)

	binLit, thickLit := 0, 0
	for i := range bin.Cov {
		if bin.Cov[i] > 0 {
			binLit++
		}
		if thick.Cov[i] > 0 {
			thickLit++
		}
		if bin.Cov[i] > 0 && bin.Cov[i] != 1 {
			t.Fatal("binary coverage must be hard 0/1, no antialiasing")
		}
	}
	if thickLit <= binLit {
		t.Fatalf("thickening should cover more pixels: %d vs %d", thickLit, binLit)
	}

	// 解析覆盖率总量接近真实几何面积，而加粗版显著超出。
	if thick.totalCoverage() <= aa.totalCoverage() {
		t.Fatal("overstroke must inflate area beyond analytic footprint")
	}
}

func TestTranslationCoverageConservation(t *testing.T) {
	// 在远离画面边界的内部区域做 1px 周期平移，解析覆盖率总量
	// 理论上恒定；允许浮点误差。
	s := Segment{Vec{4, 20}, Vec{60, 20}}
	totals := make([]float64, 8)
	for i := range totals {
		moved := Segment{
			P: Vec{s.P.X + float64(i)*0.125, s.P.Y},
			Q: Vec{s.Q.X + float64(i)*0.125, s.Q.Y},
		}
		raster := newRaster(64, 40)
		raster.rasterSegmentAnalytic(moved, 0.15)
		totals[i] = raster.totalCoverage()
	}
	for _, v := range totals[1:] {
		approxEq(t, v, totals[0], 1e-9, "conserved coverage")
	}
}

func TestClipRectContainmentAndEmpty(t *testing.T) {
	quad := ribbonQuad(Segment{Vec{2, 2}, Vec{6, 5}}, 0.3)
	clipped := clipRect(quad, Rect{100, 100, 101, 101})
	if clipped != nil {
		t.Fatalf("disjoint clip should be empty, got %v", clipped)
	}

	inside := clipRect(quad, Rect{0, 0, 10, 10})
	approxEq(t, polygonArea(inside), polygonArea(quad), 1e-9, "fully contained clip")
}
