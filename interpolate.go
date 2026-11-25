package main

import (
	"fmt"
	"math"
)

const worldScaleMM = 200.0

type StrandRef struct {
	Indices [2]int
	Weights [2]float64
}

type Link struct {
	OutputIndex   int
	A             StrandRef
	B             StrandRef
	MigrationMM   []float64
	MaxMigration  float64
	MeanMigration float64
	aControls     []Vec3
	bControls     []Vec3
	aRoot         Vec3
	bRoot         Vec3
}

type Frame struct {
	T               float64
	Pose            Pose
	Strands         []Strand
	Links           []Link
	MaxMigrationMM  float64
	MeanMigrationMM float64
}

type Interpolator struct {
	A      Groom
	B      Groom
	orderA []int
	orderB []int
}

func NewInterpolator(a, b Groom) (*Interpolator, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	if a.Levels != b.Levels {
		return nil, fmt.Errorf("control point counts differ: %d and %d", a.Levels, b.Levels)
	}
	return &Interpolator{
		A:      a,
		B:      b,
		orderA: atlasIndices(a),
		orderB: atlasIndices(b),
	}, nil
}

func (it *Interpolator) virtual(g Groom, order []int, u float64) (Strand, StrandRef) {
	n := len(order)
	p := clamp(u*float64(n-1), 0, float64(n-1))
	i0 := int(math.Floor(p))
	if i0 >= n-1 {
		i0 = n - 1
	}
	i1 := i0 + 1
	if i1 >= n {
		i1 = n - 1
	}
	w1 := p - float64(i0)
	if i0 == n-1 {
		w1 = 0
	}
	w0 := 1 - w1
	s0 := g.Strands[order[i0]]
	s1 := g.Strands[order[i1]]
	if w1 == 0 {
		controls := append([]Vec3(nil), s0.Controls...)
		return Strand{ID: -1, Controls: controls, Width: s0.Width}, StrandRef{
			Indices: [2]int{order[i0], order[i1]},
			Weights: [2]float64{1, 0},
		}
	}
	controls := make([]Vec3, g.Levels)
	for level := range controls {
		controls[level] = s0.Controls[level].Mul(w0).Add(s1.Controls[level].Mul(w1))
	}
	return Strand{
		ID:       -1,
		Controls: controls,
		Width:    s0.Width*w0 + s1.Width*w1,
	}, StrandRef{
		Indices: [2]int{order[i0], order[i1]},
		Weights: [2]float64{w0, w1},
	}
}

func (it *Interpolator) outputCount(t float64) int {
	n := int(math.Round(lerp(float64(len(it.A.Strands)), float64(len(it.B.Strands)), t)))
	return max(n, 2)
}

func (it *Interpolator) Frame(t float64) Frame {
	t = clamp(t, 0, 1)
	if t == 0 {
		return it.endpointFrame(0)
	}
	if t == 1 {
		return it.endpointFrame(1)
	}
	k := it.outputCount(t)
	frame := Frame{
		T:              t,
		Pose:           it.A.Pose.Lerp(it.B.Pose, t),
		Strands:        make([]Strand, 0, k),
		Links:          make([]Link, 0, k),
		MaxMigrationMM: 0,
	}
	totalMigration := 0.0
	for j := 0; j < k; j++ {
		u := 0.0
		if k > 1 {
			u = float64(j) / float64(k-1)
		}
		va, refA := it.virtual(it.A, it.orderA, u)
		vb, refB := it.virtual(it.B, it.orderB, u)
		controls := make([]Vec3, len(va.Controls))
		for level := range controls {
			controls[level] = va.Controls[level].Lerp(vb.Controls[level], t)
		}
		out := Strand{
			ID:       j,
			Controls: controls,
			Width:    lerp(va.Width, vb.Width, t),
		}
		link := makeLink(j, refA, refB, va, vb)
		frame.Strands = append(frame.Strands, out)
		frame.Links = append(frame.Links, link)
		frame.MaxMigrationMM = math.Max(frame.MaxMigrationMM, link.MaxMigration)
		totalMigration += link.MeanMigration
	}
	if len(frame.Links) > 0 {
		frame.MeanMigrationMM = totalMigration / float64(len(frame.Links))
	}
	return frame
}

func makeLink(index int, refA, refB StrandRef, va, vb Strand) Link {
	migration := make([]float64, len(va.Controls))
	linkMax := 0.0
	linkTotal := 0.0
	for level := range migration {
		migration[level] = va.Controls[level].Dist(vb.Controls[level]) * worldScaleMM
		linkMax = math.Max(linkMax, migration[level])
		linkTotal += migration[level]
	}
	return Link{
		OutputIndex:   index,
		A:             refA,
		B:             refB,
		MigrationMM:   migration,
		MaxMigration:  linkMax,
		MeanMigration: linkTotal / float64(len(migration)),
		aControls:     va.Controls,
		bControls:     vb.Controls,
		aRoot:         va.Root(),
		bRoot:         vb.Root(),
	}
}

func (it *Interpolator) endpointFrame(endpoint int) Frame {
	source := it.A
	other := it.B
	otherOrder := it.orderB
	refFromA := true
	if endpoint == 1 {
		source = it.B
		other = it.A
		otherOrder = it.orderA
		refFromA = false
	}
	frame := Frame{T: float64(endpoint), Pose: source.Pose}
	frame.Strands = make([]Strand, 0, len(source.Strands))
	frame.Links = make([]Link, 0, len(source.Strands))
	for j, sourceIndex := range atlasIndices(source) {
		u := float64(j) / float64(len(source.Strands)-1)
		src := source.Strands[sourceIndex]
		virtOther, otherRef := it.virtual(other, otherOrder, u)
		ref := StrandRef{Indices: [2]int{sourceIndex, sourceIndex}, Weights: [2]float64{1, 0}}
		out := Strand{ID: j, Controls: append([]Vec3(nil), src.Controls...), Width: src.Width}
		var link Link
		if refFromA {
			link = makeLink(j, ref, otherRef, src, virtOther)
		} else {
			link = makeLink(j, otherRef, ref, virtOther, src)
		}
		frame.Strands = append(frame.Strands, out)
		frame.Links = append(frame.Links, link)
		frame.MaxMigrationMM = math.Max(frame.MaxMigrationMM, link.MaxMigration)
	}
	if len(frame.Links) > 0 {
		total := 0.0
		for _, link := range frame.Links {
			total += link.MeanMigration
		}
		frame.MeanMigrationMM = total / float64(len(frame.Links))
	}
	return frame
}
