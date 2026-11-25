package main

import (
	"image"
	"image/color"
)

// bgColor / fgColor 是覆盖率可视化的底色与发丝色。
func bgColor() color.RGBA { return color.RGBA{18, 20, 26, 255} }
func fgColor() color.RGBA { return color.RGBA{232, 236, 244, 255} }

func blendOver(dst, src color.RGBA, alpha float64) color.RGBA {
	if alpha <= 0 {
		return dst
	}
	if alpha > 1 {
		alpha = 1
	}
	a := uint8(alpha*255 + 0.5)
	inv := float64(255-a) / 255
	af := float64(a) / 255
	return color.RGBA{
		R: uint8(float64(src.R)*af + float64(dst.R)*inv + 0.5),
		G: uint8(float64(src.G)*af + float64(dst.G)*inv + 0.5),
		B: uint8(float64(src.B)*af + float64(dst.B)*inv + 0.5),
		A: 255,
	}
}

// rasterToImage 把覆盖率图渲染为带背景的不透明 RGBA 图，
// 覆盖率 alpha 直接作为源透明度，细发因此呈现灰度过渡而不是亮线。
func rasterToImage(r *Raster) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	bg, fg := bgColor(), fgColor()
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			img.SetRGBA(x, y, blendOver(bg, fg, r.Cov[r.idx(x, y)]))
		}
	}
	return img
}

// nearestScale 最近邻放大，保留像素颗粒，不做平滑涂抹。
func nearestScale(src *image.RGBA, k int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w*k, h*k))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.RGBAAt(x, y)
			for dy := 0; dy < k; dy++ {
				for dx := 0; dx < k; dx++ {
					dst.SetRGBA(x*k+dx, y*k+dy, c)
				}
			}
		}
	}
	return dst
}

// fillRect 画实心矩形。
func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// hLine / vLine 画 1px 参考线。
func hLine(img *image.RGBA, x0, x1, y int, c color.RGBA) {
	for x := x0; x < x1; x++ {
		img.SetRGBA(x, y, c)
	}
}

func vLine(img *image.RGBA, x, y0, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		img.SetRGBA(x, y, c)
	}
}

// drawText 在指定位置以 5x7 位图字体绘制字符串。
// 字体为列主序：每个字节是一列，位 0 对应最上行。
func drawText(img *image.RGBA, x, y int, s string, c color.RGBA) {
	for _, ch := range s {
		glyph, ok := font5x7[ch]
		if !ok {
			x += 6
			continue
		}
		for col := 0; col < 5; col++ {
			bits := glyph[col]
			for row := 0; row < 7; row++ {
				if bits&(1<<uint(row)) != 0 {
					img.SetRGBA(x+col, y+row, c)
				}
			}
		}
		x += 6
	}
}

// heatColor 把 [0,1] 的闪烁强度映射为黑→红→黄热力色。
func heatColor(t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	if t < 0.5 {
		v := uint8(t*2*255 + 0.5)
		return color.RGBA{v, 0, 0, 255}
	}
	v := uint8((t-0.5)*2*255 + 0.5)
	return color.RGBA{255, v, 0, 255}
}
