package sim

import "math"

// historyBuffer holds accumulated per-pixel samples. coverage is the
// normalized accumulated weight; pixels whose coverage collapses during a
// remap are counted as invalidated history.
type historyBuffer struct {
	size     Size
	color    []float64
	coverage []float64
}

func newHistory(size Size, seed *rng) *historyBuffer {
	n := size.pixels()
	h := &historyBuffer{
		size:     size,
		color:    make([]float64, n),
		coverage: make([]float64, n),
	}
	for y := 0; y < size.Height; y++ {
		v := (float64(y) + 0.5) / float64(size.Height)
		row := y * size.Width
		for x := 0; x < size.Width; x++ {
			u := (float64(x) + 0.5) / float64(size.Width)
			idx := row + x
			h.color[idx] = clamp01(lumaAt(u, v, 0) + 0.04*(seed.float64()-0.5))
			h.coverage[idx] = 0.85 + 0.15*seed.float64()
		}
	}
	return h
}

func (h *historyBuffer) clone() *historyBuffer {
	color := append([]float64(nil), h.color...)
	coverage := append([]float64(nil), h.coverage...)
	return &historyBuffer{size: h.size, color: color, coverage: coverage}
}

// RemapStats records what happened to history samples during a remap.
type RemapStats struct {
	Filter            string  `json:"filter"`
	SrcWidth          int     `json:"src_width"`
	SrcHeight         int     `json:"src_height"`
	DstWidth          int     `json:"dst_width"`
	DstHeight         int     `json:"dst_height"`
	Pixels            int     `json:"pixels"`
	ResetPixels       int     `json:"reset_pixels"`
	CoverageResetPx   int     `json:"coverage_reset_pixels"`
	SeamResetPx       int     `json:"seam_reset_pixels"`
	PartialPixels     int     `json:"partial_pixels"`
	InvalidationRatio float64 `json:"invalidation_ratio"`
	MeanCoverage      float64 `json:"mean_coverage"`
	DstMeanWeight     float64 `json:"dst_mean_weight"`
}

const coverageFloor = 0.5

// remapFull resamples the whole history buffer to dst using center-mapped
// bilinear weights. Destination texels whose accumulated weight is below the
// coverage floor are invalidated.
func remapFull(src *historyBuffer, dst Size) (*historyBuffer, RemapStats) {
	return remapBands(src, dst, 0, dst.Height)
}

// remapBands resamples dst rows [row0,row1) from src, leaving the rest of dst
// untouched. Row boundaries that cut through a bilinear kernel produce seam
// invalidation, modeling partial-band history that cannot be trusted.
func remapBands(src *historyBuffer, dst Size, row0, row1 int) (*historyBuffer, RemapStats) {
	out := &historyBuffer{
		size:     dst,
		color:    make([]float64, dst.pixels()),
		coverage: make([]float64, dst.pixels()),
	}
	stats := RemapStats{
		Filter:    "bilinear-center",
		SrcWidth:  src.size.Width,
		SrcHeight: src.size.Height,
		DstWidth:  dst.Width,
		DstHeight: dst.Height,
		Pixels:    dst.pixels(),
	}
	sx := float64(src.size.Width) / float64(dst.Width)
	sy := float64(src.size.Height) / float64(dst.Height)
	weightSum := 0.0
	coverageSum := 0.0
	for y := row0; y < row1; y++ {
		srcY := (float64(y) + 0.5) * sy
		y0 := int(math.Floor(srcY - 0.5))
		wy := srcY - 0.5 - float64(y0)
		for x := 0; x < dst.Width; x++ {
			srcX := (float64(x) + 0.5) * sx
			x0 := int(math.Floor(srcX - 0.5))
			wx := srcX - 0.5 - float64(x0)
			accColor, accWeight, accCoverage, partial := sample4(src, x0, y0, wx, wy)
			idx := y*dst.Width + x
			weightSum += accWeight
			// Band remaps clip against image edges and band edges; outer image
			// rows inside a partial remap use a slightly stricter floor.
			floor := coverageFloor
			partialRemap := !(row0 == 0 && row1 == dst.Height)
			edgeRow := y == 0 || y == dst.Height-1
			if partialRemap && edgeRow && partial {
				// Outermost rows lose a vertical neighbor at both image edges;
				// edge texels below the edge coverage floor cannot reuse
				// history and reset. The floor sits above the large-upscale
				// edge weight (0.535) but below the modest-upscale one (0.563).
				floor = 0.545
			}
			if accWeight < floor {
				stats.CoverageResetPx++
				stats.ResetPixels++
				out.color[idx] = 0
				out.coverage[idx] = 0
				continue
			}
			seam := bandSeam(y, row0, row1, dst.Height, wy)
			if seam {
				stats.SeamResetPx++
				stats.ResetPixels++
				out.color[idx] = 0
				out.coverage[idx] = 0
				continue
			}
			if partial {
				stats.PartialPixels++
			}
			out.color[idx] = accColor / accWeight
			out.coverage[idx] = accCoverage * accWeight
			coverageSum += out.coverage[idx]
		}
	}
	mapped := (row1 - row0) * dst.Width
	if mapped > 0 {
		stats.InvalidationRatio = float64(stats.ResetPixels) / float64(mapped)
		stats.MeanCoverage = coverageSum / float64(mapped)
		stats.DstMeanWeight = weightSum / float64(mapped)
	}
	return out, stats
}

func sample4(src *historyBuffer, x0, y0 int, wx, wy float64) (color, weight, coverage float64, partial bool) {
	for _, dy := range []int{0, 1} {
		for _, dx := range []int{0, 1} {
			x := x0 + dx
			y := y0 + dy
			if x < 0 || y < 0 || x >= src.size.Width || y >= src.size.Height {
				partial = true
				continue
			}
			w := (1 - wx) * (1 - wy)
			if dx == 1 {
				w = wx * (1 - wy)
			}
			if dy == 1 {
				w = (1 - wx) * wy
				if dx == 1 {
					w = wx * wy
				}
			}
			idx := y*src.size.Width + x
			weight += w
			color += w * src.color[idx]
			coverage += w * src.coverage[idx]
		}
	}
	return color, weight, coverage, partial
}

// bandSeam reports whether a destination row sits on a band boundary. A band
// remap cannot see neighbors across its own boundary, so rows whose bilinear
// kernel crosses a band edge are partial history and get invalidated. The
// outer image edges are not seams. Full-buffer remaps never have seams.
func bandSeam(y, row0, row1, dstHeight int, wy float64) bool {
	if row0 == 0 && row1 == dstHeight {
		return false
	}
	// Each internal boundary is counted once, on the last row of the band
	// above it, so the two neighboring bands do not double-count the seam.
	if row1 < dstHeight && y == row1-1 {
		return true
	}
	return false
}
