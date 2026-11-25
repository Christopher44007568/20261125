package main

import "math"

type LevelMetrics struct {
	TriangleCount                  int     `json:"triangle_count"`
	VertexCount                    int     `json:"vertex_count"`
	MaxChordHeightMM               float64 `json:"max_chord_height_mm"`
	MaxProfileDeviationMM          float64 `json:"max_profile_deviation_mm"`
	MaxBoundaryContourDeviationMM  float64 `json:"max_boundary_contour_deviation_mm"`
	MaxScreenProfileDeviationPixel float64 `json:"max_screen_profile_deviation_px"`
	WorstU                         float64 `json:"worst_u"`
	WorstV                         float64 `json:"worst_v"`
	SamplePoints                   int     `json:"sample_points"`
}

type LevelResult struct {
	ID                          string       `json:"id"`
	Label                       string       `json:"label"`
	File                        string       `json:"file"`
	ChordHeightBudgetMM         float64      `json:"chord_height_budget_mm"`
	GridU                       int          `json:"grid_u"`
	GridV                       int          `json:"grid_v"`
	CameraDistanceMM            float64      `json:"camera_distance_mm"`
	MaxScreenProfileDeviationPx float64      `json:"max_screen_profile_deviation_px"`
	Metrics                     LevelMetrics `json:"metrics"`
}

type CrackMetrics struct {
	TransitionCount              int     `json:"transition_count"`
	LODBoundaryLengthMM          float64 `json:"lod_boundary_length_mm"`
	NaiveOpenEdgeLengthMM        float64 `json:"naive_open_edge_length_mm"`
	NaiveCrackCenterlineLengthMM float64 `json:"naive_crack_centerline_length_mm"`
	StitchedOpenEdgeLengthMM     float64 `json:"stitched_open_edge_length_mm"`
	CrossLODCrackLengthMM        float64 `json:"cross_lod_crack_length_mm"`
	UnmatchedStitchedEdgeCount   int     `json:"unmatched_stitched_edge_count"`
	NaiveMaxTransitionOpeningMM  float64 `json:"naive_max_transition_opening_mm"`
}

type Report struct {
	Unit                  string        `json:"unit"`
	ImageWidth            int           `json:"image_width"`
	ImageHeight           int           `json:"image_height"`
	Levels                []LevelResult `json:"levels"`
	TieredTriangleCount   int           `json:"tiered_triangle_count"`
	ZipperTriangleCount   int           `json:"zipper_triangle_count"`
	BoundaryHighlightFile string        `json:"boundary_highlight_file"`
	CrackMetrics          CrackMetrics  `json:"crack_metrics"`
	Outputs               []string      `json:"outputs"`
}

func meshIndex(u float64, count int) int {
	i := int(math.Floor(u * float64(count)))
	if i >= count {
		i = count - 1
	}
	if i < 0 {
		i = 0
	}
	return i
}

func cellVertices(mesh *Mesh, iu, iv, nu, nv int) (Vec3, Vec3, Vec3, Vec3) {
	nextIU := min(iu+1, nu)
	nextIV := min(iv+1, nv)
	a := mesh.Vertices[meshIndexID(iu, iv, nv)]
	b := mesh.Vertices[meshIndexID(iu, nextIV, nv)]
	c := mesh.Vertices[meshIndexID(nextIU, nextIV, nv)]
	d := mesh.Vertices[meshIndexID(nextIU, iv, nv)]
	return a, b, c, d
}

func meshIndexID(iu, iv, nv int) int32 {
	return int32(iu*(nv+1) + iv)
}

func distanceToCellTriangles(point, a, b, c, d Vec3) float64 {
	d1, _ := pointTriangleDistance(point, a, b, c)
	d2, _ := pointTriangleDistance(point, a, c, d)
	return math.Min(d1, d2)
}

func measureLevel(surface *Surface, mesh *Mesh, cfg LevelConfig, camera Camera) LevelResult {
	const sampleCount = 96
	metrics := LevelMetrics{
		TriangleCount: len(mesh.Tris),
		VertexCount:   len(mesh.Vertices),
	}

	for i := 0; i < sampleCount; i++ {
		for j := 0; j < sampleCount; j++ {
			u := (float64(i) + 0.5) / sampleCount
			v := (float64(j) + 0.5) / sampleCount
			point := surface.Point(u, v)
			iu := meshIndex(u, cfg.Nu)
			iv := meshIndex(v, cfg.Nv)
			a, b, c, d := cellVertices(mesh, iu, iv, cfg.Nu, cfg.Nv)
			dist3D := distanceToCellTriangles(point, a, b, c, d)
			if dist3D > metrics.MaxChordHeightMM {
				metrics.MaxChordHeightMM = dist3D
				metrics.WorstU = u
				metrics.WorstV = v
			}

			projectedSurface := camera.Project(point)
			pa := camera.Project(a)
			pb := camera.Project(b)
			pc := camera.Project(c)
			pd := camera.Project(d)
			d1, _ := pointTriangleDistance(projectedVec3(projectedSurface), projectedVec3(pa), projectedVec3(pb), projectedVec3(pc))
			d2, _ := pointTriangleDistance(projectedVec3(projectedSurface), projectedVec3(pa), projectedVec3(pc), projectedVec3(pd))
			screenDist := d1
			if d2 < d1 {
				screenDist = d2
			}
			if screenDist > metrics.MaxScreenProfileDeviationPixel {
				metrics.MaxScreenProfileDeviationPixel = screenDist
			}

			profileMM := screenDist * projectedSurface.Z / camera.Focal
			if profileMM > metrics.MaxProfileDeviationMM {
				metrics.MaxProfileDeviationMM = profileMM
			}
			metrics.SamplePoints++
		}
	}

	const boundarySamples = 1024
	for i := 0; i < boundarySamples; i++ {
		t := (float64(i) + 0.5) / boundarySamples
		boundary := [][2]float64{{0, t}, {1, t}, {t, 0}, {t, 1}}
		for _, uv := range boundary {
			point := surface.Point(uv[0], uv[1])
			iu := meshIndex(uv[0], cfg.Nu)
			iv := meshIndex(uv[1], cfg.Nv)
			a, b, c, d := cellVertices(mesh, iu, iv, cfg.Nu, cfg.Nv)
			dist3D := distanceToCellTriangles(point, a, b, c, d)
			metrics.MaxBoundaryContourDeviationMM = math.Max(metrics.MaxBoundaryContourDeviationMM, dist3D)
		}
	}

	return LevelResult{
		ID:                          cfg.ID,
		Label:                       cfg.Label,
		File:                        cfg.File,
		ChordHeightBudgetMM:         cfg.Budget,
		GridU:                       cfg.Nu,
		GridV:                       cfg.Nv,
		CameraDistanceMM:            camera.Distance,
		MaxScreenProfileDeviationPx: metrics.MaxScreenProfileDeviationPixel,
		Metrics:                     metrics,
	}
}
