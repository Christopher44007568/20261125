package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"adaptivebezier/internal/geom"
	"adaptivebezier/internal/mesh"
	"adaptivebezier/internal/render"
)

type LevelMetrics struct {
	TolerancePx          float64 `json:"tolerancePx"`
	Triangles            int     `json:"triangles"`
	Vertices             int     `json:"vertices"`
	TrimBoundarySegments int     `json:"trimBoundarySegments"`
	MaxChordPx           float64 `json:"maxChordPx"`
	MaxContourErrorPx    float64 `json:"maxContourErrorPx"`
	CrackLengthPx        float64 `json:"crackLengthPx"`
	CrackEdges           int     `json:"crackEdges"`
	Watertight           bool    `json:"watertight"`
	ClippedTriangles     int     `json:"clippedTriangleCells"`
	ClippedQuads         int     `json:"clippedQuadCells"`
	ClippedPolygons      int     `json:"clippedPolygonCells"`
	WirePpm              string  `json:"wirePpm"`
	SolidPpm             string  `json:"solidPpm"`
}

type Metrics struct {
	Patch                   string         `json:"patch"`
	Algorithm               string         `json:"algorithm"`
	CompositePpm            string         `json:"compositePpm"`
	Levels                  []LevelMetrics `json:"levels"`
	MaxChordPx              float64        `json:"maxChordPx"`
	MaxCrackLengthPx        float64        `json:"maxCrackLengthPx"`
	AllWatertight           bool           `json:"allWatertight"`
	AllChordWithinTolerance bool           `json:"allChordWithinTolerance"`
	SilhouetteRefined       bool           `json:"silhouetteRefined"`
	OutputDescription       string         `json:"outputDescription"`
}

func main() {
	outDir := flag.String("out", "artifacts", "output directory")
	flag.Parse()
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	patch := geom.DemoPatch()
	trim := geom.DemoTrim()
	project := geom.Projector{
		Yaw: 0.20, Pitch: 0.18, Roll: -0.04,
		Scale: 154, Center: geom.Vec2{X: 250, Y: 176}, ZScale: 0.56,
	}
	tolerances := []float64{0.50, 0.15, 0.05}
	meshes := make([]*mesh.Mesh, len(tolerances))
	metrics := Metrics{
		Patch:                   "bicubic rational Bezier patch, convex trimmed domain with linear and conic spans",
		Algorithm:               "adaptive boundary sampling + clipped-cell red-green subdivision with shared edge midpoints",
		CompositePpm:            "adaptive_mesh.ppm",
		AllWatertight:           true,
		AllChordWithinTolerance: true,
		SilhouetteRefined:       true,
		OutputDescription:       "top row: wireframe; bottom row: shaded solid; columns: 0.50, 0.15, 0.05 px",
	}
	composite := render.NewCanvas(1500, 720, render.RGB{R: 246, G: 243, B: 235})
	for i, tol := range tolerances {
		m, err := mesh.Build(patch, trim, project, tol)
		if err != nil {
			fatal(fmt.Errorf("tolerance %.2f: %w", tol, err))
		}
		meshes[i] = m
		wireName := fmt.Sprintf("lod%d_tol%.2f_wire.ppm", i, tol)
		solidName := fmt.Sprintf("lod%d_tol%.2f_solid.ppm", i, tol)
		wire := render.Panel(m, render.PanelSpec{Title: fmt.Sprintf("%.2f", tol), Wire: true})
		solid := render.Panel(m, render.PanelSpec{Title: fmt.Sprintf("%.2f", tol), Wire: false})
		if err := wire.WritePPM(filepath.Join(*outDir, wireName)); err != nil {
			fatal(err)
		}
		if err := solid.WritePPM(filepath.Join(*outDir, solidName)); err != nil {
			fatal(err)
		}
		composite.Blit(wire, i*500, 0)
		composite.Blit(solid, i*500, 360)
		lm := levelMetrics(m, tol, wireName, solidName)
		metrics.Levels = append(metrics.Levels, lm)
		if m.MaxChord > metrics.MaxChordPx {
			metrics.MaxChordPx = m.MaxChord
		}
		if m.Crack > metrics.MaxCrackLengthPx {
			metrics.MaxCrackLengthPx = m.Crack
		}
		if !m.Watertight || m.Crack > 0 {
			metrics.AllWatertight = false
		}
		if m.MaxChord > tol {
			metrics.AllChordWithinTolerance = false
		}
	}
	if !silhouetteChanges(meshes) {
		metrics.SilhouetteRefined = false
	}
	if err := composite.WritePPM(filepath.Join(*outDir, metrics.CompositePpm)); err != nil {
		fatal(err)
	}
	if err := writeJSON(filepath.Join(*outDir, "metrics.json"), metrics); err != nil {
		fatal(err)
	}
	if err := writeText(filepath.Join(*outDir, "metrics.txt"), metrics); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %s: %d levels, max chord %.6f px, max crack %.6f px\n",
		filepath.Join(*outDir, metrics.CompositePpm), len(metrics.Levels), metrics.MaxChordPx, metrics.MaxCrackLengthPx)
}

func levelMetrics(m *mesh.Mesh, tol float64, wire, solid string) LevelMetrics {
	return LevelMetrics{
		TolerancePx: tol, Triangles: len(m.Triangles), Vertices: len(m.Vertices),
		TrimBoundarySegments: len(m.TrimEdges), MaxChordPx: m.MaxChord,
		MaxContourErrorPx: m.Contour, CrackLengthPx: m.Crack, CrackEdges: m.CrackEdges,
		Watertight: m.Watertight, ClippedTriangles: m.Stats.ClippedTriangles,
		ClippedQuads: m.Stats.ClippedQuads, ClippedPolygons: m.Stats.ClippedPolygons,
		WirePpm: wire, SolidPpm: solid,
	}
}

func silhouetteChanges(meshes []*mesh.Mesh) bool {
	if len(meshes) < 2 {
		return false
	}
	for i := 1; i < len(meshes); i++ {
		if len(meshes[i].TrimEdges) <= len(meshes[i-1].TrimEdges) {
			return false
		}
		if meshes[i].Contour >= meshes[i-1].Contour {
			return false
		}
	}
	return true
}

func writeJSON(path string, metrics Metrics) error {
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeText(path string, metrics Metrics) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "adaptive rational Bezier tessellation\n\n")
	for _, l := range metrics.Levels {
		fmt.Fprintf(f, "tol=%.2f px  tris=%d  verts=%d  boundary=%d\n", l.TolerancePx, l.Triangles, l.Vertices, l.TrimBoundarySegments)
		fmt.Fprintf(f, "  max chord=%.6f px  contour=%.6f px  crack=%.6f px  crack edges=%d\n",
			l.MaxChordPx, l.MaxContourErrorPx, l.CrackLengthPx, l.CrackEdges)
		fmt.Fprintf(f, "  clipped triangles=%d  quads=%d  polygons=%d  watertight=%t\n",
			l.ClippedTriangles, l.ClippedQuads, l.ClippedPolygons, l.Watertight)
	}
	fmt.Fprintf(f, "\nall watertight=%t  all chord below tolerance=%t  silhouette refined=%t\n",
		metrics.AllWatertight, metrics.AllChordWithinTolerance, metrics.SilhouetteRefined)
	fmt.Fprintf(f, "max chord=%.6f px  max crack=%.6f px\n", metrics.MaxChordPx, metrics.MaxCrackLengthPx)
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
