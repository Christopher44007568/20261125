package spectralresample

import "math"

// QR decomposes the m-by-n matrix A (m >= n) into A = Q R using
// Householder reflectors. Q is returned implicitly as a compact set of
// reflectors (vectors V with V[0:k] = 0, V[k] = 1) and the upper
// triangular R; this is enough to solve least squares and null-space
// problems without any external linear algebra package.
type QR struct {
	V [][]float64
	R [][]float64
	M int
	N int
}

func NewQR(a [][]float64) QR {
	m, n := len(a), len(a[0])
	r := make([][]float64, m)
	for i := range r {
		r[i] = append([]float64(nil), a[i]...)
	}
	vs := make([][]float64, 0, n)
	for k := 0; k < n && k < m-1; k++ {
		v := make([]float64, m)
		norm := 0.0
		for i := k; i < m; i++ {
			v[i] = r[i][k]
			norm += v[i] * v[i]
		}
		norm = math.Sqrt(norm)
		if norm == 0 {
			continue
		}
		sign := 1.0
		if v[k] < 0 {
			sign = -1
		}
		v[k] += sign * norm
		vNorm := 0.0
		for i := k; i < m; i++ {
			vNorm += v[i] * v[i]
		}
		if vNorm == 0 {
			continue
		}
		scale := math.Sqrt(2 / vNorm)
		for i := k; i < m; i++ {
			v[i] *= scale
		}
		for j := k; j < n; j++ {
			dot := 0.0
			for i := k; i < m; i++ {
				dot += v[i] * r[i][j]
			}
			for i := k; i < m; i++ {
				r[i][j] -= v[i] * dot
			}
		}
		vs = append(vs, v)
	}
	return QR{V: vs, R: r, M: m, N: n}
}

// applyQ computes Q y from the stored Householder reflectors, applying them
// in reverse order.
func (q QR) applyQ(y []float64) []float64 {
	x := append([]float64(nil), y...)
	for k := len(q.V) - 1; k >= 0; k-- {
		v := q.V[k]
		dot := 0.0
		for i := range x {
			dot += v[i] * x[i]
		}
		for i := range x {
			x[i] -= v[i] * dot
		}
	}
	return x
}

// NullSpace returns an orthonormal basis of the null space of the original
// matrix A when the receiver is the QR decomposition of A^T. For an m-by-n
// A with m < n, the result spans ker(A) in the n-dimensional domain.
func (q QR) NullSpace() [][]float64 {
	rank := 0
	maxDiag := 0.0
	for k := 0; k < q.N && k < q.M; k++ {
		maxDiag = math.Max(maxDiag, math.Abs(q.R[k][k]))
	}
	if maxDiag == 0 {
		maxDiag = 1
	}
	for k := 0; k < q.N && k < q.M; k++ {
		if math.Abs(q.R[k][k]) > 1e-12*maxDiag {
			rank++
		}
	}
	var basis [][]float64
	for free := rank; free < q.M; free++ {
		// Coordinates in the transformed (Q^T) space: a unit free
		// coordinate, then back-substitute the pivot rows of R.
		y := make([]float64, q.M)
		y[free] = 1
		for k := rank - 1; k >= 0; k-- {
			sum := 0.0
			for j := k + 1; j < q.N; j++ {
				sum += q.R[k][j] * y[j]
			}
			y[k] = -sum / q.R[k][k]
		}
		x := q.applyQ(y)
		normalize(x)
		basis = append(basis, x)
	}
	return basis
}

func normalize(v []float64) {
	norm := 0.0
	for _, value := range v {
		norm += value * value
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return
	}
	for i := range v {
		v[i] /= norm
	}
}

// dot returns the Euclidean inner product of two vectors.
func dot(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}
