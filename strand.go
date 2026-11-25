package main

import (
	"math"
	"sort"
)

type StrandSpec struct {
	Prim       int
	Root       Vec3
	Normal     Vec3
	Flow       Vec3
	Length     float64
	Depth      float64
	Phase      float64
	Curl       float64
	Clump      float64
	RootJitter float64
}

func buildStrands(scene *Scene, cam Camera) []StrandSpec {
	counts := []int{5600, 760, 760, 980}
	specs := make([]StrandSpec, 0, 8100)
	const golden = 2.399963229728653
	for idx, count := range counts {
		if idx >= len(scene.Prims) {
			break
		}
		prim := scene.Prims[idx]
		for j := 0; j < count; j++ {
			u := rand013(j*3+1, idx*97+5, 17)
			v := rand013(j*3+2, idx*97+5, 31)
			w := rand013(j*3+3, idx*97+5, 47)
			y := 1 - 2*(float64(j)+0.5)/float64(count)
			y = clamp(y+(v-0.5)*0.012, -0.999, 0.999)
			r := math.Sqrt(math.Max(0, 1-y*y))
			phi := golden*float64(j) + float64(idx)*1.77 + (u-0.5)*0.025
			dl := V(r*math.Cos(phi), y, r*math.Sin(phi))
			local := dl.MulV(prim.Radii)
			root := prim.Center.Add(prim.ToWorld(local))

			nl := V(dl.X/prim.Radii.X, dl.Y/prim.Radii.Y, dl.Z/prim.Radii.Z).Norm()
			normal := prim.ToWorld(nl).Norm()
			flow := groomFlow(idx, normal)
			length := furLength(idx, root) * (0.78 + 0.40*w)
			if w > 0.975 {
				length *= 1.22
			}
			rel := root.Sub(cam.Pos)
			specs = append(specs, StrandSpec{
				Prim: idx, Root: root, Normal: normal, Flow: flow,
				Length: length, Depth: rel.Dot(cam.Fwd),
				Phase: u * 2 * math.Pi, Curl: (v - 0.5) * 1.7,
				Clump:      fbm3(root.Add(V(float64(idx)*3.3, 0, 0)), 5.8, 3),
				RootJitter: w,
			})
		}
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Depth > specs[j].Depth })
	return specs
}

func renderStrand(scene *Scene, cam Camera, w, h int, widthScale float64) *Canvas {
	out := renderUndercoat(scene, cam, w, h)
	specs := buildStrands(scene, cam)
	for _, s := range specs {
		drawStrand(out, cam, s, widthScale)
	}
	return out
}

func strandView(cam Camera, p Vec3) Vec3 { return cam.Pos.Sub(p).Norm() }

func strandPoint(s StrandSpec, t float64, cam Camera) (Vec3, RGB) {
	n := s.Normal
	flow := s.Flow
	side := n.Cross(flow).Norm()
	clumpBend := (s.Clump - 0.5) * 0.42
	rootDir := n.Mul(0.94).Add(flow.Mul(0.24 + clumpBend)).Add(side.Mul(s.Curl * 0.055)).Norm()
	tipDir := n.Mul(0.16).Add(flow.Mul(0.93)).Add(side.Mul(s.Curl * 0.14)).Norm()
	tipDir = tipDir.Add(V(0, -0.10, 0)).Norm()
	smoothT := t * t * (3 - 2*t)
	dir := rootDir.Lerp(tipDir, smoothT).Norm()

	p := s.Root.Add(dir.Mul(s.Length * t))
	wave := 0.022*math.Sin(t*3.0+s.Phase) + 0.010*math.Sin(t*8.2+s.Phase*1.7)
	p = p.Add(side.Mul(s.Length * wave * t))
	p = p.Add(n.Mul(s.Length * 0.018 * math.Sin(math.Pi*t)))

	view := strandView(cam, p)
	h := sat(t * (0.94 + 0.08*s.RootJitter))
	col := furColor(s.Prim, h, p, n, view)
	tip := smoothstep(0.68, 1.0, t)
	back := sat(n.Dot(rimDir))
	col = col.Add(furStyle(s.Prim).Tip.Mul(tip * (0.22 + 0.50*back)))
	col = col.Mul(1.24)
	if t < 0.10 {
		col = col.Mul(0.70)
	}
	return p, col
}

func drawStrand(out *Canvas, cam Camera, s StrandSpec, widthScale float64) {
	const steps = 11
	var prev Vec3
	var prevCol RGB
	var prevA float64
	for i := 0; i <= steps; i++ {
		t := float64(i) / steps
		p, col := strandPoint(s, t, cam)
		a := mix(0.50, 0.34, smoothstep(0.0, 1.0, t))
		a *= 0.72 + 0.28*s.Clump
		if i == 0 {
			prev, prevCol, prevA = p, col, a
			continue
		}
		width := mix(1.62, 0.72, t) * widthScale
		drawLineAA(out, cam, prev, p, prevCol, col, prevA, a, width)
		prev, prevCol, prevA = p, col, a
	}
}

func drawLineAA(out *Canvas, cam Camera, a, b Vec3, ca, cb RGB, aa, ab, width float64) {
	ax, ay, _, oka := cam.Project(a)
	bx, by, _, okb := cam.Project(b)
	if !oka || !okb {
		return
	}
	x0 := (ax*0.5 + 0.5) * float64(out.W)
	y0 := (1 - (ay*0.5 + 0.5)) * float64(out.H)
	x1 := (bx*0.5 + 0.5) * float64(out.W)
	y1 := (1 - (by*0.5 + 0.5)) * float64(out.H)
	dx, dy := x1-x0, y1-y0
	segLen2 := dx*dx + dy*dy
	rad := width*0.5 + 1.0
	minX := int(math.Floor(math.Min(x0, x1) - rad))
	maxX := int(math.Ceil(math.Max(x0, x1) + rad))
	minY := int(math.Floor(math.Min(y0, y1) - rad))
	maxY := int(math.Ceil(math.Max(y0, y1) + rad))
	if maxX < 0 || maxY < 0 || minX >= out.W || minY >= out.H {
		return
	}
	for py := max(0, minY); py <= min(out.H-1, maxY); py++ {
		for px := max(0, minX); px <= min(out.W-1, maxX); px++ {
			cx, cy := float64(px)+0.5, float64(py)+0.5
			tt := 0.0
			if segLen2 > 1e-9 {
				tt = sat(((cx-x0)*dx + (cy-y0)*dy) / segLen2)
			}
			qx := x0 + tt*dx
			qy := y0 + tt*dy
			dist := math.Hypot(cx-qx, cy-qy)
			cov := sat(width*0.5 + 0.65 - dist)
			if cov <= 0.001 {
				continue
			}
			col := ca.Lerp(cb, tt)
			alpha := mix(aa, ab, tt) * cov
			i := py*out.W + px
			out.Pix[i] = over(out.Pix[i], premul(col, alpha))
		}
	}
}
