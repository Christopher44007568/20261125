package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
)

func decodeChunk(blob []byte) ([]strandRecord, int, error) {
	r := bytes.NewReader(blob)
	var magic, id, bits, count uint32
	binary.Read(r, binary.LittleEndian, &magic)
	if magic != chunkMagic {
		return nil, 0, fmt.Errorf("bad chunk magic %#x", magic)
	}
	binary.Read(r, binary.LittleEndian, &id)
	binary.Read(r, binary.LittleEndian, &bits)
	binary.Read(r, binary.LittleEndian, &count)
	recs := make([]strandRecord, 0, count)
	for i := uint32(0); i < count; i++ {
		var idx uint32
		var n uint16
		binary.Read(r, binary.LittleEndian, &idx)
		binary.Read(r, binary.LittleEndian, &n)
		f := make([]float32, 12)
		binary.Read(r, binary.LittleEndian, &f)
		root := Vec3{f[0], f[1], f[2]}
		tip := Vec3{f[3], f[4], f[5]}
		lo := Vec3{f[6], f[7], f[8]}
		hi := Vec3{f[9], f[10], f[11]}
		var plen uint32
		binary.Read(r, binary.LittleEndian, &plen)
		payload := make([]byte, plen)
		r.Read(payload)
		pts := make([]Vec3, n)
		pts[0] = root
		pts[n-1] = tip
		br := NewBitReader(payload)
		for j := 1; j < int(n)-1; j++ {
			pts[j] = Vec3{
				X: dequantize(br.ReadBits(uint(bits)), lo.X, hi.X, uint(bits)),
				Y: dequantize(br.ReadBits(uint(bits)), lo.Y, hi.Y, uint(bits)),
				Z: dequantize(br.ReadBits(uint(bits)), lo.Z, hi.Z, uint(bits)),
			}
		}
		recs = append(recs, strandRecord{Index: int(idx), S: Strand{Pts: pts}})
	}
	return recs, int(id), nil
}

// FallbackEntry is a coarse per-chunk proxy used when a chunk fails to load.
type FallbackEntry struct {
	ChunkID int
	Count   int
	AvgRoot Vec3
	AvgTip  Vec3
}

func encodeFallbackAtlas(g *Groom) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(fallbackMagic))
	binary.Write(buf, binary.LittleEndian, uint32(len(g.Chunks)))
	for _, c := range g.Chunks {
		binary.Write(buf, binary.LittleEndian, uint32(c.ID))
		binary.Write(buf, binary.LittleEndian, uint32(c.Count))
		for _, v := range []float32{c.AvgRoot.X, c.AvgRoot.Y, c.AvgRoot.Z, c.AvgTip.X, c.AvgTip.Y, c.AvgTip.Z} {
			binary.Write(buf, binary.LittleEndian, v)
		}
	}
	return buf.Bytes()
}

func decodeFallbackAtlas(blob []byte) map[int]FallbackEntry {
	r := bytes.NewReader(blob)
	var magic, count uint32
	binary.Read(r, binary.LittleEndian, &magic)
	binary.Read(r, binary.LittleEndian, &count)
	out := map[int]FallbackEntry{}
	for i := uint32(0); i < count; i++ {
		var id, n uint32
		f := make([]float32, 6)
		binary.Read(r, binary.LittleEndian, &id)
		binary.Read(r, binary.LittleEndian, &n)
		binary.Read(r, binary.LittleEndian, &f)
		out[int(id)] = FallbackEntry{
			ChunkID: int(id), Count: int(n),
			AvgRoot: Vec3{f[0], f[1], f[2]}, AvgTip: Vec3{f[3], f[4], f[5]},
		}
	}
	return out
}

// fallbackStrands synthesizes coarse proxy strands for a missing chunk:
// straight root->tip hairs scattered deterministically around the chunk mean.
func fallbackStrands(e FallbackEntry, k int) []Strand {
	rng := rand.New(rand.NewSource(int64(e.ChunkID)*7919 + 13))
	out := make([]Strand, 0, k)
	for i := 0; i < k; i++ {
		jit := func() float32 { return float32(rng.Float64()*2-1) * 6 }
		root := e.AvgRoot.Add(Vec3{jit(), jit() * 0.3, jit()})
		tip := e.AvgTip.Add(Vec3{jit(), jit() * 0.3, jit()})
		pts := make([]Vec3, 4)
		for j := range pts {
			pts[j] = lerp(root, tip, float32(j)/float32(len(pts)-1))
		}
		out = append(out, Strand{Pts: pts})
	}
	return out
}

// LoadResult describes a groom rebuilt from the chunked store.
type LoadResult struct {
	Groom          *Groom
	MissingChunks  []int
	FallbackChunks []int
	// FallbackRoots[i] marks strands synthesized from the fallback atlas.
	IsFallback []bool
	// OrigIdx maps each reloaded strand to its original index, or -1 for
	// fallback proxies. Strands from missing chunks are absent entirely.
	OrigIdx []int
}

// loadStore reloads all chunks; chunks in `missing` are replaced by proxies.
func loadStore(store *Store, orig *Groom, missing map[int]bool, fallbackK int) (*LoadResult, error) {
	atlas := decodeFallbackAtlas(store.Fallback)
	res := &LoadResult{Groom: &Groom{}}
	for _, ci := range orig.Chunks {
		if missing[ci.ID] {
			res.MissingChunks = append(res.MissingChunks, ci.ID)
			res.FallbackChunks = append(res.FallbackChunks, ci.ID)
			for _, s := range fallbackStrands(atlas[ci.ID], fallbackK) {
				res.Groom.Strands = append(res.Groom.Strands, s)
				res.Groom.ChunkOf = append(res.Groom.ChunkOf, ci.ID)
				res.IsFallback = append(res.IsFallback, true)
				res.OrigIdx = append(res.OrigIdx, -1)
			}
			continue
		}
		recs, _, err := decodeChunk(store.Chunks[ci.ID])
		if err != nil {
			return nil, err
		}
		for _, rec := range recs {
			res.Groom.Strands = append(res.Groom.Strands, rec.S)
			res.Groom.ChunkOf = append(res.Groom.ChunkOf, ci.ID)
			res.IsFallback = append(res.IsFallback, false)
			res.OrigIdx = append(res.OrigIdx, rec.Index)
		}
	}
	res.Groom.buildChunks()
	return res, nil
}
