package main

import (
	"errors"
	"math"
	"sort"
)

// Basis is a clamped uniform B-spline basis on [0,1].
type Basis struct {
	Degree int
	Count  int
	Knots  []float64
}

func NewClampedBasis(count, degree int) Basis {
	if count < degree+1 {
		panic("B-spline needs count >= degree+1")
	}
	n := count + degree + 1
	knots := make([]float64, n)
	segments := count - degree
	for i := 0; i < n; i++ {
		switch {
		case i <= degree:
			knots[i] = 0
		case i >= count:
			knots[i] = 1
		default:
			knots[i] = float64(i-degree) / float64(segments)
		}
	}
	return Basis{Degree: degree, Count: count, Knots: knots}
}

func (b Basis) span(u float64) int {
	if u <= 0 {
		return b.Degree
	}
	if u >= 1 {
		return b.Count - 1
	}
	lo, hi := b.Degree, b.Count
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if u < b.Knots[mid] {
			hi = mid
		} else {
			lo = mid
		}
	}
	return lo
}

// Nonzero writes the degree+1 active basis values and returns their first index.
func (b Basis) Nonzero(u float64, dst []float64) int {
	u = clamp(u, 0, 1)
	k := b.span(u)
	p := b.Degree
	for i := range dst {
		dst[i] = 0
	}
	dst[0] = 1
	left := make([]float64, p+1)
	right := make([]float64, p+1)
	for d := 1; d <= p; d++ {
		left[d] = u - b.Knots[k+1-d]
		right[d] = b.Knots[k+d] - u
		saved := 0.0
		for r := 0; r < d; r++ {
			denom := right[r+1] + left[d-r]
			temp := 0.0
			if denom != 0 {
				temp = dst[r] / denom
			}
			dst[r] = saved + right[r+1]*temp
			saved = left[d-r] * temp
		}
		dst[d] = saved
	}
	return k - p
}

type BSplineSurface struct {
	U, V Basis
	C    [][]float64
}

func (s *BSplineSurface) Z(x, y float64) float64 {
	u := clamp((x-FitXMin)/(FitXMax-FitXMin), 0, 1)
	v := clamp((y-FitYMin)/(FitYMax-FitYMin), 0, 1)
	bu := make([]float64, s.U.Degree+1)
	bv := make([]float64, s.V.Degree+1)
	iu := s.U.Nonzero(u, bu)
	iv := s.V.Nonzero(v, bv)
	z := 0.0
	for a := 0; a <= s.U.Degree; a++ {
		for b := 0; b <= s.V.Degree; b++ {
			z += bu[a] * bv[b] * s.C[iu+a][iv+b]
		}
	}
	return z
}

func (s *BSplineSurface) Normal(x, y float64) Vec3 {
	h := 0.18
	fx := (s.Z(x+h, y) - s.Z(x-h, y)) / (2 * h)
	fy := (s.Z(x, y+h) - s.Z(x, y-h)) / (2 * h)
	return Vec3{-fx, -fy, 1}.Unit()
}

func (s *BSplineSurface) Curvature(x, y float64) (mean, gaussian float64) {
	h := 0.42
	z0 := s.Z(x, y)
	zx := (s.Z(x+h, y) - s.Z(x-h, y)) / (2 * h)
	zy := (s.Z(x, y+h) - s.Z(x, y-h)) / (2 * h)
	zxx := (s.Z(x+h, y) - 2*z0 + s.Z(x-h, y)) / (h * h)
	zyy := (s.Z(x, y+h) - 2*z0 + s.Z(x, y-h)) / (h * h)
	zxy := (s.Z(x+h, y+h) - s.Z(x+h, y-h) - s.Z(x-h, y+h) + s.Z(x-h, y-h)) / (4 * h * h)
	g := 1 + zx*zx + zy*zy
	mean = ((1+zy*zy)*zxx - 2*zx*zy*zxy + (1+zx*zx)*zyy) / (2 * math.Pow(g, 1.5))
	gaussian = (zxx*zyy - zxy*zxy) / (g * g)
	return
}

const (
	FitXMin = -PatchA
	FitXMax = PatchA
	FitYMin = -PatchB
	FitYMax = PatchB
)

type FitOptions struct {
	NU, NV      int
	Lambda      float64
	RobustIters int
}

type FitResult struct {
	Surface    *BSplineSurface
	Options    FitOptions
	InlierRMS  float64
	OutlierRMS float64
	Sigma      float64
	Iterations int
}

type fitSample struct {
	u, v   float64
	z      float64
	weight float64
}

func prepareSamples(points []ScanPoint) []fitSample {
	out := make([]fitSample, len(points))
	for i := range points {
		p := &points[i]
		out[i] = fitSample{
			u:      (p.X - FitXMin) / (FitXMax - FitXMin),
			v:      (p.Y - FitYMin) / (FitYMax - FitYMin),
			z:      p.Z,
			weight: 1,
		}
	}
	return out
}

func FitSurface(points []ScanPoint, opt FitOptions) (*FitResult, error) {
	if opt.NU < 4 || opt.NV < 4 {
		return nil, errors.New("control net must be at least 4x4")
	}
	if opt.RobustIters <= 0 {
		opt.RobustIters = 1
	}
	uBasis := NewClampedBasis(opt.NU, 3)
	vBasis := NewClampedBasis(opt.NV, 3)
	samples := prepareSamples(points)
	n := opt.NU * opt.NV
	coeff := make([]float64, n)
	var finalResiduals []float64
	iterations := 0

	for iter := 0; iter < opt.RobustIters; iter++ {
		iterations = iter + 1
		normal := make([]float64, n*n)
		rhs := make([]float64, n)
		bu := make([]float64, 4)
		bv := make([]float64, 4)
		feat := make([]float64, 16)
		idx := make([]int, 16)

		for si := range samples {
			s := &samples[si]
			iu := uBasis.Nonzero(s.u, bu)
			iv := vBasis.Nonzero(s.v, bv)
			k := 0
			for a := 0; a < 4; a++ {
				for b := 0; b < 4; b++ {
					feat[k] = bu[a] * bv[b]
					idx[k] = (iu+a)*opt.NV + iv + b
					k++
				}
			}
			w := s.weight
			for a := 0; a < 16; a++ {
				rhs[idx[a]] += w * feat[a] * s.z
				for b := 0; b < 16; b++ {
					normal[idx[a]*n+idx[b]] += w * feat[a] * feat[b]
				}
			}
		}

		addSmoothing(normal, opt.NU, opt.NV, opt.Lambda)
		for i := 0; i < n; i++ {
			normal[i*n+i] += 1e-7
		}
		sol, err := solveSPD(normal, rhs, n)
		if err != nil {
			return nil, err
		}
		copy(coeff, sol)
		surf := surfaceFromCoeff(uBasis, vBasis, opt, coeff)

		residuals := make([]float64, len(points))
		for i := range points {
			residuals[i] = points[i].Z - surf.Z(points[i].X, points[i].Y)
		}
		sigma := robustSigma(residuals)
		finalResiduals = residuals
		if iter == opt.RobustIters-1 {
			break
		}
		// Keep a material noise floor: with very clean synthetic scans a tiny
		// MAD would otherwise reject legitimate samples around occlusion holes.
		c := math.Max(2.2, 4.685*sigma)
		for i := range samples {
			r := math.Abs(residuals[i])
			if r < c {
				t := 1 - (r/c)*(r/c)
				samples[i].weight = t * t
			} else {
				samples[i].weight = 0
			}
		}
	}

	surf := surfaceFromCoeff(uBasis, vBasis, opt, coeff)
	sigma := robustSigma(finalResiduals)
	inlierN, outlierN := 0, 0
	inlierSS, outlierSS := 0.0, 0.0
	threshold := math.Max(0.95, 4.2*sigma)
	for i := range points {
		r := finalResiduals[i]
		points[i].Residual = r
		points[i].Rejected = math.Abs(r) > threshold
		if points[i].Rejected {
			outlierN++
			outlierSS += r * r
		} else {
			inlierN++
			inlierSS += r * r
		}
	}
	res := &FitResult{
		Surface: surf, Options: opt, Sigma: sigma, Iterations: iterations,
	}
	if inlierN > 0 {
		res.InlierRMS = math.Sqrt(inlierSS / float64(inlierN))
	}
	if outlierN > 0 {
		res.OutlierRMS = math.Sqrt(outlierSS / float64(outlierN))
	}
	return res, nil
}

func surfaceFromCoeff(u, v Basis, opt FitOptions, coeff []float64) *BSplineSurface {
	c := make([][]float64, opt.NU)
	for i := 0; i < opt.NU; i++ {
		c[i] = make([]float64, opt.NV)
		copy(c[i], coeff[i*opt.NV:(i+1)*opt.NV])
	}
	return &BSplineSurface{U: u, V: v, C: c}
}

func addSmoothing(a []float64, nu, nv int, lambda float64) {
	if lambda <= 0 {
		return
	}
	scale := lambda
	for i := 1; i < nu-1; i++ {
		for j := 0; j < nv; j++ {
			ids := []int{(i-1)*nv + j, i*nv + j, (i+1)*nv + j}
			row := []float64{1, -2, 1}
			addPenalty(a, nu*nv, ids, row, scale)
		}
	}
	for j := 1; j < nv-1; j++ {
		for i := 0; i < nu; i++ {
			ids := []int{i*nv + j - 1, i*nv + j, i*nv + j + 1}
			row := []float64{1, -2, 1}
			addPenalty(a, nu*nv, ids, row, scale*1.15)
		}
	}
}

func addPenalty(a []float64, n int, ids []int, row []float64, scale float64) {
	for i := range ids {
		for j := range ids {
			a[ids[i]*n+ids[j]] += scale * row[i] * row[j]
		}
	}
}

func robustSigma(residuals []float64) float64 {
	abs := make([]float64, len(residuals))
	for i, r := range residuals {
		abs[i] = math.Abs(r)
	}
	sort.Float64s(abs)
	med := abs[len(abs)/2]
	for i := range abs {
		abs[i] = math.Abs(abs[i] - med)
	}
	sort.Float64s(abs)
	mad := abs[len(abs)/2]
	return math.Max(0.18, 1.4826*mad)
}

func solveSPD(a, b []float64, n int) ([]float64, error) {
	m := make([]float64, len(a))
	copy(m, a)
	x := make([]float64, n)
	copy(x, b)
	for k := 0; k < n; k++ {
		pivot := k
		for i := k + 1; i < n; i++ {
			if math.Abs(m[i*n+k]) > math.Abs(m[pivot*n+k]) {
				pivot = i
			}
		}
		if math.Abs(m[pivot*n+k]) < 1e-12 {
			return nil, errors.New("singular fitting system")
		}
		if pivot != k {
			for j := k; j < n; j++ {
				m[k*n+j], m[pivot*n+j] = m[pivot*n+j], m[k*n+j]
			}
			x[k], x[pivot] = x[pivot], x[k]
		}
		pv := m[k*n+k]
		for i := k + 1; i < n; i++ {
			f := m[i*n+k] / pv
			if f == 0 {
				continue
			}
			for j := k; j < n; j++ {
				m[i*n+j] -= f * m[k*n+j]
			}
			x[i] -= f * x[k]
		}
	}
	for i := n - 1; i >= 0; i-- {
		s := x[i]
		for j := i + 1; j < n; j++ {
			s -= m[i*n+j] * x[j]
		}
		x[i] = s / m[i*n+i]
	}
	return x, nil
}
