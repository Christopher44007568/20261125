package geom

import "math"

type Patch struct {
	Points  [4][4]Vec3
	Weights [4][4]float64
}

func bernstein3(t float64) [4]float64 {
	u := 1 - t
	return [4]float64{
		u * u * u,
		3 * u * u * t,
		3 * u * t * t,
		t * t * t,
	}
}

func (p Patch) Eval(u, v float64) Vec3 {
	bu := bernstein3(u)
	bv := bernstein3(v)
	var sum Vec3
	var den float64
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			bw := bu[i] * bv[j] * p.Weights[j][i]
			sum = sum.Add(p.Points[j][i].Mul(bw))
			den += bw
		}
	}
	if math.Abs(den) < 1e-15 {
		return Vec3{}
	}
	return sum.Mul(1 / den)
}

func (p Patch) Normal(u, v float64) Vec3 {
	const h = 1e-5
	u0 := Clamp(u-h, 0, 1)
	u1 := Clamp(u+h, 0, 1)
	v0 := Clamp(v-h, 0, 1)
	v1 := Clamp(v+h, 0, 1)
	du := p.Eval(u1, v).Sub(p.Eval(u0, v))
	dv := p.Eval(u, v1).Sub(p.Eval(u, v0))
	n := du.Cross(dv).Normalize()
	if n.Z < 0 {
		n = n.Mul(-1)
	}
	return n
}
