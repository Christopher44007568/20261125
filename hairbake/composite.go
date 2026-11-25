package main

const renderSize = 420

var bgColor = [3]float64{0.09, 0.1, 0.13}

// scene renders either the reference groom or the baked cards from a view.
func scene(view viewDef, cards []Card, atlas *Atlas, g *Groom, res *BakeResult,
	mode string, seam bool) *RGBA {
	return sceneSize(view, cards, atlas, g, res, mode, seam, renderSize)
}

func sceneSize(view viewDef, cards []Card, atlas *Atlas, g *Groom, res *BakeResult,
	mode string, seam bool, size int) *RGBA {
	cam := makeCamera(view.az, view.el, view.dist, view.focal)
	im := newRGBA(size, size)
	im.fill(bgColor)
	renderHead(im, cam, g.Scalp)
	if mode == "ref" {
		renderStrands(im, cam, g, nil)
		markLostFlyaways(im, cam, g, res.LostFlyaways)
	} else {
		renderCards(im, cam, cards, atlas, seam)
		if seam {
			drawCardSeams(im, cam, cards)
		}
		// Overlay where the lost flyaway tips should have appeared.
		markLostFlyaways(im, cam, g, res.LostFlyaways)
	}
	return im
}

type viewDef struct {
	name  string
	az    float64
	el    float64
	dist  float64
	focal float64
}

func allViews() []viewDef {
	vs := make([]viewDef, 0, len(viewViews))
	for _, v := range viewViews {
		vs = append(vs, viewDef{v.Name, v.Az, v.El, 30, 240})
	}
	return vs
}

// sideBySide builds the front-view reference vs card audit image.
func sideBySide(g *Groom, res *BakeResult, atlas *Atlas) *RGBA {
	front := viewDef{"front", 0, 12, 30, 240}
	ref := scene(front, res.Cards, atlas, g, res, "ref", false)
	card := scene(front, res.Cards, atlas, g, res, "card", true)

	const pad, titleH = 24, 64
	out := newRGBA(renderSize*2+pad*3, renderSize+titleH+pad*2)
	out.fill([3]float64{0.05, 0.06, 0.08})
	out.blit(ref, pad, titleH)
	out.blit(card, renderSize+pad*2, titleH)

	labelPanel(ref, 8, 8, "REFERENCE STRANDS")
	labelPanel(card, 8, 8, "BAKED HAIR CARDS")
	// re-blit after labeling since labelPanel drew on sub-canvases
	out.blit(ref, pad, titleH)
	out.blit(card, renderSize+pad*2, titleH)

	legend(out, pad, 10)
	return out
}

func (im *RGBA) blit(src *RGBA, x, y int) {
	for py := 0; py < src.H; py++ {
		for px := 0; px < src.W; px++ {
			im.set(x+px, y+py, src.get(px, py))
		}
	}
}

// legend draws the bilingual annotation key across the top of an output.
func legend(im *RGBA, x, y int) {
	im.drawRing(float64(x+8), float64(y+9), 5, red)
	im.drawText(x+20, y+3, 2, "LOST FLYAWAY", red, 1)
	cx := x + 20 + textWidth("LOST FLYAWAY", 2) + 18
	cx = im.drawCJKString(cx, y, 20, "飞散发丢失", red, 1) + 18
	im.drawDashedLine(float64(cx), float64(y+9), float64(cx+26), float64(y+9), yellow, 1)
	cx += 34
	im.drawText(cx, y+3, 2, "TEXTURE SEAM", yellow, 1)
	cx += textWidth("TEXTURE SEAM", 2) + 14
	im.drawCJKString(cx, y, 20, "纹理缝隙", yellow, 1)
}
