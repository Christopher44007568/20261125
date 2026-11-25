package main

import (
	"math"
	"math/rand"
)

// Vec3 is a 3D point/vector in groom space (fp32, matching GPU vertex data).
type Vec3 struct {
	X, Y, Z float32
}

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float32) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}
func (a Vec3) Len() float32 { return float32(math.Sqrt(float64(a.X*a.X + a.Y*a.Y + a.Z*a.Z))) }
func (a Vec3) Dist(b Vec3) float32 {
	return a.Sub(b).Len()
}
func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l == 0 {
		return Vec3{0, 1, 0}
	}
	return a.Mul(1 / l)
}
func lerp(a, b Vec3, t float32) Vec3 { return a.Add(b.Sub(a).Mul(t)) }

// Strand is a single hair: Pts[0] is the root, Pts[len-1] is the tip.
type Strand struct {
	Pts []Vec3
}

func (s Strand) Root() Vec3 { return s.Pts[0] }
func (s Strand) Tip() Vec3  { return s.Pts[len(s.Pts)-1] }

// Groom is a full hair asset plus its chunk partitioning.
type Groom struct {
	Strands []Strand
	// ChunkOf[i] is the chunk id of Strands[i].
	ChunkOf []int
	Chunks  []ChunkInfo
}

// ChunkInfo describes one spatial chunk of the scalp.
type ChunkInfo struct {
	ID       int
	ThetaBin int
	PhiBin   int
	Count    int
	AvgRoot  Vec3
	AvgTip   Vec3
}

const (
	headRadius  = 50.0
	thetaChunks = 8
	phiChunks   = 4
)

// chunkID maps a scalp root position to a spherical-grid chunk id.
func chunkID(p Vec3) int {
	n := p.Norm()
	theta := math.Atan2(float64(n.Z), float64(n.X)) // [-pi, pi]
	phi := math.Acos(clampF(float64(n.Y), -1, 1))   // [0, pi], 0 = top
	tb := int((theta + math.Pi) / (2 * math.Pi) * thetaChunks)
	pb := int(phi / math.Pi * phiChunks)
	if tb >= thetaChunks {
		tb = thetaChunks - 1
	}
	if pb >= phiChunks {
		pb = phiChunks - 1
	}
	return pb*thetaChunks + tb
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Generate builds a deterministic synthetic groom on a scalp hemisphere.
func Generate(seed int64, nStrands, nPoints int) *Groom {
	rng := rand.New(rand.NewSource(seed))
	g := &Groom{}
	for i := 0; i < nStrands; i++ {
		// Scalp: upper hemisphere biased toward the top, plus sideburn falloff.
		theta := rng.Float64() * 2 * math.Pi
		phi := math.Acos(1.0 - rng.Float64()*0.72) // polar angle, denser at crown
		dir := Vec3{
			X: float32(math.Sin(phi) * math.Cos(theta)),
			Y: float32(math.Cos(phi)),
			Z: float32(math.Sin(phi) * math.Sin(theta)),
		}
		root := dir.Mul(headRadius)
		length := float32(28 + rng.Float64()*26)
		curl := float32(0.6 + rng.Float64()*1.4)
		phase := rng.Float64() * 2 * math.Pi
		side := Vec3{-dir.Z, 0, dir.X}.Norm()
		pts := make([]Vec3, nPoints)
		for j := 0; j < nPoints; j++ {
			t := float32(j) / float32(nPoints-1)
			// Grow along the normal, sag with gravity, add a curl wobble.
			p := root.Add(dir.Mul(length * t * 0.55))
			p.Y -= length * t * t * 0.55 // gravity sag
			wob := float32(math.Sin(float64(t)*math.Pi*float64(curl)+phase)) * 2.2 * t
			p = p.Add(side.Mul(wob))
			p.X += float32(math.Sin(phase+float64(t)*3.0)) * 0.6 * t
			pts[j] = p
		}
		g.Strands = append(g.Strands, Strand{Pts: pts})
		g.ChunkOf = append(g.ChunkOf, chunkID(root))
	}
	g.buildChunks()
	return g
}

func (g *Groom) buildChunks() {
	acc := map[int]*ChunkInfo{}
	for i, s := range g.Strands {
		id := g.ChunkOf[i]
		ci, ok := acc[id]
		if !ok {
			ci = &ChunkInfo{ID: id, ThetaBin: id % thetaChunks, PhiBin: id / thetaChunks}
			acc[id] = ci
		}
		ci.Count++
		ci.AvgRoot = ci.AvgRoot.Add(s.Root())
		ci.AvgTip = ci.AvgTip.Add(s.Tip())
	}
	for _, ci := range acc {
		n := float32(ci.Count)
		ci.AvgRoot = ci.AvgRoot.Mul(1 / n)
		ci.AvgTip = ci.AvgTip.Mul(1 / n)
		g.Chunks = append(g.Chunks, *ci)
	}
	// Sort chunks by id for deterministic processing.
	for i := 1; i < len(g.Chunks); i++ {
		for j := i; j > 0 && g.Chunks[j-1].ID > g.Chunks[j].ID; j-- {
			g.Chunks[j-1], g.Chunks[j] = g.Chunks[j], g.Chunks[j-1]
		}
	}
}
