package main

import (
	"flag"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	var (
		nStrands  = flag.Int("strands", 1400, "number of hair strands")
		nPoints   = flag.Int("points", 16, "points per strand")
		seed      = flag.Int64("seed", 7, "random seed")
		outDir    = flag.String("out", "out", "output directory")
		bits      = flag.Uint("bits", 12, "quantization bits for the compressed tier")
		lodBits   = flag.Uint("lod-bits", 6, "quantization bits for the low LOD tier")
		lodPoints = flag.Int("lod-points", 5, "points per strand in the low LOD tier")
		nMissing  = flag.Int("missing", 3, "number of chunks to drop (fallback demo)")
		fallbackK = flag.Int("fallback", 6, "proxy strands per missing chunk")
		size      = flag.Int("size", 640, "output image size in pixels")
	)
	flag.Parse()

	if *bits < 2 || *bits > 24 || *lodBits < 2 || *lodBits > 24 {
		fatal(fmt.Errorf("bits must be in [2,24], got bits=%d lod-bits=%d", *bits, *lodBits))
	}
	if *lodPoints < 2 {
		fatal(fmt.Errorf("lod-points must be >= 2"))
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	orig := Generate(*seed, *nStrands, *nPoints)
	origBytes := *nStrands * *nPoints * 3 * 4

	// Serialize both precision tiers into chunked stores.
	compStore := buildStore(orig, uint(*bits), 0)
	lodStore := buildStore(orig, uint(*lodBits), *lodPoints)

	// Simulate chunked loading with some chunks failing -> fallback proxies.
	missing := pickMissing(orig, *nMissing)
	compRes, err := loadStore(compStore, orig, missing, *fallbackK)
	if err != nil {
		fatal(err)
	}
	lodRes, err := loadStore(lodStore, orig, missing, *fallbackK)
	if err != nil {
		fatal(err)
	}

	// Camera shared by every view so silhouettes line up pixel-for-pixel.
	const ss = 2
	W, H := *size*ss, *size*ss
	cam := fitCamera(orig, W, H)

	compM := compare("COMPRESSED", compStore.TotalBytes(), orig, compRes)
	lodM := compare("LOW-LOD", lodStore.TotalBytes(), orig, lodRes)
	mRects := missingRects(orig, missing, cam)
	compM.SilhouetteXOR = silhouetteXOR(cam, W, H, orig, compRes.Groom, mRects)
	lodM.SilhouetteXOR = silhouetteXOR(cam, W, H, orig, lodRes.Groom, mRects)
	origM := Metrics{Name: "ORIGINAL", Bytes: origBytes}

	// Invariants: roots and silhouette endpoints must not move at all.
	for _, m := range []Metrics{compM, lodM} {
		if m.RootMaxDev != 0 || m.TipMaxDev != 0 {
			fatal(fmt.Errorf("invariant violated in %s: rootDev=%v tipDev=%v",
				m.Name, m.RootMaxDev, m.TipMaxDev))
		}
	}

	// Render the four requested images.
	memLabels := []string{
		"FP32 " + formatBytes(origBytes),
		fmt.Sprintf("%dBIT %s", *bits, formatBytes(compStore.TotalBytes())),
		fmt.Sprintf("LOD%d %s", *lodBits, formatBytes(lodStore.TotalBytes())),
	}
	memBytes := []int{origBytes, compStore.TotalBytes(), lodStore.TotalBytes()}
	memColors := []color.RGBA{
		{150, 150, 150, 255}, {240, 170, 40, 255}, {230, 70, 60, 255},
	}

	renderGroomImage(filepath.Join(*outDir, "original.png"), cam, W, H,
		orig, nil, nil, "ORIGINAL FP32", memLabels, memBytes, memColors)
	renderGroomImage(filepath.Join(*outDir, "compressed.png"), cam, W, H,
		compRes.Groom, compRes.IsFallback, mRects,
		fmt.Sprintf("COMPRESSED %dBIT CHUNKED", *bits), memLabels, memBytes, memColors)
	renderGroomImage(filepath.Join(*outDir, "low_lod.png"), cam, W, H,
		lodRes.Groom, lodRes.IsFallback, mRects,
		fmt.Sprintf("LOW LOD %dBIT %dPTS", *lodBits, *lodPoints), memLabels, memBytes, memColors)
	renderHeatmap(filepath.Join(*outDir, "error_heatmap.png"), cam, W, H,
		orig, compRes, mRects, compM.InteriorMax)
	renderHeatmap(filepath.Join(*outDir, "error_heatmap_lod.png"), cam, W, H,
		orig, lodRes, mRects, lodM.InteriorMax)

	// Console + file report.
	report := "groom compression report\n" +
		fmt.Sprintf("strands=%d points=%d chunks=%d\n", *nStrands, *nPoints, len(orig.Chunks)) +
		origM.String() + "\n" + compM.String() + "\n" + lodM.String() + "\n"
	fmt.Print(report)
	if err := os.WriteFile(filepath.Join(*outDir, "report.txt"), []byte(report), 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

// buildStore serializes every chunk of g at the given precision.
func buildStore(g *Groom, bits uint, points int) *Store {
	s := &Store{Chunks: map[int][]byte{}}
	for _, ci := range g.Chunks {
		blob := encodeChunk(g, ci.ID, bits, points)
		s.Chunks[ci.ID] = blob
		s.ChunkBytes += len(blob)
	}
	s.Fallback = encodeFallbackAtlas(g)
	s.FallbackBytes = len(s.Fallback)
	return s
}

// pickMissing chooses well-separated, populated chunks to drop.
func pickMissing(g *Groom, n int) map[int]bool {
	sorted := make([]ChunkInfo, len(g.Chunks))
	copy(sorted, g.Chunks)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a].Count > sorted[b].Count })
	out := map[int]bool{}
	step := len(sorted) / (n + 1)
	if step < 1 {
		step = 1
	}
	for i := 0; i < n && (i+1)*step < len(sorted); i++ {
		out[sorted[(i+1)*step].ID] = true
	}
	return out
}

// fitCamera frames the whole groom with margin.
func fitCamera(g *Groom, w, h int) Camera {
	lo := Vec3{1e9, 1e9, 1e9}
	hi := Vec3{-1e9, -1e9, -1e9}
	for _, s := range g.Strands {
		for _, p := range s.Pts {
			lo.X, hi.X = minF(lo.X, p.X), maxF(hi.X, p.X)
			lo.Y, hi.Y = minF(lo.Y, p.Y), maxF(hi.Y, p.Y)
		}
	}
	sx := float32(w) * 0.9 / (hi.X - lo.X)
	sy := float32(h) * 0.9 / (hi.Y - lo.Y)
	scale := minF(sx, sy)
	return Camera{
		CX:    float32(w)/2 - (lo.X+hi.X)/2*scale,
		CY:    float32(h)/2 + (lo.Y+hi.Y)/2*scale,
		Scale: scale,
	}
}

// missingRects projects each missing chunk's root cluster to a screen rect.
func missingRects(g *Groom, missing map[int]bool, cam Camera) []intRect {
	acc := map[int]*intRect{}
	for i, s := range g.Strands {
		id := g.ChunkOf[i]
		if !missing[id] {
			continue
		}
		x, y := cam.Project(s.Root())
		r, ok := acc[id]
		if !ok {
			r = &intRect{int(x), int(y), int(x), int(y)}
			acc[id] = r
		}
		r.x0 = minI(r.x0, int(x))
		r.y0 = minI(r.y0, int(y))
		r.x1 = maxI(r.x1, int(x))
		r.y1 = maxI(r.y1, int(y))
	}
	var out []intRect
	for _, r := range acc {
		pad := 14
		out = append(out, intRect{r.x0 - pad, r.y0 - pad, r.x1 + pad, r.y1 + pad})
	}
	return out
}

type intRect struct{ x0, y0, x1, y1 int }

func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}
