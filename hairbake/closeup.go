package main

// closeup renders a zoomed front view focused on the flyaway-rich upper
// silhouette, then crops the upper-right quadrant where lost tips cluster.
func closeup(g *Groom, res *BakeResult, atlas *Atlas) *RGBA {
	view := viewDef{"closeup", 28, 6, 21, 460}
	ref := scene(view, res.Cards, atlas, g, res, "ref", false)
	card := scene(view, res.Cards, atlas, g, res, "card", true)

	// Crop window: left-of-center upper silhouette where flyaway arcs diverge.
	cropX, cropY, cropW, cropH := 24, renderSize/2-150, renderSize/2+20, renderSize/2+30
	cr := crop(ref, cropX, cropY, cropW, cropH)
	cc := crop(card, cropX, cropY, cropW, cropH)
	scale := 2
	cr = scaleUp(cr, scale)
	cc = scaleUp(cc, scale)

	const pad, titleH = 24, 64
	out := newRGBA(cr.W*2+pad*3, cr.H+titleH+pad*2)
	out.fill([3]float64{0.05, 0.06, 0.08})
	labelPanel(cr, 8, 8, "REFERENCE CLOSEUP")
	labelPanel(cc, 8, 8, "CARD CLOSEUP")
	out.blit(cr, pad, titleH)
	out.blit(cc, cr.W+pad*2, titleH)

	// CJK tags directly on the crops.
	cr.drawCJKString(14, cc.H-34, 22, "飞散发", white, 1)
	cc.drawCJKString(14, cc.H-34, 22, "丢失", red, 1)
	cc.drawCJKString(14, cc.H-64, 22, "高光", yellow, 1)
	out.blit(cr, pad, titleH)
	out.blit(cc, cr.W+pad*2, titleH)
	legend(out, pad, 10)
	return out
}

func crop(im *RGBA, x, y, w, h int) *RGBA {
	out := newRGBA(w, h)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			out.set(px, py, im.get(x+px, y+py))
		}
	}
	return out
}

func scaleUp(im *RGBA, s int) *RGBA {
	out := newRGBA(im.W*s, im.H*s)
	for py := 0; py < out.H; py++ {
		for px := 0; px < out.W; px++ {
			out.set(px, py, im.get(px/s, py/s))
		}
	}
	return out
}
