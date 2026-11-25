package mesh

import (
	"testing"

	"adaptivebezier/internal/geom"
)

func testScene() (geom.Patch, geom.TrimLoop, geom.Projector) {
	project := geom.Projector{
		Yaw: 0.20, Pitch: 0.18, Roll: -0.04,
		Scale: 154, Center: geom.Vec2{X: 250, Y: 176}, ZScale: 0.56,
	}
	return geom.DemoPatch(), geom.DemoTrim(), project
}

func TestAdaptiveLevelsConform(t *testing.T) {
	patch, trim, project := testScene()
	tolerances := []float64{0.50, 0.15, 0.05}
	lastTris := 0
	lastBoundary := 0
	for _, tol := range tolerances {
		m, err := Build(patch, trim, project, tol)
		if err != nil {
			t.Fatalf("tol %.2f: %v", tol, err)
		}
		t.Logf("tol=%.2f tris=%d verts=%d boundary=%d chord=%.4f contour=%.4f crack=%.6f",
			tol, len(m.Triangles), len(m.Vertices), len(m.Boundary),
			m.MaxChord, m.Contour, m.Crack)
		if m.MaxChord > tol {
			t.Errorf("tol %.2f: chord %.6f exceeds tolerance", tol, m.MaxChord)
		}
		if !m.Watertight || m.CrackEdges != 0 || m.Crack > 0 {
			t.Errorf("tol %.2f: open mesh, crack edges=%d length=%.6f", tol, m.CrackEdges, m.Crack)
		}
		if len(m.Triangles) <= lastTris {
			t.Errorf("tolerance reduction did not densify mesh: %d -> %d", lastTris, len(m.Triangles))
		}
		if len(m.Boundary) <= lastBoundary {
			t.Errorf("silhouette did not refine: %d -> %d", lastBoundary, len(m.Boundary))
		}
		lastTris = len(m.Triangles)
		lastBoundary = len(m.Boundary)
	}
}

func TestClippedBoundaryIsMixed(t *testing.T) {
	patch, trim, project := testScene()
	for _, tol := range []float64{0.50, 0.15, 0.05} {
		m, err := Build(patch, trim, project, tol)
		if err != nil {
			t.Fatal(err)
		}
		if m.Stats.ClippedTriangles == 0 || m.Stats.ClippedQuads == 0 {
			t.Fatalf("tol %.2f: expected mixed clipped cells, got tri=%d quad=%d",
				tol, m.Stats.ClippedTriangles, m.Stats.ClippedQuads)
		}
	}
}
