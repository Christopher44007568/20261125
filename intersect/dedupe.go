package intersect

import "math"

// classifyDuplicates keeps the longest representative of each coincident
// support. Duplicates are retained and marked, because the renderer uses them
// to distinguish redundant triangle-pair output from the final valid curve.
func classifyDuplicates(segs []Segment, tolerance float64) []Segment {
	if tolerance <= 0 {
		tolerance = 2e-5
	}
	for j := range segs {
		for i := range segs {
			if i == j || segs[j].Length() > segs[i].Length() {
				continue
			}
			if i > j && math.Abs(segs[j].Length()-segs[i].Length()) < tolerance {
				continue
			}
			if segmentsCoincide(segs[i], segs[j], tolerance) {
				segs[j].Status = Duplicate
				break
			}
		}
	}
	return segs
}

func segmentsCoincide(a, b Segment, tol float64) bool {
	la, lb := a.Length(), b.Length()
	if la <= tol || lb <= tol {
		return false
	}
	ua := a.B.Sub(a.A)
	pa := math.Abs(ua.Cross(b.A.Sub(a.A)).Len() / la)
	pb := math.Abs(ua.Cross(b.B.Sub(a.A)).Len() / la)
	if math.Max(pa, pb) > tol*2 {
		return false
	}
	t0 := b.A.Sub(a.A).Dot(ua) / (la * la)
	t1 := b.B.Sub(a.A).Dot(ua) / (la * la)
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	lo, hi := math.Max(0, t0), math.Min(1, t1)
	if hi <= lo {
		return false
	}
	overlap := (hi - lo) * la
	return overlap > math.Max(3*tol, 0.25*math.Min(la, lb))
}
