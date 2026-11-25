package main

import (
	"fmt"
	"image/color"
	"math"
)

// Metrics summarizes quality + memory differences between tiers.
type Metrics struct {
	Name          string
	Bytes         int
	RootMaxDev    float32 // must be 0: roots are stored at full precision
	TipMaxDev     float32 // must be 0: tips are stored at full precision
	InteriorRMSE  float32
	InteriorMax   float32
	SilhouetteXOR int // mask pixels differing vs original render
	MissingChunks []int
	FallbackCount int
}

// compare measures reloaded strands against the originals. Strands marked
// fallback are skipped (they replace missing chunks by design).
// If the reloaded strand has fewer points (LOD), the original is resampled
// to the same count first so the comparison is apples-to-apples.
func compare(name string, bytes int, orig *Groom, res *LoadResult) Metrics {
	m := Metrics{Name: name, Bytes: bytes, MissingChunks: res.MissingChunks}
	var sumSq, maxE float64
	var n int
	for k, rs := range res.Groom.Strands {
		oi := res.OrigIdx[k]
		if oi < 0 {
			m.FallbackCount++
			continue
		}
		os := orig.Strands[oi]
		if d := os.Root().Dist(rs.Root()); d > m.RootMaxDev {
			m.RootMaxDev = d
		}
		if d := os.Tip().Dist(rs.Tip()); d > m.TipMaxDev {
			m.TipMaxDev = d
		}
		ref := os
		if len(rs.Pts) != len(os.Pts) {
			ref = resample(os, len(rs.Pts))
		}
		for j := 1; j < len(rs.Pts)-1; j++ {
			d := ref.Pts[j].Dist(rs.Pts[j])
			sumSq += float64(d * d)
			if d > float32(maxE) {
				maxE = float64(d)
			}
			n++
		}
	}
	if n > 0 {
		m.InteriorRMSE = float32(math.Sqrt(sumSq / float64(n)))
	}
	m.InteriorMax = float32(maxE)
	return m
}

// silhouetteXOR renders both grooms to binary masks and counts differing
// pixels, ignoring the regions of missing chunks (fallback is by design).
func silhouetteXOR(cam Camera, w, h int, a, b *Groom, exclude []intRect) int {
	mask := func(g *Groom) []bool {
		img := NewCanvas(w, h, color.RGBA{0, 0, 0, 255})
		DrawGroom(img, cam, g, nil)
		out := make([]bool, w*h)
		for i, p := range img.Pix {
			out[i] = p.R > 12 || p.G > 12 || p.B > 12
		}
		return out
	}
	ma, mb := mask(a), mask(b)
	diff := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if insideAny(x, y, exclude) {
				continue
			}
			i := y*w + x
			if ma[i] != mb[i] {
				diff++
			}
		}
	}
	return diff
}

func insideAny(x, y int, rects []intRect) bool {
	for _, r := range rects {
		if x >= r.x0 && x <= r.x1 && y >= r.y0 && y <= r.y1 {
			return true
		}
	}
	return false
}

func formatBytes(b int) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.2fMB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func (m Metrics) String() string {
	return fmt.Sprintf(
		"%-22s vram=%-9s rootDev=%.6f tipDev=%.6f rmse=%.4f maxErr=%.4f silhouetteXOR=%dpx missingChunks=%v fallbackStrands=%d",
		m.Name, formatBytes(m.Bytes), m.RootMaxDev, m.TipMaxDev,
		m.InteriorRMSE, m.InteriorMax, m.SilhouetteXOR, m.MissingChunks, m.FallbackCount)
}
