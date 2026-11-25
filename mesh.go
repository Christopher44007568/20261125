package main

type Tri struct {
	Vertices [3]int32
	Boundary [3]bool
	Tier     uint8
}

type Mesh struct {
	Vertices []Vec3
	Params   []Vec2
	Tris     []Tri
}

type meshBuilder struct {
	mesh Mesh
	keys map[paramKey]int32
}

func newMeshBuilder() *meshBuilder {
	return &meshBuilder{
		mesh: Mesh{},
		keys: make(map[paramKey]int32),
	}
}

type paramKey [2]int64

func makeParamKey(u, v float64) paramKey {
	const scale = 1_000_000_000_000
	return [2]int64{
		int64(u*scale + 0.5),
		int64(v*scale + 0.5),
	}
}

func (b *meshBuilder) vertex(surface *Surface, u, v float64) int32 {
	key := makeParamKey(u, v)
	if id, ok := b.keys[key]; ok {
		return id
	}
	id := int32(len(b.mesh.Vertices))
	b.keys[key] = id
	b.mesh.Vertices = append(b.mesh.Vertices, surface.Point(u, v))
	b.mesh.Params = append(b.mesh.Params, Vec2{U: u, V: v})
	return id
}

func (b *meshBuilder) tri(a, c, d int32, tier uint8, boundary [3]bool) {
	b.mesh.Tris = append(b.mesh.Tris, Tri{
		Vertices: [3]int32{a, c, d},
		Boundary: boundary,
		Tier:     tier,
	})
}

func (b *meshBuilder) regular(surface *Surface, u0, u1, v0, v1 float64, nu, nv int, tier uint8) {
	ids := make([][]int32, nu+1)
	for i := 0; i <= nu; i++ {
		ids[i] = make([]int32, nv+1)
		u := u0 + (u1-u0)*float64(i)/float64(nu)
		for j := 0; j <= nv; j++ {
			v := v0 + (v1-v0)*float64(j)/float64(nv)
			ids[i][j] = b.vertex(surface, u, v)
		}
	}

	for i := 0; i < nu; i++ {
		for j := 0; j < nv; j++ {
			b.tri(ids[i][j], ids[i][j+1], ids[i+1][j+1], tier, [3]bool{})
			b.tri(ids[i][j], ids[i+1][j+1], ids[i+1][j], tier, [3]bool{})
		}
	}
}

func (b *meshBuilder) transition(surface *Surface, u0, u1, v0, v1 float64, coarseNV int, tier uint8) {
	fineNV := coarseNV * 2
	left := make([]int32, fineNV+1)
	right := make([]int32, coarseNV+1)
	for j := 0; j <= fineNV; j++ {
		v := v0 + (v1-v0)*float64(j)/float64(fineNV)
		left[j] = b.vertex(surface, u0, v)
	}
	for j := 0; j <= coarseNV; j++ {
		v := v0 + (v1-v0)*float64(j)/float64(coarseNV)
		right[j] = b.vertex(surface, u1, v)
	}

	for j := 0; j < coarseNV; j++ {
		a := left[j*2]
		mid := left[j*2+1]
		top := left[j*2+2]
		bottom := right[j]
		rightTop := right[j+1]

		b.tri(a, bottom, rightTop, tier, [3]bool{false, true, false})
		b.tri(a, rightTop, top, tier, [3]bool{})
		b.tri(a, top, mid, tier, [3]bool{false, true, true})
	}
}

func buildFullMesh(surface *Surface, grids GridSet) *Mesh {
	b := newMeshBuilder()
	const firstBoundary = 30.0 / 96.0
	const firstTransitionEnd = 32.0 / 96.0
	const secondBoundary = 62.0 / 96.0
	const secondTransitionEnd = 64.0 / 96.0

	b.regular(surface, 0, firstBoundary, 0, 1, 30, grids.NearV, tierNear)
	b.transition(surface, firstBoundary, firstTransitionEnd, 0, 1, grids.MidV, tierTransition)
	b.regular(surface, firstTransitionEnd, secondBoundary, 0, 1, 15, grids.MidV, tierMid)
	b.transition(surface, secondBoundary, secondTransitionEnd, 0, 1, grids.FarV, tierTransition)
	b.regular(surface, secondTransitionEnd, 1, 0, 1, 8, grids.FarV, tierFar)
	return &b.mesh
}

func buildLevelMesh(surface *Surface, nu, nv int, tier uint8) *Mesh {
	b := newMeshBuilder()
	b.regular(surface, 0, 1, 0, 1, nu, nv, tier)
	return &b.mesh
}
