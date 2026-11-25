package section

import (
	"math"

	"brepsection/geom"
)

// roundedPoint rounds every coordinate through a float32 round trip when the
// mode requests the reduced-precision stress pipeline.
func roundedPoint(p geom.Vec3, use32 bool) geom.Vec3 {
	if !use32 {
		return p
	}
	return geom.V3(float64(float32(p.X)), float64(float32(p.Y)), float64(float32(p.Z)))
}

// signedDistanceRounded evaluates the plane equation with float32 round trips
// on each multiply/add when the stress pipeline is selected.
func signedDistanceRounded(pl geom.Plane, q geom.Vec3, use32 bool) float64 {
	d := q.Sub(pl.Origin)
	if !use32 {
		return d.Dot(pl.Normal)
	}
	r32 := func(v float64) float64 { return float64(float32(v)) }
	s := r32(r32(d.X*pl.Normal.X) + r32(d.Y*pl.Normal.Y))
	return r32(s + r32(d.Z*pl.Normal.Z))
}

// transition describes one place where the loop boundary enters or leaves the
// negative half-space of the cutting plane.
type transition struct {
	order    int
	intoNeg  bool
	point    geom.Vec3
	runChain []geom.Vec3 // non-nil for on-plane boundary runs
	fromRun  bool        // transition point belongs to an on-plane run
	flip     bool        // true when the half-space side actually changes
}

// loopSection walks one polygon loop and emits every straight section piece
// lying on the cutting plane. It uses only the sign of per-vertex distances,
// so loop winding does not matter. On-plane vertex runs become explicit
// entry/exit transitions, which keeps single-vertex tangencies and exact
// through-vertices numerically stable.
func loopSection(pts []geom.Vec3, loop []int, pl geom.Plane, eps float64, use32 bool, outer bool, emit func(a, b geom.Vec3, boundary, outerLoop bool)) {
	m := len(loop)
	p := make([]geom.Vec3, m)
	d := make([]float64, m)
	on := make([]bool, m)
	for i, idx := range loop {
		p[i] = roundedPoint(pts[idx], use32)
		d[i] = signedDistanceRounded(pl, p[i], use32)
		on[i] = math.Abs(d[i]) <= eps
	}

	var tr []transition
	used := make([]bool, m)

	// Whole loop lying on the plane (cap face cut exactly at its z level).
	allOn := true
	for _, v := range on {
		if !v {
			allOn = false
			break
		}
	}
	if allOn {
		for i := 0; i < m; i++ {
			emit(p[i], p[(i+1)%m], true, outer)
		}
		return
	}

	// Pass 1: maximal cyclic runs of on-plane vertices.
	for i := 0; i < m; i++ {
		if !on[i] || used[i] {
			continue
		}
		var run []int
		j := i
		for on[j] && !used[j] {
			used[j] = true
			run = append(run, j)
			j = (j + 1) % m
		}
		prev := (run[0] - 1 + m) % m
		last := run[len(run)-1]
		next := (last + 1) % m
		chain := chainPoints(p, run)
		negBefore, negAfter := d[prev] < 0, d[next] < 0

		switch {
		case negBefore == negAfter:
			// Tangential run: no half-space change. Multi-vertex runs still
			// contribute their on-plane boundary edges; a single tangent
			// vertex is inserted as a no-op transition so the alternating
			// pairing continues through it instead of truncating a chord.
			if len(run) > 1 {
				for k := 0; k+1 < len(chain); k++ {
					emit(chain[k], chain[k+1], true, outer)
				}
			} else {
				tr = append(tr, transition{
					order: run[0], point: p[run[0]], fromRun: true,
				})
			}
		default:
			// Crossing run: entry at its first vertex, exit at its last.
			tr = append(tr, transition{
				order: run[0], intoNeg: negAfter, point: p[run[0]], fromRun: true, flip: true,
			})
			tr = append(tr, transition{
				order: last, intoNeg: !negAfter, point: p[last],
				runChain: chain, fromRun: true, flip: true,
			})
		}
	}

	// Pass 2: transversal edge crossings between two off-plane vertices.
	for i := 0; i < m; i++ {
		j := (i + 1) % m
		if on[i] || on[j] || d[i]*d[j] >= 0 {
			continue
		}
		t := -d[i] / (d[j] - d[i])
		q := roundedPoint(interp(p[i], p[j], t, use32), use32)
		tr = append(tr, transition{
			order: i, intoNeg: d[j] < 0, point: q, flip: true,
		})
	}

	insertionSort(tr)

	// Drive an explicit inside/outside state. Flip transitions open or close
	// a section chord; tangent vertices do not change the state but replace
	// the open anchor so the chord extends through the touch point.
	var entry geom.Vec3
	have := false
	var entryRun bool
	for _, e := range tr {
		for k := 0; k+1 < len(e.runChain); k++ {
			emit(e.runChain[k], e.runChain[k+1], true, outer)
		}
		if !e.flip {
			if have {
				entry = e.point // extend the open chord through tangency
			}
			continue
		}
		if !have {
			entry, have, entryRun = e.point, true, e.fromRun
			continue
		}
		if entry.Approx(e.point, eps) {
			continue
		}
		// Chords ending at an on-plane run outline holes or caps and are
		// boundary geometry; pure crossing chords take the material test.
		emit(entry, e.point, entryRun || e.fromRun, outer)
		have = false
	}
}

func insertionSort(tr []transition) {
	for i := 1; i < len(tr); i++ {
		for j := i; j > 0 && tr[j-1].order > tr[j].order; j-- {
			tr[j-1], tr[j] = tr[j], tr[j-1]
		}
	}
}

func chainPoints(p []geom.Vec3, run []int) []geom.Vec3 {
	chain := make([]geom.Vec3, len(run))
	for k, v := range run {
		chain[k] = p[v]
	}
	return chain
}

// interp evaluates p + t*(q-p), rounding every arithmetic step through
// float32 in stress mode so endpoint residuals become observable.
func interp(p, q geom.Vec3, t float64, use32 bool) geom.Vec3 {
	if !use32 {
		return p.Add(q.Sub(p).Scale(t))
	}
	r := func(v float64) float64 { return float64(float32(v)) }
	return geom.V3(
		r(p.X+r(t*r(q.X-p.X))),
		r(p.Y+r(t*r(q.Y-p.Y))),
		r(p.Z+r(t*r(q.Z-p.Z))),
	)
}
