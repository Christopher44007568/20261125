package main

import "math"

var (
	red    = [3]float64{0.95, 0.22, 0.18}
	yellow = [3]float64{1.0, 0.82, 0.1}
	white  = [3]float64{0.96, 0.96, 0.96}
	blue   = [3]float64{0.35, 0.65, 1.0}
)

// drawRing strokes a small circle used to flag lost flyaway tips.
func (im *RGBA) drawRing(cx, cy, r float64, col [3]float64) {
	steps := 18
	for k := 0; k < steps; k++ {
		a0 := float64(k) * 2 * math.Pi / float64(steps)
		a1 := float64(k+1) * 2 * math.Pi / float64(steps)
		im.drawLine(cx+r*math.Cos(a0), cy+r*math.Sin(a0),
			cx+r*math.Cos(a1), cy+r*math.Sin(a1), col, 1, 1.8)
	}
}

// markLostFlyaways projects lost flyaway tips and rings them in red.
func markLostFlyaways(im *RGBA, cam Camera, g *Groom, lost []int) {
	for _, id := range lost {
		s := &g.Strands[id]
		tip := s.Pts[len(s.Pts)-1]
		x, y, z := cam.project(tip, im.W, im.H)
		if z < 0.05 {
			continue
		}
		im.drawRing(x, y, 6, red)
		// short leader line toward the upper-left for label placement
		im.drawDashedLine(x-5, y-5, x-26, y-26, red, 0.95)
	}
}

// silhouetteMask returns hair-covered pixels of a render. The canvas is
// opaque, so hair is distinguished from the dark background by luminance;
// the scalp disc is excluded by its warm red-dominant skin hue.
func silhouetteMask(im *RGBA, bg [3]float64) []bool {
	mask := make([]bool, im.W*im.H)
	for i := 0; i < im.W*im.H; i++ {
		idx := i * 4
		r := float64(im.Px[idx]) / 255
		g := float64(im.Px[idx+1]) / 255
		b := float64(im.Px[idx+2]) / 255
		dist := math.Abs(r-bg[0]) + math.Abs(g-bg[1]) + math.Abs(b-bg[2])
		skin := r > g+0.05 && g >= b
		if dist > 0.12 && !skin {
			mask[i] = true
		}
	}
	return mask
}

// maskIOU computes intersection over union of two silhouette masks.
func maskIOU(a, b []bool) float64 {
	inter, union := 0, 0
	for i := range a {
		switch {
		case a[i] && b[i]:
			inter++
			union++
		case a[i] || b[i]:
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// highlightEnergy returns mean luminance over hair pixels, used as a
// normalized specular/brightness retention metric (area-independent).
func highlightEnergy(im *RGBA, bg [3]float64) float64 {
	e, n := 0.0, 0.0
	for i := 0; i < im.W*im.H; i++ {
		idx := i * 4
		r := float64(im.Px[idx]) / 255
		g := float64(im.Px[idx+1]) / 255
		b := float64(im.Px[idx+2]) / 255
		dist := math.Abs(r-bg[0]) + math.Abs(g-bg[1]) + math.Abs(b-bg[2])
		if dist > 0.12 && !(r > g+0.05 && g >= b) {
			lum := 0.299*float64(im.Px[idx]) + 0.587*float64(im.Px[idx+1]) + 0.114*float64(im.Px[idx+2])
			e += lum
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return e / n
}

// labelPanel draws a dark tag with an English title.
func labelPanel(im *RGBA, x, y int, title string) {
	w := textWidth(title, 2) + 12
	for py := y; py < y+22; py++ {
		for px := x; px < x+w; px++ {
			im.blend(px, py, [4]float64{0.05, 0.05, 0.07, 0.72})
		}
	}
	im.drawText(x+6, y+4, 2, title, white, 1)
}
