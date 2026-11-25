package scene

import (
	"fmt"
	"math"

	"meshscope/internal/analysis"
	"meshscope/internal/mesh"
)

func expand(m *mesh.Mesh, res analysis.Result) MeshData {
	n := len(m.Tris)
	data := MeshData{
		Positions:  make([]float64, 0, n*9),
		Normals:    make([]float64, 0, n*9),
		Features:   make([]int, 0, n),
		Confidence: make([]float64, 0, n),
		Deviation:  make([]float64, 0, n),
	}
	for i, tri := range m.Tris {
		for k := 0; k < 3; k++ {
			p, nv := tri.P[k], tri.N[k]
			data.Positions = append(data.Positions, round(p.X, 4), round(p.Y, 4), round(p.Z, 4))
			data.Normals = append(data.Normals, round(nv.X, 4), round(nv.Y, 4), round(nv.Z, 4))
		}
		data.Features = append(data.Features, int(res.Faces[i].Kind))
		data.Confidence = append(data.Confidence, round(res.Faces[i].Confidence, 3))
		data.Deviation = append(data.Deviation, round(res.Faces[i].Deviation, 4))
	}
	return data
}

func boundaryLines(res analysis.Result) []BoundaryLine {
	out := make([]BoundaryLine, 0, len(res.Boundaries))
	for _, b := range res.Boundaries {
		if b.Length < 0.35 {
			continue
		}
		out = append(out, BoundaryLine{
			A: vec(b.A), B: vec(b.B), N: vec(b.N),
			KindA: featureKey(b.KindA), KindB: featureKey(b.KindB),
			Conf: round(b.Confidence, 3),
		})
	}
	return out
}

func callouts(res analysis.Result) []Callout {
	out := make([]Callout, 0, len(res.Callouts))
	for _, c := range res.Callouts {
		out = append(out, Callout{
			Position: vec(c.Position), N: vec(c.N), Label: c.Label,
			KindA: featureKey(c.KindA), KindB: featureKey(c.KindB),
			Confidence: round(c.Confidence, 2),
		})
	}
	return out
}

func probes(res analysis.Result) []Probe {
	out := make([]Probe, 0, len(res.Errors))
	for _, p := range res.Errors {
		out = append(out, Probe{
			Position: vec(p.Position), N: vec(p.N), Value: round(p.Value, 3),
			Kind: featureKey(p.Kind),
		})
	}
	return out
}

func features(res analysis.Result) []FeatureStat {
	keys := []mesh.Feature{mesh.Plane, mesh.Cylinder, mesh.Hole, mesh.Fillet}
	stats := make([]FeatureStat, 0, len(keys))
	for _, kind := range keys {
		st := FeatureStat{Key: featureKey(kind), Label: featureLabel(kind), Color: featureColor(kind)}
		var confArea, area float64
		seenRegion := map[int]bool{}
		for _, r := range res.Regions {
			if r.Kind != kind {
				continue
			}
			seenRegion[r.ID] = true
			confArea += r.Area * r.Confidence
			area += r.Area
		}
		for _, f := range res.Faces {
			if f.Kind == kind {
				st.Triangles++
			}
		}
		st.Regions = len(seenRegion)
		st.Area = round(area, 2)
		if area > 0 {
			st.Confidence = round(confArea/area, 3)
		} else {
			st.Confidence = 0
		}
		stats = append(stats, st)
	}
	return stats
}

func featureKey(f mesh.Feature) string {
	switch f {
	case mesh.Plane:
		return "plane"
	case mesh.Cylinder:
		return "cylinder"
	case mesh.Hole:
		return "hole"
	case mesh.Fillet:
		return "fillet"
	default:
		return "unknown"
	}
}

func featureLabel(f mesh.Feature) string {
	switch f {
	case mesh.Plane:
		return "平面 / PLANE"
	case mesh.Cylinder:
		return "圆柱 / CYLINDER"
	case mesh.Hole:
		return "孔 / HOLE"
	case mesh.Fillet:
		return "圆角 / FILLET"
	default:
		return "未定 / UNKNOWN"
	}
}

func featureColor(f mesh.Feature) string {
	switch f {
	case mesh.Plane:
		return "#55ccf5"
	case mesh.Cylinder:
		return "#f4b942"
	case mesh.Hole:
		return "#ff5d8f"
	case mesh.Fillet:
		return "#58e0b0"
	default:
		return "#a18cd1"
	}
}

func vec(p mesh.Vec3) [3]float64 { return [3]float64{round(p.X, 4), round(p.Y, 4), round(p.Z, 4)} }

func round(v float64, digits int) float64 {
	p := math.Pow10(digits)
	return math.Round(v*p) / p
}

func fmtConf(v float64) string { return fmt.Sprintf("%.2f", v) }
