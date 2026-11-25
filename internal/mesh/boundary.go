package mesh

import "adaptivebezier/internal/geom"

type boundaryPoint struct {
	Param  geom.Vec2
	Screen geom.Vec2
	World  geom.Vec3
	Span   int
	T      float64
}

type boundarySampler struct {
	patch    geom.Patch
	trim     geom.TrimLoop
	project  geom.Projector
	tol      float64
	maxDepth int
	points   []boundaryPoint
}

func (s *boundarySampler) run() []boundaryPoint {
	s.points = s.points[:0]
	for i, span := range s.trim.Spans {
		start := span.Eval(0)
		world := s.patch.Eval(start.X, start.Y)
		s.points = append(s.points, boundaryPoint{
			Param: start, Screen: s.project.Surface(s.patch, start.X, start.Y),
			World: world, Span: i, T: 0,
		})
		s.refine(i, span, 0, 0, 1)
	}
	if len(s.points) > 1 {
		a := s.points[0].Param
		b := s.points[len(s.points)-1].Param
		if a.Sub(b).Len() < 1e-10 {
			s.points = s.points[:len(s.points)-1]
		}
	}
	return s.points
}

func (s *boundarySampler) refine(spanID int, span geom.Span, depth int, t0, t1 float64) {
	p1 := span.Eval(t1)
	q1 := s.project.Surface(s.patch, p1.X, p1.Y)
	w1 := s.patch.Eval(p1.X, p1.Y)
	sampleT := []float64{0.25, 0.5, 0.75}
	maxErr := 0.0
	for _, f := range sampleT {
		t := t0 + (t1-t0)*f
		q := span.Eval(t)
		world := s.patch.Eval(q.X, q.Y)
		err := distPointLine3D(world, s.points[len(s.points)-1].World, w1) * s.project.Scale
		if err > maxErr {
			maxErr = err
		}
	}
	if maxErr <= s.tol || depth >= s.maxDepth {
		s.points = append(s.points, boundaryPoint{Param: p1, Screen: q1, World: w1, Span: spanID, T: t1})
		return
	}
	mid := (t0 + t1) * 0.5
	s.refine(spanID, span, depth+1, t0, mid)
	s.refine(spanID, span, depth+1, mid, t1)
}

func distPointLine3D(p, a, b geom.Vec3) float64 {
	d := b.Sub(a)
	l2 := d.Dot(d)
	if l2 < 1e-20 {
		return p.Sub(a).Len()
	}
	u := p.Sub(a).Dot(d) / l2
	q := a.Add(d.Mul(u))
	return p.Sub(q).Len()
}

func (s *boundarySampler) contourError(points []boundaryPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	maxErr := 0.0
	for i := range points {
		a := points[i]
		b := points[(i+1)%len(points)]
		if a.Span != b.Span {
			continue
		}
		t0, t1 := a.T, b.T
		if t1 < t0 {
			t0, t1 = t1, t0
		}
		for k := 1; k < 12; k++ {
			t := t0 + (t1-t0)*float64(k)/12
			p := s.trim.Spans[a.Span].Eval(t)
			q := s.project.Surface(s.patch, p.X, p.Y)
			if err := distPointSegment(q, a.Screen, b.Screen); err > maxErr {
				maxErr = err
			}
		}
	}
	return maxErr
}
