package mesh

import (
	"math"
	"sort"
)

type Feature uint8

const (
	Unknown Feature = iota
	Plane
	Cylinder
	Hole
	Fillet
)

func (f Feature) String() string {
	switch f {
	case Plane:
		return "plane"
	case Cylinder:
		return "cylinder"
	case Hole:
		return "hole"
	case Fillet:
		return "fillet"
	default:
		return "unknown"
	}
}

type Triangle struct {
	P      [3]Vec3
	N      [3]Vec3
	Normal Vec3
	Area   float64
	Center Vec3
	Hint   Feature
}

type Mesh struct {
	Tris []Triangle
}

func (m *Mesh) AddTri(p [3]Vec3, n [3]Vec3, want Vec3, hint Feature) {
	cr := p[1].Sub(p[0]).Cross(p[2].Sub(p[0]))
	area := 0.5 * cr.Len()
	if area < 1e-9 {
		return
	}
	cr = cr.Norm()
	if cr.Dot(want) < 0 {
		p[1], p[2] = p[2], p[1]
		n[1], n[2] = n[2], n[1]
		cr = cr.Mul(-1)
	}
	m.Tris = append(m.Tris, Triangle{
		P:      p,
		N:      n,
		Normal: cr,
		Area:   area,
		Center: p[0].Add(p[1]).Add(p[2]).Mul(1.0 / 3),
		Hint:   hint,
	})
}

func (m *Mesh) AddQuad(a, b, c, d Vec3, na, nb, nc, nd Vec3, want Vec3, hint Feature) {
	nwant := want.Norm()
	m.AddTri([3]Vec3{a, b, c}, [3]Vec3{na, nb, nc}, nwant, hint)
	m.AddTri([3]Vec3{a, c, d}, [3]Vec3{na, nc, nd}, nwant, hint)
}

func (m *Mesh) AddFan(center Vec3, ring []Vec3, n Vec3, hint Feature) {
	for i := 0; i < len(ring); i++ {
		j := (i + 1) % len(ring)
		m.AddTri([3]Vec3{center, ring[i], ring[j]}, [3]Vec3{n, n, n}, n, hint)
	}
}

type Bounds struct {
	Min, Max Vec3
	Center   Vec3
	Radius   float64
}

func (m *Mesh) Bounds() Bounds {
	if len(m.Tris) == 0 {
		return Bounds{}
	}
	mn := V(math.Inf(1), math.Inf(1), math.Inf(1))
	mx := V(math.Inf(-1), math.Inf(-1), math.Inf(-1))
	for _, t := range m.Tris {
		for _, p := range t.P {
			mn.X = math.Min(mn.X, p.X)
			mn.Y = math.Min(mn.Y, p.Y)
			mn.Z = math.Min(mn.Z, p.Z)
			mx.X = math.Max(mx.X, p.X)
			mx.Y = math.Max(mx.Y, p.Y)
			mx.Z = math.Max(mx.Z, p.Z)
		}
	}
	c := mn.Add(mx).Mul(0.5)
	r := 0.0
	for _, t := range m.Tris {
		for _, p := range t.P {
			r = math.Max(r, p.Dist(c))
		}
	}
	return Bounds{Min: mn, Max: mx, Center: c, Radius: r}
}

type quantKey struct {
	X, Y, Z int64
}

func qkey(p Vec3) quantKey {
	const scale = 10000.0
	return quantKey{int64(math.Round(p.X * scale)), int64(math.Round(p.Y * scale)), int64(math.Round(p.Z * scale))}
}

type EdgeKey struct {
	A, B quantKey
}

func MakeEdgeKey(a, b Vec3) EdgeKey {
	ka, kb := qkey(a), qkey(b)
	if compareKey(ka, kb) > 0 {
		ka, kb = kb, ka
	}
	return EdgeKey{A: ka, B: kb}
}

func compareKey(a, b quantKey) int {
	if a.X != b.X {
		if a.X < b.X {
			return -1
		}
		return 1
	}
	if a.Y != b.Y {
		if a.Y < b.Y {
			return -1
		}
		return 1
	}
	if a.Z < b.Z {
		return -1
	}
	if a.Z > b.Z {
		return 1
	}
	return 0
}

func EdgeMap(m *Mesh) map[EdgeKey][]int {
	out := make(map[EdgeKey][]int, len(m.Tris)*3)
	for i, t := range m.Tris {
		for e := 0; e < 3; e++ {
			k := MakeEdgeKey(t.P[e], t.P[(e+1)%3])
			out[k] = append(out[k], i)
		}
	}
	return out
}

func sortedFaceIDs(ids []int) []int {
	sort.Ints(ids)
	return ids
}
