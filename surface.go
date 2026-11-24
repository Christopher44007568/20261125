package main

import (
	"fmt"
	"math"
)

const (
	GroupA = 0
	GroupB = 1
)

type Config struct {
	Length       float64 `json:"length_mm"`
	WidthA       float64 `json:"width_a_mm"`
	WidthB       float64 `json:"width_b_mm"`
	BowMM        float64 `json:"shared_edge_bow_mm"`
	CamberA      float64 `json:"camber_a_mm"`
	CamberB      float64 `json:"camber_b_mm"`
	AngleADeg    float64 `json:"angle_a_deg"`
	AngleBDeg    float64 `json:"angle_b_deg"`
	SegAAlong    int     `json:"seg_a_along"`
	SegBAlong    int     `json:"seg_b_along"`
	SegAcross    int     `json:"seg_across"`
	HardAngleDeg float64 `json:"hard_angle_deg"`
	ChordTolMM   float64 `json:"chord_tol_mm"`
	OutputDir    string  `json:"output_dir"`
}

func DefaultConfig() Config {
	return Config{
		Length:       34,
		WidthA:       11,
		WidthB:       12,
		BowMM:        2.2,
		CamberA:      0.55,
		CamberB:      0.85,
		AngleADeg:    18,
		AngleBDeg:    40,
		SegAAlong:    4,
		SegBAlong:    18,
		SegAcross:    16,
		HardAngleDeg: 24,
		ChordTolMM:   0.01,
		OutputDir:    "artifacts",
	}
}

func (c Config) Validate() error {
	if c.Length <= 0 || c.WidthA <= 0 || c.WidthB <= 0 {
		return fmt.Errorf("dimensions must be positive")
	}
	if c.SegAAlong < 1 || c.SegBAlong < 1 || c.SegAcross < 1 {
		return fmt.Errorf("segment counts must be positive")
	}
	if c.HardAngleDeg <= 0 || c.HardAngleDeg >= 180 {
		return fmt.Errorf("hard angle must be in (0,180)")
	}
	if c.ChordTolMM < 0 {
		return fmt.Errorf("chord tolerance must be non-negative")
	}
	return nil
}

// SharedEdge is the single exact master definition of the common boundary.
// q always runs from canonical bottom to top. Patch B deliberately evaluates
// this with q=1-t so its native parameter direction is reversed.
func SharedEdge(cfg Config, q float64) Vec3 {
	q = clampFloat(q, 0, 1)
	return V(
		0.28*cfg.BowMM*math.Sin(2*math.Pi*q),
		cfg.Length*(q-0.5),
		cfg.BowMM*math.Sin(math.Pi*q),
	)
}

func surfaceA(cfg Config, u, q float64) Vec3 {
	angle := cfg.AngleADeg * math.Pi / 180
	out := V(-math.Cos(angle), 0, math.Sin(angle))
	p := SharedEdge(cfg, q).Add(out.Scale(cfg.WidthA * u))
	p.Z += cfg.CamberA * math.Sin(math.Pi*u) * math.Sin(math.Pi*q)
	return p
}

func surfaceBAtQ(cfg Config, v, q float64) Vec3 {
	angle := cfg.AngleBDeg * math.Pi / 180
	out := V(math.Cos(angle), 0, math.Sin(angle))
	p := SharedEdge(cfg, q).Add(out.Scale(cfg.WidthB * v))
	p.Z += cfg.CamberB * math.Sin(math.Pi*v) * math.Sin(math.Pi*q)
	return p
}

type PatchAssembly struct {
	MeshA  *Mesh
	MeshB  *Mesh
	ChainA BoundaryChain
	ChainB BoundaryChain
}

func BuildDemoPatches(cfg Config) PatchAssembly {
	a := buildGridPatch("patch-A-coarse-forward", cfg.SegAcross, cfg.SegAAlong, GroupA,
		func(u, native float64) Vec3 {
			q := native
			return surfaceA(cfg, u, q)
		}, false)

	b := buildGridPatch("patch-B-fine-reversed", cfg.SegAcross, cfg.SegBAlong, GroupB,
		func(v, t float64) Vec3 {
			q := 1 - t
			return surfaceBAtQ(cfg, v, q)
		}, true)

	chainA := BoundaryChain{IDs: make([]int, cfg.SegAAlong+1), Q: make([]float64, cfg.SegAAlong+1)}
	for j := 0; j <= cfg.SegAAlong; j++ {
		chainA.IDs[j] = a.id(0, j, cfg.SegAAlong)
		q := float64(j) / float64(cfg.SegAAlong)
		chainA.Q[j] = q
	}

	chainB := BoundaryChain{IDs: make([]int, cfg.SegBAlong+1), Q: make([]float64, cfg.SegBAlong+1)}
	for j := 0; j <= cfg.SegBAlong; j++ {
		chainB.IDs[j] = b.id(0, j, cfg.SegBAlong)
		t := float64(j) / float64(cfg.SegBAlong)
		chainB.Q[j] = 1 - t
	}

	return PatchAssembly{MeshA: a.mesh, MeshB: b.mesh, ChainA: chainA, ChainB: chainB}
}

type gridPatch struct {
	mesh   *Mesh
	across int
	along  int
}

func (g gridPatch) id(i, j, along int) int {
	return i*(along+1) + j
}

func buildGridPatch(name string, across, along int, group int, eval func(u, native float64) Vec3, reversedNative bool) gridPatch {
	m := &Mesh{Name: name}
	g := gridPatch{mesh: m, across: across, along: along}

	for i := 0; i <= across; i++ {
		u := float64(i) / float64(across)
		for j := 0; j <= along; j++ {
			native := float64(j) / float64(along)
			m.Pos = append(m.Pos, eval(u, native))
		}
	}

	for i := 0; i < across; i++ {
		for j := 0; j < along; j++ {
			a00 := g.id(i, j, along)
			b10 := g.id(i+1, j, along)
			c11 := g.id(i+1, j+1, along)
			d01 := g.id(i, j+1, along)
			// Reverse of parameter-order winding, chosen so both patches have
			// upward/outward face normals despite B's q=1-t direction.
			m.AddTriangle(a00, c11, b10, group, false)
			m.AddTriangle(a00, d01, c11, group, false)
		}
	}
	_ = reversedNative
	return g
}
