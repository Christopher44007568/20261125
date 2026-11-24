package geom

import "math"

// MeshTriangle carries both geometry and its originating surface parameters.
type MeshTriangle struct {
	P  [3]Vec3
	UV [3]UV
	ID int
}

func (t MeshTriangle) Normal() Vec3 {
	return t.P[1].Sub(t.P[0]).Cross(t.P[2].Sub(t.P[0])).Unit()
}

func (t MeshTriangle) Area2() float64 {
	return t.P[1].Sub(t.P[0]).Cross(t.P[2].Sub(t.P[0])).Len()
}

func (t MeshTriangle) AtUV(q UV) Vec3 {
	// The mesh parameter triangle is an affine image of the 3D triangle.
	u0, v0 := t.UV[0].U, t.UV[0].V
	u1, v1 := t.UV[1].U-t.UV[0].U, t.UV[1].V-t.UV[0].V
	u2, v2 := t.UV[2].U-t.UV[0].U, t.UV[2].V-t.UV[0].V
	du, dv := q.U-u0, q.V-v0
	det := u1*v2 - u2*v1
	if math.Abs(det) < 1e-20 {
		return t.P[0]
	}
	a := (du*v2 - u2*dv) / det
	b := (u1*dv - du*v1) / det
	return t.P[0].Add(t.P[1].Sub(t.P[0]).Mul(a)).
		Add(t.P[2].Sub(t.P[0]).Mul(b))
}

// Mesh is a rectangular tensor-product triangulation of a ruled surface.
type Mesh struct {
	Surface    *RuledSurface
	Nu, Nv     int
	Vertices   []Vec3
	Params     []UV
	Triangles  []MeshTriangle
	CellTri    [][2]int
	VertexGrid [][]int
}

func BuildMesh(s *RuledSurface) *Mesh {
	nu, nv := s.Nu, s.Nv
	if nu < 1 {
		nu = 1
	}
	if nv < 1 {
		nv = 1
	}
	verts, params := s.SampleGrid()
	m := &Mesh{
		Surface:    s,
		Nu:         nu,
		Nv:         nv,
		Vertices:   verts,
		Params:     params,
		CellTri:    make([][2]int, nu*nv),
		VertexGrid: make([][]int, nv+1),
	}
	for j := 0; j <= nv; j++ {
		m.VertexGrid[j] = make([]int, nu+1)
		for i := 0; i <= nu; i++ {
			m.VertexGrid[j][i] = j*(nu+1) + i
		}
	}
	idx := func(i, j int) int { return j*(nu+1) + i }
	for j := 0; j < nv; j++ {
		for i := 0; i < nu; i++ {
			i00, i10 := idx(i, j), idx(i+1, j)
			i11, i01 := idx(i+1, j+1), idx(i, j+1)
			t0 := m.addTri(i00, i10, i11)
			t1 := m.addTri(i00, i11, i01)
			m.CellTri[j*nu+i] = [2]int{t0, t1}
		}
	}
	return m
}

func (m *Mesh) addTri(a, b, c int) int {
	id := len(m.Triangles)
	m.Triangles = append(m.Triangles, MeshTriangle{
		P:  [3]Vec3{m.Vertices[a], m.Vertices[b], m.Vertices[c]},
		UV: [3]UV{m.Params[a], m.Params[b], m.Params[c]},
		ID: id,
	})
	return id
}

func (m *Mesh) Vertex(i, j int) int { return m.VertexGrid[j][i] }

func (m *Mesh) Bounds() (Vec3, Vec3) { return Bounds(m.Vertices) }
