package main

import "testing"

func TestCJKRender(t *testing.T) {
	im := newRGBA(900, 110)
	im.fill([3]float64{0.08, 0.09, 0.12})
	im.drawCJKString(16, 12, 64, "飞散发丢失纹理缝隙高光卡片", [3]float64{1, 0.9, 0.4}, 1)
	if err := im.writePNG("/tmp/fonttest/glyphs.png"); err != nil {
		t.Fatal(err)
	}
}
