package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Run(outDir string) (*Report, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	patch := DemoPatch()
	report := &Report{
		StandardLibraryOnly: true,
		CoordinateSystem:    "UV/XY demo domain [0,100]^2; OBJ Z from a quadratic NURBS surface",
	}
	for _, scene := range DemoScenes() {
		caseReport, err := runCase(scene, outDir, patch)
		if err != nil {
			return nil, fmt.Errorf("case %s: %w", scene.ID, err)
		}
		report.Cases = append(report.Cases, caseReport)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(outDir, "trim_report.json"), data, 0o644); err != nil {
		return nil, err
	}
	return report, nil
}

func runCase(scene SceneSpec, outDir string, patch NURBSSurface) (CaseReport, error) {
	cr := CaseReport{
		ID:             scene.ID,
		Title:          scene.Title,
		Description:    scene.Description,
		TrimCurveCount: len(scene.Curves),
	}

	loops, err := ExtractLoops(scene.Curves, attachEps)
	if err != nil {
		return cr, fmt.Errorf("extract loops: %w", err)
	}
	cr.ExtractedLoops = len(loops)
	for _, loop := range loops {
		cr.LoopVertexCounts = append(cr.LoopVertexCounts, len(loop.Points))
		cr.LoopCurveIDs = append(cr.LoopCurveIDs, loop.CurveIDs)
		cr.CrossingCount += len(loop.Crossings)
	}

	domain, err := BuildTrimDomain(loops)
	if err != nil {
		return cr, fmt.Errorf("classify trim domain: %w", err)
	}
	cr.Repaired = domain.Repaired
	cr.RepairDescription = domain.RepairDescription
	cr.OuterVertexCount = len(domain.Outer)
	for _, hole := range domain.Holes {
		cr.HoleVertexCounts = append(cr.HoleVertexCounts, len(hole))
	}

	correct, err := TriangulateDomain(domain)
	if err != nil {
		return cr, fmt.Errorf("trim-aware ear clipping: %w", err)
	}
	badCorrect := ValidateTriangles(correct, domain)
	cr.CorrectTriangles = len(correct.Triangles)
	cr.CorrectInvalid = len(badCorrect)
	cr.CorrectInvalidTris = badCorrect

	naive, err := TriangulateSimple(domain.Outer)
	if err != nil {
		return cr, fmt.Errorf("outer-only ear clipping: %w", err)
	}
	badNaive := ValidateTriangles(naive, domain)
	cr.NaiveTriangles = len(naive.Triangles)
	cr.NaiveInvalid = len(badNaive)
	cr.NaiveInvalidTris = badNaive

	objPath := filepath.Join(outDir, scene.ID+".obj")
	if err := WriteTriangulatedOBJ(objPath, scene.ID, correct, patch); err != nil {
		return cr, fmt.Errorf("write OBJ: %w", err)
	}
	reloaded, err := ReadOBJ(objPath)
	if err != nil {
		return cr, fmt.Errorf("reread OBJ: %w", err)
	}
	cr.OBJPath = objPath
	cr.OBJVertices = len(reloaded.Vertices)
	cr.OBJFaces = len(reloaded.Triangles)
	cr.OBJBoundaryEdges = len(OBJBoundaryEdges(reloaded))
	reloadedMesh := objAsMesh2(reloaded)
	badReloaded := ValidateTriangles(reloadedMesh, domain)
	cr.OBJReloadInvalid = len(badReloaded)

	directPath := filepath.Join(outDir, scene.ID+"_direct_outer.png")
	trimmedPath := filepath.Join(outDir, scene.ID+"_trimmed.png")
	objPNGPath := filepath.Join(outDir, scene.ID+"_obj_reload.png")

	directImg := RenderOverlayPNG(
		scene.Title+" DIRECT OUTER",
		"OUTER-ONLY EAR CUT + HOLE MASK + OUT-OF-BOUNDS HIGHLIGHT",
		domain, naive, badNaive,
	)
	if err := WritePNG(directPath, directImg); err != nil {
		return cr, fmt.Errorf("write direct PNG: %w", err)
	}

	trimmedImg := RenderOverlayPNG(
		scene.Title+" TRIM AWARE",
		"EXTRACTED LOOPS + BRIDGED EAR CLIPPING",
		domain, correct, badCorrect,
	)
	if err := WritePNG(trimmedPath, trimmedImg); err != nil {
		return cr, fmt.Errorf("write trimmed PNG: %w", err)
	}

	objImg := RenderOBJOverlayPNG(
		reloaded,
		domain,
		badReloaded,
		scene.Title+" OBJ RELOAD",
		"PARSED OBJ MESH + HOLE MASK + RE-VALIDATION",
	)
	if err := WritePNG(objPNGPath, objImg); err != nil {
		return cr, fmt.Errorf("write OBJ reload PNG: %w", err)
	}

	cr.Images = ImageSet{DirectOuter: directPath, Trimmed: trimmedPath, OBJReload: objPNGPath}
	return cr, nil
}
