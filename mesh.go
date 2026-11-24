package main

type Vertex struct {
	P Vec3
	N Vec3
	K float64
}

type Mesh struct {
	Vertices []Vertex
	Indices  []uint32
}

func BuildMesh(cfg Config) Mesh {
	m := Mesh{}
	appendFillet(&m, cfg)
	appendParent(&m, cfg, 1)
	appendParent(&m, cfg, -1)
	return m
}

func appendFillet(m *Mesh, cfg Config) {
	f := NewFilletSurface(cfg)
	base := uint32(len(m.Vertices))
	nu, nv := cfg.SamplesU+1, cfg.SamplesV+1
	for j := 0; j < nv; j++ {
		v := float64(j) / float64(nv-1)
		for i := 0; i < nu; i++ {
			u := float64(i) / float64(nu-1)
			s := f.Eval(u, v)
			m.Vertices = append(m.Vertices, Vertex{P: s.P, N: s.N, K: s.Curvatures().MaxAbs()})
		}
	}
	appendGridIndices(m, base, nu, nv)
}

func appendParent(m *Mesh, cfg Config, side int) {
	p := NewParentSurface(cfg, side)
	base := uint32(len(m.Vertices))
	nu, nv := 25, cfg.SamplesV+1
	for j := 0; j < nv; j++ {
		v := float64(j) / float64(nv-1)
		for i := 0; i < nu; i++ {
			s := float64(i) / float64(nu-1)
			sample := p.Eval(s, v)
			m.Vertices = append(m.Vertices, Vertex{
				P: sample.P,
				N: sample.N,
				K: sample.Curvatures().MaxAbs(),
			})
		}
	}
	appendGridIndices(m, base, nu, nv)
}

func appendGridIndices(m *Mesh, base uint32, nu, nv int) {
	for j := 0; j < nv-1; j++ {
		for i := 0; i < nu-1; i++ {
			a := base + uint32(j*nu+i)
			b := a + 1
			c := base + uint32((j+1)*nu+i+1)
			d := base + uint32((j+1)*nu+i)
			m.Indices = append(m.Indices, a, b, c, a, c, d)
		}
	}
}
