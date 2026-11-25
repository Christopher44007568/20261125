package main

import (
	"bytes"
	"encoding/binary"
	"math"
)

// Chunked store: every chunk is an independent binary blob, loaded on demand.
// Roots and tips are always stored as raw fp32 so that root positions and the
// overall silhouette survive any precision tier bit-for-bit.

const (
	chunkMagic    = 0x47524D31 // "GRM1"
	fallbackMagic = 0x46424B31 // "FBK1"
)

// Store holds serialized chunk blobs plus a coarse fallback atlas.
type Store struct {
	Chunks   map[int][]byte
	Fallback []byte
	// Bytes accounts the serialized size per tier (our "VRAM" footprint).
	ChunkBytes    int
	FallbackBytes int
}

func (s *Store) TotalBytes() int { return s.ChunkBytes + s.FallbackBytes }

type strandRecord struct {
	Index int // original strand index, kept for error metrics
	S     Strand
}

// resample reduces a strand to n points, uniformly by arc length,
// keeping root and tip exactly.
func resample(s Strand, n int) Strand {
	if len(s.Pts) <= n {
		return s
	}
	cum := make([]float32, len(s.Pts))
	for i := 1; i < len(s.Pts); i++ {
		cum[i] = cum[i-1] + s.Pts[i].Dist(s.Pts[i-1])
	}
	total := cum[len(cum)-1]
	out := make([]Vec3, n)
	out[0] = s.Pts[0]
	seg := 0
	for i := 1; i < n-1; i++ {
		target := total * float32(i) / float32(n-1)
		for seg < len(cum)-2 && cum[seg+1] < target {
			seg++
		}
		d := cum[seg+1] - cum[seg]
		t := float32(0)
		if d > 0 {
			t = (target - cum[seg]) / d
		}
		out[i] = lerp(s.Pts[seg], s.Pts[seg+1], t)
	}
	out[n-1] = s.Pts[len(s.Pts)-1]
	return Strand{Pts: out}
}

func quantize(v, lo, hi float32, bits uint) uint32 {
	maxq := float32((uint64(1) << bits) - 1)
	if hi <= lo {
		return 0
	}
	t := (v - lo) / (hi - lo)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return uint32(math.Round(float64(t * maxq)))
}

func dequantize(q uint32, lo, hi float32, bits uint) float32 {
	maxq := float32((uint64(1) << bits) - 1)
	return lo + (hi-lo)*float32(q)/maxq
}

// encodeChunk serializes one chunk at the given precision and point count.
// points <= 0 keeps the original point count.
func encodeChunk(g *Groom, chunkID int, bits uint, points int) []byte {
	buf := new(bytes.Buffer)
	w32 := func(v uint32) { binary.Write(buf, binary.LittleEndian, v) }
	wf := func(f float32) { binary.Write(buf, binary.LittleEndian, f) }

	var members []int
	for i := range g.Strands {
		if g.ChunkOf[i] == chunkID {
			members = append(members, i)
		}
	}
	w32(chunkMagic)
	w32(uint32(chunkID))
	w32(uint32(bits))
	w32(uint32(len(members)))
	for _, idx := range members {
		s := g.Strands[idx]
		if points > 0 {
			s = resample(s, points)
		}
		n := len(s.Pts)
		w32(uint32(idx))
		binary.Write(buf, binary.LittleEndian, uint16(n))
		root, tip := s.Pts[0], s.Pts[n-1]
		for _, v := range []float32{root.X, root.Y, root.Z, tip.X, tip.Y, tip.Z} {
			wf(v)
		}
		// Per-strand interior bbox: dequantized points can never leave the
		// original strand envelope, so the silhouette cannot grow.
		lo, hi := root, root
		for _, p := range s.Pts[1 : n-1] {
			lo.X, hi.X = minF(lo.X, p.X), maxF(hi.X, p.X)
			lo.Y, hi.Y = minF(lo.Y, p.Y), maxF(hi.Y, p.Y)
			lo.Z, hi.Z = minF(lo.Z, p.Z), maxF(hi.Z, p.Z)
		}
		for _, v := range []float32{lo.X, lo.Y, lo.Z, hi.X, hi.Y, hi.Z} {
			wf(v)
		}
		bw := &BitWriter{}
		for _, p := range s.Pts[1 : n-1] {
			bw.WriteBits(quantize(p.X, lo.X, hi.X, bits), bits)
			bw.WriteBits(quantize(p.Y, lo.Y, hi.Y, bits), bits)
			bw.WriteBits(quantize(p.Z, lo.Z, hi.Z, bits), bits)
		}
		payload := bw.Bytes()
		w32(uint32(len(payload)))
		buf.Write(payload)
	}
	return buf.Bytes()
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
