package main

import "math"

const (
	atlasSize = 1024
	atlasCols = 2
	atlasRows = 5
	atlasPad  = 10 // empty gutter between islands -> visible texture seams
)

// Atlas holds the two baked textures plus per-card pixel rectangles.
type Atlas struct {
	Size int
	RGBA []uint8 // color.rgb + coverage alpha
	Flow []uint8 // tangent.xy (0..1 encoded) + specular mask in B
}

func newAtlas(size int) *Atlas {
	a := &Atlas{Size: size}
	a.RGBA = make([]uint8, size*size*4)
	a.Flow = make([]uint8, size*size*4)
	// Flow neutral direction points down-island (v direction, tangent ~ (0,1)).
	for i := 0; i < size*size; i++ {
		a.Flow[i*4+0] = 128
		a.Flow[i*4+1] = 255
		a.Flow[i*4+2] = 0
		a.Flow[i*4+3] = 255
	}
	return a
}

// packAtlas assigns each card an island rectangle in a fixed 2x5 grid.
func packAtlas(cards []Card) {
	cellW := atlasSize / atlasCols
	cellH := atlasSize / atlasRows
	for i := range cards {
		col := i % atlasCols
		row := i / atlasCols
		cards[i].Rect = AtlasRect{
			X: col*cellW + atlasPad,
			Y: row*cellH + atlasPad,
			W: cellW - 2*atlasPad,
			H: cellH - 2*atlasPad,
		}
	}
}

// bakeAtlas rasterizes every assigned strand into its card island. Strands are
// drawn root(top) -> tip(bottom); per-pixel tangent is stored in the flow map.
func bakeAtlas(g *Groom, cards []Card) *Atlas {
	a := newAtlas(atlasSize)
	for ci := range cards {
		card := &cards[ci]
		r := card.Rect
		for _, si := range card.StrandIDs {
			s := &g.Strands[si]
			prevX, prevY := -1, -1
			for j := 0; j <= segments; j++ {
				p := s.Pts[j]
				d := p.Sub(card.Center[j]).Dot(card.Across[j])
				u := 0.5 + d/(2*card.HalfW[j])
				v := float64(j) / segments
				px := r.X + int(clampf(u, 0, 1)*float64(r.W-1))
				py := r.Y + int(v*float64(r.H-1))
				tan := tangentAt(card.Center, j)
				// Encode the strand's actual across-axis tangent component so
				// the flow map visibly follows strand direction; normalize to
				// unit with the along-v component.
				tx := clampf(tan.Dot(card.Across[j])*2.2, -1, 1)
				ty := math.Sqrt(math.Max(0, 1-tx*tx))
				width := s.Thickness * float64(r.W) * 0.55
				discSplat(a, px, py, width, s, tan, tx, ty, r)
				if j > 0 {
					lineSplat(a, prevX, prevY, px, py, width, s, tan, tx, ty, r)
				}
				prevX, prevY = px, py
			}
		}
		fillCardBase(a, card, g)
	}
	return a
}

// fillCardBase lays a faint translucent veil across each island so card
// interiors read as hair mass rather than sparse independent strands.
func fillCardBase(a *Atlas, card *Card, g *Groom) {
	r := card.Rect
	var col [3]float64
	for _, si := range card.StrandIDs {
		for c := 0; c < 3; c++ {
			col[c] += g.Strands[si].Color[c]
		}
	}
	n := float64(len(card.StrandIDs))
	for c := 0; c < 3; c++ {
		col[c] /= n
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			idx := (y*a.Size + x) * 4
			if a.RGBA[idx+3] == 0 {
				a.RGBA[idx+0] = uint8(col[0] * 255)
				a.RGBA[idx+1] = uint8(col[1] * 255)
				a.RGBA[idx+2] = uint8(col[2] * 255)
				a.RGBA[idx+3] = 46
				a.Flow[idx+0] = 128
				a.Flow[idx+1] = 255
				a.Flow[idx+2] = 40
			}
		}
	}
}

func insideRect(x, y int, r AtlasRect) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

func discSplat(a *Atlas, cx, cy int, rad float64, s *Strand, tan Vec3, tx, ty float64, r AtlasRect) {
	radius := int(math.Ceil(rad)) + 1
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if !insideRect(x, y, r) {
				continue
			}
			dx := float64(x-cx) + 0.5
			dy := float64(y-cy) + 0.5
			cov := clampf(rad-math.Sqrt(dx*dx+dy*dy)+0.5, 0, 1)
			if cov > 0 {
				paint(a, x, y, cov, s, tx, ty)
			}
		}
	}
}

func lineSplat(a *Atlas, x0, y0, x1, y1 int, width float64, s *Strand, tan Vec3, tx, ty float64, r AtlasRect) {
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	steps := int(math.Max(math.Abs(dx), math.Abs(dy))) + 1
	for k := 0; k <= steps; k++ {
		t := float64(k) / float64(steps)
		cx := int(math.Round(float64(x0) + dx*t))
		cy := int(math.Round(float64(y0) + dy*t))
		discSplat(a, cx, cy, width, s, tan, tx, ty, r)
	}
}

// paint alpha-over blends one strand sample into both atlas textures.
func paint(a *Atlas, x, y int, cov float64, s *Strand, tx, ty float64) {
	idx := (y*a.Size + x) * 4
	oldA := float64(a.RGBA[idx+3]) / 255
	newA := cov * 0.85
	outA := newA + oldA*(1-newA)
	if outA < 1e-4 {
		return
	}
	for c := 0; c < 3; c++ {
		nc := s.Color[c]
		oc := float64(a.RGBA[idx+c]) / 255
		blended := (nc*newA + oc*oldA*(1-newA)) / outA
		a.RGBA[idx+c] = uint8(clampf(blended, 0, 1) * 255)
	}
	a.RGBA[idx+3] = uint8(clampf(outA, 0, 1) * 255)

	// Flow tangent encoded as 0..1; specular strength accumulates in blue.
	a.Flow[idx+0] = uint8(clampf(tx*0.5+0.5, 0, 1) * 255)
	a.Flow[idx+1] = uint8(clampf(ty*0.5+0.5, 0, 1) * 255)
	spec := float64(a.Flow[idx+2])/255 + s.Spec*newA*0.6
	a.Flow[idx+2] = uint8(clampf(spec, 0, 1) * 255)
}
