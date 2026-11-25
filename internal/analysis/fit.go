package analysis

import (
	"math"

	"meshscope/internal/mesh"
)

type planeFit struct {
	normal mesh.Vec3
	offset float64
	rms    float64
}

func fitPlane(t topology, faces []int) planeFit {
	var c mesh.Vec3
	area := 0.0
	for _, f := range faces {
		w := math.Max(t.area[f], 1e-6)
		c = c.Add(t.center[f].Mul(w))
		area += w
	}
	if area > 0 {
		c = c.Mul(1 / area)
	}
	var cov [3][3]float64
	for _, f := range faces {
		w := math.Max(t.area[f], 1e-6)
		d := t.center[f].Sub(c)
		cov[0][0] += w * d.X * d.X
		cov[0][1] += w * d.X * d.Y
		cov[0][2] += w * d.X * d.Z
		cov[1][1] += w * d.Y * d.Y
		cov[1][2] += w * d.Y * d.Z
		cov[2][2] += w * d.Z * d.Z
	}
	cov[1][0], cov[2][0], cov[2][1] = cov[0][1], cov[0][2], cov[1][2]
	vals, vecs := eigenSym(cov)
	_ = vals
	n := mesh.V(VecAt(vecs, 0, 0), VecAt(vecs, 1, 0), VecAt(vecs, 2, 0)).Norm()
	if n.Dot(t.averageNormal(faces)) < 0 {
		n = n.Mul(-1)
	}
	off := n.Dot(c)
	rms := 0.0
	for _, f := range faces {
		d := n.Dot(t.center[f]) - off
		rms += t.area[f] * d * d
	}
	total := 0.0
	for _, f := range faces {
		total += t.area[f]
	}
	if total > 0 {
		rms = math.Sqrt(rms / total)
	}
	return planeFit{normal: n, offset: off, rms: rms}
}

type cylinderFit struct {
	axis      mesh.Vec3
	center    mesh.Vec3
	u, v      mesh.Vec3
	radius    float64
	normalRMS float64
	radialRMS float64
	span      float64
	signed    float64
	valid     bool
}

func fitCylinder(t topology, faces []int) cylinderFit {
	var cov [3][3]float64
	for _, f := range faces {
		w := math.Max(t.area[f], 1e-6)
		n := t.normal[f]
		cov[0][0] += w * n.X * n.X
		cov[0][1] += w * n.X * n.Y
		cov[0][2] += w * n.X * n.Z
		cov[1][1] += w * n.Y * n.Y
		cov[1][2] += w * n.Y * n.Z
		cov[2][2] += w * n.Z * n.Z
	}
	cov[1][0], cov[2][0], cov[2][1] = cov[0][1], cov[0][2], cov[1][2]
	vals, vecs := eigenSym(cov)
	axis := mesh.V(VecAt(vecs, 0, 0), VecAt(vecs, 1, 0), VecAt(vecs, 2, 0)).Norm()
	if vals[1] < vals[0]*1.0001 {
		_ = vals
	}
	u := axis.Cross(mesh.V(0, 1, 0))
	if u.Len() < 1e-5 {
		u = axis.Cross(mesh.V(1, 0, 0))
	}
	u = u.Norm()
	v := axis.Cross(u).Norm()

	var best cylinderFit
	bestScore := math.Inf(1)
	for _, sign := range []float64{1, -1} {
		var a00, a01, a02, a11, a12, a22, b0, b1, b2 float64
		for _, f := range faces {
			w := math.Max(t.area[f], 1e-6)
			n, p := t.normal[f], t.center[f]
			nu, nv := n.Dot(u), n.Dot(v)
			a00 += w * nu * nu
			a01 += w * nu * nv
			a02 += w * nu * sign
			a11 += w * nv * nv
			a12 += w * nv * sign
			a22 += w
			np := n.Dot(p)
			b0 += w * nu * np
			b1 += w * nv * np
			b2 += w * sign * np
		}
		sol, ok := solve3([3][3]float64{
			{a00, a01, a02}, {a01, a11, a12}, {a02, a12, a22},
		}, [3]float64{b0, b1, b2})
		if !ok || sol[2] <= 0 {
			continue
		}
		center := u.Mul(sol[0]).Add(v.Mul(sol[1]))
		cf, score := evalCylinder(t, faces, axis, center, u, v, sol[2])
		if score < bestScore {
			best, bestScore = cf, score
		}
	}
	if math.IsInf(bestScore, 1) {
		return cylinderFit{}
	}
	return best
}

func evalCylinder(t topology, faces []int, axis, center, u, v mesh.Vec3, radius float64) (cylinderFit, float64) {
	var nAng, rErr, total, orientSum, angSum float64
	for _, f := range faces {
		w := math.Max(t.area[f], 1e-6)
		d := t.center[f].Sub(center)
		d = d.Sub(axis.Mul(d.Dot(axis)))
		dist := d.Len()
		orientSum += w * t.normal[f].Dot(d.Norm())
		rErr += w * (dist - radius) * (dist - radius)
		total += w
	}
	if total <= 0 {
		return cylinderFit{}, math.Inf(1)
	}
	orient := 1.0
	if orientSum < 0 {
		orient = -1
	}
	for _, f := range faces {
		w := math.Max(t.area[f], 1e-6)
		d := t.center[f].Sub(center)
		d = d.Sub(axis.Mul(d.Dot(axis)))
		rn := d.Norm().Mul(orient)
		ang := angleDeg(t.normal[f], rn)
		nAng += w * ang * ang
		angSum += w * ang
	}
	normalRMS := math.Sqrt(nAng / total)
	radialRMS := math.Sqrt(rErr / total)

	var angles []float64
	for _, f := range faces {
		d := t.center[f].Sub(center)
		d = d.Sub(axis.Mul(d.Dot(axis)))
		angles = append(angles, math.Atan2(d.Dot(v), d.Dot(u)))
	}
	span := angularSpan(angles)
	signed := 0.0
	for _, f := range faces {
		d := t.center[f].Sub(center)
		d = d.Sub(axis.Mul(d.Dot(axis)))
		signed += math.Max(t.area[f], 1e-6) * t.normal[f].Dot(d.Norm())
	}
	signed /= areaSum(t, faces)
	score := normalRMS + radialRMS*8
	return cylinderFit{
		axis: axis, center: center, u: u, v: v, radius: radius,
		normalRMS: normalRMS, radialRMS: radialRMS, span: span,
		signed: signed, valid: normalRMS < 17 && span > 5,
	}, score
}

func solve3(a [3][3]float64, b [3]float64) ([3]float64, bool) {
	m := [3][4]float64{
		{a[0][0], a[0][1], a[0][2], b[0]},
		{a[1][0], a[1][1], a[1][2], b[1]},
		{a[2][0], a[2][1], a[2][2], b[2]},
	}
	for col := 0; col < 3; col++ {
		pivot := col
		for r := col + 1; r < 3; r++ {
			if math.Abs(m[r][col]) > math.Abs(m[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(m[pivot][col]) < 1e-7 {
			return [3]float64{}, false
		}
		m[col], m[pivot] = m[pivot], m[col]
		for r := 0; r < 3; r++ {
			if r == col {
				continue
			}
			f := m[r][col] / m[col][col]
			for c := col; c < 4; c++ {
				m[r][c] -= f * m[col][c]
			}
		}
	}
	var out [3]float64
	for i := 0; i < 3; i++ {
		out[i] = m[i][3] / m[i][i]
	}
	return out, true
}

func angularSpan(angles []float64) float64 {
	if len(angles) == 0 {
		return 0
	}
	for i, a := range angles {
		if a < 0 {
			angles[i] = a + 2*math.Pi
		}
	}
	sortFloats(angles)
	maxGap := angles[0] + 2*math.Pi - angles[len(angles)-1]
	for i := 1; i < len(angles); i++ {
		if g := angles[i] - angles[i-1]; g > maxGap {
			maxGap = g
		}
	}
	return (2*math.Pi - maxGap) * 180 / math.Pi
}

func areaSum(t topology, faces []int) float64 {
	sum := 0.0
	for _, f := range faces {
		sum += t.area[f]
	}
	return math.Max(sum, 1e-9)
}

func VecAt(m [3][3]float64, r, c int) float64 { return m[r][c] }

func sortFloats(v []float64) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
