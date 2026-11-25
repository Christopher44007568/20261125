package main

import "math"

const (
	rachisLength = 1040
	barbCount    = 56
	windStrength = 92
)

type Barb struct {
	Side   float64
	T      float64
	Base   Vec
	Ctrl   Vec
	End    Vec
	Width  float64
	Sweep  float64
	HalfW  float64
	Anchor int
}

type Feather struct {
	Base    Vec
	Tip     Vec
	C1      Vec
	C2      Vec
	Barbs   []Barb
	Anchors []Vec
	Wind    Vec
}

func rachisPoint(t float64) Vec {
	u := 1 - t
	return Vec{0, -520}.Scale(u * u * u).
		Add(Vec{46, -180}.Scale(3 * u * u * t)).
		Add(Vec{36, 200}.Scale(3 * u * t * t)).
		Add(Vec{16, 520}.Scale(t * t * t))
}

func rachisTangent(t float64) Vec {
	u := 1 - t
	return Vec{46, 340}.Scale(3 * u * u).
		Add(Vec{-10, 380}.Scale(6 * u * t)).
		Add(Vec{-20, 320}.Scale(3 * t * t))
}

func rachisWidth(t float64) float64 {
	return lerp(15, 2.4, math.Pow(t, 0.75))
}

func vaneHalfWidth(t float64, side float64) float64 {
	env := math.Pow(math.Sin(math.Pi*math.Pow(t, 0.82)), 0.72)
	asym := 1.0
	if side < 0 {
		asym = 1.18
	} else {
		asym = 0.84
	}
	return 152 * env * asym * (1 - 0.1*t)
}

func barbSweep(t float64) float64 {
	return lerp(0.16, 0.98, smooth(t))
}

func buildFeather() Feather {
	f := Feather{
		Base: rachisPoint(0),
		Tip:  rachisPoint(1),
		C1:   Vec{46, -180},
		C2:   Vec{36, 200},
		Wind: Vec{1, -0.14}.Norm(),
	}
	for i := 0; i < barbCount; i++ {
		t := lerp(0.035, 0.965, float64(i)/float64(barbCount-1))
		base := rachisPoint(t)
		tan := rachisTangent(t).Norm()
		sweep := barbSweep(t)
		gust := 0.72 + 0.28*math.Sin(t*9.0)
		windDisp := f.Wind.Scale(windStrength * gust * (0.3 + 0.7*t))
		for _, side := range []float64{1, -1} {
			dir := tan.Rot(-math.Pi / 2 * side).Rot(side * sweep)
			half := vaneHalfWidth(t, side)
			length := half / math.Cos(sweep)
			end := base.Add(dir.Scale(length)).Add(windDisp)
			ctrl := base.Add(dir.Scale(length * 0.42)).Add(windDisp.Scale(0.46))
			b := Barb{
				Side:   side,
				T:      t,
				Base:   base,
				Ctrl:   ctrl,
				End:    end,
				Width:  lerp(3.7, 1.5, t),
				Sweep:  sweep,
				HalfW:  half,
				Anchor: i,
			}
			f.Barbs = append(f.Barbs, b)
		}
		f.Anchors = append(f.Anchors, base)
	}
	return f
}

func barbPoint(b Barb, u float64) Vec {
	return quadPoint(b.Base, b.Ctrl, b.End, u)
}

func barbTangent(b Barb, u float64) Vec {
	return quadTangent(b.Base, b.Ctrl, b.End, u).Norm()
}

func barbWeight(b Barb, u float64) float64 {
	taper := lerp(1, 0.16, smooth(u))
	edge := 1.0
	if u < 0.06 {
		edge = u / 0.06
	} else if u > 0.92 {
		edge = (1 - u) / 0.08
	}
	return clamp01(taper * edge)
}
