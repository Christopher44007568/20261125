package scenes

import (
	"math"

	"ruledintersect/geom"
)

type Kind string

const (
	Tangent      Kind = "tangent"
	NearParallel Kind = "near_parallel"
	Crossing     Kind = "crossing"
)

type Scene struct {
	Name        string
	Kind        Kind
	Description string
	A, B        *geom.RuledSurface
}

func All() []Scene {
	return []Scene{
		tangentScene(),
		nearParallelScene(),
		crossingScene(),
	}
}

func ByKind(kind Kind) (Scene, bool) {
	for _, s := range All() {
		if s.Kind == kind {
			return s, true
		}
	}
	return Scene{}, false
}

func tangentScene() Scene {
	a := strip("plane_z035",
		func(u float64) geom.Vec3 { return geom.V(0.55*u+0.2, 0.1+0.72*u, 0.35) },
		func(u float64) geom.Vec3 { return geom.V(0.0, 0.68, 0.0) })
	b := strip("tangent_parabolic_strip",
		func(u float64) geom.Vec3 {
			return geom.V(0.55*u+0.2, 0.1+0.72*u, 0.35+0.24*(u-0.5)*(u-0.5))
		},
		func(u float64) geom.Vec3 { return geom.V(0.0, 0.68, 0.0) })
	a.Nu, a.Nv = 100, 28
	b.Nu, b.Nv = 100, 28
	return Scene{
		Name:        "tangent",
		Kind:        Tangent,
		Description: "Parabolic ruled strip touches a plane along its u=0.5 ruling.",
		A:           a,
		B:           b,
	}
}

func nearParallelScene() Scene {
	a := strip("tilted_plane_low",
		func(u float64) geom.Vec3 { return geom.V(0.04+0.92*u, 0.06, 0.3+0.12*u) },
		func(u float64) geom.Vec3 { return geom.V(0.0, 0.82, 0.02) })
	b := strip("tilted_plane_high",
		func(u float64) geom.Vec3 { return geom.V(0.04+0.92*u, 0.06, 0.305+0.122*u) },
		func(u float64) geom.Vec3 { return geom.V(0.0, 0.82, -0.018) })
	a.Nu, a.Nv = 96, 24
	b.Nu, b.Nv = 96, 24
	return Scene{
		Name:        "near_parallel",
		Kind:        NearParallel,
		Description: "Two shallow-angle planar strips intersect in a long, stable line.",
		A:           a,
		B:           b,
	}
}

func crossingScene() Scene {
	a := strip("wavy_rule_strip",
		func(u float64) geom.Vec3 {
			return geom.V(0.04+0.92*u, 0.08, 0.34+0.09*math.Cos(2*math.Pi*u))
		},
		func(u float64) geom.Vec3 {
			return geom.V(0.0, 0.84, 0.23*math.Sin(2*math.Pi*u))
		})
	b := strip("crossing_curved_strip",
		func(u float64) geom.Vec3 {
			return geom.V(0.16+0.68*u, 0.12+0.06*math.Cos(2*math.Pi*u), 0.2+0.66*u)
		},
		func(u float64) geom.Vec3 {
			return geom.V(0.04*math.Sin(2*math.Pi*u), 0.72, -0.62)
		})
	a.Nu, a.Nv = 110, 38
	b.Nu, b.Nv = 92, 34
	return Scene{
		Name:        "crossing",
		Kind:        Crossing,
		Description: "A curved transverse strip passes through a wavy ruled strip.",
		A:           a,
		B:           b,
	}
}

func strip(name string, base, dir geom.CurveFunc) *geom.RuledSurface {
	s := geom.NewRuledSurface(name, base, dir)
	s.Nu, s.Nv = 80, 30
	return s
}
