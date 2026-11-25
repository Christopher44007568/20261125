package main

import "math"

// Card is a ribbon hair card: a center polyline with a width axis per knot.
type Card struct {
	ID        int
	StrandIDs []int
	Center    []Vec3
	Across    []Vec3 // unit width axis at each knot
	HalfW     []float64
	Rect      AtlasRect
	SpecMean  float64
	FlyCount  int
}

// AtlasRect is the card's island rectangle in the atlas (pixels).
type AtlasRect struct {
	X, Y, W, H int
}

// BakeResult holds the reduced representation plus reduction statistics.
type BakeResult struct {
	Cards         []Card
	Assign        []int
	MaxDist       float64
	MeanDist      float64
	LostFlyaways  []int
	KeptFlyaways  []int
	SilhouetteIOU float64 // filled later by the render metric pass
	SpecRetention float64
}

const cardCount = 10

// bakeGroom clusters strands into a small number of ribbon cards. Card extents
// are derived from the outermost assigned strands so silhouette coverage is
// prioritized even with a tight card budget.
func bakeGroom(g *Groom) *BakeResult {
	n := len(g.Strands)
	k := cardCount
	rng := randSeed{771}

	centers := initCenters(g, k, &rng)
	assign := make([]int, n)
	for iter := 0; iter < 30; iter++ {
		moved := false
		for i, s := range g.Strands {
			best, bestD := 0, math.Inf(1)
			for c := 0; c < k; c++ {
				d := s.Root.Sub(centers[c]).Len()
				if d < bestD {
					best, bestD = c, d
				}
			}
			if assign[i] != best {
				assign[i] = best
				moved = true
			}
		}
		if !moved && iter > 0 {
			break
		}
		for c := 0; c < k; c++ {
			var sum Vec3
			cnt := 0
			for i := range g.Strands {
				if assign[i] == c {
					sum = sum.Add(g.Strands[i].Root)
					cnt++
				}
			}
			if cnt > 0 {
				centers[c] = sum.Scale(1 / float64(cnt))
			}
		}
	}

	cards := make([]Card, 0, k)
	for c := 0; c < k; c++ {
		ids := make([]int, 0)
		for i := range g.Strands {
			if assign[i] == c {
				ids = append(ids, i)
			}
		}
		if len(ids) == 0 {
			continue
		}
		card := buildCard(g, ids, len(cards))
		cards = append(cards, card)
	}

	res := &BakeResult{Cards: cards, Assign: assign}
	collectReductionStats(g, res)
	return res
}

// initCenters uses deterministic k-means++ seeding on strand roots.
func initCenters(g *Groom, k int, rng *randSeed) []Vec3 {
	centers := make([]Vec3, 0, k)
	first := rng.s % uint64(len(g.Strands))
	centers = append(centers, g.Strands[first].Root)
	dist := make([]float64, len(g.Strands))
	for len(centers) < k {
		total := 0.0
		for i, s := range g.Strands {
			d := math.Inf(1)
			for _, c := range centers {
				if dd := s.Root.Sub(c).Len(); dd < d {
					d = dd
				}
			}
			dist[i] = d * d
			total += dist[i]
		}
		pick := rng.float() * total
		acc := 0.0
		idx := len(g.Strands) - 1
		for i, d := range dist {
			acc += d
			if acc >= pick {
				idx = i
				break
			}
		}
		centers = append(centers, g.Strands[idx].Root)
	}
	return centers
}

func buildCard(g *Groom, ids []int, id int) Card {
	card := Card{ID: id, StrandIDs: ids}
	knots := segments + 1
	card.Center = make([]Vec3, knots)
	card.Across = make([]Vec3, knots)
	card.HalfW = make([]float64, knots)

	for j := 0; j < knots; j++ {
		var sum Vec3
		for _, si := range ids {
			sum = sum.Add(g.Strands[si].Pts[j])
		}
		card.Center[j] = sum.Scale(1 / float64(len(ids)))
	}

	for j := 0; j < knots; j++ {
		tan := tangentAt(card.Center, j)
		// Width axis follows the scalp's horizontal "around the head" ring
		// direction (up x radial), projected off the strand tangent. This
		// keeps ribbons tangent to the head instead of forming flat planes.
		radial := card.Center[j].Sub(g.Scalp.Center)
		radial.Y = 0
		radial = radial.Norm()
		ring := v3(0, 1, 0).Cross(radial)
		across := ring.Sub(tan.Scale(ring.Dot(tan)))
		if across.Len() < 0.25 {
			across = v3(1, 0, 0).Sub(tan.Scale(tan.X))
		}
		across = across.Norm()
		// Keep orientation consistent knot-to-knot.
		if j > 0 && across.Dot(card.Across[j-1]) < 0 {
			across = across.Neg()
		}
		card.Across[j] = across

		// Silhouette priority: extend to the outermost strand + thickness margin.
		maxExt := 0.0
		for _, si := range ids {
			d := g.Strands[si].Pts[j].Sub(card.Center[j]).Dot(across)
			if math.Abs(d) > maxExt {
				maxExt = math.Abs(d)
			}
			s := g.Strands[si]
			if s.Flyaway {
				card.FlyCount++
			}
			card.SpecMean += s.Spec
		}
		// Tip taper: narrow the ribbon at the free end so it hugs the groom
		// instead of flaring into wide wings, while keeping the root broad for
		// silhouette coverage.
		t := float64(j) / float64(knots-1)
		taper := 1.0 - 0.35*t
		// Overlap factor lets adjacent ribbons interpenetrate so the card
		// shell has no silhouette gaps under the tight 10-card budget.
		card.HalfW[j] = (maxExt + 0.12) * taper * 1.3
	}
	card.SpecMean /= float64(len(ids) * knots)
	card.FlyCount /= knots
	hugScalp(g, &card)
	return card
}

// hugScalp pushes the upper ribbon knots outward along the scalp radial so
// the card shell covers the crown instead of leaving bare skin between the
// strand roots and the ribbon surface.
func hugScalp(g *Groom, card *Card) {
	for j := range card.Center {
		t := float64(j) / float64(len(card.Center)-1)
		if t > 0.85 {
			return
		}
		n := g.Scalp.surfaceNormal(card.Center[j])
		// Lift upper knots to sit just above the ellipsoid, guaranteeing the
		// ribbons cover the crown rather than starting under the hairline.
		push := (0.85 - t) * 3.4
		card.Center[j] = card.Center[j].Add(n.Scale(push))
	}
}

func tangentAt(poly []Vec3, j int) Vec3 {
	if j == 0 {
		return poly[1].Sub(poly[0]).Norm()
	}
	if j == len(poly)-1 {
		return poly[j].Sub(poly[j-1]).Norm()
	}
	return poly[j+1].Sub(poly[j-1]).Norm()
}

func collectReductionStats(g *Groom, res *BakeResult) {
	total, sumD, maxD := 0.0, 0.0, 0.0
	for _, c := range res.Cards {
		for _, si := range c.StrandIDs {
			s := g.Strands[si]
			for j, p := range s.Pts {
				d := math.Abs(p.Sub(c.Center[j]).Dot(c.Across[j]))
				sumD += d
				if d > maxD {
					maxD = d
				}
				total++
			}
		}
	}
	res.MeanDist = sumD / total
	res.MaxDist = maxD

	// Flyaways whose tip escapes the ribbon by a clear margin are "lost":
	// the tight card budget cannot represent their outward arc. A threshold
	// keeps the audit focused on visibly missing arcs rather than noise.
	for i, s := range g.Strands {
		if !s.Flyaway {
			continue
		}
		cid := res.Assign[i]
		card := res.Cards[cid]
		esc := 0.0
		for j := 1; j < len(card.Center); j++ {
			p := s.Pts[j]
			d := math.Abs(p.Sub(card.Center[j]).Dot(card.Across[j])) - card.HalfW[j]
			if d > esc {
				esc = d
			}
		}
		if esc > 0.45 {
			res.LostFlyaways = append(res.LostFlyaways, i)
		} else {
			res.KeptFlyaways = append(res.KeptFlyaways, i)
		}
	}
}
