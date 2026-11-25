package geom

import "math"

type Projector struct {
	Yaw, Pitch, Roll float64
	Scale            float64
	Center           Vec2
	ZScale           float64
}

func (p Projector) Project(v Vec3) Vec2 {
	x := v.X
	y := v.Y
	z := v.Z * p.ZScale
	x, y = x*math.Cos(p.Yaw)-y*math.Sin(p.Yaw), x*math.Sin(p.Yaw)+y*math.Cos(p.Yaw)
	y, z = y*math.Cos(p.Pitch)-z*math.Sin(p.Pitch), y*math.Sin(p.Pitch)+z*math.Cos(p.Pitch)
	x, y = x*math.Cos(p.Roll)-y*math.Sin(p.Roll), x*math.Sin(p.Roll)+y*math.Cos(p.Roll)
	return Vec2{
		p.Center.X + x*p.Scale,
		p.Center.Y - y*p.Scale,
	}
}

func (p Projector) Surface(s Patch, u, v float64) Vec2 {
	return p.Project(s.Eval(u, v))
}
