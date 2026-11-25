package main

import "math"

func meanFurLuma(c *Canvas) float64 {
	sum, n := 0.0, 0
	for _, p := range c.Pix {
		if p.A < 0.55 {
			continue
		}
		sum += (0.2126*p.R + 0.7152*p.G + 0.0722*p.B) / p.A
		n++
	}
	if n == 0 {
		return 1
	}
	return sum / float64(n)
}

func calibrateFurPair(shell, strand *Canvas) (float64, float64) {
	ls, lt := meanFurLuma(shell), meanFurLuma(strand)
	if ls <= 1e-5 || lt <= 1e-5 {
		return 1, 1
	}
	target := (ls + lt) * 0.5
	gs := clamp(target/ls, 0.58, 1.55)
	gt := clamp(target/lt, 0.58, 1.55)
	for i := range shell.Pix {
		if shell.Pix[i].A > 0 {
			shell.Pix[i].R *= gs
			shell.Pix[i].G *= gs
			shell.Pix[i].B *= gs
		}
	}
	for i := range strand.Pix {
		if strand.Pix[i].A > 0 {
			strand.Pix[i].R *= gt
			strand.Pix[i].G *= gt
			strand.Pix[i].B *= gt
		}
	}
	return gs, gt
}

func hybridLOD(depth float64) float64 {
	if math.IsInf(depth, 1) {
		return 0.5
	}
	return 1 - smoothstep(3.68, 4.86, depth)
}

func compositeHybrid(shell, strand *Canvas) *Canvas {
	out := NewCanvas(shell.W, shell.H)
	for y := 0; y < out.H; y++ {
		ny := float64(y) / float64(out.H)
		for x := 0; x < out.W; x++ {
			nx := float64(x) / float64(out.W)
			i := y*out.W + x
			sp, tp := shell.Pix[i], strand.Pix[i]
			if sp.A < 1e-5 && tp.A < 1e-5 {
				continue
			}
			depth := shell.Depth[i]
			if math.IsInf(depth, 1) {
				depth = strand.Depth[i]
			}
			organic := fbm2(nx*7.2+2.7, ny*7.2+8.1, 3) - 0.5
			w := hybridLOD(depth + organic*0.46)

			// Coverage-preserving replacement: opacity is normalized over the
			// representations that actually cover this pixel. The two layers
			// are never added, and a hole in one representation cannot fade a
			// valid tip in the other into a background ghost.
			blended, ok := blendHybridPixel(sp, tp, w)
			if !ok {
				continue
			}
			out.Pix[i] = blended
			if !math.IsInf(shell.Depth[i], 1) {
				out.Depth[i] = shell.Depth[i]
			} else {
				out.Depth[i] = strand.Depth[i]
			}
		}
	}
	return out
}

func coverageStep(a float64) float64 {
	if a < 1e-5 {
		return 0
	}
	return 1
}

func blendHybridPixel(sp, tp HDR, w float64) (HDR, bool) {
	ws := (1 - w) * coverageStep(sp.A)
	wt := w * coverageStep(tp.A)
	den := ws + wt
	if den < 1e-6 {
		return HDR{}, false
	}
	a := (sp.A*ws + tp.A*wt) / den
	cs := C(0, 0, 0)
	ct := C(0, 0, 0)
	if sp.A > 1e-6 {
		cs = C(sp.R/sp.A, sp.G/sp.A, sp.B/sp.A)
	}
	if tp.A > 1e-6 {
		ct = C(tp.R/tp.A, tp.G/tp.A, tp.B/tp.A)
	}
	col := cs.Lerp(ct, wt/den)
	return HDR{R: col.R * a, G: col.G * a, B: col.B * a, A: a}, true
}
