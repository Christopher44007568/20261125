package main

import (
	"fmt"
	"math"
)

var scalpRadii = V(0.47, 0.58, 0.45)

type Strand struct {
	ID       int
	Controls []Vec3
	Width    float64
}

func (s Strand) Root() Vec3 {
	return s.Controls[0]
}

func (s Strand) Tip() Vec3 {
	return s.Controls[len(s.Controls)-1]
}

type Groom struct {
	Name    string
	Pose    Pose
	Strands []Strand
	Levels  int
}

type HairStyle struct {
	Count      int
	Levels     int
	Length     float64
	LengthVar  float64
	CurlAmp    float64
	Turns      float64
	Bulge      float64
	TipFlip    float64
	Width      float64
	PhaseShift float64
}

const goldenAngle = 2.399963229728653

func buildGroom(name string, pose Pose, style HairStyle) (Groom, error) {
	if style.Count < 2 {
		return Groom{}, fmt.Errorf("groom %q needs at least two strands", name)
	}
	if style.Levels < 3 {
		return Groom{}, fmt.Errorf("groom %q needs at least three control points", name)
	}
	strands := make([]Strand, 0, style.Count)
	roots := scalpRoots(style.Count)
	for i, root := range roots {
		strands = append(strands, makeStrand(i, root, style))
	}
	return Groom{Name: name, Pose: pose, Strands: strands, Levels: style.Levels}, nil
}

func scalpRoots(count int) []Vec3 {
	roots := make([]Vec3, 0, count)
	for i := 0; len(roots) < count && i < count*16; i++ {
		u := math.Mod((float64(i)+0.5)*0.6180339887498949, 1)
		y := 1.0 - 1.20*u
		if y < -0.035 {
			continue
		}
		r := math.Sqrt(math.Max(0, 1-y*y))
		az := float64(i)*goldenAngle + 0.41
		dir := V(r*math.Cos(az), y, r*math.Sin(az))
		if dir.Z > 0.28 && dir.Y < 0.52 {
			continue
		}
		roots = append(roots, V(
			dir.X*scalpRadii.X,
			dir.Y*scalpRadii.Y,
			dir.Z*scalpRadii.Z,
		))
	}
	for len(roots) < count {
		i := len(roots)
		az := float64(i) * goldenAngle
		u := float64(i) / float64(count)
		dir := V(math.Cos(az)*0.78, 0.44+0.12*u, math.Sin(az)*0.78).Norm()
		roots = append(roots, V(dir.X*scalpRadii.X, dir.Y*scalpRadii.Y, dir.Z*scalpRadii.Z))
	}
	return roots
}

func scalpNormal(root Vec3) Vec3 {
	return V(
		root.X/(scalpRadii.X*scalpRadii.X),
		root.Y/(scalpRadii.Y*scalpRadii.Y),
		root.Z/(scalpRadii.Z*scalpRadii.Z),
	).Norm()
}

func makeStrand(index int, root Vec3, style HairStyle) Strand {
	normal := scalpNormal(root)
	phase := math.Atan2(root.Z, root.X) + style.PhaseShift
	outward := V(normal.X, 0, normal.Z).Norm()
	if outward.Len2() < 1e-8 {
		outward = V(math.Cos(phase), 0, math.Sin(phase))
	}
	side := V(outward.Z, 0, -outward.X)
	variation := math.Sin(float64(index)*1.713 + 0.37)
	length := style.Length * (1 + style.LengthVar*variation)
	width := style.Width * (0.88 + 0.18*(0.5+0.5*math.Sin(float64(index)*2.17)))
	controls := make([]Vec3, style.Levels)
	for level := 0; level < style.Levels; level++ {
		s := float64(level) / float64(style.Levels-1)
		drop := math.Pow(s, 1.18) * length
		envelope := s
		wave := style.CurlAmp * envelope
		angle := 2*math.Pi*style.Turns*s + phase
		p := root.
			Add(V(0, -drop, 0)).
			Add(outward.Mul(style.Bulge * math.Sin(math.Pi*s))).
			Add(normal.Mul(style.Bulge * 0.28 * math.Sin(math.Pi*s))).
			Add(side.Mul(wave * math.Sin(angle))).
			Add(outward.Mul(wave * 0.62 * math.Cos(angle)))
		flip := style.TipFlip * math.Pow(s, 3)
		p = p.Add(normal.Mul(flip))
		controls[level] = p
	}
	return Strand{ID: index, Controls: controls, Width: width}
}

func (g Groom) Validate() error {
	if len(g.Strands) < 2 {
		return fmt.Errorf("%s: not enough strands", g.Name)
	}
	if g.Levels < 3 {
		return fmt.Errorf("%s: bad level count", g.Name)
	}
	for i, strand := range g.Strands {
		if len(strand.Controls) != g.Levels {
			return fmt.Errorf("%s: strand %d has %d controls, want %d", g.Name, i, len(strand.Controls), g.Levels)
		}
		for level, p := range strand.Controls {
			if math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsNaN(p.Z) ||
				math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) || math.IsInf(p.Z, 0) {
				return fmt.Errorf("%s: non-finite point at strand %d level %d", g.Name, i, level)
			}
		}
	}
	return nil
}

func maxControlRadius(g Groom) float64 {
	best := 0.0
	for _, strand := range g.Strands {
		root := strand.Root()
		for _, p := range strand.Controls {
			best = math.Max(best, p.Dist(root))
		}
	}
	return best
}

func maxSegmentSpan(g Groom) float64 {
	best := 0.0
	for _, strand := range g.Strands {
		for i := 1; i < len(strand.Controls); i++ {
			best = math.Max(best, strand.Controls[i].Dist(strand.Controls[i-1]))
		}
	}
	return best
}
