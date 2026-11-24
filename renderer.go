package main

import "math/rand/v2"

type blockKey struct {
	ox, oy, bs int
}

func jitterRng(seed uint64, parts ...uint64) *rand.Rand {
	h := seed
	for _, p := range parts {
		h ^= p
		h *= 1099511628211
	}
	h ^= h >> 33
	return rand.New(rand.NewPCG(h, h^0x9e3779b97f4a7c15))
}

func renderGroundTruth(cfg Config, sc Scene, eye Eye, frame int) *Frame {
	f := newFrame(cfg.W, cfg.H)
	n := cfg.GTSpp
	for y := 0; y < cfg.H; y++ {
		for x := 0; x < cfg.W; x++ {
			rng := jitterRng(cfg.Seed, uint64(x)*73856093,
				uint64(y)*19349663, uint64(len(eye.Name)))
			var acc Color
			for j := 0; j < n; j++ {
				for i := 0; i < n; i++ {
					px := float64(x) + (float64(i)+rng.Float64())/float64(n) + eye.Offset.X
					py := float64(y) + (float64(j)+rng.Float64())/float64(n) + eye.Offset.Y
					acc = acc.Add(sc.sample(Point{px, py}))
				}
			}
			f.Pix[f.index(x, y)] = acc.Scale(1 / float64(n*n))
		}
	}
	return f
}

func blockSample(cfg Config, sc Scene, p Point, bs, frame int, keyParts ...uint64) Color {
	parts := append([]uint64{uint64(frame), uint64(bs)}, keyParts...)
	rng := jitterRng(cfg.Seed, parts...)
	jx := (rng.Float64() - 0.5) * float64(bs)
	jy := (rng.Float64() - 0.5) * float64(bs)
	return sc.sample(Point{p.X + jx, p.Y + jy})
}

func renderFoveated(cfg Config, sc Scene, eye Eye, frame int) *Frame {
	f := newFrame(cfg.W, cfg.H)
	cache := map[blockKey]Color{}
	for y := 0; y < cfg.H; y++ {
		for x := 0; x < cfg.W; x++ {
			bs := cfg.blockSize(x, y)
			ox, oy := (x/bs)*bs, (y/bs)*bs
			key := blockKey{ox, oy, bs}
			col, ok := cache[key]
			if !ok {
				p := Point{float64(ox) + float64(bs)/2 + eye.Offset.X,
					float64(oy) + float64(bs)/2 + eye.Offset.Y}
				col = blockSample(cfg, sc, p, bs, frame, uint64(ox), uint64(oy), 1)
				cache[key] = col
			}
			f.Pix[f.index(x, y)] = col
		}
	}
	return f
}

func buildSharedCache(cfg Config, sc Scene, frame int) map[blockKey]Color {
	cache := map[blockKey]Color{}
	for y := 0; y < cfg.H; y++ {
		for x := 0; x < cfg.W; x++ {
			bs := cfg.blockSize(x, y)
			ox, oy := (x/bs)*bs, (y/bs)*bs
			key := blockKey{ox, oy, bs}
			if _, ok := cache[key]; ok {
				continue
			}
			p := Point{float64(ox) + float64(bs)/2, float64(oy) + float64(bs)/2}
			cache[key] = blockSample(cfg, sc, p, bs, frame, uint64(ox), uint64(oy), 2)
		}
	}
	return cache
}

func fillFromCache(cfg Config, f *Frame, eye Eye, cache map[blockKey]Color) {
	for y := 0; y < cfg.H; y++ {
		for x := 0; x < cfg.W; x++ {
			bs := cfg.blockSize(x, y)
			ox, oy := (x/bs)*bs, (y/bs)*bs
			f.Pix[f.index(x, y)] = cache[blockKey{ox, oy, bs}]
		}
	}
}

func renderShared(cfg Config, sc Scene, frame int) (left, right *Frame) {
	cache := buildSharedCache(cfg, sc, frame)
	left, right = newFrame(cfg.W, cfg.H), newFrame(cfg.W, cfg.H)
	fillFromCache(cfg, left, bothEyes(cfg)[0], cache)
	fillFromCache(cfg, right, bothEyes(cfg)[1], cache)
	return left, right
}

func boxBlur5(f *Frame) *Frame {
	out := newFrame(f.W, f.H)
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			var acc Color
			n := 0
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					xx, yy := x+dx, y+dy
					if xx < 0 || yy < 0 || xx >= f.W || yy >= f.H {
						continue
					}
					acc = acc.Add(f.Pix[f.index(xx, yy)])
					n++
				}
			}
			out.Pix[out.index(x, y)] = acc.Scale(1 / float64(n))
		}
	}
	return out
}
