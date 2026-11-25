package main

import (
	"math"
	"sort"
)

const atlasOrder = 6

func hilbertXY2D(n, x, y uint32) uint32 {
	var d uint32
	for s := n / 2; s > 0; s /= 2 {
		var rx, ry uint32
		if x&s != 0 {
			rx = 1
		}
		if y&s != 0 {
			ry = 1
		}
		d += s * s * ((3 * rx) ^ ry)
		if ry == 0 {
			if rx == 1 {
				x = n - 1 - x
				y = n - 1 - y
			}
			x, y = y, x
		}
	}
	return d
}

func atlasCell(root Vec3) (uint32, uint32, float64, float64) {
	az := math.Atan2(root.Z, root.X)
	u := (az + math.Pi) / (2 * math.Pi)
	v := clamp((root.Y/scalpRadii.Y+1)*0.5, 0, 1)
	const side = 1 << atlasOrder
	x := uint32(clamp(math.Floor(u*float64(side)), 0, float64(side-1)))
	y := uint32(clamp(math.Floor(v*float64(side)), 0, float64(side-1)))
	return x, y, u, v
}

func atlasIndices(g Groom) []int {
	type keyed struct {
		index int
		h     uint32
		y     float64
		x     float64
	}
	keys := make([]keyed, len(g.Strands))
	for i, strand := range g.Strands {
		root := strand.Root()
		x, y, _, _ := atlasCell(root)
		keys[i] = keyed{
			index: i,
			h:     hilbertXY2D(1<<atlasOrder, x, y),
			y:     root.Y,
			x:     root.X,
		}
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if keys[i].h != keys[j].h {
			return keys[i].h < keys[j].h
		}
		if keys[i].y != keys[j].y {
			return keys[i].y > keys[j].y
		}
		if keys[i].x != keys[j].x {
			return keys[i].x < keys[j].x
		}
		return keys[i].index < keys[j].index
	})
	out := make([]int, len(keys))
	for i := range keys {
		out[i] = keys[i].index
	}
	return out
}
