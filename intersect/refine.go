package intersect

import (
	"math"

	"ruledintersect/geom"
)

// refineIntersection solves S1(u1,v1)=S2(u2,v2) with damped minimum-norm
// Newton steps. The system has four parameters and three equations; choosing
// the minimum-norm correction keeps the solution near the seed on the
// intersection curve instead of drifting along it.
func refineIntersection(s1, s2 *geom.RuledSurface, p geom.Vec3,
	p1, p2 geom.UV, opts Options) (geom.Vec3, float64, bool) {
	q := p
	uva, uvb := p1, p2
	best := math.Inf(1)
	bestQ := q
	for iter := 0; iter < opts.MaxRefineIterations; iter++ {
		a := s1.Point(uva.U, uva.V)
		b := s2.Point(uvb.U, uvb.V)
		r := a.Sub(b)
		gap := r.Len()
		if gap < best {
			best, bestQ = gap, a.Lerp(b, 0.5)
		}
		if gap <= 1e-10 {
			return bestQ, gap, true
		}
		daU, daV := s1.Derivatives(uva.U, uva.V)
		dbU, dbV := s2.Derivatives(uvb.U, uvb.V)
		cols := [4]geom.Vec3{daU, daV, dbU.Mul(-1), dbV.Mul(-1)}
		delta, ok := minNormSolve(cols, r.Mul(-1))
		if !ok {
			break
		}
		scale := 1.0
		for k := 0; k < 4; k++ {
			if math.Abs(delta[k]) > 0.2 {
				scale = math.Min(scale, 0.2/math.Abs(delta[k]))
			}
		}
		uva.U = geom.Clamp(uva.U+scale*delta[0], s1.U0, s1.U1)
		uva.V = geom.Clamp(uva.V+scale*delta[1], s1.V0, s1.V1)
		uvb.U = geom.Clamp(uvb.U+scale*delta[2], s2.U0, s2.U1)
		uvb.V = geom.Clamp(uvb.V+scale*delta[3], s2.V0, s2.V1)
	}
	if best <= opts.OnSurfaceTolerance*0.5 {
		return bestQ, best, true
	}
	return alternateProject(s1, s2, q, opts)
}

func alternateProject(s1, s2 *geom.RuledSurface, p geom.Vec3,
	opts Options) (geom.Vec3, float64, bool) {
	q := p
	gap := math.Inf(1)
	for i := 0; i < opts.MaxRefineIterations*2; i++ {
		_, _, a := s1.Project(q)
		_, _, b := s2.Project(q)
		gap = a.Dist(b)
		nq := a.Lerp(b, 0.5)
		if nq.Dist(q) < 1e-10 {
			q = nq
			break
		}
		q = nq
	}
	if gap <= opts.OnSurfaceTolerance*2 {
		return q, gap, true
	}
	return q, gap, false
}

// minNormSolve returns the minimum Euclidean-norm solution of J*delta=r,
// where J is a 3x4 matrix supplied by columns.
func minNormSolve(cols [4]geom.Vec3, r geom.Vec3) ([4]float64, bool) {
	var g [3][3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 4; k++ {
				g[i][j] += cols[k].At(i) * cols[k].At(j)
			}
		}
	}
	// Light damping makes tangent/near-parallel systems well behaved.
	trace := g[0][0] + g[1][1] + g[2][2]
	damp := math.Max(1e-12, trace*1e-10)
	g[0][0] += damp
	g[1][1] += damp
	g[2][2] += damp
	lambda, ok := solve3(g, [3]float64{r.X, r.Y, r.Z})
	if !ok {
		return [4]float64{}, false
	}
	var delta [4]float64
	for k := 0; k < 4; k++ {
		delta[k] = cols[k].X*lambda[0] + cols[k].Y*lambda[1] + cols[k].Z*lambda[2]
	}
	if math.IsNaN(delta[0]) || math.IsInf(delta[0], 0) {
		return [4]float64{}, false
	}
	return delta, true
}

func solve3(a [3][3]float64, b [3]float64) ([3]float64, bool) {
	m := a
	rhs := b
	for col := 0; col < 3; col++ {
		p := col
		for row := col + 1; row < 3; row++ {
			if math.Abs(m[row][col]) > math.Abs(m[p][col]) {
				p = row
			}
		}
		if math.Abs(m[p][col]) < 1e-18 {
			return [3]float64{}, false
		}
		if p != col {
			m[p], m[col] = m[col], m[p]
			rhs[p], rhs[col] = rhs[col], rhs[p]
		}
		for row := col + 1; row < 3; row++ {
			f := m[row][col] / m[col][col]
			for k := col; k < 3; k++ {
				m[row][k] -= f * m[col][k]
			}
			rhs[row] -= f * rhs[col]
		}
	}
	var x [3]float64
	for row := 2; row >= 0; row-- {
		sum := rhs[row]
		for k := row + 1; k < 3; k++ {
			sum -= m[row][k] * x[k]
		}
		x[row] = sum / m[row][row]
	}
	return x, true
}
