package analysis

import "math"

func eigenSym(a [3][3]float64) ([3]float64, [3][3]float64) {
	m := a
	v := [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	for iter := 0; iter < 32; iter++ {
		p, q := 0, 1
		max := math.Abs(m[0][1])
		if math.Abs(m[0][2]) > max {
			p, q, max = 0, 2, math.Abs(m[0][2])
		}
		if math.Abs(m[1][2]) > max {
			p, q, max = 1, 2, math.Abs(m[1][2])
		}
		if max < 1e-12 {
			break
		}
		phi := 0.5 * math.Atan2(2*m[p][q], m[q][q]-m[p][p])
		c, s := math.Cos(phi), math.Sin(phi)
		for k := 0; k < 3; k++ {
			mkp, mkq := m[k][p], m[k][q]
			m[k][p] = c*mkp - s*mkq
			m[k][q] = s*mkp + c*mkq
		}
		for k := 0; k < 3; k++ {
			mpk, mqk := m[p][k], m[q][k]
			m[p][k] = c*mpk - s*mqk
			m[q][k] = s*mpk + c*mqk
		}
		for k := 0; k < 3; k++ {
			vkp, vkq := v[k][p], v[k][q]
			v[k][p] = c*vkp - s*vkq
			v[k][q] = s*vkp + c*vkq
		}
	}
	vals := [3]float64{m[0][0], m[1][1], m[2][2]}
	for i := 0; i < 2; i++ {
		for j := i + 1; j < 3; j++ {
			if vals[j] < vals[i] {
				vals[i], vals[j] = vals[j], vals[i]
				for k := 0; k < 3; k++ {
					v[k][i], v[k][j] = v[k][j], v[k][i]
				}
			}
		}
	}
	return vals, v
}
