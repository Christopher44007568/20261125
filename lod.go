package main

import "math"

const (
	nearMorphStart = 2.35
	nearMorphEnd   = 3.45
	midMorphStart  = 3.90
	midMorphEnd    = 5.05
)

type LODFrame struct {
	Strips    []Strip
	HairTris  int
	FromLevel int
	ToLevel   int
	Blend     float64
	State     string
}

func BuildLODFrame(model *HairModel, distance float64) LODFrame {
	switch {
	case distance < nearMorphStart:
		return LODFrame{
			Strips: model.LOD0, HairTris: model.Tris0,
			FromLevel: 0, ToLevel: 0, Blend: 0, State: "LOD0-stable",
		}
	case distance < nearMorphEnd:
		b := smoothstep(nearMorphStart, nearMorphEnd, distance)
		strips := morphStrips(model.LOD0, model.LOD1, b)
		return LODFrame{
			Strips: strips, HairTris: countStripTriangles(strips),
			FromLevel: 0, ToLevel: 1, Blend: b, State: "geomorph-0-to-1",
		}
	case distance < midMorphStart:
		return LODFrame{
			Strips: model.LOD1, HairTris: model.Tris1,
			FromLevel: 1, ToLevel: 1, Blend: 0, State: "LOD1-stable",
		}
	case distance < midMorphEnd:
		b := smoothstep(midMorphStart, midMorphEnd, distance)
		strips := morphStrips(model.LOD1, model.LOD2, b)
		return LODFrame{
			Strips: strips, HairTris: countStripTriangles(strips),
			FromLevel: 1, ToLevel: 2, Blend: b, State: "geomorph-1-to-2",
		}
	default:
		return LODFrame{
			Strips: model.LOD2, HairTris: model.Tris2,
			FromLevel: 2, ToLevel: 2, Blend: 0, State: "LOD2-stable",
		}
	}
}

func morphStrips(sources, parents []Strip, blend float64) []Strip {
	if blend >= 1 {
		return parents
	}
	out := make([]Strip, len(sources))
	for i, src := range sources {
		parent := parents[src.Parent]
		mp := make([]Vec3, len(src.Points))
		mw := make([]float64, len(src.Widths))
		for j, p := range src.Points {
			u := 0.0
			if len(src.Points) > 1 {
				u = float64(j) / float64(len(src.Points)-1)
			}
			targetPoint, targetWidth := sampleStrip(parent, u)
			mp[j] = vlerp(p, targetPoint, blend)
			mw[j] = lerpFloat(src.Widths[j], targetWidth, blend)
		}
		color := RGB{
			R: lerpFloat(src.Color.R, parent.Color.R, blend),
			G: lerpFloat(src.Color.G, parent.Color.G, blend),
			B: lerpFloat(src.Color.B, parent.Color.B, blend),
		}
		out[i] = Strip{
			ID: src.ID, Points: mp, Widths: mw, Color: color, Parent: src.Parent,
		}
	}
	return out
}

func sampleStrip(s Strip, u float64) (Vec3, float64) {
	if len(s.Points) == 1 {
		return s.Points[0], s.Widths[0]
	}
	u = clampFloat(u, 0, 1)
	f := u * float64(len(s.Points)-1)
	i := int(math.Floor(f))
	if i >= len(s.Points)-1 {
		return s.Points[len(s.Points)-1], s.Widths[len(s.Widths)-1]
	}
	t := f - float64(i)
	return vlerp(s.Points[i], s.Points[i+1], t),
		lerpFloat(s.Widths[i], s.Widths[i+1], t)
}
