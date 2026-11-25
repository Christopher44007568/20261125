package main

// montage builds the multi-angle puzzle: two rows (reference, cards) across
// five camera views. Card-row panels include the seam overlay.
func montage(g *Groom, res *BakeResult, atlas *Atlas) *RGBA {
	views := allViews()
	const cell = 260
	const pad = 18
	const headH = 46
	cols := len(views)
	const footH = 34
	out := newRGBA(cols*cell+(cols+1)*pad, 2*cell+3*pad+headH+footH)
	out.fill([3]float64{0.05, 0.06, 0.08})

	for i, v := range views {
		x := pad + i*(cell+pad)
		ref := sceneSize(v, res.Cards, atlas, g, res, "ref", false, cell)
		card := sceneSize(v, res.Cards, atlas, g, res, "card", true, cell)
		labelPanel(ref, 6, 6, "REF "+v.name)
		labelPanel(card, 6, 6, "CARD "+v.name)
		out.blit(ref, x, headH)
		out.blit(card, x, headH+cell+pad)
	}
	// Row titles.
	out.drawText(pad, 14, 2, "TOP ROW: REFERENCE STRANDS   BOTTOM ROW: BAKED CARDS", white, 1)
	// Compact bilingual legend strip.
	ly := headH + 2*cell + 2*pad + 10
	out.drawRing(float64(pad+8), float64(ly+7), 4, red)
	out.drawText(pad+18, ly+2, 2, "LOST FLYAWAY", red, 1)
	lx := pad + 18 + textWidth("LOST FLYAWAY", 2) + 14
	lx = out.drawCJKString(lx, ly, 18, "飞散发丢失", red, 1) + 16
	out.drawDashedLine(float64(lx), float64(ly+7), float64(lx+22), float64(ly+7), yellow, 1)
	lx += 30
	out.drawText(lx, ly+2, 2, "SEAM", yellow, 1)
	lx += textWidth("SEAM", 2) + 12
	out.drawCJKString(lx, ly, 18, "纹理缝隙", yellow, 1)
	return out
}
