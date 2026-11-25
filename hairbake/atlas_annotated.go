package main

// annotateAtlas draws island rectangles and gutters over the color atlas and
// stamps per-card ids so the texture seams are explicit.
func annotateAtlas(a *Atlas, cards []Card) *RGBA {
	im := atlasToRGBA(a, false)
	// Checkerboard behind transparent texels to expose coverage gaps.
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			idx := (y*im.W + x) * 4
			if im.Px[idx+3] == 0 {
				c := uint8(26)
				if (x/16+y/16)%2 == 0 {
					c = 40
				}
				im.Px[idx+0], im.Px[idx+1], im.Px[idx+2], im.Px[idx+3] = c, c, c+6, 255
			}
		}
	}
	for _, card := range cards {
		r := card.Rect
		// Yellow rectangle: baked island boundary; outside it is empty gutter.
		im.drawRect(r.X-atlasPad/2, r.Y-atlasPad/2,
			r.W+atlasPad, r.H+atlasPad, yellow, 0.95)
		tag := "CARD " + pad2(card.ID)
		im.drawText(r.X+6, r.Y+r.H-20, 2, tag, white, 0.95)
	}
	// Grid lines make the 2x5 packing and inter-card seams obvious.
	for c := 1; c < atlasCols; c++ {
		x := c * (atlasSize / atlasCols)
		im.drawDashedLine(float64(x), 0, float64(x), atlasSize, yellow, 0.85)
	}
	for r := 1; r < atlasRows; r++ {
		y := r * (atlasSize / atlasRows)
		im.drawDashedLine(0, float64(y), atlasSize, float64(y), yellow, 0.85)
	}
	// Title is drawn into the top gutter band on a dark tag so it stays legible.
	title := "HAIR CARD ATLAS - YELLOW: ISLAND EDGES / TEXTURE SEAMS"
	tw := textWidth(title, 2) + 16
	for py := 4; py < 26; py++ {
		for px := 8; px < 8+tw; px++ {
			im.blend(px, py, [4]float64{0.04, 0.04, 0.06, 0.78})
		}
	}
	im.drawText(14, 8, 2, title, yellow, 1)
	return im
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [12]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
