package main

import "math"

type FurStyle struct {
	Root RGB
	Mid  RGB
	Tip  RGB
	Len  float64
}

func furStyle(idx int) FurStyle {
	switch idx {
	case 1, 2:
		return FurStyle{
			Root: C(0.080, 0.040, 0.022),
			Mid:  C(0.40, 0.215, 0.086),
			Tip:  C(1.32, 0.91, 0.49),
			Len:  0.15,
		}
	case 3:
		return FurStyle{
			Root: C(0.095, 0.052, 0.029),
			Mid:  C(0.48, 0.285, 0.125),
			Tip:  C(1.38, 1.01, 0.61),
			Len:  0.085,
		}
	default:
		return FurStyle{
			Root: C(0.075, 0.038, 0.020),
			Mid:  C(0.40, 0.218, 0.090),
			Tip:  C(1.36, 0.95, 0.54),
			Len:  0.205,
		}
	}
}

var (
	keyDir  = V(-0.48, 0.74, 0.48).Norm()
	fillDir = V(0.72, 0.16, 0.48).Norm()
	rimDir  = V(-0.18, 0.52, -0.84).Norm()
)

func groomFlow(idx int, n Vec3) Vec3 {
	var base Vec3
	switch idx {
	case 1:
		base = V(-0.72, 0.60, -0.05)
	case 2:
		base = V(0.72, 0.60, -0.05)
	case 3:
		base = V(0, -0.42, -0.90)
	default:
		base = V(-0.06, -0.96, 0.16)
	}
	flow := base.Sub(n.Mul(base.Dot(n)))
	if flow.Len() < 0.08 {
		axis := V(1, 0, 0)
		if math.Abs(n.X) > 0.8 {
			axis = V(0, 0, 1)
		}
		flow = axis.Sub(n.Mul(axis.Dot(n)))
	}
	return flow.Norm()
}

func furLength(idx int, p Vec3) float64 {
	st := furStyle(idx)
	n := fbm3(p.Add(V(float64(idx)*7.3, 0, 0)), 7.0, 3)
	fine := fbm3(p.Add(V(0.2, 9.1, 0)), 22.0, 2)
	scale := 0.66 + 0.52*n + 0.12*(fine-0.5)
	if idx == 3 {
		scale *= 0.88 + 0.22*sat((p.Z-0.6)*2)
	}
	return st.Len * clamp(scale, 0.48, 1.12)
}

func surfaceLight(n, view Vec3) (float64, float64, float64) {
	wrap := sat((n.Dot(keyDir) + 0.38) / 1.38)
	key := math.Pow(wrap, 1.25) * 0.98
	fill := sat((n.Dot(fillDir)+0.35)/1.35) * 0.30
	back := math.Pow(sat(n.Dot(rimDir)), 2.2) * 0.26
	grazing := math.Pow(1-sat(n.Dot(view)), 2.0) * 0.15
	return key, fill, back + grazing
}

func furColor(idx int, h float64, p, n, view Vec3) RGB {
	st := furStyle(idx)
	h = sat(h)
	col := st.Root.Lerp(st.Mid, smoothstep(0.02, 0.46, h))
	col = col.Lerp(st.Tip, smoothstep(0.70, 1.0, h))

	key, fill, rim := surfaceLight(n, view)
	rootAO := 0.30 + 0.70*smoothstep(0.0, 0.38, h)
	occ := 0.72 + 0.28*fbm3(p.Add(V(5.2, 0.4, 1.7)), 18, 2)
	ambient := 0.20 + 0.15*sat(n.Y*0.5+0.5)
	col = col.Mul((ambient + key*1.20 + fill*0.55) * rootAO * occ)
	col = col.Add(st.Tip.Mul(rim * (0.08 + 0.72*h*h*h)))
	col = col.Add(st.Tip.Mul(math.Pow(h, 5.0) * 0.20))
	backRim := math.Pow(sat(n.Dot(rimDir)), 3.0)*0.66 + math.Pow(1-sat(n.Dot(view)), 2.0)*0.30
	col = col.Add(st.Tip.Mul(backRim * smoothstep(0.72, 1.0, h) * 0.62))
	return col
}

func shellLayerAlpha(idx int, h float64, p, n, flow Vec3) float64 {
	seed := V(float64(idx)*11.7, float64(idx)*3.1, 0)
	clumpP := V(p.X*6.2, p.Y*2.6, p.Z*6.2).Add(seed)
	fineP := V(p.X*57, p.Y*9.5, p.Z*57).Add(seed.Mul(1.7))
	clumpN := fbm3(clumpP, 1.0, 3)
	fiber := fbm3(fineP, 1.0, 2)
	mixField := 0.30*clumpN + 0.70*fiber
	cover := smoothstep(0.34, 0.64, mixField)
	cover = smoothstep(0.12, 0.92, cover)

	root := 0.020 + 0.30*cover
	tip := 0.003 + 0.090*cover
	a := mix(root, tip, smoothstep(0.24, 1.0, h))
	a *= 0.84 + 0.16*math.Sin(float64(idx)+p.X*4.1)
	return sat(a)
}
